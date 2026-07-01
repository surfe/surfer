package cmd

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/Surfe/surfer/pkg/output"
)

var creditsCmd = &cobra.Command{
	Use:   "credits",
	Short: "Check remaining email and mobile credits",
	RunE: func(cmd *cobra.Command, _ []string) error {
		var result any
		if err := newAPIClient().Get("/v1/credits", &result); err != nil {
			return err
		}

		return output.Print(os.Stdout, result, outputFormat(cmd))
	},
}

func init() {
	creditsCmd.GroupID = "account"
	rootCmd.AddCommand(creditsCmd)
}
