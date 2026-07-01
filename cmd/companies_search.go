package cmd

import (
	"encoding/json"
	"os"

	"github.com/spf13/cobra"

	"github.com/Surfe/surfer/pkg/output"
)

type companiesSearchRequest struct {
	Filters   companiesSearchFilters `json:"filters"`
	Limit     int                    `json:"limit,omitempty"`
	PageToken string                 `json:"pageToken,omitempty"`
}

type companiesSearchFilters struct {
	Domains         []string     `json:"domains,omitempty"`
	Names           []string     `json:"names,omitempty"`
	Countries       []string     `json:"countries,omitempty"`
	Industries      []string     `json:"industries,omitempty"`
	DomainsExcluded []string     `json:"domainsExcluded,omitempty"`
	EmployeeCount   *rangeFilter `json:"employeeCount,omitempty"`
}

var companiesSearchCmd = &cobra.Command{
	Use:   "companies [company names...]",
	Short: "Search for companies by name, industry, size, location",
	Long: `Search for companies using various filters. Positional args are treated as company names.

Examples:
  surfer search companies apple
  surfer search companies --domains "stripe.com,google.com"
  surfer search companies --industries "Fintech" --countries "US"
  surfer search companies --min-employees 50 --max-employees 500`,
	RunE: func(cmd *cobra.Command, args []string) error {
		var body any
		if jsonInput, _ := cmd.Flags().GetString("json"); jsonInput != "" {
			if err := json.Unmarshal([]byte(jsonInput), &body); err != nil {
				return err
			}
		} else {
			body = buildCompaniesSearchRequest(cmd, args)
		}

		var result any
		if err := newAPIClient().Post("/v2/companies/search", body, &result); err != nil {
			return err
		}

		return output.Print(os.Stdout, result, outputFormat(cmd))
	},
}

func buildCompaniesSearchRequest(cmd *cobra.Command, args []string) companiesSearchRequest {
	req := companiesSearchRequest{}

	limit, _ := cmd.Flags().GetInt("limit")
	if limit > 0 {
		req.Limit = limit
	}
	pageToken, _ := cmd.Flags().GetString("page-token")
	req.PageToken = pageToken

	req.Filters.Domains, _ = cmd.Flags().GetStringSlice("domains")
	req.Filters.Names, _ = cmd.Flags().GetStringSlice("names")
	req.Filters.Countries, _ = cmd.Flags().GetStringSlice("countries")
	req.Filters.Industries, _ = cmd.Flags().GetStringSlice("industries")
	req.Filters.DomainsExcluded, _ = cmd.Flags().GetStringSlice("domains-excluded")

	// Positional args are treated as company names
	req.Filters.Names = append(req.Filters.Names, args...)

	minEmp, _ := cmd.Flags().GetInt("min-employees")
	maxEmp, _ := cmd.Flags().GetInt("max-employees")
	if minEmp > 0 || maxEmp > 0 {
		req.Filters.EmployeeCount = &rangeFilter{From: minEmp, To: maxEmp}
	}

	return req
}

func init() {
	companiesSearchCmd.Flags().StringSlice("domains", nil, "Filter by company domains")
	companiesSearchCmd.Flags().StringSlice("names", nil, "Filter by company names")
	companiesSearchCmd.Flags().StringSlice("countries", nil, "Filter by countries")
	companiesSearchCmd.Flags().StringSlice("industries", nil, "Filter by industries")
	companiesSearchCmd.Flags().StringSlice("domains-excluded", nil, "Exclude these domains")
	companiesSearchCmd.Flags().Int("min-employees", 0, "Minimum employee count")
	companiesSearchCmd.Flags().Int("max-employees", 0, "Maximum employee count")
	companiesSearchCmd.Flags().Int("limit", 10, "Max results to return")
	companiesSearchCmd.Flags().String("page-token", "", "Pagination token")

	companiesSearchCmd.Flags().String("json", "", `Raw JSON request body (e.g. '{"filters":{"names":["apple"]}}')`)

	searchCmd.AddCommand(companiesSearchCmd)
}
