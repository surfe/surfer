package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"text/template"

	"github.com/logrusorgru/aurora/v3"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/Surfe/surfer/internal/client"
	"github.com/Surfe/surfer/pkg"
	"github.com/Surfe/surfer/pkg/config"
	"github.com/Surfe/surfer/pkg/output"
	"github.com/Surfe/surfer/pkg/version"
)

// outputFormat returns the --output flag value from any command.
func outputFormat(cmd *cobra.Command) string {
	f, _ := cmd.Flags().GetString("output")
	if f == "" {
		return "json"
	}
	return f
}

// newAPIClient returns the API client used by all commands.
// Override in tests to inject a mock server.
var newAPIClient = func() *client.Client {
	return client.New()
}

type redErrWriter struct{}

func (e *redErrWriter) Write(p []byte) (int, error) {
	return fmt.Fprint(os.Stderr, aurora.Red(string(p)))
}

var rootCmd = &cobra.Command{
	Use:               "surfer",
	CompletionOptions: cobra.CompletionOptions{DisableDefaultCmd: true},
	Short:             "Surfer — Surfe API CLI",
	Long: fmt.Sprintf(`
   _____ __  ______  ________________
  / ___// / / / __ \/ ____/ ____/ __ \
  \__ \/ / / / /_/ / /_  / __/ / /_/ /
 ___/ / /_/ / _, _/ __/ / /___/ _, _/
/____/\____/_/ |_/_/   /_____/_/ |_|

  v%s (%s)`, pkg.BuildVersion, pkg.BuildCommit),
}

const colorHelpTemplate = `{{cyan .Long}}

{{heading "Usage:"}}
  {{yellow .UseLine}}{{if .HasAvailableSubCommands}} [command]{{end}}
{{- if .HasAvailableSubCommands}}
{{- range .Groups}}

{{heading .Title}}
{{- range (cmdsInGroup $.Commands .ID)}}
  {{green (rpad .Name .NamePadding)}}  {{.Short}}
{{- end}}
{{- end}}

{{- if (cmdsNotInGroup .Commands)}}

{{heading "Additional Commands:"}}
{{- range (cmdsNotInGroup .Commands)}}
  {{green (rpad .Name .NamePadding)}}  {{.Short}}
{{- end}}
{{- end}}
{{- end}}
{{- if .HasAvailableLocalFlags}}

{{heading "Flags:"}}
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}
{{- end}}

Use "{{yellow (print .CommandPath " [command] --help")}}" for more information about a command.
`

func cmdsInGroup(cmds []*cobra.Command, groupID string) []*cobra.Command {
	var result []*cobra.Command
	for _, c := range cmds {
		if c.IsAvailableCommand() && c.GroupID == groupID {
			result = append(result, c)
		}
	}
	return result
}

func cmdsNotInGroup(cmds []*cobra.Command) []*cobra.Command {
	var result []*cobra.Command
	for _, c := range cmds {
		if c.IsAvailableCommand() && c.GroupID == "" {
			result = append(result, c)
		}
	}
	return result
}

func colorFuncMap() template.FuncMap {
	au := aurora.NewAurora(true)
	return template.FuncMap{
		"cyan":           func(s any) string { return au.Cyan(s).String() },
		"green":          func(s any) string { return au.Green(s).String() },
		"yellow":         func(s any) string { return au.Yellow(s).String() },
		"heading":        func(s string) string { return au.Bold(au.Blue(s)).String() },
		"rpad":           func(s string, padding int) string { return fmt.Sprintf("%-*s", padding, s) },
		"cmdsInGroup":    cmdsInGroup,
		"cmdsNotInGroup": cmdsNotInGroup,
	}
}

func init() {
	rootCmd.AddGroup(
		&cobra.Group{ID: "data", Title: "🔍 Search & Enrich:"},
		&cobra.Group{ID: "auth", Title: "🔑 Authentication:"},
		&cobra.Group{ID: "account", Title: "⚙️  Account:"},
	)

	cobra.AddTemplateFuncs(colorFuncMap())
	rootCmd.SetHelpTemplate(colorHelpTemplate)
}

func Execute() error {
	err := config.Setup()
	if err != nil {
		return fmt.Errorf("could not setup config: %w", err)
	}

	rootCmd.SetErr(&redErrWriter{})

	rootCmd.PersistentPostRun = func(cmd *cobra.Command, args []string) {
		if pkg.BuildVersion == "main" {
			return
		}
		latest, hasUpdate := version.CheckForUpdate(pkg.BuildVersion)
		if hasUpdate {
			fmt.Fprintln(os.Stderr, aurora.Red(
				fmt.Sprintf("\n%s A new version is available: v%s %s Run 'surfer update' to upgrade.", output.IconRocket, latest, output.IconArrow),
			))
		}
	}

	rootCmd.PersistentFlags().Bool("debug", false, "Enable debug logging")
	rootCmd.PersistentFlags().StringP("output", "o", "json", "Output format: json or csv")

	consoleWriter := zerolog.NewConsoleWriter(func(w *zerolog.ConsoleWriter) {
		w.PartsExclude = []string{zerolog.TimestampFieldName}
		w.Out = os.Stderr
		w.TimeFormat = "15:04:05"
	})

	log.Logger = log.With().Timestamp().Logger().Output(consoleWriter)
	zerolog.SetGlobalLevel(zerolog.Disabled)

	cobra.OnInitialize(func() {
		if rootCmd.Flag("debug").Value.String() == "true" {
			zerolog.SetGlobalLevel(zerolog.DebugLevel)
		}
	})

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	return rootCmd.ExecuteContext(ctx)
}
