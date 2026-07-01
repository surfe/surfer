package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"

	"github.com/Surfe/surfer/internal/auth"
)

const UserAgent = "surfer-cli"

// APIBaseURL returns the base URL for Surfe API calls.
// Override via config file (api-url), env var (SURFER_API_URL), or --api-url flag.
var APIBaseURL = func() string {
	if v := viper.GetString("api-url"); v != "" {
		return v
	}
	return "https://api.surfe.com"
}

// TokenFunc returns an access token. Defaults to auth.GetAccessToken.
type TokenFunc func() (string, error)

type Client struct {
	httpClient *http.Client
	baseURL    string
	tokenFunc  TokenFunc
}

func New() *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		baseURL:    APIBaseURL(),
		tokenFunc:  auth.GetAccessToken,
	}
}

// NewWithOptions creates a client with custom base URL and token function (for testing).
func NewWithOptions(baseURL string, tokenFunc TokenFunc) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		baseURL:    baseURL,
		tokenFunc:  tokenFunc,
	}
}

func (c *Client) Do(method, path string, body any, result any) error {
	token, err := c.tokenFunc()
	if err != nil {
		return err
	}

	var reqBody io.Reader
	var reqBytes []byte
	if body != nil {
		reqBytes, err = json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(reqBytes)
	}

	url := c.baseURL + path
	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", UserAgent)

	log.Debug().
		Str("method", method).
		Str("url", url).
		RawJSON("body", appendOrNull(reqBytes)).
		Msg("API request")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	log.Debug().
		Int("status", resp.StatusCode).
		RawJSON("body", appendOrNull(respBody)).
		Msg("API response")

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("API error (%d): %s", resp.StatusCode, string(respBody))
	}

	if result != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("parse response: %w", err)
		}
	}

	return nil
}

func (c *Client) Get(path string, result any) error {
	return c.Do("GET", path, nil, result)
}

func (c *Client) Post(path string, body any, result any) error {
	return c.Do("POST", path, body, result)
}

func appendOrNull(b []byte) []byte {
	if len(b) == 0 {
		return []byte("null")
	}
	return b
}
