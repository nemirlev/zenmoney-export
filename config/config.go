package config

import (
	"errors"
	"fmt"
	"github.com/spf13/viper"
	"log/slog"
	"os"
)

type Config struct {
	DBType        string `mapstructure:"db_type"`
	DBConfig      string `mapstructure:"db_config"`
	ZenMoneyToken string `mapstructure:"token"`
	LogLevel      string `mapstructure:"log_level"`
	Format        string `mapstructure:"format"`
}

type CommandOptions struct {
	ConfigFile string
	Token      string
	LogLevel   string
	Format     string
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

func NewConfigFromViper() (*Config, error) {
	if err := initViper(); err != nil {
		return nil, err
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config error: %w", err)
	}

	if err := ValidateConfig(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func initViper() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	viper.AddConfigPath(home)
	viper.SetConfigName(".zenexport")
	viper.SetConfigType("yaml")
	err = viper.BindEnv("db_type", "DB_TYPE")
	if err != nil {
		slog.Error("error binding env", "error", err)
		return err
	}
	err = viper.BindEnv("db_config", "DB_CONFIG")
	if err != nil {
		slog.Error("error binding env", "error", err)
		return err
	}
	err = viper.BindEnv("token", "TOKEN")
	if err != nil {
		slog.Error("error binding env", "error", err)
		return err
	}
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if !errors.As(err, &configFileNotFoundError) {
			return err
		}
	}

	return nil
}

func ValidateConfig(cfg *Config) error {
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
