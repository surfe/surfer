package cmd

import (
	"fmt"

	"github.com/logrusorgru/aurora/v3"
	"github.com/spf13/cobra"

	"github.com/Surfe/surfer/internal/auth"
	"github.com/Surfe/surfer/pkg/output"
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with Surfe",
	Long:  "Opens a browser to authenticate via OAuth 2.0 (PKCE). Tokens are stored locally.",
	RunE: func(cmd *cobra.Command, _ []string) error {
		if auth.IsLoggedIn() {
			fmt.Printf("%s Already logged in. Use 'surfer logout' to sign out first.\n", output.IconInfo)
			return nil
		}

		store, err := auth.StartLogin()
		if err != nil {
			return fmt.Errorf("login failed: %w", err)
		}

		if err := auth.SaveTokens(store); err != nil {
			return fmt.Errorf("failed to save tokens: %w", err)
		}

		fmt.Printf("%s %s\n", output.IconSuccess, aurora.Green("Successfully logged in!"))
		return nil
	},
}

func init() {
	loginCmd.GroupID = "auth"
	rootCmd.AddCommand(loginCmd)
}
