package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Surfe/surfer/internal/auth"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRootCommand_HasSubcommands(t *testing.T) {
	commands := rootCmd.Commands()
	names := make([]string, 0, len(commands))
	for _, c := range commands {
		names = append(names, c.Name())
	}

	assert.Contains(t, names, "login")
	assert.Contains(t, names, "logout")
	assert.Contains(t, names, "search")
	assert.Contains(t, names, "enrich")
	assert.Contains(t, names, "credits")
	assert.Contains(t, names, "version")
	assert.Contains(t, names, "update")
}

func TestRootCommand_Groups(t *testing.T) {
	groups := rootCmd.Groups()
	groupIDs := make([]string, 0, len(groups))
	for _, g := range groups {
		groupIDs = append(groupIDs, g.ID)
	}

	assert.Contains(t, groupIDs, "auth")
	assert.Contains(t, groupIDs, "data")
	assert.Contains(t, groupIDs, "account")
}

func TestLoginCommand_GroupID(t *testing.T) {
	assert.Equal(t, "auth", loginCmd.GroupID)
}

func TestLogoutCommand_GroupID(t *testing.T) {
	assert.Equal(t, "auth", logoutCmd.GroupID)
}

func TestSearchCommand_GroupID(t *testing.T) {
	assert.Equal(t, "data", searchCmd.GroupID)
}

func TestEnrichCommand_GroupID(t *testing.T) {
	assert.Equal(t, "data", enrichCmd.GroupID)
}

func TestCreditsCommand_GroupID(t *testing.T) {
	assert.Equal(t, "account", creditsCmd.GroupID)
}

func TestSearchCommand_HasSubcommands(t *testing.T) {
	commands := searchCmd.Commands()
	names := make([]string, 0, len(commands))
	for _, c := range commands {
		names = append(names, c.Name())
	}

	assert.Contains(t, names, "companies")
	assert.Contains(t, names, "people")
}

func TestEnrichCommand_HasSubcommands(t *testing.T) {
	commands := enrichCmd.Commands()
	names := make([]string, 0, len(commands))
	for _, c := range commands {
		names = append(names, c.Name())
	}

	assert.Contains(t, names, "companies")
	assert.Contains(t, names, "people")
}

func TestEnrichSubcommands_HaveStatus(t *testing.T) {
	for _, parent := range []*cobra.Command{companiesEnrichCmd, peopleEnrichCmd} {
		names := make([]string, 0)
		for _, c := range parent.Commands() {
			names = append(names, c.Name())
		}
		assert.Contains(t, names, "status", "enrich %s should have a status subcommand", parent.Name())
	}
}

func TestPeopleSearchCommand_Flags(t *testing.T) {
	flags := peopleSearchCmd.Flags()
	assert.NotNil(t, flags.Lookup("job-titles"))
	assert.NotNil(t, flags.Lookup("seniorities"))
	assert.NotNil(t, flags.Lookup("departments"))
	assert.NotNil(t, flags.Lookup("company-domains"))
	assert.NotNil(t, flags.Lookup("company-names"))
	assert.NotNil(t, flags.Lookup("countries"))
	assert.NotNil(t, flags.Lookup("industries"))
	assert.NotNil(t, flags.Lookup("limit"))
	assert.NotNil(t, flags.Lookup("people-per-company"))
	assert.NotNil(t, flags.Lookup("page-token"))
}

func TestPeopleEnrichCommand_Flags(t *testing.T) {
	flags := peopleEnrichCmd.Flags()
	assert.NotNil(t, flags.Lookup("first-name"))
	assert.NotNil(t, flags.Lookup("last-name"))
	assert.NotNil(t, flags.Lookup("linkedin"))
	assert.NotNil(t, flags.Lookup("company-name"))
	assert.NotNil(t, flags.Lookup("company-domain"))
	assert.NotNil(t, flags.Lookup("include-email"))
	assert.NotNil(t, flags.Lookup("include-mobile"))
}

func TestPeopleStatusCommand_Args(t *testing.T) {
	// Should require exactly 1 arg
	err := peopleStatusCmd.Args(peopleStatusCmd, []string{})
	assert.Error(t, err)

	err = peopleStatusCmd.Args(peopleStatusCmd, []string{"id-123"})
	assert.NoError(t, err)

	err = peopleStatusCmd.Args(peopleStatusCmd, []string{"a", "b"})
	assert.Error(t, err)
}

func TestCompaniesSearchCommand_Flags(t *testing.T) {
	flags := companiesSearchCmd.Flags()
	assert.NotNil(t, flags.Lookup("domains"))
	assert.NotNil(t, flags.Lookup("countries"))
	assert.NotNil(t, flags.Lookup("industries"))
	assert.NotNil(t, flags.Lookup("domains-excluded"))
	assert.NotNil(t, flags.Lookup("min-employees"))
	assert.NotNil(t, flags.Lookup("max-employees"))
	assert.NotNil(t, flags.Lookup("limit"))
	assert.NotNil(t, flags.Lookup("page-token"))
}

func TestCompaniesEnrichCommand_Flags(t *testing.T) {
	flags := companiesEnrichCmd.Flags()
	assert.NotNil(t, flags.Lookup("domain"))
	assert.NotNil(t, flags.Lookup("external-id"))
}

