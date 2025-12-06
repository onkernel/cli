package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/onkernel/cli/pkg/util"
	"github.com/onkernel/kernel-go-sdk"
	"github.com/onkernel/kernel-go-sdk/option"
	"github.com/pkg/browser"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

// AgentsAuthService defines the subset of the Kernel SDK agent auth client that we use.
type AgentsAuthService interface {
	Start(ctx context.Context, body kernel.AgentsAuthStartParams, opts ...option.RequestOption) (res *kernel.AgentsAuthStartResponse, err error)
	Retrieve(ctx context.Context, id string, opts ...option.RequestOption) (res *kernel.AgentsAuthRetrieveResponse, err error)
}

// AgentsAuthInvocationsService defines the subset we use for agent auth invocations.
type AgentsAuthInvocationsService interface {
	Retrieve(ctx context.Context, invocationId string, opts ...option.RequestOption) (res *kernel.AgentsAuthInvocationsRetrieveResponse, err error)
	Exchange(ctx context.Context, invocationId string, body kernel.AgentsAuthInvocationsExchangeParams, opts ...option.RequestOption) (res *kernel.AgentsAuthInvocationsExchangeResponse, err error)
	Discover(ctx context.Context, invocationId string, body kernel.AgentsAuthInvocationsDiscoverParams, opts ...option.RequestOption) (res *kernel.AgentsAuthInvocationsDiscoverResponse, err error)
	Submit(ctx context.Context, invocationId string, body kernel.AgentsAuthInvocationsSubmitParams, opts ...option.RequestOption) (res *kernel.AgentsAuthInvocationsSubmitResponse, err error)
}

type AgentsAuthStartInput struct {
	TargetDomain string
	ProfileName  string
	LoginURL     string
	ProxyID      string
	Hosted       bool
}

type AgentsAuthStatusInput struct {
	ID string
}

// AgentsAuthCmd handles agent auth operations independent of cobra.
type AgentsAuthCmd struct {
	auth        AgentsAuthService
	invocations AgentsAuthInvocationsService
	browsers    BrowsersService
}

func (a AgentsAuthCmd) Start(ctx context.Context, in AgentsAuthStartInput) error {
	pterm.Info.Println("Starting agent authentication flow...")
	pterm.Println(fmt.Sprintf("  Target domain: %s", in.TargetDomain))
	pterm.Println(fmt.Sprintf("  Profile name: %s", in.ProfileName))
	if in.LoginURL != "" {
		pterm.Println(fmt.Sprintf("  Login URL: %s", in.LoginURL))
	}
	if in.ProxyID != "" {
		pterm.Println(fmt.Sprintf("  Proxy ID: %s", in.ProxyID))
	}
	pterm.Println()

	params := kernel.AgentsAuthStartParams{
		TargetDomain: in.TargetDomain,
		ProfileName:  in.ProfileName,
	}
	if in.LoginURL != "" {
		params.LoginURL = kernel.Opt(in.LoginURL)
	}
	if in.ProxyID != "" {
		params.Proxy = kernel.AgentsAuthStartParamsProxy{
			ProxyID: kernel.Opt(in.ProxyID),
		}
	}

	startResp, err := a.auth.Start(ctx, params)
	if err != nil {
		return util.CleanedUpSdkError{Err: err}
	}

	pterm.Success.Println("Auth flow started successfully!")
	pterm.Println(fmt.Sprintf("  Invocation ID: %s", startResp.InvocationID))
	pterm.Println(fmt.Sprintf("  Auth Agent ID: %s", startResp.AuthAgentID))
	pterm.Println()

	if in.Hosted {
		return a.handleHostedMode(ctx, startResp)
	}

	return a.handleInteractiveMode(ctx, startResp)
}

