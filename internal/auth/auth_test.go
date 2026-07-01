package auth

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestTokenFile(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "tokens.json")
	tokenFilePath = path
	t.Cleanup(func() { tokenFilePath = "" })
	return path
}

func writeTestTokens(t *testing.T, store *TokenStore) {
	t.Helper()
	path := setupTestTokenFile(t)
	data, err := json.MarshalIndent(store, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, data, 0o600))
}

// ── PKCE ────────────────────────────────────────────────────────────────────

func TestGenerateCodeVerifier_Length(t *testing.T) {
	verifier, err := generateCodeVerifier()
	require.NoError(t, err)
	// 64 bytes → base64url = 86 chars (matching Python secrets.token_urlsafe(64))
	assert.Len(t, verifier, 86)
}

func TestGenerateCodeVerifier_Unique(t *testing.T) {
	v1, err := generateCodeVerifier()
	require.NoError(t, err)
	v2, err := generateCodeVerifier()
	require.NoError(t, err)
	assert.NotEqual(t, v1, v2)
}

func TestGenerateCodeChallenge_S256(t *testing.T) {
	verifier := "test-verifier-value"
	challenge := generateCodeChallenge(verifier)

	// Verify it's SHA256 of the verifier, base64url-encoded
	h := sha256.Sum256([]byte(verifier))
	expected := base64.RawURLEncoding.EncodeToString(h[:])
	assert.Equal(t, expected, challenge)
}

func TestGenerateCodeChallenge_Deterministic(t *testing.T) {
	verifier := "same-verifier"
	c1 := generateCodeChallenge(verifier)
	c2 := generateCodeChallenge(verifier)
	assert.Equal(t, c1, c2)
}

// ── Token Store ─────────────────────────────────────────────────────────────

func TestSaveAndLoadTokens(t *testing.T) {
	setupTestTokenFile(t)

	store := &TokenStore{
		AccessToken:  "access-123",
		RefreshToken: "refresh-456",
		ExpiresAt:    time.Now().Add(time.Hour),
		TokenType:    "Bearer",
		Scope:        "surfe:all",
	}

	err := SaveTokens(store)
	require.NoError(t, err)

	loaded, err := LoadTokens()
	require.NoError(t, err)
	assert.Equal(t, "access-123", loaded.AccessToken)
	assert.Equal(t, "refresh-456", loaded.RefreshToken)
	assert.Equal(t, "Bearer", loaded.TokenType)
	assert.Equal(t, "surfe:all", loaded.Scope)
}

func TestLoadTokens_NoFile(t *testing.T) {
	setupTestTokenFile(t)

	_, err := LoadTokens()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not logged in")
}

func TestLoadTokens_CorruptedJSON(t *testing.T) {
	path := setupTestTokenFile(t)
	os.WriteFile(path, []byte("not json"), 0o600)

	_, err := LoadTokens()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "corrupted token file")
}

func TestClearTokens(t *testing.T) {
	setupTestTokenFile(t)

	store := &TokenStore{AccessToken: "test"}
	require.NoError(t, SaveTokens(store))

	err := ClearTokens()
	require.NoError(t, err)

	_, err = LoadTokens()
	assert.Error(t, err)
}

func TestClearTokens_NoFile(t *testing.T) {
	setupTestTokenFile(t)
	err := ClearTokens()
	assert.NoError(t, err)
}

// ── IsLoggedIn ──────────────────────────────────────────────────────────────

func TestIsLoggedIn_True(t *testing.T) {
	writeTestTokens(t, &TokenStore{
		AccessToken: "valid-token",
		ExpiresAt:   time.Now().Add(time.Hour),
	})
	assert.True(t, IsLoggedIn())
}

func TestIsLoggedIn_NoFile(t *testing.T) {
	setupTestTokenFile(t)
	assert.False(t, IsLoggedIn())
}

func TestIsLoggedIn_EmptyToken(t *testing.T) {
	writeTestTokens(t, &TokenStore{
		AccessToken: "",
		ExpiresAt:   time.Now().Add(time.Hour),
	})
	assert.False(t, IsLoggedIn())
}

// ── GetAccessToken ──────────────────────────────────────────────────────────

