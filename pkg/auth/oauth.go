package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/pkg/browser"
	"github.com/pterm/pterm"
	"golang.org/x/oauth2"
)

//go:embed success.html
var successHTML string

const (
	// MCP Server OAuth endpoints (which proxy to Clerk)
	AuthURL  = "https://auth.onkernel.com/authorize"
	TokenURL = "https://auth.onkernel.com/token"

	// OAuth client configuration
	ClientID    = "hmFrJn9hKDV2N02M" // Prod Kernel CLI OAuth Client ID
	RedirectURI = "http://localhost"

	// OAuth scopes - openid for the MCP server flow
	DefaultScope = "openid email"
)

// OAuthConfig represents the OAuth2 configuration
type OAuthConfig struct {
	Config   *oauth2.Config
	Verifier string
	State    string
}

// TokenResponse represents the OAuth token response
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
	OrgID        string `json:"org_id"`
}

// AuthResult represents the result data passed through the callback channel
type AuthResult struct {
	Code  string `json:"code"`
	OrgID string `json:"org_id,omitempty"`
}

// NewOAuthConfig creates a new OAuth configuration with PKCE
func NewOAuthConfig() (*OAuthConfig, error) {
	// Generate PKCE code verifier and challenge
	verifier, err := generateCodeVerifier()
	if err != nil {
		return nil, fmt.Errorf("failed to generate code verifier: %w", err)
	}

	// Generate random CSRF token for state protection
	csrfToken, err := generateRandomString(32)
	if err != nil {
		return nil, fmt.Errorf("failed to generate CSRF token: %w", err)
	}

	// Create state as base64-encoded JSON containing the CSRF token
	stateData := map[string]string{
		"csrf": csrfToken,
	}
	stateJSON, err := json.Marshal(stateData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal state data: %w", err)
	}
	state := base64.StdEncoding.EncodeToString(stateJSON)

	// Try to find an available port from our allowed range
	// Note: We'll get the actual port later when starting the server to avoid race conditions
	redirectURI := fmt.Sprintf("%s:0/callback", RedirectURI)

	config := &oauth2.Config{
		ClientID:    ClientID,
		RedirectURL: redirectURI,
		Scopes:      strings.Split(DefaultScope, " "),
		Endpoint: oauth2.Endpoint{
			AuthURL:  AuthURL,
			TokenURL: TokenURL,
		},
	}

	return &OAuthConfig{
		Config:   config,
		Verifier: verifier,
		State:    state, // Store the encoded state for OAuth URL
	}, nil
}

// StartOAuthFlow initiates the OAuth flow with browser redirect
func (oc *OAuthConfig) StartOAuthFlow(ctx context.Context) (*TokenStorage, error) {
	// Find an available port and get a listener to prevent race conditions
	listener, port, err := findAvailablePortListener()
	if err != nil {
		return nil, fmt.Errorf("failed to find available port: %w", err)
	}
	// Note: listener will be closed when server.Shutdown() is called

	// Update the config with the actual port
	oc.Config.RedirectURL = fmt.Sprintf("%s:%d/callback", RedirectURI, port)

	// Generate authorization URL with PKCE
	challenge := generateCodeChallenge(oc.Verifier)
	authURL := oc.Config.AuthCodeURL(oc.State,
		oauth2.SetAuthURLParam("code_challenge", challenge),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
	)

	// Print URL immediately for manual access (especially useful for headless environments)
	pterm.Info.Println("Authentication URL:")
	// Use ANSI hyperlink for modern terminals, falls back to plain URL for others
	pterm.Printf("  \033]8;;%s\033\\%s\033]8;;\033\\\n\n", authURL, authURL)

	// Try to open browser automatically
	if err := browser.OpenURL(authURL); err != nil {
		// Browser launch failed - likely a headless/server environment
		pterm.Warning.Println("Could not open browser automatically")
		pterm.Info.Println("Please manually open the URL above (Cmd/Ctrl+Click if supported)")
	}

	// Start local callback server
	codeChan := make(chan string, 1)
	errChan := make(chan error, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		// Extract and decode state parameter to get CSRF token and org_id
		encodedState := r.URL.Query().Get("state")
		var csrfToken, orgID string

		if encodedState != "" {
			// Try to decode the state parameter
			if decodedBytes, err := base64.StdEncoding.DecodeString(encodedState); err == nil {
				var stateData map[string]string
				if json.Unmarshal(decodedBytes, &stateData) == nil {
					csrfToken = stateData["csrf"]
					orgID = stateData["org_id"]
				}
			}

			// Fallback to treating the entire state as CSRF token if decoding fails
			if csrfToken == "" {
				csrfToken = encodedState
			}
		}

		// Verify CSRF token to prevent CSRF attacks
		// Extract the expected CSRF token from our stored state
		var expectedCSRF string
		if decodedBytes, err := base64.StdEncoding.DecodeString(oc.State); err == nil {
			var stateData map[string]string
			if json.Unmarshal(decodedBytes, &stateData) == nil {
				expectedCSRF = stateData["csrf"]
			}
		}

		if csrfToken != expectedCSRF || expectedCSRF == "" {
			http.Error(w, "Invalid state parameter", http.StatusBadRequest)
			errChan <- fmt.Errorf("invalid state parameter")
			return
		}

		code := r.URL.Query().Get("code")
		if code == "" {
			http.Error(w, "Missing authorization code", http.StatusBadRequest)
			errChan <- fmt.Errorf("missing authorization code")
			return
		}

		// Success page
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(successHTML))

		// Pass both code and org_id to the channel using JSON encoding
		result := AuthResult{
			Code:  code,
			OrgID: orgID,
		}
		resultJSON, err := json.Marshal(result)
		if err != nil {
			errChan <- fmt.Errorf("failed to encode auth result: %w", err)
			return
		}
		codeChan <- string(resultJSON)
	})

	server := &http.Server{Handler: mux}

	// Start server in background using our already-bound listener
	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			errChan <- fmt.Errorf("callback server error: %w", err)
		}
	}()

	// Wait for callback or timeout
	var authCode, orgID string
	select {
	case resultJSON := <-codeChan:
		// Success - shutdown server
		server.Shutdown(context.Background())
		// Parse JSON result containing both code and org_id
		var result AuthResult
		if err := json.Unmarshal([]byte(resultJSON), &result); err != nil {
			return nil, fmt.Errorf("failed to decode auth result: %w", err)
		}
		authCode = result.Code
		orgID = result.OrgID
	case err := <-errChan:
		server.Shutdown(context.Background())
		return nil, err
	case <-time.After(5 * time.Minute):
		server.Shutdown(context.Background())
		return nil, fmt.Errorf("authentication timeout after 5 minutes")
	case <-ctx.Done():
		server.Shutdown(context.Background())
		return nil, ctx.Err()
	}

	// Exchange authorization code for tokens
	return oc.exchangeCodeForTokens(ctx, authCode, orgID)
}

