// config/config.go
package config

import (
	"fmt"
	"github.com/spf13/viper"
	"log/slog"
	"os"
)

const (
	// ConfigFileName is the name of the config file without extension
	ConfigFileName = ".zenexport"
)

// Config holds all configuration for the application
type Config struct {
	// Database configuration
	DBType   string `mapstructure:"db_type"`   // postgres, mysql, clickhouse, etc.
	DBConfig string `mapstructure:"db_config"` // connection string

	// API configuration
	ZenMoneyToken string `mapstructure:"token"` // ZenMoney API token

	// Logging configuration
	LogLevel string `mapstructure:"log_level"` // debug, info, warn, error

	// Output configuration
	OutputFormat string `mapstructure:"format"` // text, json
}

// LoadConfig loads the configuration from all available sources
func LoadConfig() (*Config, error) {
	v := viper.New()

	// Set default values
	setDefaults(v)

	// Setup viper
	setupViper(v)

	// Read config
	config := &Config{}
	if err := v.Unmarshal(config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return config, nil
}

// setDefaults sets default values for configuration
func setDefaults(v *viper.Viper) {
	v.SetDefault("db_type", "postgres")
	v.SetDefault("log_level", "info")
	v.SetDefault("format", "json")
}

// setupViper configures Viper instance
func setupViper(v *viper.Viper) {
	v.AutomaticEnv()

	// Find home directory for default config location
	home, err := os.UserHomeDir()
	if err == nil {
		// Search config in home directory with name ".zenexport" (without extension)
		v.AddConfigPath(home)
		v.SetConfigName(ConfigFileName)
	}

	// Read config file if exists
	if err := v.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", v.ConfigFileUsed())
	}
}

// ValidateConfig validates the configuration
func ValidateConfig(cfg *Config) error {
	// Validate log level
	switch cfg.LogLevel {
	case "debug", "info", "warn", "error":
		// Valid log level
	default:
		return fmt.Errorf("invalid log level: %s, must be one of: debug, info, warn, error", cfg.LogLevel)
	}

	// Validate output format
	switch cfg.OutputFormat {
	case "text", "json":
		// Valid format
	default:
		return fmt.Errorf("invalid output format: %s, must be one of: text, json", cfg.OutputFormat)
	}

	return nil
}

// NewLogger creates a new logger instance with the configured level
func NewLogger(cfg *Config) *slog.Logger {
	var level slog.Level
	switch cfg.LogLevel {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: level,
	})
	logger := slog.New(handler)
	slog.SetDefault(logger)
	return logger
}