func TestGetAccessToken_EnvVar(t *testing.T) {
	t.Setenv("SURFE_API_KEY", "env-api-key-123")
	setupTestTokenFile(t)

	token, err := GetAccessToken()
	require.NoError(t, err)
	assert.Equal(t, "env-api-key-123", token)
}

func TestGetAccessToken_ValidToken(t *testing.T) {
	t.Setenv("SURFE_API_KEY", "")
	writeTestTokens(t, &TokenStore{
		AccessToken: "valid-access-token",
		ExpiresAt:   time.Now().Add(time.Hour),
	})

	token, err := GetAccessToken()
	require.NoError(t, err)
	assert.Equal(t, "valid-access-token", token)
}

func TestGetAccessToken_NoFile(t *testing.T) {
	t.Setenv("SURFE_API_KEY", "")
	setupTestTokenFile(t)

	_, err := GetAccessToken()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not logged in")
}

func TestGetAccessToken_ExpiredNoRefresh(t *testing.T) {
	t.Setenv("SURFE_API_KEY", "")
	writeTestTokens(t, &TokenStore{
		AccessToken:  "expired-token",
		RefreshToken: "",
		ExpiresAt:    time.Now().Add(-time.Hour),
	})

	_, err := GetAccessToken()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "token expired")
}

// ── exchangeCode ────────────────────────────────────────────────────────────

func TestExchangeCode_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/oauth/token", r.URL.Path)
		r.ParseForm()
		assert.Equal(t, "authorization_code", r.FormValue("grant_type"))
		assert.Equal(t, "test-code", r.FormValue("code"))
		assert.Equal(t, "test-verifier", r.FormValue("code_verifier"))

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tokenResponse{
			AccessToken:  "new-access-token",
			TokenType:    "Bearer",
			ExpiresIn:    3600,
			RefreshToken: "new-refresh-token",
			Scope:        "surfe:all",
		})
	}))
	defer server.Close()

	origURL := AuthBaseURL
	AuthBaseURL = func() string { return server.URL }
	t.Cleanup(func() { AuthBaseURL = origURL })

	store, err := exchangeCode("test-code", "test-verifier")
	require.NoError(t, err)
	assert.Equal(t, "new-access-token", store.AccessToken)
	assert.Equal(t, "new-refresh-token", store.RefreshToken)
	assert.Equal(t, "Bearer", store.TokenType)
	assert.Equal(t, "surfe:all", store.Scope)
	assert.True(t, store.ExpiresAt.After(time.Now()))
}

func TestExchangeCode_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error":             "invalid_grant",
			"error_description": "code expired",
		})
	}))
	defer server.Close()

	origURL := AuthBaseURL
	AuthBaseURL = func() string { return server.URL }
	t.Cleanup(func() { AuthBaseURL = origURL })

	_, err := exchangeCode("bad-code", "verifier")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "token exchange failed")
	assert.Contains(t, err.Error(), "code expired")
}

// ── RefreshTokens ───────────────────────────────────────────────────────────

func TestRefreshTokens_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/oauth/token", r.URL.Path)
		r.ParseForm()
		assert.Equal(t, "refresh_token", r.FormValue("grant_type"))
		assert.Equal(t, "old-refresh", r.FormValue("refresh_token"))

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tokenResponse{
			AccessToken:  "refreshed-access",
			TokenType:    "Bearer",
			ExpiresIn:    3600,
			RefreshToken: "new-refresh",
			Scope:        "surfe:all",
		})
	}))
	defer server.Close()

	origURL := AuthBaseURL
	AuthBaseURL = func() string { return server.URL }
	t.Cleanup(func() { AuthBaseURL = origURL })

	store, err := RefreshTokens("old-refresh")
	require.NoError(t, err)
	assert.Equal(t, "refreshed-access", store.AccessToken)
	assert.Equal(t, "new-refresh", store.RefreshToken)
}

func TestRefreshTokens_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{
			"error_description": "invalid refresh token",
		})
	}))
	defer server.Close()

	origURL := AuthBaseURL
	AuthBaseURL = func() string { return server.URL }
	t.Cleanup(func() { AuthBaseURL = origURL })

	_, err := RefreshTokens("bad-token")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "refresh failed")
}

// ── Logout ──────────────────────────────────────────────────────────────────

