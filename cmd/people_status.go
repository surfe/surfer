package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/Surfe/surfer/pkg/output"
)

var peopleStatusCmd = &cobra.Command{
	Use:   "status <enrichment-id>",
	Short: "Check people enrichment status and retrieve results",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		enrichmentID := args[0]

		var result any
		if err := newAPIClient().Get(fmt.Sprintf("/v2/people/enrich/%s", enrichmentID), &result); err != nil {
			return err
		}

		return output.Print(os.Stdout, result, outputFormat(cmd))
	},
}

func init() {
	peopleEnrichCmd.AddCommand(peopleStatusCmd)
}
