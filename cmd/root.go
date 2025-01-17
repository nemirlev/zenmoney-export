package cmd

import (
	"fmt"
	"github.com/nemirlev/zenmoney-export/config"
	"github.com/spf13/cobra"
	"os"
)

// rootCmd is the base command
var rootCmd = newRootCmd()

// newRootCmd creates the root command
func newRootCmd() *cobra.Command {
	cfg := config.LoadConfig()
	logger := config.NewLogger(cfg)

	rootCmd := &cobra.Command{
		Use:   "zenexport",
		Short: "CLI for synchronizing ZenMoney data",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Ensure the token is set
			if cfg.ZenMoneyToken == "" {
				return fmt.Errorf("ZenMoney token is required")
			}

			logger.Info("Configuration loaded", "config", cfg)
			return nil
		},
	}

	// Example flag for overriding token (optional)
	rootCmd.PersistentFlags().String("token", cfg.ZenMoneyToken, "ZenMoney API token")
	rootCmd.PersistentFlags().String("log-level", cfg.LogLevel, "Log level")

	return rootCmd
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main().
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
