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
	"github.com/onkernel/kernel-go-sdk/option"
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

func runDeploy(cmd *cobra.Command, args []string) (err error) {
	startTime := time.Now()
	client := getKernelClient(cmd)
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
	spinner.Info("Compressed files")
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

	logger.Debug("deploying app", logger.Args("version", version, "force", force, "entrypoint", filepath.Base(resolvedEntrypoint)))
	pterm.Info.Println("Deploying...")

	resp, err := client.Deployments.New(cmd.Context(), kernel.DeploymentNewParams{
		File:              file,
		Version:           kernel.Opt(version),
		Force:             kernel.Opt(force),
		EntrypointRelPath: filepath.Base(resolvedEntrypoint),
		EnvVars:           envVars,
	}, option.WithMaxRetries(0))
	if err != nil {
		return &util.CleanedUpSdkError{Err: err}
	}

	// Follow deployment events via SSE
	stream := client.Deployments.FollowStreaming(cmd.Context(), resp.ID, option.WithMaxRetries(0))
	for stream.Next() {
		data := stream.Current()
		switch data.Event {
		case "log":
			logEv := data.AsLog()
			msg := strings.TrimSuffix(logEv.Message, "\n")
			pterm.Info.Println(pterm.Gray(msg))
		case "deployment_state":
			deploymentState := data.AsDeploymentState()
			status := deploymentState.Deployment.Status
			if status == string(kernel.DeploymentGetResponseStatusFailed) ||
				status == string(kernel.DeploymentGetResponseStatusStopped) {
				pterm.Error.Println("✖ Deployment failed")
				err = fmt.Errorf("Deployment %s: %s", status, deploymentState.Deployment.StatusReason)
				return err
			}
			if status == string(kernel.DeploymentGetResponseStatusRunning) {
				duration := time.Since(startTime)
				pterm.Success.Printfln("✔ Deployment complete in %s", duration.Round(time.Millisecond))
				return nil
			}
		case "app_version_summary":
			appVersionSummary := data.AsDeploymentFollowResponseAppVersionSummaryEvent()
			pterm.Info.Printf("App \"%s\" deployed (version: %s)\n", appVersionSummary.AppName, appVersionSummary.Version)
			if len(appVersionSummary.Actions) > 0 {
				action0Name := appVersionSummary.Actions[0].Name
				pterm.Info.Printf("Invoke with: kernel invoke %s %s --payload '{...}'\n", quoteIfNeeded(appVersionSummary.AppName), quoteIfNeeded(action0Name))
			}
		case "error":
			errorEv := data.AsErrorEvent()
			err = fmt.Errorf("%s: %s", errorEv.Error.Code, errorEv.Error.Message)
			return err
		}
	}

	if serr := stream.Err(); serr != nil {
		pterm.Error.Println("✖ Stream error")
		return fmt.Errorf("stream error: %w", serr)
	}
	return nil
}

func quoteIfNeeded(s string) string {
	if len(s) > 0 && (s[0] == ' ' || s[len(s)-1] == ' ') {
		return fmt.Sprintf("\"%s\"", s)
	}
	return s
}
