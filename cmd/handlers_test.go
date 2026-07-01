package cmd

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Surfe/surfer/internal/auth"
	"github.com/Surfe/surfer/internal/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeTokenFile creates a tokens.json under a temp XDG config dir and points the
// CLI at it via XDG_CONFIG_HOME. Returns the token file path.
func writeTokenFile(t *testing.T, accessToken string) string {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "surfer"), 0o755))
	store, err := json.Marshal(map[string]any{
		"access_token": accessToken,
		"expires_at":   "2030-01-01T00:00:00Z",
	})
	require.NoError(t, err)
	path := filepath.Join(dir, "surfer", "tokens.json")
	require.NoError(t, os.WriteFile(path, store, 0o600))
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("SURFE_API_KEY", "")
	return path
}

// makeJWT builds an unsigned JWT with the given claims for tests (display-only decode).
func makeJWT(t *testing.T, claims map[string]any) string {
	t.Helper()
	payload, err := json.Marshal(claims)
	require.NoError(t, err)
	seg := func(b []byte) string { return base64.RawURLEncoding.EncodeToString(b) }
	return seg([]byte(`{"alg":"none","typ":"JWT"}`)) + "." + seg(payload) + ".sig"
}

func withMockAPI(t *testing.T, handler http.HandlerFunc) {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	orig := newAPIClient
	newAPIClient = func() *client.Client {
		return client.NewWithOptions(server.URL, func() (string, error) {
			return "test-token", nil
		})
	}
	t.Cleanup(func() { newAPIClient = orig })
}

func TestCreditsCommand_RunE(t *testing.T) {
	withMockAPI(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/credits", r.URL.Path)
		assert.Equal(t, "GET", r.Method)
		json.NewEncoder(w).Encode(map[string]int{"emailCredits": 500, "mobileCredits": 100})
	})

	rootCmd.SetArgs([]string{"credits"})
	err := rootCmd.Execute()
	require.NoError(t, err)
}

func TestPeopleSearchCommand_RunE(t *testing.T) {
	withMockAPI(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v2/people/search", r.URL.Path)
		assert.Equal(t, "POST", r.Method)
		json.NewEncoder(w).Encode(map[string]any{"people": []any{}})
	})

	rootCmd.SetArgs([]string{"search", "people", "--company-domains", "surfe.com"})
	err := rootCmd.Execute()
	require.NoError(t, err)
}

func TestPeopleEnrichCommand_RunE(t *testing.T) {
	withMockAPI(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v2/people/enrich", r.URL.Path)
		json.NewEncoder(w).Encode(map[string]string{"enrichmentID": "e-123"})
	})

	rootCmd.SetArgs([]string{"enrich", "people", "--linkedin", "https://linkedin.com/in/test"})
	err := rootCmd.Execute()
	require.NoError(t, err)
}

func TestPeopleEnrichCommand_MissingArgs(t *testing.T) {
	withMockAPI(t, func(w http.ResponseWriter, r *http.Request) {})

	// Reset flags from previous test
	peopleEnrichCmd.Flags().Set("linkedin", "")
	peopleEnrichCmd.Flags().Set("first-name", "")
	peopleEnrichCmd.Flags().Set("last-name", "")

	rootCmd.SetArgs([]string{"enrich", "people"})
	err := rootCmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "provide --linkedin")
}

func TestPeopleStatusCommand_RunE(t *testing.T) {
	withMockAPI(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v2/people/enrich/abc-123", r.URL.Path)
		json.NewEncoder(w).Encode(map[string]string{"status": "COMPLETED"})
	})

	rootCmd.SetArgs([]string{"enrich", "people", "status", "abc-123"})
	err := rootCmd.Execute()
	require.NoError(t, err)
}

