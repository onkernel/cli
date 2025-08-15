package cmd

import (
	"fmt"

	"github.com/onkernel/kernel-go-sdk"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

var browsersCmd = &cobra.Command{
	Use:   "browsers",
	Short: "Manage browsers",
	Long:  "Commands for managing Kernel browsers",
}

var browsersListCmd = &cobra.Command{
	Use:   "list",
	Short: "List running or persistent browsers",
	RunE:  runBrowsersList,
}

var browsersCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new browser session",
	RunE:  runBrowsersCreate,
}

var browsersDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a browser",
	RunE:  runBrowsersDelete,
}

var browsersViewCmd = &cobra.Command{
	Use:   "view",
	Short: "Get the live view URL for a browser",
	RunE:  runBrowsersView,
}

func init() {
	browsersCmd.AddCommand(browsersListCmd)
	browsersCmd.AddCommand(browsersCreateCmd)
	browsersCmd.AddCommand(browsersDeleteCmd)
	browsersCmd.AddCommand(browsersViewCmd)

	// Add flags for create command
	browsersCreateCmd.Flags().String("persistence-id", "", "Unique identifier for browser session persistence")
	browsersCreateCmd.Flags().Bool("stealth", false, "Launch browser in stealth mode to avoid detection")
	browsersCreateCmd.Flags().Bool("headless", false, "Launch browser without GUI access")
	browsersCreateCmd.Flags().Int("timeout", 60, "Timeout in seconds for the browser session")

	// Add flags for delete command
	browsersDeleteCmd.Flags().String("by-persistent-id", "", "Delete browser by persistent ID")
	browsersDeleteCmd.Flags().String("by-id", "", "Delete browser by ID")
	browsersDeleteCmd.Flags().BoolP("yes", "y", false, "Skip confirmation prompt")
	browsersDeleteCmd.MarkFlagsMutuallyExclusive("by-persistent-id", "by-id")

	// Add flags for view command
	browsersViewCmd.Flags().String("by-persistent-id", "", "View browser by persistent ID")
	browsersViewCmd.Flags().String("by-id", "", "View browser by ID")
	browsersViewCmd.MarkFlagsMutuallyExclusive("by-persistent-id", "by-id")
}

func runBrowsersList(cmd *cobra.Command, args []string) error {
	client := getKernelClient(cmd)

	pterm.Info.Println("Fetching browsers...")

	browsers, err := client.Browsers.List(cmd.Context())
	if err != nil {
		pterm.Error.Printf("Failed to list browsers: %v\n", err)
		return nil
	}

	if browsers == nil || len(*browsers) == 0 {
		pterm.Info.Println("No running or persistent browsers found")
		return nil
	}

	// Prepare table data
	tableData := pterm.TableData{
		{"Browser ID", "Created At", "Persistent ID", "CDP WS URL", "Live View URL"},
	}

	for _, browser := range *browsers {
		persistentID := "-"
		if browser.Persistence.ID != "" {
			persistentID = browser.Persistence.ID
		}

		tableData = append(tableData, []string{
			browser.SessionID,
			browser.CreatedAt.Format("2006-01-02 15:04:05"),
			persistentID,
			truncateURL(browser.CdpWsURL, 50),
			truncateURL(browser.BrowserLiveViewURL, 50),
		})
	}

	pterm.DefaultTable.WithHasHeader().WithData(tableData).Render()
	return nil
}

func runBrowsersCreate(cmd *cobra.Command, args []string) error {
	client := getKernelClient(cmd)

	// Get flag values
	persistenceID, _ := cmd.Flags().GetString("persistence-id")
	stealth, _ := cmd.Flags().GetBool("stealth")
	headless, _ := cmd.Flags().GetBool("headless")
	timeout, _ := cmd.Flags().GetInt("timeout")

	pterm.Info.Println("Creating browser session...")

	// Build browser creation parameters
	params := kernel.BrowserNewParams{}

	if persistenceID != "" {
		params.Persistence = kernel.BrowserPersistenceParam{
			ID: persistenceID,
		}
	}

	if timeout > 0 {
		params.TimeoutSeconds = kernel.Opt(int64(timeout))
	}

	// Always set stealth parameter if the flag was explicitly provided
	if cmd.Flags().Changed("stealth") {
		params.Stealth = kernel.Opt(stealth)
	}

	// Always set headless parameter if the flag was explicitly provided
	if cmd.Flags().Changed("headless") {
		params.Headless = kernel.Opt(headless)
	}

	// Create the browser
	browser, err := client.Browsers.New(cmd.Context(), params)
	if err != nil {
		pterm.Error.Printf("Failed to create browser: %v\n", err)
		return nil
	}

	// Display browser information
	tableData := pterm.TableData{
		{"Property", "Value"},
		{"Session ID", browser.SessionID},
		{"CDP WebSocket URL", browser.CdpWsURL},
	}

	if browser.BrowserLiveViewURL != "" {
		tableData = append(tableData, []string{"Live View URL", browser.BrowserLiveViewURL})
	}

	if browser.Persistence.ID != "" {
		tableData = append(tableData, []string{"Persistent ID", browser.Persistence.ID})
	}

	pterm.DefaultTable.WithHasHeader().WithData(tableData).Render()

	return nil
}

