package client

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mockToken() (string, error) {
	return "test-bearer-token", nil
}

func failingToken() (string, error) {
	return "", fmt.Errorf("not authenticated")
}

func TestNew(t *testing.T) {
	c := New()
	assert.Equal(t, APIBaseURL(), c.baseURL)
	assert.Equal(t, "https://api.surfe.com", c.baseURL)
	assert.NotNil(t, c.httpClient)
	assert.NotNil(t, c.tokenFunc)
}

func TestNewWithOptions(t *testing.T) {
	c := NewWithOptions("http://custom.api", mockToken)
	assert.Equal(t, "http://custom.api", c.baseURL)
}

func TestDo_GetSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/v1/credits", r.URL.Path)
		assert.Equal(t, "Bearer test-bearer-token", r.Header.Get("Authorization"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "surfer-cli", r.Header.Get("User-Agent"))

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]int{"emailCredits": 500})
	}))
	defer server.Close()

	c := NewWithOptions(server.URL, mockToken)

	var result map[string]int
	err := c.Get("/v1/credits", &result)
	require.NoError(t, err)
	assert.Equal(t, 500, result["emailCredits"])
}

func TestDo_PostWithBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/v2/people/search", r.URL.Path)

		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		assert.NotNil(t, body["people"])

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"people": []map[string]string{
				{"firstName": "John", "lastName": "Doe"},
			},
		})
	}))
	defer server.Close()

	c := NewWithOptions(server.URL, mockToken)

	reqBody := map[string]any{"people": map[string]any{"jobTitles": []string{"CTO"}}}
	var result map[string]any
	err := c.Post("/v2/people/search", reqBody, &result)
	require.NoError(t, err)
	assert.NotNil(t, result["people"])
}

func TestDo_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"error": "insufficient credits"}`))
	}))
	defer server.Close()

	c := NewWithOptions(server.URL, mockToken)

	var result any
	err := c.Get("/v2/people/search", &result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "API error (403)")
	assert.Contains(t, err.Error(), "insufficient credits")
}

func TestDo_TokenError(t *testing.T) {
	c := NewWithOptions("http://unused", failingToken)

	var result any
	err := c.Get("/test", &result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not authenticated")
}

func TestDo_NilResult(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	c := NewWithOptions(server.URL, mockToken)
	err := c.Do("DELETE", "/resource", nil, nil)
	require.NoError(t, err)
}

func TestDo_EmptyResponseBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c := NewWithOptions(server.URL, mockToken)
	var result map[string]any
	err := c.Get("/empty", &result)
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestDo_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{invalid json`))
	}))
	defer server.Close()

	c := NewWithOptions(server.URL, mockToken)
	var result map[string]any
	err := c.Get("/bad-json", &result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "parse response")
}

func TestDo_NilBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, int64(0), r.ContentLength)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok": true}`))
	}))
	defer server.Close()

	c := NewWithOptions(server.URL, mockToken)
	var result map[string]bool
	err := c.Do("POST", "/test", nil, &result)
	require.NoError(t, err)
	assert.True(t, result["ok"])
}

func TestGet(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		w.Write([]byte(`{"status": "ok"}`))
	}))
	defer server.Close()

	c := NewWithOptions(server.URL, mockToken)
	var result map[string]string
	err := c.Get("/health", &result)
	require.NoError(t, err)
	assert.Equal(t, "ok", result["status"])
}

func TestPost(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		w.Write([]byte(`{"id": "123"}`))
	}))
	defer server.Close()

	c := NewWithOptions(server.URL, mockToken)
	var result map[string]string
	err := c.Post("/create", map[string]string{"name": "test"}, &result)
	require.NoError(t, err)
	assert.Equal(t, "123", result["id"])
}

func TestDo_MarshalError(t *testing.T) {
	c := NewWithOptions("http://unused", mockToken)
	// channels can't be marshaled to JSON
	err := c.Post("/test", make(chan int), nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "marshal request body")
}

func TestDo_InvalidMethod(t *testing.T) {
	c := NewWithOptions("http://unused", mockToken)
	err := c.Do("BAD METHOD", "/test", nil, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "create request")
}

func TestDo_ConnectionError(t *testing.T) {
	c := NewWithOptions("http://127.0.0.1:1", mockToken)
	err := c.Get("/test", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "request failed")
}

func TestDo_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal error"))
	}))
	defer server.Close()

	c := NewWithOptions(server.URL, mockToken)
	err := c.Get("/fail", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "API error (500)")
}