func TestCompaniesStatusCommand_Args(t *testing.T) {
	err := companiesStatusCmd.Args(companiesStatusCmd, []string{})
	assert.Error(t, err)

	err = companiesStatusCmd.Args(companiesStatusCmd, []string{"id-456"})
	assert.NoError(t, err)
}

func TestVersionCommand_Runs(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"version"})
	err := rootCmd.Execute()
	require.NoError(t, err)
}

func TestRootCommand_Help(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"--help"})
	err := rootCmd.Execute()
	require.NoError(t, err)
}

func TestRootCommand_DebugFlag(t *testing.T) {
	// The debug flag is registered in Execute(), so register it here for testing
	if rootCmd.PersistentFlags().Lookup("debug") == nil {
		rootCmd.PersistentFlags().Bool("debug", false, "Enable debug logging")
	}
	flag := rootCmd.PersistentFlags().Lookup("debug")
	assert.NotNil(t, flag)
	assert.Equal(t, "false", flag.DefValue)
}

func TestOutputFormat(t *testing.T) {
	c := &cobra.Command{}
	c.Flags().String("output", "", "")

	// No flag set → defaults to json.
	assert.Equal(t, "json", outputFormat(c))

	// Explicit value is returned.
	_ = c.Flags().Set("output", "csv")
	assert.Equal(t, "csv", outputFormat(c))
}

func TestColorFuncMap(t *testing.T) {
	funcs := colorFuncMap()
	assert.Contains(t, funcs, "cyan")
	assert.Contains(t, funcs, "green")
	assert.Contains(t, funcs, "yellow")
	assert.Contains(t, funcs, "heading")
	assert.Contains(t, funcs, "rpad")
	assert.Contains(t, funcs, "cmdsInGroup")
	assert.Contains(t, funcs, "cmdsNotInGroup")
}

func TestCmdsInGroup(t *testing.T) {
	cmds := cmdsInGroup(rootCmd.Commands(), "auth")
	names := make([]string, 0, len(cmds))
	for _, c := range cmds {
		names = append(names, c.Name())
	}
	assert.Contains(t, names, "login")
	assert.Contains(t, names, "logout")
}

func TestCmdsNotInGroup(t *testing.T) {
	cmds := cmdsNotInGroup(rootCmd.Commands())
	names := make([]string, 0, len(cmds))
	for _, c := range cmds {
		names = append(names, c.Name())
	}
	assert.Contains(t, names, "version")
	assert.Contains(t, names, "update")
}

func TestBuildPeopleSearchRequest(t *testing.T) {
	// Note: *peopleSearchCmd shares flag state, so this test modifies the global.
	// Cobra's StringSlice flag appends, not replaces, so we test with fresh values.
	peopleSearchCmd.Flags().Set("job-titles", "CTO")
	peopleSearchCmd.Flags().Set("countries", "US")
	peopleSearchCmd.Flags().Set("limit", "20")
	peopleSearchCmd.Flags().Set("people-per-company", "3")

	req := buildPeopleSearchRequest(peopleSearchCmd)
	assert.Equal(t, 20, req.Limit)
	assert.Equal(t, 3, req.PeoplePerCompany)
	assert.NotNil(t, req.People)
	assert.NotNil(t, req.Companies)
	assert.Contains(t, req.Companies.Countries, "US")
}

func TestRedErrWriter(t *testing.T) {
	w := &redErrWriter{}
	n, err := w.Write([]byte("test error"))
	assert.NoError(t, err)
	assert.Greater(t, n, 0)
}

func TestLoginCommand_AlreadyLoggedIn(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	configDir := filepath.Join(dir, "surfer")
	os.MkdirAll(configDir, 0o750)

	// Write a token file
	store := &auth.TokenStore{
		AccessToken: "existing-token",
		ExpiresAt:   time.Now().Add(time.Hour),
	}
	data, _ := json.MarshalIndent(store, "", "  ")
	os.WriteFile(filepath.Join(configDir, "tokens.json"), data, 0o600)

	// The login command should detect already logged in
	// (We can't fully test the browser flow, but we can test the IsLoggedIn check)
	assert.True(t, auth.IsLoggedIn())
}

func TestBuildCompaniesSearchRequest(t *testing.T) {
	cmd := *companiesSearchCmd
	cmd.Flags().Set("domains", "stripe.com")
	cmd.Flags().Set("min-employees", "50")
	cmd.Flags().Set("max-employees", "500")

	req := buildCompaniesSearchRequest(&cmd, nil)
	assert.Contains(t, req.Filters.Domains, "stripe.com")
	assert.NotNil(t, req.Filters.EmployeeCount)
	assert.Equal(t, 50, req.Filters.EmployeeCount.From)
	assert.Equal(t, 500, req.Filters.EmployeeCount.To)
}

func TestBuildCompaniesSearchRequest_PositionalArgs(t *testing.T) {
	cmd := *companiesSearchCmd
	req := buildCompaniesSearchRequest(&cmd, []string{"apple", "google"})
	assert.Contains(t, req.Filters.Names, "apple")
	assert.Contains(t, req.Filters.Names, "google")
}