func runBrowsersDelete(cmd *cobra.Command, args []string) error {
	client := getKernelClient(cmd)

	persistentID, _ := cmd.Flags().GetString("by-persistent-id")
	sessionID, _ := cmd.Flags().GetString("by-id")
	skipConfirm, _ := cmd.Flags().GetBool("yes")

	// Show confirmation prompt unless --yes flag is provided
	if !skipConfirm {
		var confirmMsg string
		if persistentID != "" {
			confirmMsg = fmt.Sprintf("Are you sure you want to delete browser with persistent ID '%s'?", persistentID)
		} else {
			confirmMsg = fmt.Sprintf("Are you sure you want to delete browser with ID '%s'?", sessionID)
		}

		pterm.DefaultInteractiveConfirm.DefaultText = confirmMsg
		result, _ := pterm.DefaultInteractiveConfirm.Show()
		if !result {
			pterm.Info.Println("Deletion cancelled")
			return nil
		}
	}

	if persistentID != "" {
		pterm.Info.Printf("Deleting browser with persistent ID: %s\n", persistentID)
		err := client.Browsers.Delete(cmd.Context(), kernel.BrowserDeleteParams{
			PersistentID: persistentID,
		})
		if err != nil {
			pterm.Error.Printf("Failed to delete browser: %v\n", err)
			return nil
		}
		pterm.Success.Printf("Successfully deleted browser with persistent ID: %s\n", persistentID)
	} else {
		pterm.Info.Printf("Deleting browser with ID: %s\n", sessionID)
		err := client.Browsers.DeleteByID(cmd.Context(), sessionID)
		if err != nil {
			pterm.Error.Printf("Failed to delete browser: %v\n", err)
			return nil
		}
		pterm.Success.Printf("Successfully deleted browser with ID: %s\n", sessionID)
	}

	return nil
}

func runBrowsersView(cmd *cobra.Command, args []string) error {
	client := getKernelClient(cmd)

	persistentID, _ := cmd.Flags().GetString("by-persistent-id")
	sessionID, _ := cmd.Flags().GetString("by-id")

	if persistentID == "" && sessionID == "" {
		return fmt.Errorf("must specify either --by-persistent-id or --by-id")
	}

	// List all browsers and filter client-side
	browsers, err := client.Browsers.List(cmd.Context())
	if err != nil {
		pterm.Error.Printf("Failed to list browsers: %v\n", err)
		return nil
	}

	if browsers == nil || len(*browsers) == 0 {
		pterm.Error.Println("No browsers found")
		return nil
	}

	// Find the matching browser
	var foundBrowser *kernel.BrowserListResponse
	for _, browser := range *browsers {
		if persistentID != "" && browser.Persistence.ID == persistentID {
			foundBrowser = &browser
			break
		} else if sessionID != "" && browser.SessionID == sessionID {
			foundBrowser = &browser
			break
		}
	}

	if foundBrowser == nil {
		if persistentID != "" {
			pterm.Error.Printf("Browser with persistent ID '%s' not found\n", persistentID)
		} else {
			pterm.Error.Printf("Browser with ID '%s' not found\n", sessionID)
		}
		return nil
	}

	// Output just the URL
	pterm.Info.Println(foundBrowser.BrowserLiveViewURL)
	return nil
}

func truncateURL(url string, maxLen int) string {
	if len(url) <= maxLen {
		return url
	}
	return url[:maxLen-3] + "..."
}
