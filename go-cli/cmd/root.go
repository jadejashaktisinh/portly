package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var version = "dev"

var rootCmd = &cobra.Command{
	Use:     "portly",
	Short:   "Portly forwards webhooks to a local server",
	Long: `Portly is a lightweight tunneling and webhook forwarding service.
Use "portly login" to authenticate and "portly forward <local-url>" to expose a local server.`,
	Version: version,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	loadDotEnv()
	rootCmd.SetOut(os.Stdout)
	rootCmd.SetErr(os.Stderr)
	rootCmd.PersistentFlags().StringVar(&flagAPIURL, "api-url", "", "Portly API server URL")
	rootCmd.PersistentFlags().StringVar(&flagRelayURL, "relay-url", "", "Portly relay server URL")
	rootCmd.PersistentFlags().StringVar(&flagKeyringService, "keyring-service", "", "OS keyring service name")
	rootCmd.PersistentFlags().StringVar(&flagKeyringUser, "keyring-user", "", "OS keyring account name")
}