// exchangeCodeForTokens exchanges the authorization code for access and refresh tokens
func (oc *OAuthConfig) exchangeCodeForTokens(ctx context.Context, code, orgID string) (*TokenStorage, error) {
	// Use PKCE verifier in token exchange, and include org_id if available
	var opts []oauth2.AuthCodeOption
	opts = append(opts, oauth2.SetAuthURLParam("code_verifier", oc.Verifier))
	if orgID != "" {
		opts = append(opts, oauth2.SetAuthURLParam("org_id", orgID))
	}

	token, err := oc.Config.Exchange(ctx, code, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code for token: %w", err)
	}

	return &TokenStorage{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		ExpiresAt:    token.Expiry,
		OrgID:        orgID,
	}, nil
}

// RefreshTokens refreshes the access token using the refresh token
func RefreshTokens(ctx context.Context, tokens *TokenStorage) (*TokenStorage, error) {
	if tokens.RefreshToken == "" {
		return nil, fmt.Errorf("no refresh token available")
	}

	values := url.Values{}
	values.Set("grant_type", "refresh_token")
	values.Set("refresh_token", tokens.RefreshToken)
	values.Set("client_id", ClientID)
	values.Set("scope", DefaultScope)

	// Make the token request manually to ensure client_id is included
	req, err := http.NewRequestWithContext(ctx, "POST", TokenURL, strings.NewReader(values.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create refresh request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send refresh request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("refresh request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse the response
	var tokenResponse map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResponse); err != nil {
		return nil, fmt.Errorf("failed to decode refresh response: %w", err)
	}

	// Create oauth2.Token from response for compatibility
	newToken := &oauth2.Token{}
	if accessToken, ok := tokenResponse["access_token"].(string); ok {
		newToken.AccessToken = accessToken
	}
	if refreshToken, ok := tokenResponse["refresh_token"].(string); ok {
		newToken.RefreshToken = refreshToken
	}
	if expiresIn, ok := tokenResponse["expires_in"].(float64); ok {
		newToken.Expiry = time.Now().Add(time.Duration(expiresIn) * time.Second)
	}
	// Add extra fields
	newToken = newToken.WithExtra(tokenResponse)

	return &TokenStorage{
		AccessToken:  newToken.AccessToken,
		RefreshToken: newToken.RefreshToken,
		ExpiresAt:    newToken.Expiry,
	}, nil
}

// generateCodeVerifier generates a cryptographically secure random string for PKCE
func generateCodeVerifier() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(bytes), nil
}

// generateCodeChallenge generates the code challenge from the verifier
func generateCodeChallenge(verifier string) string {
	hash := sha256.Sum256([]byte(verifier))
	return base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(hash[:])
}

// findAvailablePortListener tries to find an available port and returns a listener
// This prevents race conditions by keeping the port bound until the server is ready
func findAvailablePortListener() (net.Listener, int, error) {
	// Uncommon ports that are unlikely to conflict with dev servers
	// These should be added to your Clerk redirect URIs
	preferredPorts := []int{58432, 58433, 58434, 58435, 58436, 58437, 58438, 58439}

	for _, port := range preferredPorts {
		listener, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", port))
		if err == nil {
			return listener, port, nil
		}
	}

	// If all preferred ports are taken, try any available port
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return nil, 0, fmt.Errorf("no available ports found")
	}
	port := listener.Addr().(*net.TCPAddr).Port
	return listener, port, nil
}

// generateRandomString generates a cryptographically secure random string
func generateRandomString(length int) (string, error) {
	// Base64 encoding expands data by 4/3, so to get at least 'length' characters,
	// we need at least (length * 3 + 3) / 4 bytes (adding 3 for ceiling division)
	bytesNeeded := (length*3 + 3) / 4

	bytes := make([]byte, bytesNeeded)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	encoded := base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(bytes)
	if len(encoded) > length {
		encoded = encoded[:length]
	}
	return encoded, nil
}