func (a AgentsAuthCmd) handleHostedMode(ctx context.Context, startResp *kernel.AgentsAuthStartResponse) error {
	pterm.Info.Println("Hosted UI Mode")
	pterm.Println(strings.Repeat("=", 60))
	pterm.Println()
	pterm.Println("Please open this URL in your browser:")
	pterm.Println()
	pterm.Println(fmt.Sprintf("  %s", startResp.HostedURL))
	pterm.Println()
	pterm.Println(strings.Repeat("=", 60))
	pterm.Println()

	// Try to open browser automatically
	if err := browser.OpenURL(startResp.HostedURL); err != nil {
		pterm.Warning.Printf("Could not open browser automatically: %v\n", err)
		pterm.Info.Println("Please copy the URL above and open it manually.")
	}

	// Wait for user to confirm they've opened it
	pterm.DefaultInteractiveTextInput.DefaultText = "Press Enter once you've completed authentication in the browser..."
	_, _ = pterm.DefaultInteractiveTextInput.Show()
	pterm.Println()

	// Poll for completion
	pterm.Info.Println("Polling for completion...")
	pterm.Println("  Poll interval: 2s")
	pterm.Println("  Max wait time: 5 minutes")
	pterm.Println()

	startTime := time.Now()
	maxWaitTime := 5 * time.Minute
	pollInterval := 2 * time.Second

	for time.Since(startTime) < maxWaitTime {
		invocation, err := a.invocations.Retrieve(ctx, startResp.InvocationID)
		if err != nil {
			return util.CleanedUpSdkError{Err: err}
		}

		elapsed := int(time.Since(startTime).Seconds())
		pterm.Println(fmt.Sprintf("  [%ds] Status: %s", elapsed, invocation.Status))

		switch invocation.Status {
		case kernel.AgentsAuthInvocationsRetrieveResponseStatusSuccess:
			pterm.Println()
			pterm.Success.Println("Success! Profile is ready.")
			// Get profile name from auth agent
			authAgent, err := a.auth.Retrieve(ctx, startResp.AuthAgentID)
			if err != nil {
				return util.CleanedUpSdkError{Err: err}
			}
			return a.showSuccessAndOfferBrowser(ctx, startResp.AuthAgentID, authAgent.ProfileName)
		case kernel.AgentsAuthInvocationsRetrieveResponseStatusExpired:
			pterm.Println()
			pterm.Error.Println("Error: Invocation expired before completion")
			return nil
		case kernel.AgentsAuthInvocationsRetrieveResponseStatusCanceled:
			pterm.Println()
			pterm.Error.Println("Error: Invocation was canceled")
			return nil
		}

		time.Sleep(pollInterval)
	}

	pterm.Error.Println("Error: Polling timed out")
	return nil
}

