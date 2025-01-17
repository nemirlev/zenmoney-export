package config

import (
	"log/slog"
	"os"

	"github.com/spf13/viper"
)

type Config struct {
	DBType        string // postgres, mysql, clickhouse, etc.
	DBConfig      string // connection string, example: "host=localhost port=5432 user=postgres password=postgres dbname=zenmoney sslmode=disable"
	ZenMoneyToken string // ZenMoney API token, get on https://zerro.app/token
	LogLevel      string // debug, info, warn, error
}

func LoadConfig() *Config {
	viper.SetDefault("DB_TYPE", "postgres")
	viper.SetDefault("LOG_LEVEL", "info")
	viper.AutomaticEnv()

	config := &Config{
		DBType:        viper.GetString("DB_TYPE"),
		DBConfig:      viper.GetString("DB_CONFIG"),
		ZenMoneyToken: viper.GetString("ZENMONEY_TOKEN"),
		LogLevel:      viper.GetString("LOG_LEVEL"),
	}

	return config
}

// NewLogger creates a new logger instance
// TODO: make logic for setting log level from config
func NewLogger(config *Config) *slog.Logger {
	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	logger := slog.New(handler)
	slog.SetDefault(logger)
	return logger
}
