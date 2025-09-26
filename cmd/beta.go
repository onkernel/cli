package cmd

import (
	"github.com/onkernel/cli/cmd/proxies"
	"github.com/spf13/cobra"
)

// betaCmd is the parent command for experimental features
var betaCmd = &cobra.Command{
	Use:   "beta",
	Short: "Experimental features (subject to change)",
	Long: `The beta command provides access to experimental features that are still under development.
These features may change, break, or be removed in future versions without notice.`,
	Run: func(cmd *cobra.Command, args []string) {
		// If called without subcommands, show help
		_ = cmd.Help()
	},
}

func init() {
	// Add proxy commands under beta
	betaCmd.AddCommand(proxies.ProxiesCmd)
}