func (a AgentsAuthCmd) handleInteractiveMode(ctx context.Context, startResp *kernel.AgentsAuthStartResponse) error {
	pterm.Info.Println("Interactive Mode")
	pterm.Println()

	// Step 2: Exchange handoff code for JWT
	pterm.Info.Println("Exchanging handoff code for JWT...")

	exchangeResp, err := a.invocations.Exchange(ctx, startResp.InvocationID, kernel.AgentsAuthInvocationsExchangeParams{
		Code: startResp.HandoffCode,
	})
	if err != nil {
		return util.CleanedUpSdkError{Err: err}
	}

	jwt := exchangeResp.JWT
	pterm.Success.Println("JWT obtained successfully!")
	pterm.Println()

	// Create JWT-authenticated client
	client := getKernelClientFromJWT(jwt)
	jwtInvocations := client.Agents.Auth.Invocations

	// Step 3: Discover login fields
	pterm.Info.Println("Discovering login fields...")
	pterm.Println()

	discoverParams := kernel.AgentsAuthInvocationsDiscoverParams{}
	if startResp.LoginURL != nil && *startResp.LoginURL != "" {
		discoverParams.LoginURL = kernel.Opt(*startResp.LoginURL)
	}

	discoverResp, err := jwtInvocations.Discover(ctx, startResp.InvocationID, discoverParams)
	if err != nil {
		return util.CleanedUpSdkError{Err: err}
	}

	if discoverResp.LoggedIn != nil && *discoverResp.LoggedIn {
		pterm.Success.Println("Already logged in! Profile saved.")
		// Get profile name from auth agent
		authAgent, err := a.auth.Retrieve(ctx, startResp.AuthAgentID)
		if err != nil {
			return util.CleanedUpSdkError{Err: err}
		}
		return a.showSuccessAndOfferBrowser(ctx, startResp.AuthAgentID, authAgent.ProfileName)
	}

	if discoverResp.Success == nil || !*discoverResp.Success {
		errorMsg := "Discovery failed"
		if discoverResp.ErrorMessage != nil {
			errorMsg = fmt.Sprintf("%s: %s", errorMsg, *discoverResp.ErrorMessage)
		}
		pterm.Error.Println(errorMsg)
		return nil
	}

	pterm.Success.Println("Login fields discovered!")
	if discoverResp.LoginURL != nil {
		pterm.Println(fmt.Sprintf("  Login URL: %s", *discoverResp.LoginURL))
	}
	if discoverResp.PageTitle != nil {
		pterm.Println(fmt.Sprintf("  Page title: %s", *discoverResp.PageTitle))
	}
	pterm.Println()

	fields := discoverResp.Fields
	if fields == nil || len(*fields) == 0 {
		pterm.Error.Println("No fields discovered!")
		return nil
	}

	pterm.Info.Println("Discovered fields:")
	for _, field := range *fields {
		label := "-"
		if field.Label != nil {
			label = *field.Label
		}
		pterm.Println(fmt.Sprintf("  - %s (type: %s, label: \"%s\")", field.Name, field.Type, label))
	}
	pterm.Println()

	// Step 4: Collect credentials
	pterm.Info.Println("Collecting credentials...")
	pterm.Println()

	userCredentials := make(map[string]string)
	for _, field := range *fields {
		fieldLabel := field.Name
		if field.Label != nil && *field.Label != "" {
			fieldLabel = *field.Label
		}

		isPassword := field.Type == "password" || strings.Contains(strings.ToLower(field.Name), "password")

		if isPassword {
			pterm.Warning.Println("  (Note: Password will be visible as you type)")
		}

		prompt := fmt.Sprintf("  Enter %s: ", fieldLabel)
		value, err := pterm.DefaultInteractiveTextInput.WithDefaultText(prompt).Show()
		if err != nil {
			return fmt.Errorf("failed to read input: %w", err)
		}
		userCredentials[field.Name] = value
	}

	pterm.Println()

	// Step 5: Submit credentials
	pterm.Info.Println("Submitting credentials...")
	pterm.Println()

	fieldValues := make(map[string]string)
	for _, field := range *fields {
		if val, ok := userCredentials[field.Name]; ok {
			fieldValues[field.Name] = val
		}
	}

	submitResp, err := jwtInvocations.Submit(ctx, startResp.InvocationID, kernel.AgentsAuthInvocationsSubmitParams{
		FieldValues: fieldValues,
	})
	if err != nil {
		return util.CleanedUpSdkError{Err: err}
	}

	// Handle multi-step auth flows
	for submitResp.NeedsAdditionalAuth != nil && *submitResp.NeedsAdditionalAuth {
		if submitResp.AdditionalFields == nil || len(*submitResp.AdditionalFields) == 0 {
			break
		}

		pterm.Info.Println("Additional authentication required!")
		pterm.Info.Println("Additional fields:")
		for _, field := range *submitResp.AdditionalFields {
			label := "-"
			if field.Label != nil {
				label = *field.Label
			}
			pterm.Println(fmt.Sprintf("  - %s (type: %s, label: \"%s\")", field.Name, field.Type, label))
		}
		pterm.Println()

		additionalValues := make(map[string]string)
		for _, field := range *submitResp.AdditionalFields {
			fieldLabel := field.Name
			if field.Label != nil && *field.Label != "" {
				fieldLabel = *field.Label
			}

			prompt := fmt.Sprintf("  Enter %s: ", fieldLabel)
			value, err := pterm.DefaultInteractiveTextInput.WithDefaultText(prompt).Show()
			if err != nil {
				return fmt.Errorf("failed to read input: %w", err)
			}
			additionalValues[field.Name] = value
		}

		pterm.Println()
		pterm.Info.Println("Submitting additional authentication...")

		submitResp, err = jwtInvocations.Submit(ctx, startResp.InvocationID, kernel.AgentsAuthInvocationsSubmitParams{
			FieldValues: additionalValues,
		})
		if err != nil {
			return util.CleanedUpSdkError{Err: err}
		}

		pterm.Println()
	}

	// Check final result
	if submitResp.LoggedIn != nil && *submitResp.LoggedIn {
		// Get profile name from auth agent
		authAgent, err := a.auth.Retrieve(ctx, startResp.AuthAgentID)
		if err != nil {
			return util.CleanedUpSdkError{Err: err}
		}
		return a.showSuccessAndOfferBrowser(ctx, startResp.AuthAgentID, authAgent.ProfileName)
	}

	if submitResp.ErrorMessage != nil {
		pterm.Error.Println(strings.Repeat("=", 60))
		pterm.Error.Println("LOGIN FAILED")
		pterm.Error.Println(strings.Repeat("=", 60))
		pterm.Error.Printf("Error: %s\n", *submitResp.ErrorMessage)
		return nil
	}

	pterm.Error.Println("Unexpected state - not logged in but no error message")
	return nil
}

