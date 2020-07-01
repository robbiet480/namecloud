package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/billputer/go-namecheap"
	"github.com/cloudflare/cloudflare-go"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var cloudflareContactID string
var domainYears int
var enablePrivacy bool
var enableAutoRenew bool
var enableImportDNS bool

// transferCmd represents the transfer command
var transferCmd = &cobra.Command{
	Use:    "transfer",
	Short:  "transfer will complete most of the process of transferring a domain to Cloudflare Registrar.",
	Long:   `transfer will unlock your Namecheap domain, wait for you to provide a auth code and begin the transfer to Cloudflare Registrar.`,
	PreRun: preRun,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			log.Fatalln("You must provide the domain name to transfer as the last argument.")
		}

		namecheapDomain, domainErr := namecheapClient.DomainGetInfo(args[0])
		if domainErr != nil {
			log.Fatalln("Error getting Namecheap domain info for:", args[0], domainErr)
		}

		if namecheapDomain == nil {
			log.Fatalf("Didn't find domain name %s in your Namecheap account!\n", args[0])
		}

		if namecheapDomain.IsExpired {
			log.Fatalln("You can't transfer an expired domain!")
		}

		daysSince := time.Since(namecheapDomain.Created).Hours() / 24
		if daysSince <= 60 {
			log.Fatalf("Transfer can not begin until at least 60 days since initial registration. It has only been %f days\n", daysSince)
		}

		cloudflareZones, cfZonesErr := cloudflareClient.ListZonesContext(context.TODO(), cloudflare.WithZoneFilters(args[0], cloudflareAccount.ID, ""), cloudflare.WithPagination(cloudflare.PaginationOptions{
			PerPage: 50,
		}))
		if cfZonesErr != nil {
			log.Fatalln("Error getting all zones", cfZonesErr)
		}

		if len(cloudflareZones.Result) == 0 {
			log.Fatalf("Didn't find Cloudflare zone for %s\n", args[0])
		}

		cloudflareZone := cloudflareZones.Result[0]

		eppCodeUrl := fmt.Sprintf("https://ap.www.namecheap.com/domains/dcp/share/%s/rights", namecheapDomain.Name)
		reader := bufio.NewReader(os.Stdin)
		log.Infof("We need the EPP/Auth code for %s. You can get this at the bottom of %s. It will be emailed to you within a few minutes. You must then enter it below.", namecheapDomain.Name, eppCodeUrl)
		fmt.Printf("Enter the EPP/Auth Code for %s to continue: ", namecheapDomain.Name)
		eppCode, eppCodeErr := reader.ReadString('\n')
		if eppCodeErr != nil {
			log.Fatalln("Error accepting EPP code input", eppCodeErr)
		}

		eppCode = strings.Replace(eppCode, "\n", "", -1)

		log.Infof("Received EPP/auth code for %s: %#q\n", namecheapDomain.Name, eppCode)

		codeIsGood, codeCheckErr := cloudflareClient.CheckRegistrarTransferAuthCode(cloudflareClient.AccountID, namecheapDomain.Name, eppCode)
		if codeCheckErr != nil {
			log.Fatalf("Error checking EPP/auth code validity for domain %s: %v\n", namecheapDomain.Name, codeCheckErr)
		}

		if !codeIsGood {
			log.Fatalf(`Cloudflare reported the EPP/auth code "%s" is invalid for the domain %s. Please try again.`, eppCode, namecheapDomain.Name)
		}

		if namecheapDomain.IsLocked {
			log.Infof("%s: Unlocking", namecheapDomain.Name)
			setLockStatus, setLockStatusErr := namecheapClient.DomainSetRegistrarLock(namecheapDomain.Name, false)
			if setLockStatusErr != nil {
				bailOut(namecheapDomain, "Error unlocking domain %s: %v\n", namecheapDomain.Name, setLockStatusErr)
			}

			log.Infoln("setLockStatus", setLockStatus.Name, "is success?", setLockStatus.IsSuccess)
		} else {
			log.Infof("%s: Domain is already unlocked", namecheapDomain.Name)
		}

		if namecheapDomain.Whoisguard.Enabled {
			log.Infof("%s: Disabling WhoisGuard", namecheapDomain.Name)
			if setWhoisguardErr := namecheapClient.WhoisguardDisable(namecheapDomain.Whoisguard.ID); setWhoisguardErr != nil {
				bailOut(namecheapDomain, "Error disabling Whoisguard on domain %s: %v\n", namecheapDomain.Name, setWhoisguardErr)
			}
		} else {
			log.Infof("%s: WhoisGuard is already disabled", namecheapDomain.Name)
		}

		log.Infoln("Domain is ready to be transferred to Cloudflare, continuing...")

		success, transferReqErr := cloudflareClient.TransferRegistrarDomain(cloudflareZone, cloudflare.TransferRegistrarDomainRequest{
			RegistrantContactID: cloudflareContactID,
			AutoRenew:           enableAutoRenew,
			Years:               domainYears,
			Privacy:             enablePrivacy,
			ImportDNS:           enableImportDNS,
			Name:                namecheapDomain.Name,
			AuthCode:            eppCode,
		})
		if transferReqErr != nil {
			bailOut(namecheapDomain, "Error beginning Cloudflare Registrar transfer of %s: %v\n", namecheapDomain.Name, transferReqErr)
		}

		if success {
			log.Infoln("Cloudflare has started the process. Keep an eye on your email for a confirmation email from Namecheap. Inside, you'll find a link that must be clicked to continue the process.")
		} else {
			bailOut(namecheapDomain, "Cloudflare reported the transfer of %s failed!\n", namecheapDomain.Name)
		}

	},
}

