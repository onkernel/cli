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
	invokeCmd.Flags().String("version", "", "Specify a version of the app to invoke")
	invokeCmd.Flags().String("payload", "", "JSON payload for the invocation")
}

func runInvoke(cmd *cobra.Command, args []string) error {
	apiKey := os.Getenv("KERNEL_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("KERNEL_API_KEY environment variable is not set")
	}
	client := kernel.NewClient() // defaults to look at KERNEL_API_KEY
	appName := args[0]
	actionName := args[1]

	payloadStr, _ := cmd.Flags().GetString("payload")
	var payload interface{}
	if payloadStr == "" {
		// If no payload specified, default to an empty JSON object
		payloadStr = "{}"
	}

	if err := json.Unmarshal([]byte(payloadStr), &payload); err != nil {
		return fmt.Errorf("invalid JSON payload: %w", err)
	}
	version, _ := cmd.Flags().GetString("version")
	pterm.Info.Printf("Invoking \"%s\" (action: %s) ...\n", appName, actionName)
	resp, err := client.Apps.Invoke(cmd.Context(), kernel.AppInvokeParams{
		AppName:    appName,
		ActionName: actionName,
		Payload:    payload,
		Version:    version,
	})
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

		return fmt.Errorf("invocation failed: %w", err)
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
