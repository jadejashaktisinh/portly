package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "portly",
	Short: "Portly forwards webhooks to a local server",
	Long: `Portly is a lightweight tunneling and webhook forwarding service.
Use "portly login" to authenticate and "portly forward <local-url>" to expose a local server.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.SetOut(os.Stdout)
	rootCmd.SetErr(os.Stderr)
}
