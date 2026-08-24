package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func TestNewConfigFromViperUsesSelectedFile(t *testing.T) {
	resetViperAndEnvironment(t)

	configPath := writeConfig(t, `
token: file-token
db_url: postgres://file.example/zenmoney
log_level: debug
`)

	cfg, err := NewConfigFromViper(configPath)

	require.NoError(t, err)
	require.Equal(t, "file-token", cfg.ZenMoneyToken)
	require.Equal(t, "postgres://file.example/zenmoney", cfg.DBConfig)
	require.Equal(t, "postgres", cfg.DBType)
	require.Equal(t, "debug", cfg.LogLevel)
	require.Equal(t, DefaultMaxResponseSizeMB, cfg.MaxResponseSizeMB)
}

func TestNewConfigFromViperEnvironmentOverridesFile(t *testing.T) {
	resetViperAndEnvironment(t)

	configPath := writeConfig(t, `
token: file-token
db_url: postgres://file.example/zenmoney
log_level: debug
max_response_size_mb: 128
`)
	t.Setenv("ZEN_API_TOKEN", "env-token")
	t.Setenv("DB_URL", "postgres://env.example/zenmoney")
	t.Setenv("LOG_LEVEL", "warn")
	t.Setenv("ZEN_MAX_RESPONSE_SIZE_MB", "512")

	cfg, err := NewConfigFromViper(configPath)

	require.NoError(t, err)
	require.Equal(t, "env-token", cfg.ZenMoneyToken)
	require.Equal(t, "postgres://env.example/zenmoney", cfg.DBConfig)
	require.Equal(t, "warn", cfg.LogLevel)
	require.Equal(t, int64(512), cfg.MaxResponseSizeMB)
}

func TestNewConfigFromViperSupportsLegacyNames(t *testing.T) {
	resetViperAndEnvironment(t)

	t.Setenv("TOKEN", "legacy-token")
	t.Setenv("DB_CONFIG", "postgres://legacy.example/zenmoney")

	configPath := writeConfig(t, "db_config: postgres://file.example/zenmoney\ntoken: file-token\n")
	cfg, err := NewConfigFromViper(configPath)
	require.NoError(t, err)
	require.Equal(t, "legacy-token", cfg.ZenMoneyToken)
	require.Equal(t, "postgres://legacy.example/zenmoney", cfg.DBConfig)
}

func TestNewConfigFromViperReportsMissingSelectedFile(t *testing.T) {
	resetViperAndEnvironment(t)

	_, err := NewConfigFromViper(filepath.Join(t.TempDir(), "missing.yaml"))

	require.ErrorContains(t, err, "read config file")
}

func TestCanonicalEnvironmentNamesTakePriorityOverLegacyNames(t *testing.T) {
	resetViperAndEnvironment(t)

	t.Setenv("ZEN_API_TOKEN", "canonical-token")
	t.Setenv("TOKEN", "legacy-token")
	t.Setenv("DB_URL", "postgres://canonical.example/zenmoney")
	t.Setenv("DB_CONFIG", "postgres://legacy.example/zenmoney")
	configPath := writeConfig(t, "{}")

	cfg, err := NewConfigFromViper(configPath)

	require.NoError(t, err)
	require.Equal(t, "canonical-token", cfg.ZenMoneyToken)
	require.Equal(t, "postgres://canonical.example/zenmoney", cfg.DBConfig)
}

func TestValidateConfig(t *testing.T) {
	valid := Config{
		DBType:            "postgres",
		DBConfig:          "postgres://localhost/zenmoney",
		ZenMoneyToken:     "secret-token",
		LogLevel:          "info",
		MaxResponseSizeMB: DefaultMaxResponseSizeMB,
	}

	tests := []struct {
		name    string
		mutate  func(*Config)
		message string
	}{
		{
			name:    "token is required",
			mutate:  func(cfg *Config) { cfg.ZenMoneyToken = " " },
			message: "API token is required",
		},
		{
			name:    "database URL is required",
			mutate:  func(cfg *Config) { cfg.DBConfig = "" },
			message: "database URL is required",
		},
		{
			name:    "database type is supported",
			mutate:  func(cfg *Config) { cfg.DBType = "mysql" },
			message: "unsupported database type",
		},
		{
			name:    "log level is valid",
			mutate:  func(cfg *Config) { cfg.LogLevel = "verbose" },
			message: "invalid log level",
		},
		{
			name:    "response size is positive",
			mutate:  func(cfg *Config) { cfg.MaxResponseSizeMB = 0 },
			message: "response size must be greater than zero",
		},
		{
			name:    "response size does not overflow",
			mutate:  func(cfg *Config) { cfg.MaxResponseSizeMB = 1 << 44 },
			message: "response size is too large",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := valid
			tt.mutate(&cfg)

			err := ValidateConfig(&cfg)

			require.ErrorContains(t, err, tt.message)
			require.NotContains(
				t,
				err.Error(),
				valid.ZenMoneyToken,
				"validation errors must not expose the API token",
			)
		})
	}
}

func TestMaxResponseSizeBytes(t *testing.T) {
	cfg := Config{MaxResponseSizeMB: 256}

	got, err := cfg.MaxResponseSizeBytes()

	require.NoError(t, err)
	require.Equal(t, int64(256<<20), got)
}

func resetViperAndEnvironment(t *testing.T) {
	t.Helper()
	viper.Reset()
	t.Cleanup(viper.Reset)
	for _, name := range []string{"ZEN_API_TOKEN", "TOKEN", "DB_URL", "DB_CONFIG", "DB_TYPE", "LOG_LEVEL", "ZEN_MAX_RESPONSE_SIZE_MB"} {
		t.Setenv(name, "")
	}
}

func writeConfig(t *testing.T, contents string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "zenexport.yaml")
	require.NoError(t, os.WriteFile(path, []byte(strings.TrimSpace(contents)+"\n"), 0o600))
	return path
}
