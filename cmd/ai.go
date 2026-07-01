package cmd

import (
	"fmt"

	"github.com/Surfe/surfer/pkg"
	"github.com/spf13/cobra"
)

const aiPrompt = `You have access to the Surfer CLI (v%s), a command-line tool for the Surfe API.
Use it to search contacts, enrich leads with verified emails/phones, and search companies.

## Authentication
The CLI is already authenticated. If you get a 401 error, run: surfer login

## Available commands

### Search people
surfer search people [flags]
  --job-titles strings        Filter by job titles (comma-separated)
  --seniorities strings       Filter by seniority levels (e.g. C-Level,VP,Head,Director)
  --departments strings       Filter by departments
  --company-domains strings   Filter by company domains
  --company-names strings     Filter by company names
  --countries strings         Filter by country codes (e.g. US,UK,DE,FR)
  --industries strings        Filter by industries
  --limit int                 Max results (default 10)
  --people-per-company int    Max people per company (default 5)
  --json string               Raw JSON request body (overrides all flags)
  -o, --output string         Output format: json (default) or csv

### Enrich a person (get email, phone, LinkedIn)
surfer enrich people [flags]
  --linkedin string           LinkedIn profile URL
  --first-name string         Person's first name (use with --last-name)
  --last-name string          Person's last name (use with --first-name)
  --company-name string       Company name
  --company-domain string     Company domain
  --include-email bool        Include email (default true)
  --include-mobile bool       Include mobile phone (default false)
  --json string               Raw JSON request body
  --wait                      Poll until enrichment completes, then print the final result
  --poll-interval duration    How often to poll while waiting (default 2s)
  --timeout duration          Max time to wait for completion (default 2m)
  -o, --output string         Output format: json or csv

### Check people enrichment status
surfer enrich people status <enrichment-id>

### Search companies
surfer search companies [names...] [flags]
  --domains strings           Filter by company domains
  --names strings             Filter by company names
  --countries strings         Filter by countries
  --industries strings        Filter by industries
  --min-employees int         Minimum employee count
  --max-employees int         Maximum employee count
  --limit int                 Max results (default 10)
  --json string               Raw JSON request body
  -o, --output string         Output format: json or csv

### Enrich a company
surfer enrich companies --domain <domain> [--external-id <id>]
  --wait                      Poll until enrichment completes, then print the final result
  --poll-interval duration    How often to poll while waiting (default 2s)
  --timeout duration          Max time to wait for completion (default 2m)
  --json string               Raw JSON request body

### Check company enrichment status
surfer enrich companies status <enrichment-id>

### Check current account / who am I
surfer whoami
  Shows the authenticated account (email, orgId, userId, roles) decoded locally
  from the access token. No API call. Use to confirm which account commands run as.

### Check credit balance
surfer credits

## Output
- All commands return JSON by default. Use -o csv for CSV output.
- Pipe to jq for field extraction: surfer search people ... | jq '.people[].firstName'
- Use --debug for full HTTP request/response logging.

## Tips
- Enrichment is async: either pass --wait to block until it finishes and get the
  result in one call, or call enrich, take the enrichmentID, and poll status until COMPLETED.
- Use --json for complex queries: surfer search people --json '{"companies":{"domains":["surfe.com"]},"people":{"jobTitles":["CTO"]}}'
- CSV output is useful for bulk exports: surfer search people ... -o csv > leads.csv
- surfer whoami shows which account you're authenticated as (no API call).
`

var aiCmd = &cobra.Command{
	Use:    "ai",
	Short:  "Print a system prompt for AI agents to use this CLI",
	Long:   "Outputs a structured prompt that describes all available commands, flags, and usage patterns. Feed this to an AI agent so it knows how to use surfer.",
	Hidden: false,
	Run: func(cmd *cobra.Command, _ []string) {
		fmt.Printf(aiPrompt, pkg.BuildVersion)
	},
}

func init() {
	rootCmd.AddCommand(aiCmd)
}
