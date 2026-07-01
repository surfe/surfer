package config

import (
	"os"
	"path"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

func GetDir() string {
	if os.Getenv("XDG_CONFIG_HOME") != "" {
		return path.Join(os.Getenv("XDG_CONFIG_HOME"), "surfer")
	}
	return path.Join(os.Getenv("HOME"), ".surfer")
}

// Defaults — can be overridden via config file (~/.surfer/config.yaml)
// or env vars (SURFER_API_URL, SURFER_AUTH_URL, SURFER_CLIENT_ID).
func setDefaults() {
	viper.SetDefault("api-url", "https://api.surfe.com")
	viper.SetDefault("auth-url", "https://eu.prod.surfe.com")
	viper.SetDefault("client-id", "hubspot")
}

func Setup() error {
	viper.SetEnvPrefix("surfer")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.SetConfigType("yaml")
	viper.SetConfigName("config")
	viper.AddConfigPath(GetDir())
	viper.AutomaticEnv()

	setDefaults()

	if _, err := os.Stat(GetDir()); os.IsNotExist(err) {
		if err := os.Mkdir(GetDir(), 0o750); err != nil {
			return err
		}
	}

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return err
		}
	}

	return nil
}

func WriteToDisk() error {
	if err := viper.WriteConfig(); err != nil {
		if err := viper.SafeWriteConfig(); err != nil {
			return err
		}
	}
	return nil
}

func BindFlags(cmd *cobra.Command) {
	flags := cmd.Flags()
	flags.VisitAll(func(f *pflag.Flag) {
		_ = viper.BindPFlag(f.Name, flags.Lookup(f.Name))
	})
}
