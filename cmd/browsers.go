package cmd

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/onkernel/kernel-go-sdk"
	"github.com/onkernel/kernel-go-sdk/option"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

// BrowsersService defines the subset of the Kernel SDK browser client that we use.
// See https://github.com/onkernel/kernel-go-sdk/blob/main/browser.go
type BrowsersService interface {
	List(ctx context.Context, opts ...option.RequestOption) (res *[]kernel.BrowserListResponse, err error)
	New(ctx context.Context, body kernel.BrowserNewParams, opts ...option.RequestOption) (res *kernel.BrowserNewResponse, err error)
	Delete(ctx context.Context, body kernel.BrowserDeleteParams, opts ...option.RequestOption) (err error)
	DeleteByID(ctx context.Context, id string, opts ...option.RequestOption) (err error)
}

// BoolFlag captures whether a boolean flag was set explicitly and its value.
type BoolFlag struct {
	Set   bool
	Value bool
}

// Inputs for each command
type BrowsersCreateInput struct {
	PersistenceID  string
	TimeoutSeconds int
	Stealth        BoolFlag
	Headless       BoolFlag
}

type BrowsersDeleteInput struct {
	Identifier  string
	SkipConfirm bool
}

type BrowsersViewInput struct {
	Identifier string
}

// BrowsersCmd is a cobra-independent command handler for browsers operations.
type BrowsersCmd struct {
	browsers BrowsersService
}