func (a AgentsAuthCmd) showSuccessAndOfferBrowser(ctx context.Context, authAgentID string, profileName string) error {
	pterm.Success.Println(strings.Repeat("=", 60))
	pterm.Success.Println("SUCCESS! Profile saved and ready for use.")
	pterm.Success.Println(strings.Repeat("=", 60))
	pterm.Println()

	// Verify auth agent status
	pterm.Info.Println("Verifying auth agent status...")
	authAgent, err := a.auth.Retrieve(ctx, authAgentID)
	if err != nil {
		return util.CleanedUpSdkError{Err: err}
	}

	pterm.Println(fmt.Sprintf("  Auth Agent ID: %s", authAgent.ID))
	pterm.Println(fmt.Sprintf("  Profile: %s", authAgent.ProfileName))
	pterm.Println(fmt.Sprintf("  Domain: %s", authAgent.Domain))
	pterm.Println(fmt.Sprintf("  Status: %s", authAgent.Status))

	if authAgent.Status != kernel.AgentsAuthRetrieveResponseStatusAuthenticated {
		pterm.Warning.Printf("Warning: Expected status AUTHENTICATED, got %s\n", authAgent.Status)
	} else {
		pterm.Success.Println("Auth agent status confirmed: AUTHENTICATED")
	}
	pterm.Println()

	pterm.Info.Printf("You can now create browsers with profile: %s\n", profileName)
	pterm.Println()

	// Offer to create browser
	pterm.DefaultInteractiveConfirm.DefaultText = "Would you like to create a browser with the saved profile? (y/n)"
	result, _ := pterm.DefaultInteractiveConfirm.Show()

	if result {
		pterm.Println()
		pterm.Info.Println("Creating browser with saved profile...")

		if a.browsers == nil {
			pterm.Warning.Println("Browser service not available")
			return nil
		}

		browserResp, err := a.browsers.New(ctx, kernel.BrowserNewParams{
			Stealth: kernel.Opt(true),
			Profile: kernel.BrowserProfileParam{
				Name: kernel.Opt(profileName),
			},
		})
		if err != nil {
			return util.CleanedUpSdkError{Err: err}
		}

		pterm.Success.Println("Browser created successfully!")
		pterm.Println(fmt.Sprintf("  Session ID: %s", browserResp.SessionID))
		pterm.Println(fmt.Sprintf("  CDP WebSocket URL: %s", browserResp.CdpWsURL))
		if browserResp.BrowserLiveViewURL != "" {
			pterm.Println(fmt.Sprintf("  Live View URL: %s", browserResp.BrowserLiveViewURL))
		}
		pterm.Println()
	}

	return nil
}

