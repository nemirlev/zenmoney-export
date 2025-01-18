// cmd/root.go
package cmd

import (
	"fmt"
	"os"

	"github.com/nemirlev/zenmoney-export/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string
	cfg     *config.Config
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "zenexport",
	Short: "A tool for exporting and syncing ZenMoney data",
	Long: `ZenExport is a CLI tool that allows you to export and synchronize data 
from ZenMoney to your local database. It supports multiple database types 
and provides various options for data synchronization and export.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Skip config validation for help and version commands
		if cmd.Name() == "help" || cmd.Name() == "version" {
			return nil
		}

		var err error
		// Load config after flags are parsed
		cfg, err = config.LoadConfig()
		if err != nil {
			return fmt.Errorf("error loading config: %w", err)
		}

		// Initialize logger
		logger := config.NewLogger(cfg)

		// Validate config
		if err := config.ValidateConfig(cfg); err != nil {
			return fmt.Errorf("invalid configuration: %w", err)
		}

		logger.Info("configuration loaded successfully",
			"db_type", cfg.DBType,
			"log_level", cfg.LogLevel,
			"format", cfg.OutputFormat,
		)

		return validateToken(cmd)
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.zenexport.yaml)")
	rootCmd.PersistentFlags().StringP("token", "t", "", "ZenMoney API token")
	rootCmd.PersistentFlags().String("log-level", "info", "log level (debug, info, warn, error)")
	rootCmd.PersistentFlags().StringP("format", "f", "json", "output format (text, json)")

	// Bind flags to viper
	viper.BindPFlag("token", rootCmd.PersistentFlags().Lookup("token"))
	viper.BindPFlag("log_level", rootCmd.PersistentFlags().Lookup("log-level"))
	viper.BindPFlag("format", rootCmd.PersistentFlags().Lookup("format"))
}

func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Search config in home directory
		viper.AddConfigPath(home)
		viper.SetConfigName(".zenexport")
	}

	// Read environment variables
	viper.AutomaticEnv()

	// If a config file is found, read it in
	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}

// validateToken checks if token is provided and valid
func validateToken(cmd *cobra.Command) error {
	// Skip token validation for certain commands
	if cmd.Name() == "check" || cmd.Name() == "migrate" || cmd.Name() == "config" {
		return nil
	}

	// Check if token is provided
	if cfg.ZenMoneyToken == "" {
		return fmt.Errorf("ZenMoney token is required. Set it via --token flag, ZENEXPORT_TOKEN environment variable, or config file")
	}

	// TODO: Add actual token validation logic when implementing API client
	return nil
}
