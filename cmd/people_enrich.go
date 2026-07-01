package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

type enrichPeopleRequest struct {
	People  []enrichPerson `json:"people"`
	Include *enrichInclude `json:"include,omitempty"`
}

type enrichPerson struct {
	FirstName     string `json:"firstName,omitempty"`
	LastName      string `json:"lastName,omitempty"`
	LinkedinURL   string `json:"linkedinUrl,omitempty"`
	CompanyName   string `json:"companyName,omitempty"`
	CompanyDomain string `json:"companyDomain,omitempty"`
}

type enrichInclude struct {
	Email      bool `json:"email"`
	Mobile     bool `json:"mobile"`
	LinkedInURL bool `json:"linkedInUrl"`
	JobHistory bool `json:"jobHistory"`
}

var peopleEnrichCmd = &cobra.Command{
	Use:   "people",
	Short: "Enrich people to get emails, phones, LinkedIn profiles",
	Long: `Start enrichment for one or more people. Returns an enrichment ID to check status.

Examples:
  surfer enrich people --linkedin "https://linkedin.com/in/johndoe"
  surfer enrich people --first-name John --last-name Doe --company-domain google.com
  surfer enrich people --linkedin "https://linkedin.com/in/johndoe" --include-mobile
  surfer enrich people --json '{"people":[{"linkedinUrl":"https://linkedin.com/in/johndoe"}],"include":{"email":true}}'
  surfer enrich people --linkedin "https://linkedin.com/in/johndoe" --wait   # poll until complete
  surfer enrich people status <enrichment-id>`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		var body any
		if jsonInput, _ := cmd.Flags().GetString("json"); jsonInput != "" {
			if err := json.Unmarshal([]byte(jsonInput), &body); err != nil {
				return err
			}
		} else {
			person := enrichPerson{}
			person.FirstName, _ = cmd.Flags().GetString("first-name")
			person.LastName, _ = cmd.Flags().GetString("last-name")
			person.LinkedinURL, _ = cmd.Flags().GetString("linkedin")
			person.CompanyName, _ = cmd.Flags().GetString("company-name")
			person.CompanyDomain, _ = cmd.Flags().GetString("company-domain")

			if person.LinkedinURL == "" && (person.FirstName == "" || person.LastName == "") {
				return fmt.Errorf("provide --linkedin URL or --first-name and --last-name")
			}

			includeEmail, _ := cmd.Flags().GetBool("include-email")
			includeMobile, _ := cmd.Flags().GetBool("include-mobile")
			includeLinkedin, _ := cmd.Flags().GetBool("include-linkedin")
			includeHistory, _ := cmd.Flags().GetBool("include-job-history")

			body = enrichPeopleRequest{
				People: []enrichPerson{person},
				Include: &enrichInclude{
					Email:       includeEmail,
					Mobile:      includeMobile,
					LinkedInURL: includeLinkedin,
					JobHistory:  includeHistory,
				},
			}
		}

		var start map[string]any
		if err := newAPIClient().Post("/v2/people/enrich", body, &start); err != nil {
			return err
		}

		return emitEnrichResult(cmd, "/v2/people/enrich", start)
	},
}

func init() {
	peopleEnrichCmd.Flags().String("first-name", "", "Person's first name")
	peopleEnrichCmd.Flags().String("last-name", "", "Person's last name")
	peopleEnrichCmd.Flags().String("linkedin", "", "LinkedIn profile URL")
	peopleEnrichCmd.Flags().String("company-name", "", "Company name")
	peopleEnrichCmd.Flags().String("company-domain", "", "Company domain")
	peopleEnrichCmd.Flags().Bool("include-email", true, "Include email in enrichment")
	peopleEnrichCmd.Flags().Bool("include-mobile", false, "Include mobile phone")
	peopleEnrichCmd.Flags().Bool("include-linkedin", true, "Include LinkedIn URL")
	peopleEnrichCmd.Flags().Bool("include-job-history", false, "Include job history")

	peopleEnrichCmd.Flags().String("json", "", `Raw JSON request body`)
	addWaitFlags(peopleEnrichCmd)

	enrichCmd.AddCommand(peopleEnrichCmd)
}