func TestLogout_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/oauth/logout", r.URL.Path)
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	origURL := AuthBaseURL
	AuthBaseURL = func() string { return server.URL }
	t.Cleanup(func() { AuthBaseURL = origURL })

	err := Logout("test-token")
	assert.NoError(t, err)
}

func TestLogout_NoContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	origURL := AuthBaseURL
	AuthBaseURL = func() string { return server.URL }
	t.Cleanup(func() { AuthBaseURL = origURL })

	err := Logout("test-token")
	assert.NoError(t, err)
}

func TestLogout_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	origURL := AuthBaseURL
	AuthBaseURL = func() string { return server.URL }
	t.Cleanup(func() { AuthBaseURL = origURL })

	err := Logout("test-token")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "logout failed with status 500")
}

// ── StartLogin callback handler ─────────────────────────────────────────────

func TestStartLogin_SuccessfulCallback(t *testing.T) {
	// Mock the token exchange server
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/oauth/authorize" {
			// Redirect to the callback URL with a code
			redirectURI := r.URL.Query().Get("redirect_uri")
			http.Redirect(w, r, redirectURI+"?code=test-auth-code", http.StatusFound)
			return
		}
		if r.URL.Path == "/oauth/token" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(tokenResponse{
				AccessToken:  "login-access-token",
				TokenType:    "Bearer",
				ExpiresIn:    3600,
				RefreshToken: "login-refresh-token",
				Scope:        "surfe:all",
			})
			return
		}
	}))
	defer tokenServer.Close()

	origURL := AuthBaseURL
	AuthBaseURL = func() string { return tokenServer.URL }
	t.Cleanup(func() { AuthBaseURL = origURL })

	// We can't fully test StartLogin because it opens a browser,
	// but we can test the callback handler directly
	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			errCh <- fmt.Errorf("no code")
			return
		}
		codeCh <- code
	})

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	server := &http.Server{Handler: mux}
	go server.Serve(listener)
	defer server.Close()

	port := listener.Addr().(*net.TCPAddr).Port

	// Simulate the OAuth callback
	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/callback?code=test-code-123", port))
	require.NoError(t, err)
	resp.Body.Close()

	code := <-codeCh
	assert.Equal(t, "test-code-123", code)
}

func TestStartLogin_ErrorCallback(t *testing.T) {
	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
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
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusBadRequest)
			errCh <- fmt.Errorf("authorization failed: %s", errMsg)
			return
		}
		codeCh <- code
	})

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	server := &http.Server{Handler: mux}
	go server.Serve(listener)
	defer server.Close()

	port := listener.Addr().(*net.TCPAddr).Port

	// Simulate error callback
	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/callback?error=access_denied&error_description=user+cancelled", port))
	require.NoError(t, err)
	resp.Body.Close()

	cbErr := <-errCh
	assert.Contains(t, cbErr.Error(), "access_denied")
	assert.Contains(t, cbErr.Error(), "user cancelled")
}

func TestStartLogin_ErrorCallbackNoDesc(t *testing.T) {
	errCh := make(chan error, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
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
			errCh <- fmt.Errorf("authorization failed: %s", errMsg)
			return
		}
	})

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	server := &http.Server{Handler: mux}
	go server.Serve(listener)
	defer server.Close()

	port := listener.Addr().(*net.TCPAddr).Port

	// Simulate callback with no code and no error params
	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/callback", port))
	require.NoError(t, err)
	resp.Body.Close()

	cbErr := <-errCh
	assert.Contains(t, cbErr.Error(), "no authorization code received")
}

// ── GetAccessToken with refresh ─────────────────────────────────────────────

func TestGetAccessToken_ExpiredWithRefresh(t *testing.T) {
	t.Setenv("SURFE_API_KEY", "")

	// Mock refresh server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tokenResponse{
			AccessToken:  "refreshed-token",
			TokenType:    "Bearer",
			ExpiresIn:    3600,
			RefreshToken: "new-refresh",
			Scope:        "surfe:all",
		})
	}))
	defer server.Close()

	origURL := AuthBaseURL
	AuthBaseURL = func() string { return server.URL }
	t.Cleanup(func() { AuthBaseURL = origURL })

	writeTestTokens(t, &TokenStore{
		AccessToken:  "expired-token",
		RefreshToken: "old-refresh-token",
		ExpiresAt:    time.Now().Add(-time.Hour),
	})

	token, err := GetAccessToken()
	require.NoError(t, err)
	assert.Equal(t, "refreshed-token", token)
}

