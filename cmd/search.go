package cmd

import (
	"github.com/spf13/cobra"
)

var searchCmd = &cobra.Command{
	Use:   "search",
	Short: "Search for companies or people",
	Long:  `Search Surfe for companies or people using filters like domain, name, industry, size, job title, or location.`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		return cmd.Help()
	},
}

func init() {
	searchCmd.GroupID = "data"
	rootCmd.AddCommand(searchCmd)
}
