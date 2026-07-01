package cmd

import (
	"github.com/spf13/cobra"

	"github.com/Surfe/surfer/pkg"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the current version",
	Run: func(_ *cobra.Command, _ []string) {
		pkg.PrintVersion()
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