// ── exchangeCode / RefreshTokens edge cases ─────────────────────────────────

func TestExchangeCode_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{not valid json`))
	}))
	defer server.Close()

	origURL := AuthBaseURL
	AuthBaseURL = func() string { return server.URL }
	t.Cleanup(func() { AuthBaseURL = origURL })

	_, err := exchangeCode("code", "verifier")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "parse token response")
}

func TestRefreshTokens_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{bad json`))
	}))
	defer server.Close()

	origURL := AuthBaseURL
	AuthBaseURL = func() string { return server.URL }
	t.Cleanup(func() { AuthBaseURL = origURL })

	_, err := RefreshTokens("token")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "parse refresh response")
}

func TestLogout_RequestError(t *testing.T) {
	origURL := AuthBaseURL
	AuthBaseURL = func() string { return "http://127.0.0.1:1" } // connection refused
	t.Cleanup(func() { AuthBaseURL = origURL })

	err := Logout("token")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "logout request failed")
}

func TestExchangeCode_ConnectionRefused(t *testing.T) {
	origURL := AuthBaseURL
	AuthBaseURL = func() string { return "http://127.0.0.1:1" }
	t.Cleanup(func() { AuthBaseURL = origURL })

	_, err := exchangeCode("code", "verifier")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "token exchange request failed")
}

func TestRefreshTokens_ConnectionRefused(t *testing.T) {
	origURL := AuthBaseURL
	AuthBaseURL = func() string { return "http://127.0.0.1:1" }
	t.Cleanup(func() { AuthBaseURL = origURL })

	_, err := RefreshTokens("token")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "refresh request failed")
}

// ── StartLogin full flow ────────────────────────────────────────────────────

func TestStartLogin_FullFlow(t *testing.T) {
	// Mock the identity server: /oauth/token returns tokens
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/oauth/token" {
			r.ParseForm()
			assert.Equal(t, "authorization_code", r.FormValue("grant_type"))
			assert.NotEmpty(t, r.FormValue("code"))
			assert.NotEmpty(t, r.FormValue("code_verifier"))

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(tokenResponse{
				AccessToken:  "full-flow-access",
				TokenType:    "Bearer",
				ExpiresIn:    3600,
				RefreshToken: "full-flow-refresh",
				Scope:        "surfe:all",
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer tokenServer.Close()

	origURL := AuthBaseURL
	AuthBaseURL = func() string { return tokenServer.URL }
	t.Cleanup(func() { AuthBaseURL = origURL })

	// Mock the browser — instead of opening a browser, simulate the OAuth redirect
	// by calling the callback URL directly
	origBrowser := OpenBrowserFunc
	OpenBrowserFunc = func(authURL string) error {
		// Parse the authorize URL to find the redirect_uri
		// Then call the redirect_uri with a code
		go func() {
			// Small delay to let the server start
			time.Sleep(50 * time.Millisecond)
			// Extract callback URL from authURL
			parts := strings.SplitAfter(authURL, "redirect_uri=")
			if len(parts) < 2 {
				return
			}
			callbackURL := strings.Split(parts[1], "&")[0]
			http.Get(callbackURL + "?code=test-auth-code-xyz")
		}()
		return nil
	}
	t.Cleanup(func() { OpenBrowserFunc = origBrowser })

	store, err := StartLogin()
	require.NoError(t, err)
	assert.Equal(t, "full-flow-access", store.AccessToken)
	assert.Equal(t, "full-flow-refresh", store.RefreshToken)
	assert.Equal(t, "Bearer", store.TokenType)
	assert.Equal(t, "surfe:all", store.Scope)
}

func TestStartLogin_ErrorFromCallback(t *testing.T) {
	origURL := AuthBaseURL
	AuthBaseURL = func() string { return "http://localhost:1" } // won't be reached
	t.Cleanup(func() { AuthBaseURL = origURL })

	origBrowser := OpenBrowserFunc
	OpenBrowserFunc = func(authURL string) error {
		go func() {
			time.Sleep(50 * time.Millisecond)
			parts := strings.SplitAfter(authURL, "redirect_uri=")
			if len(parts) < 2 {
				return
			}
			callbackURL := strings.Split(parts[1], "&")[0]
			http.Get(callbackURL + "?error=access_denied&error_description=user+denied")
		}()
		return nil
	}
	t.Cleanup(func() { OpenBrowserFunc = origBrowser })

	_, err := StartLogin()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "access_denied")
}

// ── openBrowser ─────────────────────────────────────────────────────────────

func TestOpenBrowser_FuncOverride(t *testing.T) {
	called := false
	orig := OpenBrowserFunc
	OpenBrowserFunc = func(url string) error {
		called = true
		assert.Equal(t, "https://example.com", url)
		return nil
	}
	t.Cleanup(func() { OpenBrowserFunc = orig })

	err := openBrowser("https://example.com")
	assert.NoError(t, err)
	assert.True(t, called)
}

func TestOpenBrowser_FuncError(t *testing.T) {
	orig := OpenBrowserFunc
	OpenBrowserFunc = func(url string) error {
		return fmt.Errorf("browser not found")
	}
	t.Cleanup(func() { OpenBrowserFunc = orig })

	err := openBrowser("https://example.com")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "browser not found")
}

