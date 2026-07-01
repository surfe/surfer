package cmd

import (
	"github.com/spf13/cobra"
)

var enrichCmd = &cobra.Command{
	Use:   "enrich",
	Short: "Enrich companies or people",
	Long:  `Enrich companies (firmographic data) or people (emails, phones, LinkedIn profiles). Enrichment is asynchronous: each command returns an enrichment ID you can poll with the "status" subcommand.`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		return cmd.Help()
	},
}

func init() {
	enrichCmd.GroupID = "data"
	rootCmd.AddCommand(enrichCmd)
}