func TestCompaniesSearchCommand_RunE(t *testing.T) {
	withMockAPI(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v2/companies/search", r.URL.Path)
		json.NewEncoder(w).Encode(map[string]any{"companies": []any{}})
	})

	rootCmd.SetArgs([]string{"search", "companies", "--industries", "Fintech"})
	err := rootCmd.Execute()
	require.NoError(t, err)
}

func TestCompaniesEnrichCommand_RunE(t *testing.T) {
	withMockAPI(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v2/companies/enrich", r.URL.Path)
		json.NewEncoder(w).Encode(map[string]string{"enrichmentID": "c-456"})
	})

	rootCmd.SetArgs([]string{"enrich", "companies", "--domain", "stripe.com"})
	err := rootCmd.Execute()
	require.NoError(t, err)
}

// Regression: --json must work without --domain. domain is enforced manually in
// RunE (not via MarkFlagRequired), so the JSON branch is reachable.
func TestCompaniesEnrichCommand_JSONWithoutDomain(t *testing.T) {
	withMockAPI(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v2/companies/enrich", r.URL.Path)
		json.NewEncoder(w).Encode(map[string]string{"enrichmentID": "c-789"})
	})

	companiesEnrichCmd.Flags().Set("domain", "")
	t.Cleanup(func() { companiesEnrichCmd.Flags().Set("json", "") })

	rootCmd.SetArgs([]string{"enrich", "companies", "--json", `{"companies":[{"domain":"stripe.com"}]}`})
	err := rootCmd.Execute()
	require.NoError(t, err)
}

func TestCompaniesStatusCommand_RunE(t *testing.T) {
	withMockAPI(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v2/companies/enrich/xyz-789", r.URL.Path)
		json.NewEncoder(w).Encode(map[string]string{"status": "IN_PROGRESS"})
	})

	rootCmd.SetArgs([]string{"enrich", "companies", "status", "xyz-789"})
	err := rootCmd.Execute()
	require.NoError(t, err)
}

func TestUpdateCommand_DevBuild(t *testing.T) {
	rootCmd.SetArgs([]string{"update"})
	err := rootCmd.Execute()
	// BuildVersion is "main" in test, so it should error
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot update a development build")
}

func TestSearchCommand_ShowsHelp(t *testing.T) {
	rootCmd.SetArgs([]string{"search"})
	err := rootCmd.Execute()
	require.NoError(t, err)
}

func TestEnrichCommand_ShowsHelp(t *testing.T) {
	rootCmd.SetArgs([]string{"enrich"})
	err := rootCmd.Execute()
	require.NoError(t, err)
}

func TestCompaniesEnrichCommand_MissingDomain(t *testing.T) {
	withMockAPI(t, func(w http.ResponseWriter, r *http.Request) {})
	companiesEnrichCmd.Flags().Set("domain", "")
	companiesEnrichCmd.Flags().Set("json", "")
	rootCmd.SetArgs([]string{"enrich", "companies"})
	err := rootCmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "--domain is required")
}

func TestCreditsCommand_APIError(t *testing.T) {
	withMockAPI(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error": "unauthorized"}`))
	})

	rootCmd.SetArgs([]string{"credits"})
	err := rootCmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "API error (401)")
}

func TestWhoami_CurrentIdentity_JWT(t *testing.T) {
	tok := makeJWT(t, map[string]any{
		"email": "x@surfe.com", "org_id": "org_1", "sub": "user_1",
		"roles": []any{"member", "admin"}, "scope": "surfe:all",
	})
	t.Setenv("SURFE_API_KEY", tok)

	info, err := currentIdentity()
	require.NoError(t, err)
	assert.Equal(t, "x@surfe.com", info.Email)
	assert.Equal(t, "org_1", info.OrgID)
	assert.Equal(t, "user_1", info.UserID)
	assert.Equal(t, "member, admin", info.Roles)
	assert.Equal(t, "surfe:all", info.Scope)
	assert.Contains(t, info.AuthMethod, "api_key")
}