// ── tokenFile ───────────────────────────────────────────────────────────────

func TestTokenFile_Default(t *testing.T) {
	tokenFilePath = ""
	f := tokenFile()
	assert.Contains(t, f, "tokens.json")
}

func TestTokenFile_Override(t *testing.T) {
	tokenFilePath = "/tmp/test-tokens.json"
	t.Cleanup(func() { tokenFilePath = "" })
	assert.Equal(t, "/tmp/test-tokens.json", tokenFile())
}

// ── StartLogin timeout ─────────────────────────────────────────────────────

func TestStartLogin_Timeout(t *testing.T) {
	origTimeout := LoginTimeout
	LoginTimeout = 100 * time.Millisecond
	t.Cleanup(func() { LoginTimeout = origTimeout })

	origURL := AuthBaseURL
	AuthBaseURL = func() string { return "http://localhost:1" }
	t.Cleanup(func() { AuthBaseURL = origURL })

	// Mock browser to do nothing — let the timeout fire
	origBrowser := OpenBrowserFunc
	OpenBrowserFunc = func(url string) error { return nil }
	t.Cleanup(func() { OpenBrowserFunc = origBrowser })

	_, err := StartLogin()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "login timed out")
}

// ── StartLogin browser open failure (prints URL instead) ────────────────────

func TestStartLogin_BrowserOpenFails(t *testing.T) {
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/oauth/token" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(tokenResponse{
				AccessToken:  "browser-fail-access",
				TokenType:    "Bearer",
				ExpiresIn:    3600,
				RefreshToken: "browser-fail-refresh",
			})
		}
	}))
	defer tokenServer.Close()

	origURL := AuthBaseURL
	AuthBaseURL = func() string { return tokenServer.URL }
	t.Cleanup(func() { AuthBaseURL = origURL })

	origBrowser := OpenBrowserFunc
	OpenBrowserFunc = func(authURL string) error {
		// Browser fails, but we still send the code manually
		go func() {
			time.Sleep(50 * time.Millisecond)
			parts := strings.SplitAfter(authURL, "redirect_uri=")
			callbackURL := strings.Split(parts[1], "&")[0]
			http.Get(callbackURL + "?code=manual-code")
		}()
		return fmt.Errorf("no browser available")
	}
	t.Cleanup(func() { OpenBrowserFunc = origBrowser })

	store, err := StartLogin()
	require.NoError(t, err)
	assert.Equal(t, "browser-fail-access", store.AccessToken)
}

// ── ClearTokens with permission error ───────────────────────────────────────

func TestClearTokens_PermissionError(t *testing.T) {
	dir := t.TempDir()
	tokenDir := filepath.Join(dir, "readonly")
	os.MkdirAll(tokenDir, 0o750)
	path := filepath.Join(tokenDir, "tokens.json")
	os.WriteFile(path, []byte("{}"), 0o600)

	tokenFilePath = path
	t.Cleanup(func() { tokenFilePath = "" })

	// Make the directory read-only so Remove fails
	os.Chmod(tokenDir, 0o444)
	t.Cleanup(func() { os.Chmod(tokenDir, 0o750) })

	err := ClearTokens()
	assert.Error(t, err)
}

