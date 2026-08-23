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
	for _, name := range []string{"ZEN_API_TOKEN", "TOKEN", "DB_URL", "DB_CONFIG", "DB_TYPE", "LOG_LEVEL", "ZEN_MAX_RESPONSE_SIZE_MB"} {
		t.Setenv(name, "")
	}

	configPath := filepath.Join(t.TempDir(), "zenexport.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte(`
token: file-token
db_url: postgres://file.example/zenmoney
log_level: debug
max_response_size_mb: 128
`), 0o600))
	t.Setenv("ZEN_API_TOKEN", "env-token")
	t.Setenv("DB_URL", "postgres://env.example/zenmoney")
	t.Setenv("LOG_LEVEL", "warn")
	t.Setenv("ZEN_MAX_RESPONSE_SIZE_MB", "512")

	root := &RootCommand{opts: &config.CommandOptions{}}
	root.cmd = &cobra.Command{Use: "zenexport"}
	root.addFlags()
	require.NoError(t, root.cmd.PersistentFlags().Set("config", configPath))
	require.NoError(t, root.cmd.PersistentFlags().Set("token", "flag-token"))
	require.NoError(t, root.cmd.PersistentFlags().Set("db-type", "postgres"))
	require.NoError(t, root.cmd.PersistentFlags().Set("db-url", "postgres://flag.example/zenmoney"))
	require.NoError(t, root.cmd.PersistentFlags().Set("log-level", "error"))
	require.NoError(t, root.cmd.PersistentFlags().Set("max-response-size-mb", "1024"))

	cfg, err := config.NewConfigFromViper(root.opts.ConfigFile)

	require.NoError(t, err)
	require.Equal(t, "flag-token", cfg.ZenMoneyToken)
	require.Equal(t, "postgres", cfg.DBType)
	require.Equal(t, "postgres://flag.example/zenmoney", cfg.DBConfig)
	require.Equal(t, "error", cfg.LogLevel)
	require.Equal(t, int64(1024), cfg.MaxResponseSizeMB)
}

func TestRootExposesOnlyImplementedGlobalOptions(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)

	root := NewRootCommand()
	for _, name := range []string{"config", "token", "db-type", "db-url", "log-level", "max-response-size-mb"} {
		require.NotNilf(t, root.PersistentFlags().Lookup(name), "missing --%s", name)
	}
	require.Nil(t, root.PersistentFlags().Lookup("format"))
	require.Equal(t, "256", root.PersistentFlags().Lookup("max-response-size-mb").DefValue)
}
