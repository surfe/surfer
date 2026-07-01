package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Surfe/surfer/pkg/config"
	"github.com/spf13/viper"
)

var LoginTimeout = 2 * time.Minute

// httpClient is used for OAuth token exchange, refresh, and logout requests.
// It carries a timeout so a hung identity service can't block the CLI forever.
var httpClient = &http.Client{Timeout: 30 * time.Second}

// AuthBaseURL returns the base URL for authentication and API calls.
// Override via config file (auth-url), env var (SURFER_AUTH_URL), or --auth-url flag.
var AuthBaseURL = func() string {
	if v := viper.GetString("auth-url"); v != "" {
		return v
	}
	return "https://eu.prod.surfe.com"
}

// ClientID returns the OAuth client ID.
// Override via config file (client-id), env var (SURFER_CLIENT_ID), or --client-id flag.
var ClientID = func() string {
	if v := viper.GetString("client-id"); v != "" {
		return v
	}
	return "hubspot"
}

// tokenFilePath can be overridden in tests.
var tokenFilePath string

type TokenStore struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
	TokenType    string    `json:"token_type"`
	Scope        string    `json:"scope"`
}

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
}

func tokenFile() string {
	if tokenFilePath != "" {
		return tokenFilePath
	}
	return filepath.Join(config.GetDir(), "tokens.json")
}

func LoadTokens() (*TokenStore, error) {
	data, err := os.ReadFile(tokenFile())
	if err != nil {
		return nil, fmt.Errorf("not logged in — run 'surfer login' first")
	}
	var store TokenStore
	if err := json.Unmarshal(data, &store); err != nil {
		return nil, fmt.Errorf("corrupted token file: %w", err)
	}
	return &store, nil
}

func SaveTokens(store *TokenStore) error {
	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(tokenFile(), data, 0o600)
}

func ClearTokens() error {
	if err := os.Remove(tokenFile()); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// DecodeTokenClaims parses the claims from a JWT access token WITHOUT verifying
// its signature. It is for local display only (e.g. `surfer whoami`) — never use
// it for authorization decisions. Returns an error for opaque (non-JWT) tokens.
func DecodeTokenClaims(token string) (map[string]any, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("not a JWT")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("decode token payload: %w", err)
	}
	var claims map[string]any
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, fmt.Errorf("parse token claims: %w", err)
	}
	return claims, nil
}

func IsLoggedIn() bool {
	store, err := LoadTokens()
	if err != nil {
		return false
	}
	return store.AccessToken != ""
}

// GetAccessToken returns a valid access token, refreshing if expired.
// Falls back to SURFE_API_KEY env var for server-to-server usage.
func GetAccessToken() (string, error) {
	if key := os.Getenv("SURFE_API_KEY"); key != "" {
		return key, nil
	}

	store, err := LoadTokens()
	if err != nil {
		return "", err
	}

	if time.Now().Before(store.ExpiresAt.Add(-30 * time.Second)) {
		return store.AccessToken, nil
	}

	// Token expired or about to expire — refresh it
	if store.RefreshToken == "" {
		return "", fmt.Errorf("token expired and no refresh token available — run 'surfer login'")
	}

	newStore, err := RefreshTokens(store.RefreshToken)
	if err != nil {
		return "", fmt.Errorf("token refresh failed — run 'surfer login': %w", err)
	}

	if err := SaveTokens(newStore); err != nil {
		return "", fmt.Errorf("failed to save refreshed tokens: %w", err)
	}

	return newStore.AccessToken, nil
}