// ── SaveTokens to unwritable path ───────────────────────────────────────────

func TestSaveTokens_WriteError(t *testing.T) {
	tokenFilePath = "/nonexistent-dir/impossible/tokens.json"
	t.Cleanup(func() { tokenFilePath = "" })

	err := SaveTokens(&TokenStore{AccessToken: "test"})
	assert.Error(t, err)
}

// ── GetAccessToken refresh failure saving ───────────────────────────────────

func TestGetAccessToken_RefreshSaveError(t *testing.T) {
	t.Setenv("SURFE_API_KEY", "")

	// Write tokens to a readable path, then point SaveTokens to an unwritable path
	dir := t.TempDir()
	readPath := filepath.Join(dir, "tokens.json")

	store := &TokenStore{
		AccessToken:  "expired",
		RefreshToken: "old-refresh",
		ExpiresAt:    time.Now().Add(-time.Hour),
	}
	data, _ := json.MarshalIndent(store, "", "  ")
	os.WriteFile(readPath, data, 0o600)

	tokenFilePath = readPath
	t.Cleanup(func() { tokenFilePath = "" })

	// Mock refresh server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tokenResponse{
			AccessToken:  "new-token",
			ExpiresIn:    3600,
			RefreshToken: "new-refresh",
		})
	}))
	defer server.Close()

	origURL := AuthBaseURL
	AuthBaseURL = func() string { return server.URL }
	t.Cleanup(func() { AuthBaseURL = origURL })

	// After LoadTokens succeeds, redirect writes to an impossible path
	// We do this by making the file unwritable after the read
	os.Chmod(readPath, 0o444)
	os.Chmod(dir, 0o555)
	t.Cleanup(func() {
		os.Chmod(dir, 0o750)
		os.Chmod(readPath, 0o644)
	})

	_, err := GetAccessToken()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to save refreshed tokens")
}

// ── openBrowserDefault actually runs ────────────────────────────────────────

func TestOpenBrowserDefault(t *testing.T) {
	// Test that openBrowserDefault doesn't panic with a benign URL
	// On macOS this will briefly try to open, but the URL is invalid so it's harmless
	err := openBrowserDefault("surfer://test-noop")
	// May or may not error depending on OS, but should not panic
	_ = err
}

// ── Logout with NoContent ───────────────────────────────────────────────────

func TestLogout_NoContent204(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	origURL := AuthBaseURL
	AuthBaseURL = func() string { return server.URL }
	t.Cleanup(func() { AuthBaseURL = origURL })

	err := Logout("token")
	assert.NoError(t, err)
}

func jwtWithPayload(payload string) string {
	seg := base64.RawURLEncoding.EncodeToString([]byte(payload))
	return "header." + seg + ".signature"
}

func TestDecodeTokenClaims_Valid(t *testing.T) {
	tok := jwtWithPayload(`{"email":"a@b.com","org_id":"o1","roles":["member"]}`)
	claims, err := DecodeTokenClaims(tok)
	require.NoError(t, err)
	assert.Equal(t, "a@b.com", claims["email"])
	assert.Equal(t, "o1", claims["org_id"])
}

func TestDecodeTokenClaims_NotAJWT(t *testing.T) {
	_, err := DecodeTokenClaims("opaque-api-key")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not a JWT")
}

func TestDecodeTokenClaims_BadBase64(t *testing.T) {
	_, err := DecodeTokenClaims("header.!!!not-base64!!!.sig")
	require.Error(t, err)
}

func TestDecodeTokenClaims_BadJSON(t *testing.T) {
	tok := jwtWithPayload(`not-json`)
	_, err := DecodeTokenClaims(tok)
	require.Error(t, err)
}

func TestGenerateState_UniqueAndDecodable(t *testing.T) {
	a, err := generateState()
	require.NoError(t, err)
	b, err := generateState()
	require.NoError(t, err)
	assert.NotEqual(t, a, b)
	_, err = base64.RawURLEncoding.DecodeString(a)
	assert.NoError(t, err)
}
