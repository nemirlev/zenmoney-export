package config

import (
	"errors"
	"fmt"
	"log/slog"
	"math"
	"os"
	"strings"

	"github.com/spf13/viper"
)

const (
	DefaultMaxResponseSizeMB int64 = 256
	bytesPerMiB                    = int64(1 << 20)
)

type Config struct {
	DBType            string `mapstructure:"db_type"`
	DBConfig          string `mapstructure:"db_config"`
	ZenMoneyToken     string `mapstructure:"token"`
	LogLevel          string `mapstructure:"log_level"`
	MaxResponseSizeMB int64  `mapstructure:"max_response_size_mb"`
}

type CommandOptions struct {
	ConfigFile        string
	Token             string
	LogLevel          string
	DBType            string
	DBConfig          string
	MaxResponseSizeMB int64
}

type SyncOptions struct {
	CommandOptions
	IsDaemon  bool
	Interval  int
	Entities  string
	BatchSize int
	WriteMode string
	Force     bool
	DryRun    bool
}

func NewConfigFromViper(configFiles ...string) (*Config, error) {
	configFile := ""
	if len(configFiles) > 0 {
		configFile = configFiles[0]
	}

	if err := initViper(configFile); err != nil {
		return nil, err
	}

	dbConfig := viper.GetString("db_url")
	if dbConfig == "" {
		// Keep accepting the original YAML key. Environment variables and CLI
		// flags are bound to db_url so that their precedence remains explicit.
		dbConfig = viper.GetString("db_config")
	}

	cfg := Config{
		DBType:            viper.GetString("db_type"),
		DBConfig:          dbConfig,
		ZenMoneyToken:     viper.GetString("token"),
		LogLevel:          viper.GetString("log_level"),
		MaxResponseSizeMB: viper.GetInt64("max_response_size_mb"),
	}

	if err := ValidateConfig(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func initViper(configFile string) error {
	viper.SetDefault("db_type", "postgres")
	viper.SetDefault("log_level", "info")
	viper.SetDefault("max_response_size_mb", DefaultMaxResponseSizeMB)

	if configFile != "" {
		viper.SetConfigFile(configFile)
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		viper.AddConfigPath(home)
		viper.SetConfigName(".zenexport")
		viper.SetConfigType("yaml")
	}

	envBindings := []struct {
		key   string
		names []string
	}{
		{key: "db_type", names: []string{"DB_TYPE"}},
		{key: "db_url", names: []string{"DB_URL", "DB_CONFIG"}},
		{key: "token", names: []string{"ZEN_API_TOKEN", "TOKEN"}},
		{key: "log_level", names: []string{"LOG_LEVEL"}},
		{key: "max_response_size_mb", names: []string{"ZEN_MAX_RESPONSE_SIZE_MB"}},
	}
	for _, binding := range envBindings {
		args := append([]string{binding.key}, binding.names...)
		if err := viper.BindEnv(args...); err != nil {
			return fmt.Errorf("bind environment variable for %s: %w", binding.key, err)
		}
	}

	if err := viper.ReadInConfig(); err != nil {
		if configFile != "" {
			return fmt.Errorf("read config file %q: %w", configFile, err)
		}
		if _, ok := errors.AsType[viper.ConfigFileNotFoundError](err); !ok {
			return err
		}
	}

	return nil
}

func ValidateConfig(cfg *Config) error {
	if strings.TrimSpace(cfg.ZenMoneyToken) == "" {
		return errors.New("ZenMoney API token is required")
	}
	if strings.TrimSpace(cfg.DBConfig) == "" {
		return errors.New("database URL is required")
	}
	if cfg.DBType != "postgres" {
		return fmt.Errorf("unsupported database type %q (supported: postgres)", cfg.DBType)
	}
	if _, err := cfg.MaxResponseSizeBytes(); err != nil {
		return err
	}

	switch cfg.LogLevel {
	case "debug", "info", "warn", "error", "":
	default:
		return fmt.Errorf("invalid log level: %s", cfg.LogLevel)
	}
	return nil
}

// MaxResponseSizeBytes converts the configured MiB limit to the byte value
// expected by the ZenMoney SDK without allowing an int64 overflow.
func (cfg *Config) MaxResponseSizeBytes() (int64, error) {
	if cfg.MaxResponseSizeMB <= 0 {
		return 0, errors.New("maximum response size must be greater than zero")
	}
	if cfg.MaxResponseSizeMB > math.MaxInt64/bytesPerMiB {
		return 0, errors.New("maximum response size is too large")
	}
	return cfg.MaxResponseSizeMB * bytesPerMiB, nil
}

func NewLogger(cfg *Config) *slog.Logger {
	level := slog.LevelInfo
	switch cfg.LogLevel {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}

	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: level,
	})
	return slog.New(handler)
}