func (a AgentsAuthCmd) Status(ctx context.Context, in AgentsAuthStatusInput) error {
	authAgent, err := a.auth.Retrieve(ctx, in.ID)
	if err != nil {
		return util.CleanedUpSdkError{Err: err}
	}

	rows := pterm.TableData{{"Property", "Value"}}
	rows = append(rows, []string{"ID", authAgent.ID})
	rows = append(rows, []string{"Profile Name", authAgent.ProfileName})
	rows = append(rows, []string{"Domain", authAgent.Domain})
	rows = append(rows, []string{"Status", string(authAgent.Status)})

	PrintTableNoPad(rows, true)
	return nil
}

// getKernelClientFromJWT creates a new Kernel client with a JWT token
func getKernelClientFromJWT(jwt string) kernel.Client {
	return kernel.NewClient(option.WithAPIKey(jwt))
}

// --- Cobra wiring ---

var agentsCmd = &cobra.Command{
	Use:   "agents",
	Short: "Manage agents",
	Long:  "Commands for managing Kernel agents",
}

var agentsAuthCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage agent authentication",
	Long:  "Commands for managing agent authentication flows",
}

var agentsAuthStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start an agent authentication flow",
	Long:  "Start an interactive authentication flow for a website. Use --hosted to open the authentication flow in a browser.",
	Args:  cobra.NoArgs,
	RunE:  runAgentsAuthStart,
}

var agentsAuthStatusCmd = &cobra.Command{
	Use:   "status <auth-agent-id>",
	Short: "Get auth agent status",
	Args:  cobra.ExactArgs(1),
	RunE:  runAgentsAuthStatus,
}

func init() {
	agentsAuthCmd.AddCommand(agentsAuthStartCmd)
	agentsAuthCmd.AddCommand(agentsAuthStatusCmd)
	agentsCmd.AddCommand(agentsAuthCmd)

	agentsAuthStartCmd.Flags().String("target-domain", "", "Target domain to authenticate with (required)")
	agentsAuthStartCmd.Flags().String("profile-name", "", "Profile name to use or create (required)")
	agentsAuthStartCmd.Flags().String("login-url", "", "Optional login URL to skip discovery")
	agentsAuthStartCmd.Flags().String("proxy-id", "", "Optional proxy ID to use")
	agentsAuthStartCmd.Flags().Bool("hosted", false, "Use hosted UI mode (opens browser)")

	_ = agentsAuthStartCmd.MarkFlagRequired("target-domain")
	_ = agentsAuthStartCmd.MarkFlagRequired("profile-name")
}

func runAgentsAuthStart(cmd *cobra.Command, args []string) error {
	client := getKernelClient(cmd)
	targetDomain, _ := cmd.Flags().GetString("target-domain")
	profileName, _ := cmd.Flags().GetString("profile-name")
	loginURL, _ := cmd.Flags().GetString("login-url")
	proxyID, _ := cmd.Flags().GetString("proxy-id")
	hosted, _ := cmd.Flags().GetBool("hosted")

	svc := client.Agents.Auth
	invocationsSvc := client.Agents.Auth.Invocations
	browsersSvc := client.Browsers
	a := AgentsAuthCmd{auth: &svc, invocations: &invocationsSvc, browsers: &browsersSvc}

	return a.Start(cmd.Context(), AgentsAuthStartInput{
		TargetDomain: targetDomain,
		ProfileName:  profileName,
		LoginURL:     loginURL,
		ProxyID:      proxyID,
		Hosted:       hosted,
	})
}

func runAgentsAuthStatus(cmd *cobra.Command, args []string) error {
	client := getKernelClient(cmd)
	svc := client.Agents.Auth
	a := AgentsAuthCmd{auth: &svc}

	return a.Status(cmd.Context(), AgentsAuthStatusInput{
		ID: args[0],
	})
}
