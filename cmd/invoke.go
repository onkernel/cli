package cmd

import (
	"encoding/json"
	"fmt"
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
	invokeCmd.Flags().BoolP("async", "a", true, "Invoke asynchronously (default true). Use --async=false if invocations are expected to last less than 60 seconds to wait synchronously")
}

func runInvoke(cmd *cobra.Command, args []string) error {
	client := kernel.NewClient()
	appName := args[0]
	actionName := args[1]
	version, _ := cmd.Flags().GetString("version")
	asyncFlag, _ := cmd.Flags().GetBool("async")
	if version == "" {
		return fmt.Errorf("version cannot be an empty string")
	}
	params := kernel.AppInvocationNewParams{
		AppName:    appName,
		ActionName: actionName,
		Version:    version,
	}
	payloadStr, _ := cmd.Flags().GetString("payload")
	payloadProvided := cmd.Flags().Changed("payload")
	switch {
	case !payloadProvided:
		// user did NOT pass --payload at all
	case payloadStr == "":
		// user passed --payload ""  (or --payload=) – empty string explicitly requested
		params.Payload = kernel.Opt("")
	default:
		// user passed a non-empty payload
		var i interface{}
		if err := json.Unmarshal([]byte(payloadStr), &i); err != nil {
			return fmt.Errorf("invalid JSON payload: %w", err)
		}
		params.Payload = kernel.Opt(payloadStr)
	}
	if asyncFlag {
		params.Async = kernel.Opt(true)
	}

	pterm.Info.Printf("Invoking \"%s\" (action: %s, version: %s) ...\n", appName, actionName, version)
	requestOpts := []option.RequestOption{option.WithMaxRetries(0)}
	if !asyncFlag {
		requestOpts = append(requestOpts, option.WithRequestTimeout(10*time.Minute))
	}
	resp, err := client.Apps.Invocations.New(cmd.Context(), params, requestOpts...)
	if err != nil {
		pterm.Error.Printf("Failed to invoke application: %v\n", err)

		// Try to extract more detailed error information
		if apiErr, ok := err.(*kernel.Error); ok {
			pterm.Error.Printf("API Error Details:\n")
			pterm.Error.Printf("  Status: %d\n", apiErr.StatusCode)
			pterm.Error.Printf("  Response: %s\n", apiErr.DumpResponse(true))
		}

		// Print troubleshooting tips
		pterm.Info.Println("Troubleshooting tips:")
		pterm.Info.Println("- Check that your API key is valid")
		pterm.Info.Println("- Verify that the app name and action name are correct")
		pterm.Info.Println("- Ensure the app version exists")
		pterm.Info.Println("- Validate that your payload is properly formatted")
		return nil
	}

	// if not queued, print the result
	if resp.Status != kernel.AppInvocationNewResponseStatusQueued {
		printResult(resp.Status == kernel.AppInvocationNewResponseStatusSucceeded, resp.Output)
		return nil
	}

	// invocation is queued--poll until we have output or terminal status
	pterm.Info.Println("Invocation queued, polling for result...")
	for {
		time.Sleep(2 * time.Second)
		invocation, err := client.Apps.Invocations.Get(cmd.Context(), resp.ID, option.WithMaxRetries(0), option.WithRequestTimeout(2*time.Minute))
		if err != nil {
			pterm.Error.Printf("Polling failed: %v\n", err)
			return nil
		}
		if invocation.Status == kernel.AppInvocationGetResponseStatusSucceeded || invocation.Status == kernel.AppInvocationGetResponseStatusFailed {
			printResult(invocation.Status == kernel.AppInvocationGetResponseStatusSucceeded, invocation.Output)
			return nil
		}
	}
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
