package cmd

import (
	"fmt"
	"os"

	"github.com/billputer/go-namecheap"
	"github.com/cloudflare/cloudflare-go"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var namecheapApiUser string
var namecheapApiToken string
var namecheapUsername string
var cloudflareAPIKey string
var cloudflareEmail string
var cloudflareAccountId string
var namecheapClient *namecheap.Client
var cloudflareClient *cloudflare.API
var cloudflareAccount cloudflare.Account

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "namecloud",
	Short: "namecloud is a little utility to point all your Namecheap domains to Cloudflare nameservers and optionally, transfer them.",
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&namecheapApiUser, "namecheap.api-user", "", "Namecheap API User")
	rootCmd.PersistentFlags().StringVar(&namecheapApiToken, "namecheap.api-token", "", "Namecheap API Token")
	rootCmd.PersistentFlags().StringVar(&namecheapUsername, "namecheap.username", "", "Namecheap Username")
	rootCmd.PersistentFlags().StringVar(&cloudflareAPIKey, "cloudflare.api-key", "", "Cloudflare API Key")
	rootCmd.PersistentFlags().StringVar(&cloudflareEmail, "cloudflare.email", "", "Cloudflare Email")
	rootCmd.PersistentFlags().StringVar(&cloudflareAccountId, "cloudflare.account-id", "", "Cloudflare Account ID")
}

func preRun(cmd *cobra.Command, args []string) {
	namecheapClient = namecheap.NewClient(namecheapApiUser, namecheapApiToken, namecheapUsername)

	var cfClientErr error
	cloudflareClient, cfClientErr = cloudflare.New(cloudflareAPIKey, cloudflareEmail, cloudflare.UsingAccount(cloudflareAccountId))
	if cfClientErr != nil {
		log.Fatalln("Error creating Cloudflare API Client", cfClientErr)
		return
	}

	var cfAccountErr error
	cloudflareAccount, _, cfAccountErr = cloudflareClient.Account(cloudflareClient.AccountID)
	if cfAccountErr != nil {
		log.Fatalln("Error getting Cloudflare accout", cfAccountErr)
	}
}