func bailOut(domain *namecheap.DomainInfo, format string, args ...interface{}) {
	log.Warnln("Bailout: Something went seriously wrong, entering bailout. We will attempt to lock the domain and re-enable WhoisGuard (if it exists on domain) but this may fail. Double check with the Namecheap control panel. Error details to follow.")

	if _, setLockStatusErr := namecheapClient.DomainSetRegistrarLock(domain.Name, true); setLockStatusErr != nil {
		log.Fatalf("Bailout: Error unlocking domain %s. YOUR DOMAIN IS UNLOCKED!! %v\n", domain.Name, setLockStatusErr)
	}

	log.Infoln("Bailout: Successfully re-locked domain")

	if domain.Whoisguard.Enabled {
		if setWhoisguardErr := namecheapClient.WhoisguardEnable(domain.Whoisguard.ID, domain.Whoisguard.EmailDetails.ForwardedTo); setWhoisguardErr != nil {
			log.Fatalf("Bailout: Error disabling Whoisguard on domain %s. YOUR WHOIS INFORMATION IS UNPROTECTED!! %v\n", domain.Name, setWhoisguardErr)
		}
	}

	log.Infoln("Bailout: Successfully re-enabled WhoisGuard")

	log.Infoln("Bailout complete, error details to follow")

	log.Fatalf(format, args...)
}

func init() {
	rootCmd.AddCommand(transferCmd)

	transferCmd.Flags().StringVar(&cloudflareContactID, "cloudflare.contact-id", "", "Contact ID to use for domain WHOIS")
	transferCmd.Flags().IntVar(&domainYears, "cloudflare.years", 1, "Number of years to register domain for")
	transferCmd.Flags().BoolVar(&enablePrivacy, "cloudflare.privacy", true, "Whether WHOIS privacy should be enabled")
	transferCmd.Flags().BoolVar(&enableAutoRenew, "cloudflare.auto-renew", true, "Whether auto renewal should be enabled")
	transferCmd.Flags().BoolVar(&enableImportDNS, "cloudflare.import", true, "Whether existing DNS records should be imported")
}
