package cmd

import (
	"context"
	"fmt"
	"os"
	"runtime"

	"github.com/charmbracelet/fang"
	"github.com/onkernel/cli/pkg/auth"
	"github.com/onkernel/kernel-go-sdk"
	"github.com/onkernel/kernel-go-sdk/option"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

type Metadata struct {
	Version   string
	Commit    string
	Date      string
	GoVersion string
}

var metadata = Metadata{
	// these are set at build-time via ldflags.
	// https://goreleaser.com/cookbooks/using-main.version/
	Version:   "dev",
	Commit:    "none",
	Date:      "unknown",
	GoVersion: runtime.Version(),
}

// rootCmd is the base command for the CLI.
var rootCmd = &cobra.Command{
	Use:   "kernel",
	Short: "CLI for Kernel deployment and invocation",
	Run: func(cmd *cobra.Command, args []string) {
		// If called without any subcommands, just show help.
		_ = cmd.Help()
	},
}

var logger *pterm.Logger

func logLevelToPterm(level string) pterm.LogLevel {
	switch level {
	case "trace":
		return pterm.LogLevelTrace
	case "debug":
		return pterm.LogLevelDebug
	case "info":
		return pterm.LogLevelInfo
	case "warn":
		return pterm.LogLevelWarn
	case "error":
		return pterm.LogLevelError
	case "fatal":
		return pterm.LogLevelFatal
	case "print":
		return pterm.LogLevelPrint
	default:
		return pterm.LogLevelInfo
	}
}

type contextKey string

const KernelClientKey contextKey = "kernel_client"

func getKernelClient(cmd *cobra.Command) kernel.Client {
	return cmd.Context().Value(KernelClientKey).(kernel.Client)
}

// isAuthExempt returns true if the command or any of its parents should skip auth.
func isAuthExempt(cmd *cobra.Command) bool {
	if cmd == rootCmd { // only 'kernel' with no subcommand
		return true
	}
	for c := cmd; c != nil; c = c.Parent() {
		switch c.Name() {
		case "login", "logout", "auth", "help", "completion":
			return true
		}
	}
	return false
}

func init() {
	rootCmd.PersistentFlags().BoolP("version", "v", false, "Print the CLI version")
	rootCmd.PersistentFlags().BoolP("no-color", "", false, "Disable color output")
	rootCmd.PersistentFlags().String("log-level", "warn", "Set the log level (trace, debug, info, warn, error, fatal, print)")
	rootCmd.SilenceUsage = true
	rootCmd.SilenceErrors = true
	cobra.OnInitialize(initConfig)

	// Version flag handling: we use our own persistent pre-run to handle it globally.
	// We also inject a Kernel client object into the command context for commands to use
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		logLevel, _ := cmd.Flags().GetString("log-level")
		logger = pterm.DefaultLogger.WithLevel(logLevelToPterm(logLevel))
		if noColor, _ := cmd.Flags().GetBool("no-color"); noColor {
			pterm.DisableStyling()
		}

		// Skip auth check for commands that don't need it (including children, e.g., "completion zsh")
		if isAuthExempt(cmd) {
			return nil
		}

		// Get authenticated client with OAuth tokens or API key fallback
		client, err := auth.GetAuthenticatedClient(option.WithHeader("X-Kernel-Cli-Version", metadata.Version))
		if err != nil {
			return fmt.Errorf("authentication required: %w", err)
		}

		ctx := context.WithValue(cmd.Context(), KernelClientKey, *client)
		cmd.SetContext(ctx)
		return nil
	}

	// Register subcommands
	rootCmd.AddCommand(deployCmd)
	rootCmd.AddCommand(invokeCmd)
	rootCmd.AddCommand(browsersCmd)
	rootCmd.AddCommand(appCmd)
}

func initConfig() {
	// Placeholder for future configuration (env vars, config files, etc.)
	pterm.EnableStyling() // ensure pterm is initialised in case env disables it
}

// Execute executes the root command.
func Execute(m Metadata) {
	metadata = m
	vt := "kernel"
	if metadata.Version != "" {
		vt += " " + metadata.Version
	}
	if metadata.Commit != "" {
		vt += " (" + metadata.Commit + ")"
	}
	if metadata.GoVersion != "" {
		vt += " " + metadata.GoVersion
	}
	if metadata.Date != "" {
		vt += " " + metadata.Date
	}
	vt += "\n"
	rootCmd.SetVersionTemplate(vt)
	if err := fang.Execute(context.Background(), rootCmd, fang.WithVersion(metadata.Version), fang.WithCommit(metadata.Commit)); err != nil {
		// fang takes care of printing the error
		os.Exit(1)
	}
}

// onCancel runs a function when the provided context is cancelled
func onCancel(ctx context.Context, fn func()) {
	go func() {
		<-ctx.Done()
		if ctx.Err() == context.Canceled {
			fn()
		}
	}()
}
