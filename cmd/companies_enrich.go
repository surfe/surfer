package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

type enrichCompaniesRequest struct {
	Companies []enrichCompany `json:"companies"`
}

type enrichCompany struct {
	Domain     string `json:"domain"`
	ExternalID string `json:"externalID,omitempty"`
}

var companiesEnrichCmd = &cobra.Command{
	Use:   "companies",
	Short: "Enrich companies to get firmographic data",
	Long: `Start enrichment for one or more companies. Returns an enrichment ID to check status.

Examples:
  surfer enrich companies --domain stripe.com
  surfer enrich companies --domain google.com --external-id company-123
  surfer enrich companies --json '{"companies":[{"domain":"stripe.com"}]}'
  surfer enrich companies --domain stripe.com --wait   # poll until complete
  surfer enrich companies status <enrichment-id>`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		var body any
		if jsonInput, _ := cmd.Flags().GetString("json"); jsonInput != "" {
			if err := json.Unmarshal([]byte(jsonInput), &body); err != nil {
				return err
			}
		} else {
			domain, _ := cmd.Flags().GetString("domain")
			externalID, _ := cmd.Flags().GetString("external-id")
			if domain == "" {
				return fmt.Errorf("--domain is required")
			}
			body = enrichCompaniesRequest{
				Companies: []enrichCompany{
					{Domain: domain, ExternalID: externalID},
				},
			}
		}

		var start map[string]any
		if err := newAPIClient().Post("/v2/companies/enrich", body, &start); err != nil {
			return err
		}

		return emitEnrichResult(cmd, "/v2/companies/enrich", start)
	},
}

func init() {
	// Note: --domain is not marked required via MarkFlagRequired because cobra
	// validates required flags before RunE, which would make --json unusable.
	// The non-JSON path enforces --domain manually in RunE instead.
	companiesEnrichCmd.Flags().String("domain", "", "Company domain (required unless --json)")
	companiesEnrichCmd.Flags().String("external-id", "", "External ID for tracking")

	companiesEnrichCmd.Flags().String("json", "", `Raw JSON request body`)
	addWaitFlags(companiesEnrichCmd)

	enrichCmd.AddCommand(companiesEnrichCmd)
}
