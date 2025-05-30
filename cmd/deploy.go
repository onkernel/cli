package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/onkernel/cli/pkg/util"
	kernel "github.com/onkernel/kernel-go-sdk"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

var deployCmd = &cobra.Command{
	Use:   "deploy <entrypoint>",
	Short: "Deploy a Kernel application",
	Args:  cobra.ExactArgs(1),
	RunE:  runDeploy,
}

func init() {
	deployCmd.Flags().String("version", "latest", "Specify a version for the app (default: latest)")
	deployCmd.Flags().Bool("force", false, "Allow overwrite of an existing version with the same name")
	deployCmd.Flags().StringArrayP("env", "e", []string{}, "Set environment variables (e.g., KEY=value). May be specified multiple times")
	deployCmd.Flags().StringArray("env-file", []string{}, "Read environment variables from a file (.env format). May be specified multiple times")
}

func runDeploy(cmd *cobra.Command, args []string) error {
	startTime := time.Now()
	client := kernel.NewClient() // defaults to look at KERNEL_API_KEY
	entrypoint := args[0]
	version, _ := cmd.Flags().GetString("version")
	force, _ := cmd.Flags().GetBool("force")
	if version == "" {
		version = "latest"
	}
	resolvedEntrypoint, err := filepath.Abs(entrypoint)
	if err != nil {
		return fmt.Errorf("failed to resolve entrypoint: %w", err)
	}
	if _, err := os.Stat(resolvedEntrypoint); err != nil {
		return fmt.Errorf("entrypoint %s does not exist", resolvedEntrypoint)
	}

	sourceDir := filepath.Dir(resolvedEntrypoint)
	spinner, _ := pterm.DefaultSpinner.Start("Compressing files...")
	tmpFile := filepath.Join(os.TempDir(), fmt.Sprintf("kernel_%d.zip", time.Now().UnixNano()))
	logger.Debug("compressing files", logger.Args("sourceDir", sourceDir, "tmpFile", tmpFile))
	if err := util.ZipDirectory(sourceDir, tmpFile); err != nil {
		spinner.Fail("Failed to compress files")
		return err
	}
	spinner.Success("Compressed files")
	defer os.Remove(tmpFile)

	// make io.Reader from tmpFile
	file, err := os.Open(tmpFile)
	if err != nil {
		return fmt.Errorf("failed to open tmpFile: %w", err)
	}
	defer file.Close()

	// Gather environment variables from --env and --env-file flags
	envPairs, _ := cmd.Flags().GetStringArray("env")
	envFiles, _ := cmd.Flags().GetStringArray("env-file")

	envVars := make(map[string]string)

	// Load from env files first so that explicit --env overrides them
	for _, envFile := range envFiles {
		fileVars, err := godotenv.Read(envFile)
		if err != nil {
			return fmt.Errorf("failed to read env file %s: %w", envFile, err)
		}
		for k, v := range fileVars {
			envVars[k] = v
		}
	}

	// Parse KEY=value pairs provided via --env
	for _, kv := range envPairs {
		parts := strings.SplitN(kv, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid env variable format: %s (expected KEY=value)", kv)
		}
		envVars[parts[0]] = parts[1]
	}

	spinner, _ = pterm.DefaultSpinner.Start("Deploying app...")
	logger.Debug("deploying app", logger.Args("version", version, "force", force, "entrypoint", filepath.Base(resolvedEntrypoint)))
	resp, err := client.Apps.Deployments.New(cmd.Context(), kernel.AppDeploymentNewParams{
		File:              file,
		Version:           kernel.Opt(version),
		Force:             kernel.Opt(force),
		EntrypointRelPath: filepath.Base(resolvedEntrypoint),
		EnvVars:           envVars,
	})
	if err != nil {
		return &util.CleanedUpSdkError{Err: err}
	}
	spinner.Success("Deployment successful")
	logger.Debug("deployment successful", logger.Args("resp", resp))
	for _, app := range resp.Apps {
		actions := make([]string, 0, len(app.Actions))
		for _, a := range app.Actions {
			actions = append(actions, a.Name)
		}
		pterm.Success.Printf("App \"%s\" deployed with action(s): %s\n", app.Name, actions)
		if len(actions) > 0 {
			pterm.Info.Printf("Invoke with: kernel invoke %s %s --payload '{...}'\n", quoteIfNeeded(app.Name), quoteIfNeeded(actions[0]))
		} else {
			pterm.Warning.Printf("App \"%s\" has no actions available to invoke.\n", app.Name)
		}
	}

	_ = os.Remove(tmpFile)
	duration := time.Since(startTime)
	pterm.Success.Printf("Total deployment time: %s\n", duration.Round(time.Millisecond))
	return nil
}

func quoteIfNeeded(s string) string {
	if len(s) > 0 && (s[0] == ' ' || s[len(s)-1] == ' ') {
		return fmt.Sprintf("\"%s\"", s)
	}
	return s
}
