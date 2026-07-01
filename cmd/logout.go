package cmd

import (
	"fmt"

	"github.com/logrusorgru/aurora/v3"
	"github.com/spf13/cobra"

	"github.com/Surfe/surfer/internal/auth"
	"github.com/Surfe/surfer/pkg/output"
)

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Sign out of Surfe",
	RunE: func(cmd *cobra.Command, _ []string) error {
		store, err := auth.LoadTokens()
		if err != nil {
			fmt.Printf("%s Not currently logged in.\n", output.IconInfo)
			return nil
		}

		// Best-effort server-side logout
		if store.AccessToken != "" {
			_ = auth.Logout(store.AccessToken)
		}

		if err := auth.ClearTokens(); err != nil {
			return fmt.Errorf("failed to clear local tokens: %w", err)
		}

		fmt.Printf("%s %s\n", output.IconSuccess, aurora.Green("Successfully logged out."))
		return nil
	},
}

func init() {
	logoutCmd.GroupID = "auth"
	rootCmd.AddCommand(logoutCmd)
}