func generateCodeVerifier() (string, error) {
	// Match Python's secrets.token_urlsafe(64): 64 random bytes → base64url (86 chars)
	b := make([]byte, 64)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func generateCodeChallenge(verifier string) string {
	h := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(h[:])
}

// generateState returns a random opaque value for the OAuth state parameter.
func generateState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// StartLogin opens the browser for OAuth authorization and waits for the callback.
// Uses port 0 (OS-assigned) like wrangler/gh/gcloud to avoid port conflicts.
func StartLogin() (*TokenStore, error) {
	verifier, err := generateCodeVerifier()
	if err != nil {
		return nil, fmt.Errorf("generate PKCE verifier: %w", err)
	}
	challenge := generateCodeChallenge(verifier)

	state, err := generateState()
	if err != nil {
		return nil, fmt.Errorf("generate state: %w", err)
	}

	// Bind port 0 — OS picks a free port.
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("start callback server: %w", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	callbackURL := fmt.Sprintf("http://localhost:%d/callback", port)

	authURL := fmt.Sprintf("%s/oauth/authorize?client_id=%s&response_type=code&redirect_uri=%s&code_challenge=%s&code_challenge_method=S256&state=%s",
		AuthBaseURL(),
		url.QueryEscape(ClientID()),
		callbackURL,
		challenge,
		url.QueryEscape(state),
	)

	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		// Validate the OAuth state to guard against CSRF / login injection.
		// Only enforced when the server echoes a state back, since PKCE is the
		// primary protection for this loopback flow.
		if got := r.URL.Query().Get("state"); got != "" && got != state {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			fmt.Fprint(w, `<!DOCTYPE html>
<html><head><meta charset="utf-8"></head>
<body style="font-family:system-ui,sans-serif;text-align:center;padding:60px;color:#1a1a1a">
	<h2 style="color:#dc2626">&#10008; Authentication failed</h2>
	<p>State mismatch — possible CSRF. Please try again.</p>
	<p>You can close this tab.</p>
</body></html>`)
			errCh <- fmt.Errorf("authorization failed: state mismatch (possible CSRF)")
			return
		}

		code := r.URL.Query().Get("code")
		if code == "" {
			errMsg := r.URL.Query().Get("error")
			if errMsg == "" {
				errMsg = "no authorization code received"
			}
			errDesc := r.URL.Query().Get("error_description")
			if errDesc != "" {
				errMsg = errMsg + ": " + errDesc
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			fmt.Fprintf(w, `<!DOCTYPE html>
<html><head><meta charset="utf-8"></head>
<body style="font-family:system-ui,sans-serif;text-align:center;padding:60px;color:#1a1a1a">
	<h2 style="color:#dc2626">&#10008; Authentication failed</h2>
	<p>%s</p>
	<p>You can close this tab.</p>
</body></html>`, html.EscapeString(errMsg))
			errCh <- fmt.Errorf("authorization failed: %s", errMsg)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, `<!DOCTYPE html>
<html><head><meta charset="utf-8"></head>
<body style="font-family:system-ui,sans-serif;text-align:center;padding:60px;color:#1a1a1a">
	<h2 style="color:#16a34a">&#10004; Authentication successful!</h2>
	<p>This tab will close automatically...</p>
	<script>setTimeout(function(){window.close()},1500)</script>
</body></html>`)
		codeCh <- code
	})

	server := &http.Server{Handler: mux}

	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			errCh <- fmt.Errorf("callback server failed: %w", err)
		}
	}()

	// Open browser
	fmt.Printf("Opening browser to login...\n\n  %s\n\n", authURL)
	if err := openBrowser(authURL); err != nil {
		fmt.Printf("Could not open browser automatically. Copy the URL above.\n")
	}

	// Wait for callback
	var code string
	select {
	case code = <-codeCh:
	case err := <-errCh:
		_ = server.Close()
		return nil, err
	case <-time.After(LoginTimeout):
		_ = server.Close()
		return nil, fmt.Errorf("login timed out after 2 minutes")
	}

	// Give the browser a moment to receive the success HTML before shutting down.
	time.Sleep(500 * time.Millisecond)
	_ = server.Close()

	return exchangeCode(code, verifier)
}

func exchangeCode(code, verifier string) (*TokenStore, error) {
	data := url.Values{
		"grant_type":    {"authorization_code"},
		"client_id":     {ClientID()},
		"code":          {code},
		"code_verifier": {verifier},
	}

	resp, err := httpClient.PostForm(AuthBaseURL()+"/oauth/token", data)
	if err != nil {
		return nil, fmt.Errorf("token exchange request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errBody map[string]string
		_ = json.NewDecoder(resp.Body).Decode(&errBody)
		return nil, fmt.Errorf("token exchange failed (%d): %s", resp.StatusCode, errBody["error_description"])
	}

	var tokenResp tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("parse token response: %w", err)
	}

	return &TokenStore{
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		ExpiresAt:    time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second),
		TokenType:    tokenResp.TokenType,
		Scope:        tokenResp.Scope,
	}, nil
}

func RefreshTokens(refreshToken string) (*TokenStore, error) {
	data := url.Values{
		"grant_type":    {"refresh_token"},
		"client_id":     {ClientID()},
		"refresh_token": {refreshToken},
	}

	resp, err := httpClient.PostForm(AuthBaseURL()+"/oauth/token", data)
	if err != nil {
		return nil, fmt.Errorf("refresh request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errBody map[string]string
		_ = json.NewDecoder(resp.Body).Decode(&errBody)
		return nil, fmt.Errorf("refresh failed (%d): %s", resp.StatusCode, errBody["error_description"])
	}

	var tokenResp tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("parse refresh response: %w", err)
	}

	return &TokenStore{
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		ExpiresAt:    time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second),
		TokenType:    tokenResp.TokenType,
		Scope:        tokenResp.Scope,
	}, nil
}

func Logout(accessToken string) error {
	req, err := http.NewRequest("POST", AuthBaseURL()+"/oauth/logout", strings.NewReader(""))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("logout request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("logout failed with status %d", resp.StatusCode)
	}

	return nil
}