func (b BrowsersCmd) List(ctx context.Context) error {
	pterm.Info.Println("Fetching browsers...")

	browsers, err := b.browsers.List(ctx)
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

func (b BrowsersCmd) Create(ctx context.Context, in BrowsersCreateInput) error {
	pterm.Info.Println("Creating browser session...")

	params := kernel.BrowserNewParams{}

	if in.PersistenceID != "" {
		params.Persistence = kernel.BrowserPersistenceParam{ID: in.PersistenceID}
	}
	if in.TimeoutSeconds > 0 {
		params.TimeoutSeconds = kernel.Opt(int64(in.TimeoutSeconds))
	}
	if in.Stealth.Set {
		params.Stealth = kernel.Opt(in.Stealth.Value)
	}
	if in.Headless.Set {
		params.Headless = kernel.Opt(in.Headless.Value)
	}

	browser, err := b.browsers.New(ctx, params)
	if err != nil {
		pterm.Error.Printf("Failed to create browser: %v\n", err)
		return nil
	}

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

func (b BrowsersCmd) Delete(ctx context.Context, in BrowsersDeleteInput) error {
	isNotFound := func(err error) bool {
		if err == nil {
			return false
		}
		var apierr *kernel.Error
		if errors.As(err, &apierr) {
			return apierr != nil && apierr.StatusCode == http.StatusNotFound
		}
		return false
	}

	if !in.SkipConfirm {
		browsers, err := b.browsers.List(ctx)
		if err != nil {
			pterm.Error.Printf("Failed to list browsers: %v\n", err)
			return nil
		}
		if browsers == nil || len(*browsers) == 0 {
			pterm.Error.Println("No browsers found")
			return nil
		}

		var found *kernel.BrowserListResponse
		for _, br := range *browsers {
			if br.SessionID == in.Identifier || br.Persistence.ID == in.Identifier {
				bCopy := br
				found = &bCopy
				break
			}
		}
		if found == nil {
			pterm.Error.Printf("Browser '%s' not found\n", in.Identifier)
			return nil
		}

		var confirmMsg string
		if found.Persistence.ID == in.Identifier {
			confirmMsg = fmt.Sprintf("Are you sure you want to delete browser with persistent ID \"%s\"?", in.Identifier)
		} else {
			confirmMsg = fmt.Sprintf("Are you sure you want to delete browser with ID \"%s\"?", in.Identifier)
		}
		pterm.DefaultInteractiveConfirm.DefaultText = confirmMsg
		result, _ := pterm.DefaultInteractiveConfirm.Show()
		if !result {
			pterm.Info.Println("Deletion cancelled")
			return nil
		}

		if found.Persistence.ID == in.Identifier {
			pterm.Info.Printf("Deleting browser with persistent ID: %s\n", in.Identifier)
			err = b.browsers.Delete(ctx, kernel.BrowserDeleteParams{PersistentID: in.Identifier})
			if err != nil && !isNotFound(err) {
				pterm.Error.Printf("Failed to delete browser: %v\n", err)
				return nil
			}
			pterm.Success.Printf("Successfully deleted browser with persistent ID: %s\n", in.Identifier)
			return nil
		}

		pterm.Info.Printf("Deleting browser with ID: %s\n", in.Identifier)
		err = b.browsers.DeleteByID(ctx, in.Identifier)
		if err != nil && !isNotFound(err) {
			pterm.Error.Printf("Failed to delete browser: %v\n", err)
			return nil
		}
		pterm.Success.Printf("Successfully deleted browser with ID: %s\n", in.Identifier)
		return nil
	}

	// Skip confirmation: try both deletion modes without listing first
	// Treat not found as a success (idempotent delete)
	var nonNotFoundErrors []error

	// Attempt by session ID
	if err := b.browsers.DeleteByID(ctx, in.Identifier); err != nil {
		if !isNotFound(err) {
			nonNotFoundErrors = append(nonNotFoundErrors, err)
		}
	}

	// Attempt by persistent ID
	if err := b.browsers.Delete(ctx, kernel.BrowserDeleteParams{PersistentID: in.Identifier}); err != nil {
		if !isNotFound(err) {
			nonNotFoundErrors = append(nonNotFoundErrors, err)
		}
	}

	if len(nonNotFoundErrors) >= 2 {
		// Both failed with meaningful errors; report one
		pterm.Error.Printf("Failed to delete browser: %v\n", nonNotFoundErrors[0])
		return nil
	}

	pterm.Success.Printf("Successfully deleted (or already absent) browser: %s\n", in.Identifier)
	return nil
}

func (b BrowsersCmd) View(ctx context.Context, in BrowsersViewInput) error {
	browsers, err := b.browsers.List(ctx)
	if err != nil {
		pterm.Error.Printf("Failed to list browsers: %v\n", err)
		return nil
	}

	if browsers == nil || len(*browsers) == 0 {
		pterm.Error.Println("No browsers found")
		return nil
	}

	var foundBrowser *kernel.BrowserListResponse
	for _, browser := range *browsers {
		if browser.Persistence.ID == in.Identifier || browser.SessionID == in.Identifier {
			foundBrowser = &browser
			break
		}
	}

	if foundBrowser == nil {
		pterm.Error.Printf("Browser '%s' not found\n", in.Identifier)
		return nil
	}

	// Output just the URL
	pterm.Info.Println(foundBrowser.BrowserLiveViewURL)
	return nil
}

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
	Use:   "delete <id-or-persistent-id>",
	Short: "Delete a browser",
	Args:  cobra.ExactArgs(1),
	RunE:  runBrowsersDelete,
}

var browsersViewCmd = &cobra.Command{
	Use:   "view <id-or-persistent-id>",
	Short: "Get the live view URL for a browser",
	Args:  cobra.ExactArgs(1),
	RunE:  runBrowsersView,
}

func init() {
	browsersCmd.AddCommand(browsersListCmd)
	browsersCmd.AddCommand(browsersCreateCmd)
	browsersCmd.AddCommand(browsersDeleteCmd)
	browsersCmd.AddCommand(browsersViewCmd)

	// Add flags for create command
	browsersCreateCmd.Flags().StringP("persistent-id", "p", "", "Unique identifier for browser session persistence")
	browsersCreateCmd.Flags().BoolP("stealth", "s", false, "Launch browser in stealth mode to avoid detection")
	browsersCreateCmd.Flags().BoolP("headless", "H", false, "Launch browser without GUI access")
	browsersCreateCmd.Flags().IntP("timeout", "t", 60, "Timeout in seconds for the browser session")

	// Add flags for delete command
	browsersDeleteCmd.Flags().BoolP("yes", "y", false, "Skip confirmation prompt")

	// no flags for view; it takes a single positional argument
}

func runBrowsersList(cmd *cobra.Command, args []string) error {
	client := getKernelClient(cmd)
	svc := client.Browsers
	b := BrowsersCmd{browsers: &svc}
	return b.List(cmd.Context())
}

func runBrowsersCreate(cmd *cobra.Command, args []string) error {
	client := getKernelClient(cmd)

	// Get flag values
	persistenceID, _ := cmd.Flags().GetString("persistent-id")
	stealthVal, _ := cmd.Flags().GetBool("stealth")
	headlessVal, _ := cmd.Flags().GetBool("headless")
	timeout, _ := cmd.Flags().GetInt("timeout")

	in := BrowsersCreateInput{
		PersistenceID:  persistenceID,
		TimeoutSeconds: timeout,
		Stealth:        BoolFlag{Set: cmd.Flags().Changed("stealth"), Value: stealthVal},
		Headless:       BoolFlag{Set: cmd.Flags().Changed("headless"), Value: headlessVal},
	}

	svc := client.Browsers
	b := BrowsersCmd{browsers: &svc}
	return b.Create(cmd.Context(), in)
}

func runBrowsersDelete(cmd *cobra.Command, args []string) error {
	client := getKernelClient(cmd)

	identifier := args[0]
	skipConfirm, _ := cmd.Flags().GetBool("yes")

	in := BrowsersDeleteInput{Identifier: identifier, SkipConfirm: skipConfirm}
	svc := client.Browsers
	b := BrowsersCmd{browsers: &svc}
	return b.Delete(cmd.Context(), in)
}

func runBrowsersView(cmd *cobra.Command, args []string) error {
	client := getKernelClient(cmd)

	identifier := args[0]

	in := BrowsersViewInput{Identifier: identifier}
	svc := client.Browsers
	b := BrowsersCmd{browsers: &svc}
	return b.View(cmd.Context(), in)
}

func truncateURL(url string, maxLen int) string {
	if len(url) <= maxLen {
		return url
	}
	return url[:maxLen-3] + "..."
}
