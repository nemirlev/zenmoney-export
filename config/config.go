package config

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	DBType        string `mapstructure:"db_type"`
	DBConfig      string `mapstructure:"db_config"`
	ZenMoneyToken string `mapstructure:"token"`
	LogLevel      string `mapstructure:"log_level"`
}

type CommandOptions struct {
	ConfigFile string
	Token      string
	LogLevel   string
	DBType     string
	DBConfig   string
}

type SyncOptions struct {
	CommandOptions
	IsDaemon  bool
	Interval  int
	Entities  string
	BatchSize int
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
		DBType:        viper.GetString("db_type"),
		DBConfig:      dbConfig,
		ZenMoneyToken: viper.GetString("token"),
		LogLevel:      viper.GetString("log_level"),
	}

	if err := ValidateConfig(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func initViper(configFile string) error {
	viper.SetDefault("db_type", "postgres")
	viper.SetDefault("log_level", "info")

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

	switch cfg.LogLevel {
	case "debug", "info", "warn", "error", "":
	default:
		return fmt.Errorf("invalid log level: %s", cfg.LogLevel)
	}
	return nil
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
