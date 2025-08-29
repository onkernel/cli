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

var deployLogsCmd = &cobra.Command{
	Use:   "logs <deployment_id>",
	Short: "Stream logs for a deployment",
	Args:  cobra.ExactArgs(1),
	RunE:  runDeployLogs,
}

var deployHistoryCmd = &cobra.Command{
	Use:   "history [app_name]",
	Short: "Show deployment history",
	Args:  cobra.RangeArgs(0, 1),
	RunE:  runDeployHistory,
}

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

	// Subcommands under deploy
	deployLogsCmd.Flags().BoolP("follow", "f", false, "Follow logs in real-time (stream continuously)")
	deployLogsCmd.Flags().StringP("since", "s", "", "How far back to retrieve logs. Supports duration formats: ns, us, ms, s, m, h (e.g., 5m, 2h, 1h30m). Note: 'd' not supported; use hours instead. Can also specify timestamps: 2006-01-02, 2006-01-02T15:04, 2006-01-02T15:04:05, 2006-01-02T15:04:05.000. Max lookback ~167h.")
	deployLogsCmd.Flags().BoolP("with-timestamps", "t", false, "Include timestamps in each log line")
	deployCmd.AddCommand(deployLogsCmd)

	deployHistoryCmd.Flags().Bool("all", false, "Show deployment history for all applications")
	deployCmd.AddCommand(deployHistoryCmd)
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
		return util.CleanedUpSdkError{Err: err}
	}

	// Follow deployment events via SSE
	stream := client.Deployments.FollowStreaming(cmd.Context(), resp.ID, kernel.DeploymentFollowParams{}, option.WithMaxRetries(0))
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
				pterm.Error.Printf("Deployment ID: %s\n", resp.ID)
				pterm.Info.Printf("View logs: kernel deploy logs %s --since 1h\n", resp.ID)
				err = fmt.Errorf("deployment %s: %s", status, deploymentState.Deployment.StatusReason)
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
			pterm.Error.Printf("Deployment ID: %s\n", resp.ID)
			pterm.Info.Printf("View logs: kernel deploy logs %s --since 1h\n", resp.ID)
			err = fmt.Errorf("%s: %s", errorEv.Error.Code, errorEv.Error.Message)
			return err
		}
	}

	if serr := stream.Err(); serr != nil {
		pterm.Error.Println("✖ Stream error")
		pterm.Error.Printf("Deployment ID: %s\n", resp.ID)
		pterm.Info.Printf("View logs: kernel deploy logs %s --since 1h\n", resp.ID)
		return fmt.Errorf("stream error: %w", serr)
	}
	return nil
}

func quoteIfNeeded(s string) string {
	if strings.ContainsRune(s, ' ') {
		return fmt.Sprintf("\"%s\"", s)
	}
	return s
}

func runDeployLogs(cmd *cobra.Command, args []string) error {
	client := getKernelClient(cmd)

	deploymentID := args[0]
	pterm.Info.Printf("Streaming logs for deployment %s...\n", deploymentID)

	since, _ := cmd.Flags().GetString("since")
	follow, _ := cmd.Flags().GetBool("follow")
	ts, _ := cmd.Flags().GetBool("with-timestamps")

	stream := client.Deployments.FollowStreaming(cmd.Context(), deploymentID, kernel.DeploymentFollowParams{Since: kernel.Opt(since)}, option.WithMaxRetries(0))
	defer func() { _ = stream.Close() }()
	if stream.Err() != nil {
		return fmt.Errorf("failed to open log stream: %w", stream.Err())
	}

	if follow {
		for stream.Next() {
			data := stream.Current()
			switch data.Event {
			case "log":
				logEntry := data.AsLog()
				if ts {
					fmt.Printf("%s %s\n", logEntry.Timestamp.Format(time.RFC3339Nano), strings.TrimSuffix(logEntry.Message, "\n"))
				} else {
					fmt.Println(strings.TrimSuffix(logEntry.Message, "\n"))
				}
			case "error":
				errEvt := data.AsErrorEvent()
				return fmt.Errorf("%s: %s", errEvt.Error.Code, errEvt.Error.Message)
			}
		}
	} else {
		// Non-follow: exit after brief inactivity window (3s) like app logs
		timeout := time.NewTimer(3 * time.Second)
		defer timeout.Stop()
		for {
			nextCh := make(chan bool, 1)
			go func() { nextCh <- stream.Next() }()
			select {
			case hasNext := <-nextCh:
				if !hasNext {
					return nil
				}
				data := stream.Current()
				switch data.Event {
				case "log":
					logEntry := data.AsLog()
					if ts {
						fmt.Printf("%s %s\n", logEntry.Timestamp.Format(time.RFC3339Nano), strings.TrimSuffix(logEntry.Message, "\n"))
					} else {
						fmt.Println(strings.TrimSuffix(logEntry.Message, "\n"))
					}
				case "error":
					errEvt := data.AsErrorEvent()
					return fmt.Errorf("%s: %s", errEvt.Error.Code, errEvt.Error.Message)
				}
				timeout.Reset(3 * time.Second)
			case <-timeout.C:
				_ = stream.Close()
				return nil
			}
		}
	}

	if stream.Err() != nil {
		return fmt.Errorf("failed while streaming logs: %w", stream.Err())
	}
	return nil
}

func runDeployHistory(cmd *cobra.Command, args []string) error {
	client := getKernelClient(cmd)

	all, _ := cmd.Flags().GetBool("all")

	var appNames []string
	if len(args) == 1 {
		appNames = []string{args[0]}
	} else if all {
		apps, err := client.Apps.List(cmd.Context(), kernel.AppListParams{})
		if err != nil {
			pterm.Error.Printf("Failed to list applications: %v\n", err)
			return nil
		}
		for _, a := range *apps {
			appNames = append(appNames, a.AppName)
		}
		// de-duplicate app names
		seen := map[string]struct{}{}
		uniq := make([]string, 0, len(appNames))
		for _, n := range appNames {
			if _, ok := seen[n]; ok {
				continue
			}
			seen[n] = struct{}{}
			uniq = append(uniq, n)
		}
		appNames = uniq
	} else {
		pterm.Error.Println("Either provide an app name or use --all")
		return nil
	}

	table := pterm.TableData{{"Deployment ID", "Created At", "Region", "Status", "Entrypoint", "Reason"}}
	for _, appName := range appNames {
		params := kernel.DeploymentListParams{AppName: kernel.Opt(appName)}
		pterm.Debug.Printf("Listing deployments for app '%s'...\n", appName)
		deployments, err := client.Deployments.List(cmd.Context(), params)
		if err != nil {
			pterm.Error.Printf("Failed to list deployments for '%s': %v\n", appName, err)
			continue
		}
		for _, dep := range *deployments {
			created := dep.CreatedAt.Format(time.RFC3339)
			status := string(dep.Status)
			table = append(table, []string{
				dep.ID,
				created,
				string(dep.Region),
				status,
				dep.EntrypointRelPath,
				dep.StatusReason,
			})
		}
	}
	if len(table) == 1 {
		pterm.Info.Println("No deployments found")
		return nil
	}
	pterm.DefaultTable.WithHasHeader().WithData(table).Render()
	return nil
}
