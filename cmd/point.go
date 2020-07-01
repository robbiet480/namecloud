package cmd

import (
	"context"
	"strings"

	"github.com/cloudflare/cloudflare-go"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/weppos/publicsuffix-go/publicsuffix"
)

// pointCmd represents the point command
var pointCmd = &cobra.Command{
	Use:    "point",
	Short:  "point will configure your Namecheap domain(s) for use with Cloudflare",
	Long:   `point creates a Cloudflare zone for your Namecheap domain(s) (if required) and sets the nameservers of the domain(s) to Cloudflare nameservers.`,
	PreRun: preRun,
	Run: func(cmd *cobra.Command, args []string) {
		cloudflareDomains := []string{}

		cfZonesResp, cfZonesErr := cloudflareClient.ListZonesContext(context.TODO(), cloudflare.WithZoneFilters("", cloudflareAccount.ID, ""), cloudflare.WithPagination(cloudflare.PaginationOptions{
			PerPage: 50,
		}))
		if cfZonesErr != nil {
			log.Fatalln("Error getting all zones", cfZonesErr)
		}

		for _, z := range cfZonesResp.Result {
			cloudflareDomains = append(cloudflareDomains, z.Name)
		}

		domainsToAddToCF := map[string][]string{}
		domainList := []string{}

		namecheapDomainsResp, _ := namecheapClient.DomainsGetList()
		for _, domain := range namecheapDomainsResp {
			if contains(cloudflareDomains, domain.Name) {
				continue
			}

			domainInfo, domainInfoErr := namecheapClient.DomainGetInfo(domain.Name)
			if domainInfoErr != nil {
				log.Fatalln("Error getting Namecheap domain info for domain:", domain.Name, domainInfoErr)
			}
			log.Infoln("Namecheap Domain:", domainInfo.Name, "has name servers:", strings.Join(domainInfo.DNSDetails.Nameservers, ", "))

			domainsToAddToCF[domainInfo.Name] = domainInfo.DNSDetails.Nameservers
			domainList = append(domainList, domainInfo.Name)
		}

		log.Infoln("Zones that will be created in Cloudflare:", domainList)

		for domainName, currentNameServers := range domainsToAddToCF {
			newZone, newZoneErr := cloudflareClient.CreateZone(domainName, true, cloudflareAccount, "full")
			if newZoneErr != nil {
				log.Errorln("Error when adding new zone", domainName, newZoneErr)
				continue
			}

			if !equal(newZone.NameServers, currentNameServers) {
				log.Infof("New zone %s nameservers don't match whats on Namecheap! Cloudflare wants %v, Namecheap has %v\n", newZone.Name, newZone.NameServers, currentNameServers)

				parsedDomain, parsedDomainErr := publicsuffix.Parse(domainName)
				if parsedDomainErr != nil {
					log.Errorln("Error when parsing domain name", domainName, parsedDomainErr)
					continue
				}

				setNameserversResp, setNameserversErr := namecheapClient.DomainDNSSetCustom(parsedDomain.SLD, parsedDomain.TLD, strings.Join(newZone.NameServers, ","))
				if setNameserversErr != nil {
					log.Errorln("Error when setting Namecheap nameservers for domain", domainName, setNameserversErr)
					continue
				}

				if !setNameserversResp.Update {
					log.Warnln("Namecheap nameservers update failed for domain", domainName)
				}
			}

		}
	},
}

func init() {
	rootCmd.AddCommand(pointCmd)
}
