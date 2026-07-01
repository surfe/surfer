package cmd

import (
	"encoding/json"
	"os"

	"github.com/spf13/cobra"

	"github.com/Surfe/surfer/pkg/output"
)

type peopleSearchRequest struct {
	People           *peopleFilter   `json:"people,omitempty"`
	Companies        *companyFilter  `json:"companies,omitempty"`
	PeoplePerCompany int             `json:"peoplePerCompany,omitempty"`
	Limit            int             `json:"limit,omitempty"`
	PageToken        string          `json:"pageToken,omitempty"`
}

type peopleFilter struct {
	JobTitles   []string `json:"jobTitles,omitempty"`
	Seniorities []string `json:"seniorities,omitempty"`
	Departments []string `json:"departments,omitempty"`
}

type companyFilter struct {
	Domains    []string    `json:"domains,omitempty"`
	Names      []string    `json:"names,omitempty"`
	Countries  []string    `json:"countries,omitempty"`
	Industries []string    `json:"industries,omitempty"`
	EmployeeCount *rangeFilter `json:"employeeCount,omitempty"`
}

type rangeFilter struct {
	From int `json:"from,omitempty"`
	To   int `json:"to,omitempty"`
}

var peopleSearchCmd = &cobra.Command{
	Use:   "people",
	Short: "Search for people by job title, company, location",
	Long: `Search for contacts using various filters.

Examples:
  surfer search people --job-titles "CTO,VP Engineering" --countries "US,UK"
  surfer search people --company-domains "google.com" --seniorities "C-Level"
  surfer search people --company-names "Stripe" --limit 20
  surfer search people --json '{"companies":{"domains":["surfe.com"]}}'`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		var body any
		if jsonInput, _ := cmd.Flags().GetString("json"); jsonInput != "" {
			if err := json.Unmarshal([]byte(jsonInput), &body); err != nil {
				return err
			}
		} else {
			body = buildPeopleSearchRequest(cmd)
		}

		var result any
		if err := newAPIClient().Post("/v2/people/search", body, &result); err != nil {
			return err
		}

		return output.Print(os.Stdout, result, outputFormat(cmd))
	},
}

func buildPeopleSearchRequest(cmd *cobra.Command) peopleSearchRequest {
	req := peopleSearchRequest{}

	limit, _ := cmd.Flags().GetInt("limit")
	if limit > 0 {
		req.Limit = limit
	}
	pageToken, _ := cmd.Flags().GetString("page-token")
	req.PageToken = pageToken

	ppc, _ := cmd.Flags().GetInt("people-per-company")
	if ppc > 0 {
		req.PeoplePerCompany = ppc
	}

	jobTitles, _ := cmd.Flags().GetStringSlice("job-titles")
	seniorities, _ := cmd.Flags().GetStringSlice("seniorities")
	departments, _ := cmd.Flags().GetStringSlice("departments")
	if len(jobTitles) > 0 || len(seniorities) > 0 || len(departments) > 0 {
		req.People = &peopleFilter{
			JobTitles:   jobTitles,
			Seniorities: seniorities,
			Departments: departments,
		}
	}

	domains, _ := cmd.Flags().GetStringSlice("company-domains")
	names, _ := cmd.Flags().GetStringSlice("company-names")
	countries, _ := cmd.Flags().GetStringSlice("countries")
	industries, _ := cmd.Flags().GetStringSlice("industries")
	if len(domains) > 0 || len(names) > 0 || len(countries) > 0 || len(industries) > 0 {
		req.Companies = &companyFilter{
			Domains:    domains,
			Names:      names,
			Countries:  countries,
			Industries: industries,
		}
	}

	return req
}

func init() {
	peopleSearchCmd.Flags().StringSlice("job-titles", nil, "Filter by job titles (comma-separated)")
	peopleSearchCmd.Flags().StringSlice("seniorities", nil, "Filter by seniority levels")
	peopleSearchCmd.Flags().StringSlice("departments", nil, "Filter by departments")
	peopleSearchCmd.Flags().StringSlice("company-domains", nil, "Filter by company domains")
	peopleSearchCmd.Flags().StringSlice("company-names", nil, "Filter by company names")
	peopleSearchCmd.Flags().StringSlice("countries", nil, "Filter by countries")
	peopleSearchCmd.Flags().StringSlice("industries", nil, "Filter by industries")
	peopleSearchCmd.Flags().Int("limit", 10, "Max results to return")
	peopleSearchCmd.Flags().Int("people-per-company", 5, "Max people per company")
	peopleSearchCmd.Flags().String("page-token", "", "Pagination token")

	peopleSearchCmd.Flags().String("json", "", `Raw JSON request body`)

	searchCmd.AddCommand(peopleSearchCmd)
}
