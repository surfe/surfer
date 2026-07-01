package cmd

import (
	"context"
	"fmt"
	"runtime"

	"github.com/creativeprojects/go-selfupdate"
	"github.com/spf13/cobra"

	"github.com/Surfe/surfer/pkg"
	"github.com/Surfe/surfer/pkg/output"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update surfer to the latest version",
	Long:  "Checks GitHub releases for a newer version and updates the binary in-place.",
	RunE: func(cmd *cobra.Command, _ []string) error {
		return runUpdate(cmd.Context())
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)
}

func runUpdate(ctx context.Context) error {
	currentVersion := pkg.BuildVersion
	if currentVersion == "main" {
		return fmt.Errorf("cannot update a development build — install a released version first")
	}

	fmt.Printf("Current version: %s (%s/%s)\n", currentVersion, runtime.GOOS, runtime.GOARCH)
	fmt.Println(output.IconInfo + " Checking for updates...")

	source, err := selfupdate.NewGitHubSource(selfupdate.GitHubConfig{})
	if err != nil {
		return fmt.Errorf("create update source: %w", err)
	}

	updater, err := selfupdate.NewUpdater(selfupdate.Config{
		Source: source,
		Validator: &selfupdate.ChecksumValidator{
			UniqueFilename: "checksums.txt",
		},
	})
	if err != nil {
		return fmt.Errorf("create updater: %w", err)
	}

	latest, found, err := updater.DetectLatest(ctx, selfupdate.NewRepositorySlug("Surfe", "surfer"))
	if err != nil {
		return fmt.Errorf("detect latest version: %w", err)
	}

	if !found {
		return fmt.Errorf("no releases found for Surfe/surfer")
	}

	if !latest.GreaterThan(currentVersion) {
		fmt.Printf("%s Already up to date (v%s)\n", output.IconSuccess, currentVersion)
		return nil
	}

	fmt.Printf("%s Updating to v%s...\n", output.IconRocket, latest.Version())

	exe, err := selfupdate.ExecutablePath()
	if err != nil {
		return fmt.Errorf("locate executable: %w", err)
	}

	if err := updater.UpdateTo(ctx, latest, exe); err != nil {
		return fmt.Errorf("update failed: %w", err)
	}

	fmt.Printf("%s Successfully updated to v%s\n", output.IconSuccess, latest.Version())
	return nil
}
