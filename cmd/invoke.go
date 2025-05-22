package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/onkernel/kernel-go-sdk"
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
}

func runInvoke(cmd *cobra.Command, args []string) error {
	apiKey := os.Getenv("KERNEL_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("KERNEL_API_KEY environment variable is not set")
	}
	client := kernel.NewClient()
	appName := args[0]
	actionName := args[1]
	version, _ := cmd.Flags().GetString("version")
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

	pterm.Info.Printf("Invoking \"%s\" (action: %s, version: %s) ...\n", appName, actionName, version)
	resp, err := client.Apps.Invocations.New(cmd.Context(), params)
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

	if resp.Output == "" {
		pterm.Warning.Println("No output returned")
		return nil
	}

	var prettyJSON map[string]interface{}
	if err := json.Unmarshal([]byte(resp.Output), &prettyJSON); err != nil {
		pterm.Success.Printf("Result: %s\n", resp.Output)
		return nil
	}

	pretty, _ := json.MarshalIndent(prettyJSON, "", "  ")
	pterm.Success.Printf("Result:\n%s\n", string(pretty))
	return nil
}