func TestWhoami_CurrentIdentity_OpaqueKey(t *testing.T) {
	t.Setenv("SURFE_API_KEY", "opaque-secret-value-1234")

	info, err := currentIdentity()
	require.NoError(t, err)
	assert.Empty(t, info.Email)
	assert.NotEmpty(t, info.APIKey)               // masked
	assert.NotContains(t, info.APIKey, "secret") // raw secret never shown
}

func TestWaitForEnrichment_PollsUntilDone(t *testing.T) {
	var calls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&calls, 1) < 2 {
			json.NewEncoder(w).Encode(map[string]any{"status": "IN_PROGRESS"})
			return
		}
		json.NewEncoder(w).Encode(map[string]any{"status": "COMPLETED", "companies": []any{map[string]any{"domain": "surfe.com"}}})
	}))
	t.Cleanup(server.Close)

	c := client.NewWithOptions(server.URL, func() (string, error) { return "t", nil })
	res, err := waitForEnrichment(c, "/v2/companies/enrich/x", time.Millisecond, 5*time.Second)
	require.NoError(t, err)
	m := res.(map[string]any)
	assert.Equal(t, "COMPLETED", m["status"])
	assert.GreaterOrEqual(t, atomic.LoadInt32(&calls), int32(2))
}

func TestWaitForEnrichment_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"status": "IN_PROGRESS"})
	}))
	t.Cleanup(server.Close)

	c := client.NewWithOptions(server.URL, func() (string, error) { return "t", nil })
	_, err := waitForEnrichment(c, "/v2/companies/enrich/x", time.Millisecond, 20*time.Millisecond)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "timed out")
}

// --wait wiring end-to-end: start enrichment, then poll status to completion.
func TestCompaniesEnrichCommand_Wait(t *testing.T) {
	withMockAPI(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			assert.Equal(t, "/v2/companies/enrich", r.URL.Path)
			json.NewEncoder(w).Encode(map[string]any{"enrichmentID": "wait-1"})
			return
		}
		assert.Equal(t, "/v2/companies/enrich/wait-1", r.URL.Path)
		json.NewEncoder(w).Encode(map[string]any{"status": "COMPLETED", "companies": []any{map[string]any{"domain": "stripe.com"}}})
	})

	companiesEnrichCmd.Flags().Set("json", "")
	t.Cleanup(func() { companiesEnrichCmd.Flags().Set("wait", "false") })

	rootCmd.SetArgs([]string{"enrich", "companies", "--domain", "stripe.com", "--wait"})
	require.NoError(t, rootCmd.Execute())
}

// --wait with no enrichmentID in the start response: falls back to printing it.
func TestCompaniesEnrichCommand_WaitNoID(t *testing.T) {
	withMockAPI(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"message": "started"})
	})

	companiesEnrichCmd.Flags().Set("json", "")
	t.Cleanup(func() { companiesEnrichCmd.Flags().Set("wait", "false") })

	rootCmd.SetArgs([]string{"enrich", "companies", "--domain", "x.com", "--wait"})
	require.NoError(t, rootCmd.Execute())
}

func TestWhoamiCommand_RunE(t *testing.T) {
	t.Setenv("SURFE_API_KEY", makeJWT(t, map[string]any{"email": "x@surfe.com", "roles": []any{"member"}}))
	rootCmd.SetArgs([]string{"whoami"})
	require.NoError(t, rootCmd.Execute())
}

func TestAICommand_Runs(t *testing.T) {
	rootCmd.SetArgs([]string{"ai"})
	require.NoError(t, rootCmd.Execute())
}

func TestStartSpinner_NoTTYIsNoop(t *testing.T) {
	// stderr is not a terminal under `go test`, so the spinner must be a no-op
	// and the stop func must return cleanly.
	stop := startSpinner("working")
	stop()
}

