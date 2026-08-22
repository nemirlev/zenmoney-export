package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/nemirlev/zenmoney-export/v2/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func TestRootFlagsOverrideEnvironmentAndConfigFile(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)
	for _, name := range []string{"ZEN_API_TOKEN", "TOKEN", "DB_URL", "DB_CONFIG", "DB_TYPE", "LOG_LEVEL"} {
		t.Setenv(name, "")
	}

	configPath := filepath.Join(t.TempDir(), "zenexport.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte(`
token: file-token
db_url: postgres://file.example/zenmoney
log_level: debug
`), 0o600))
	t.Setenv("ZEN_API_TOKEN", "env-token")
	t.Setenv("DB_URL", "postgres://env.example/zenmoney")
	t.Setenv("LOG_LEVEL", "warn")

	root := &RootCommand{opts: &config.CommandOptions{}}
	root.cmd = &cobra.Command{Use: "zenexport"}
	root.addFlags()
	require.NoError(t, root.cmd.PersistentFlags().Set("config", configPath))
	require.NoError(t, root.cmd.PersistentFlags().Set("token", "flag-token"))
	require.NoError(t, root.cmd.PersistentFlags().Set("db-type", "postgres"))
	require.NoError(t, root.cmd.PersistentFlags().Set("db-url", "postgres://flag.example/zenmoney"))
	require.NoError(t, root.cmd.PersistentFlags().Set("log-level", "error"))

	cfg, err := config.NewConfigFromViper(root.opts.ConfigFile)

	require.NoError(t, err)
	require.Equal(t, "flag-token", cfg.ZenMoneyToken)
	require.Equal(t, "postgres", cfg.DBType)
	require.Equal(t, "postgres://flag.example/zenmoney", cfg.DBConfig)
	require.Equal(t, "error", cfg.LogLevel)
}

func TestRootExposesOnlyImplementedGlobalOptions(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)

	root := NewRootCommand()
	for _, name := range []string{"config", "token", "db-type", "db-url", "log-level"} {
		require.NotNilf(t, root.PersistentFlags().Lookup(name), "missing --%s", name)
	}
	require.Nil(t, root.PersistentFlags().Lookup("format"))
}
