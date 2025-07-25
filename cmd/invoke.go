package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/onkernel/kernel-go-sdk"
	"github.com/onkernel/kernel-go-sdk/option"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

var invokeCmd = &cobra.Command{
	Use:   "invoke <app_name> <action_name> [flags]",
	Short: "Invoke a deployed Kernel application",
	Args:  cobra.ExactArgs(2),
	RunE:  runInvoke,
}

func init() {
	invokeCmd.Flags().StringP("version", "v", "latest", "Specify a version of the app to invoke (optional, defaults to 'latest')")
	invokeCmd.Flags().StringP("payload", "p", "", "JSON payload for the invocation (optional)")
	invokeCmd.Flags().BoolP("sync", "s", false, "Invoke synchronously (default false). A synchronous invocation will open a long-lived HTTP POST to the Kernel API to wait for the invocation to complete. This will time out after 60 seconds, so only use this option if you expect your invocation to complete in less than 60 seconds. The default is to invoke asynchronously, in which case the CLI will open an SSE connection to the Kernel API after submitting the invocation and wait for the invocation to complete.")
}

func runInvoke(cmd *cobra.Command, args []string) error {
	startTime := time.Now()
	client := getKernelClient(cmd)
	appName := args[0]
	actionName := args[1]
	version, _ := cmd.Flags().GetString("version")
	if version == "" {
		return fmt.Errorf("version cannot be an empty string")
	}
	isSync, _ := cmd.Flags().GetBool("sync")
	params := kernel.InvocationNewParams{
		AppName:    appName,
		ActionName: actionName,
		Version:    version,
		Async:      kernel.Opt(!isSync),
	}

	payloadStr, _ := cmd.Flags().GetString("payload")
	if cmd.Flags().Changed("payload") {
		// validate JSON unless empty string explicitly set
		if payloadStr != "" {
			var v interface{}
			if err := json.Unmarshal([]byte(payloadStr), &v); err != nil {
				return fmt.Errorf("invalid JSON payload: %w", err)
			}
		}
		params.Payload = kernel.Opt(payloadStr)
	}
	// we don't really care to cancel the context, we just want to handle signals
	ctx, _ := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
	cmd.SetContext(ctx)

	pterm.Info.Printf("Invoking \"%s\" (action: %s, version: %s)…\n", appName, actionName, version)

	// Create the invocation
	resp, err := client.Invocations.New(cmd.Context(), params, option.WithMaxRetries(0))
	if err != nil {
		return handleSdkError(err)
	}
	// coordinate the cleanup with the polling loop to ensure this is given enough time to run
	// before this function returns
	cleanupDone := make(chan struct{})
	cleanupStarted := atomic.Bool{}
	defer func() {
		if cleanupStarted.Load() {
			<-cleanupDone
		}
	}()

	// this is a little indirect but we try to fail out of the invocation by deleting the
	// underlying browser sessions
	once := sync.Once{}
	onCancel(cmd.Context(), func() {
		once.Do(func() {
			cleanupStarted.Store(true)
			defer close(cleanupDone)
			pterm.Warning.Println("Invocation cancelled...cleaning up...")
			if err := client.Invocations.DeleteBrowsers(context.Background(), resp.ID, option.WithRequestTimeout(30*time.Second)); err != nil {
				pterm.Error.Printf("Failed to cancel invocation: %v\n", err)
			}
		})
	})

	// Start following events
	stream := client.Invocations.FollowStreaming(cmd.Context(), resp.ID, option.WithMaxRetries(0))
	for stream.Next() {
		ev := stream.Current()

		switch ev.Event {
		case "log":
			logEv := ev.AsLog()
			msg := strings.TrimSuffix(logEv.Message, "\n")
			pterm.Info.Println(pterm.Gray(msg))

		case "invocation_state":
			stateEv := ev.AsInvocationState()
			status := stateEv.Invocation.Status
			if status == string(kernel.InvocationGetResponseStatusSucceeded) || status == string(kernel.InvocationGetResponseStatusFailed) {
				// Finished – print output and exit accordingly
				succeeded := status == string(kernel.InvocationGetResponseStatusSucceeded)
				printResult(succeeded, stateEv.Invocation.Output)

				duration := time.Since(startTime)
				if succeeded {
					pterm.Success.Printfln("✔ Completed in %s", duration.Round(time.Millisecond))
					return nil
				}
				return nil
			}

		case "error":
			errEv := ev.AsError()
			return fmt.Errorf("%s: %s", errEv.Error.Code, errEv.Error.Message)
		}
	}

	if serr := stream.Err(); serr != nil {
		return fmt.Errorf("stream error: %w", serr)
	}
	return nil
}

// handleSdkError prints helpful diagnostics similar to runDeploy
func handleSdkError(err error) error {
	pterm.Error.Printf("Failed to invoke application: %v\n", err)
	if apiErr, ok := err.(*kernel.Error); ok {
		pterm.Error.Printf("API Error Details:\n")
		pterm.Error.Printf("  Status: %d\n", apiErr.StatusCode)
		pterm.Error.Printf("  Response: %s\n", apiErr.DumpResponse(true))
	}

	pterm.Info.Println("Troubleshooting tips:")
	pterm.Info.Println("- Check that your API key is valid")
	pterm.Info.Println("- Verify that the app name and action name are correct")
	pterm.Info.Println("- Validate that your payload is properly formatted")
	pterm.Info.Println("- Check `kernel app history <app name>` to see if the app is deployed")
	pterm.Info.Println("- Try redeploying the app")
	pterm.Info.Println("- Make sure you're on the latest version of the CLI: `brew upgrade onkernel/tap/kernel`")
	return nil
}

func printResult(success bool, output string) {
	var prettyJSON map[string]interface{}
	if err := json.Unmarshal([]byte(output), &prettyJSON); err == nil {
		bs, _ := json.MarshalIndent(prettyJSON, "", "  ")
		output = string(bs)
	}
	// use pterm.Success if succeeded, pterm.Error if failed
	if success {
		pterm.Success.Printf("Result:\n%s\n", output)
	} else {
		pterm.Error.Printf("Result:\n%s\n", output)
	}
}