// currentIdentity decoding an OAuth token from disk (the non-API-key branch).
func TestWhoami_CurrentIdentity_OAuthTokenFile(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "surfer"), 0o755))
	tok := makeJWT(t, map[string]any{"email": "o@auth.com", "org_id": "org_x", "sub": "user_x"})
	store, err := json.Marshal(map[string]any{
		"access_token": tok,
		"expires_at":   "2030-01-01T00:00:00Z",
	})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "surfer", "tokens.json"), store, 0o600))

	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("SURFE_API_KEY", "")

	info, err := currentIdentity()
	require.NoError(t, err)
	assert.Equal(t, "oauth", info.AuthMethod)
	assert.Equal(t, "o@auth.com", info.Email)
	assert.Equal(t, "org_x", info.OrgID)
	assert.NotEmpty(t, info.ExpiresAt)
}

func TestWhoami_CurrentIdentity_NotLoggedIn(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("SURFE_API_KEY", "")

	_, err := currentIdentity()
	require.Error(t, err)
}

func TestSpin_RendersFramesAndClears(t *testing.T) {
	var buf bytes.Buffer
	done := make(chan struct{})
	finished := make(chan struct{})
	go func() {
		defer close(finished)
		spin(&buf, "enriching", time.Millisecond, done)
	}()
	time.Sleep(20 * time.Millisecond) // allow several ticks
	close(done)
	<-finished // spin has returned: no concurrent writes, safe to read buf

	out := buf.String()
	assert.Contains(t, out, "enriching") // at least one frame rendered
	assert.Contains(t, out, "\033[K")    // line cleared on stop
}

// --wait that never completes returns a timeout error (emitEnrichResult error path).
func TestCompaniesEnrichCommand_WaitTimeout(t *testing.T) {
	withMockAPI(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			json.NewEncoder(w).Encode(map[string]any{"enrichmentID": "stuck-1"})
			return
		}
		json.NewEncoder(w).Encode(map[string]any{"status": "IN_PROGRESS"})
	})

	companiesEnrichCmd.Flags().Set("json", "")
	t.Cleanup(func() {
		companiesEnrichCmd.Flags().Set("wait", "false")
		companiesEnrichCmd.Flags().Set("timeout", "2m")
		companiesEnrichCmd.Flags().Set("poll-interval", "2s")
	})

	rootCmd.SetArgs([]string{"enrich", "companies", "--domain", "x.com", "--wait", "--poll-interval", "1ms", "--timeout", "15ms"})
	err := rootCmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "timed out")
}

func TestMaskSecret_Short(t *testing.T) {
	assert.Equal(t, "****", maskSecret("short"))
}

func TestToStringSlice_NonSlice(t *testing.T) {
	assert.Nil(t, toStringSlice("not-a-slice"))
	assert.Nil(t, toStringSlice(nil))
}

func TestSpinAsync_RendersAndStops(t *testing.T) {
	var buf bytes.Buffer
	stop := spinAsync(&buf, "loading", time.Millisecond)
	time.Sleep(20 * time.Millisecond)
	stop() // blocks until cleared; no concurrent writes after this

	out := buf.String()
	assert.Contains(t, out, "loading")
	assert.Contains(t, out, "\033[K")
}

func TestLogoutCommand_LoggedIn(t *testing.T) {
	path := writeTokenFile(t, "tok-abc")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/oauth/logout", r.URL.Path)
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(server.Close)

	orig := auth.AuthBaseURL
	auth.AuthBaseURL = func() string { return server.URL }
	t.Cleanup(func() { auth.AuthBaseURL = orig })

	rootCmd.SetArgs([]string{"logout"})
	require.NoError(t, rootCmd.Execute())

	_, err := os.Stat(path)
	assert.True(t, os.IsNotExist(err), "token file should be removed on logout")
}

func TestLogoutCommand_NotLoggedIn(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("SURFE_API_KEY", "")

	rootCmd.SetArgs([]string{"logout"})
	require.NoError(t, rootCmd.Execute())
}
