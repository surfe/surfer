package cmd

import (
	"encoding/json"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/Surfe/surfer/internal/auth"
	"github.com/Surfe/surfer/pkg/output"
)

// Roles is a comma-joined string (not a slice) so the record stays flat — the CSV
// formatter treats the first array field as the row set, which would break here.
type whoamiInfo struct {
	Email      string `json:"email,omitempty"`
	OrgID      string `json:"orgId,omitempty"`
	UserID     string `json:"userId,omitempty"`
	Roles      string `json:"roles,omitempty"`
	Scope      string `json:"scope,omitempty"`
	AuthMethod string `json:"authMethod"`
	APIKey     string `json:"apiKey,omitempty"` // masked; only for opaque (non-JWT) keys
	ExpiresAt  string `json:"expiresAt,omitempty"`
}

var whoamiCmd = &cobra.Command{
	Use:   "whoami",
	Short: "Show the account you're currently authenticated as",
	Long: `Show the account behind the current credentials (decoded locally from the
access token — no API call). Useful to confirm which Surfe account a command runs against.

Examples:
  surfer whoami
  surfer whoami -o csv`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		info, err := currentIdentity()
		if err != nil {
			return err
		}

		// Round-trip through JSON so output.Print gets a map[string]any,
		// keeping both json and csv formatting consistent with other commands.
		var generic any
		b, err := json.Marshal(info)
		if err != nil {
			return err
		}
		if err := json.Unmarshal(b, &generic); err != nil {
			return err
		}

		return output.Print(os.Stdout, generic, outputFormat(cmd))
	},
}

func currentIdentity() (*whoamiInfo, error) {
	info := &whoamiInfo{}
	var token string

	if key := os.Getenv("SURFE_API_KEY"); key != "" {
		info.AuthMethod = "api_key (SURFE_API_KEY)"
		token = key
	} else {
		store, err := auth.LoadTokens()
		if err != nil {
			return nil, err
		}
		info.AuthMethod = "oauth"
		token = store.AccessToken
		if !store.ExpiresAt.IsZero() {
			info.ExpiresAt = store.ExpiresAt.Format(time.RFC3339)
		}
	}

	claims, err := auth.DecodeTokenClaims(token)
	if err != nil {
		// Opaque token (non-JWT API key): can't derive the account locally.
		info.APIKey = maskSecret(token)
		return info, nil
	}

	if v, ok := claims["email"].(string); ok {
		info.Email = v
	}
	if v, ok := claims["org_id"].(string); ok {
		info.OrgID = v
	}
	if v, ok := claims["sub"].(string); ok {
		info.UserID = v
	}
	if v, ok := claims["scope"].(string); ok {
		info.Scope = v
	}
	info.Roles = strings.Join(toStringSlice(claims["roles"]), ", ")
	if info.ExpiresAt == "" {
		if exp, ok := claims["exp"].(float64); ok {
			info.ExpiresAt = time.Unix(int64(exp), 0).UTC().Format(time.RFC3339)
		}
	}

	return info, nil
}

func toStringSlice(v any) []string {
	arr, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(arr))
	for _, e := range arr {
		if s, ok := e.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func maskSecret(s string) string {
	if len(s) <= 8 {
		return "****"
	}
	return s[:4] + "…" + s[len(s)-4:]
}

func init() {
	whoamiCmd.GroupID = "auth"
	rootCmd.AddCommand(whoamiCmd)
}
