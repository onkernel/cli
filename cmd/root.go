package cmd

import (
	"context"
	"fmt"
	"os"
	"runtime"

	"github.com/charmbracelet/fang"
	"github.com/onkernel/cli/pkg/util"
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

const KernelClientKey = "kernel_client"

func getKernelClient(cmd *cobra.Command) kernel.Client {
	return cmd.Context().Value(KernelClientKey).(kernel.Client)
}

func init() {
	rootCmd.PersistentFlags().BoolP("version", "v", false, "Print the CLI version")
	rootCmd.PersistentFlags().BoolP("no-color", "", false, "Disable color output")
	rootCmd.PersistentFlags().String("log-level", "warn", "Set the log level (trace, debug, info, warn, error, fatal, print)")
	rootCmd.SilenceUsage = true
	rootCmd.SilenceErrors = true
	cobra.OnInitialize(initConfig)

	// Version flag handling: we use our own persistent pre-run to handle it globally.
	// We also inject a Kernel client object into the command context for commands to sue
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		logLevel, _ := cmd.Flags().GetString("log-level")
		logger = pterm.DefaultLogger.WithLevel(logLevelToPterm(logLevel))
		if v, _ := cmd.Flags().GetBool("version"); v {
			fmt.Printf("kernel %s", metadata.Version)
			if metadata.Commit != "" {
				fmt.Printf(" (%s)", metadata.Commit)
			}
			if metadata.GoVersion != "" {
				fmt.Printf(" %s", metadata.GoVersion)
			}
			if metadata.Date != "" {
				fmt.Printf(" %s", metadata.Date)
			}
			fmt.Println()
			os.Exit(0)
		}
		if noColor, _ := cmd.Flags().GetBool("no-color"); noColor {
			pterm.DisableStyling()
		}

		// Check if this command requires API key
		// Skip API key check for root command (help) and any command that doesn't need it
		if cmd == rootCmd {
			return nil
		}

		// Commands that don't require API key can be added here
		// For now, all subcommands require API key
		apiKey := os.Getenv("KERNEL_API_KEY")
		if apiKey == "" {
			return fmt.Errorf("KERNEL_API_KEY environment variable is not set")
		}

		ctx := context.WithValue(cmd.Context(), KernelClientKey, util.NewClient(option.WithHeader("X-Kernel-Cli-Version", metadata.Version)))
		cmd.SetContext(ctx)
		return nil
	}

	// Register subcommands
	rootCmd.AddCommand(deployCmd)
	rootCmd.AddCommand(invokeCmd)
	rootCmd.AddCommand(browsersCmd)
	rootCmd.AddCommand(appsCmd)
}

func initConfig() {
	// Placeholder for future configuration (env vars, config files, etc.)
	pterm.EnableStyling() // ensure pterm is initialised in case env disables it
}

// Execute executes the root command.
func Execute(m Metadata) {
	metadata = m
	if err := fang.Execute(context.Background(), rootCmd); err != nil {
		pterm.Error.Println(err)
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
