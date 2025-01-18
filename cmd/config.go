package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Show current configuration",
	Long: `Display current configuration and its sources. 
This command helps to understand where each configuration value comes from:
- Command line flags
- Environment variables
- Configuration file`,
	RunE: runConfig,
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.Flags().Bool("show-sources", false, "show configuration sources")
}

type configSource struct {
	Value  interface{} `json:"value"`
	Source string      `json:"source"`
}

func runConfig(cmd *cobra.Command, args []string) error {
	showSources, _ := cmd.Flags().GetBool("show-sources")

	if showSources {
		return showConfigWithSources()
	}

	// Show just the values
	config := make(map[string]interface{})
	for _, key := range viper.AllKeys() {
		config[key] = viper.Get(key)
	}

	// Mask sensitive values
	maskSensitiveValues(config)

	return printJSON(config)
}

func showConfigWithSources() error {
	sources := make(map[string]configSource)

	// Check all known configuration keys
	for _, key := range viper.AllKeys() {
		value := viper.Get(key)
		source := determineSource(key)

		if key == "token" || key == "db_password" {
			if strVal, ok := value.(string); ok && strVal != "" {
				value = maskValue(strVal)
			}
		}

		sources[key] = configSource{
			Value:  value,
			Source: source,
		}
	}

	return printJSON(sources)
}

func determineSource(key string) string {
	// Check if value is set by flag
	if cmd := rootCmd.Flags().Lookup(key); cmd != nil {
		if cmd.Changed {
			return "flag"
		}
	}

	// Check if value is set by environment variable
	if os.Getenv(key) != "" {
		return "environment"
	}

	// If value exists in viper but not in flags or env, it's from config file
	if viper.ConfigFileUsed() != "" {
		return fmt.Sprintf("config_file (%s)", viper.ConfigFileUsed())
	}

	return "default"
}

func maskSensitiveValues(config map[string]interface{}) {
	sensitiveKeys := []string{"token", "db_password", "api_key"}
	for _, key := range sensitiveKeys {
		if value, ok := config[key].(string); ok && value != "" {
			config[key] = maskValue(value)
		}
	}
}

func maskValue(value string) string {
	if len(value) <= 4 {
		return "****"
	}
	return value[:4] + "****"
}

func printJSON(v interface{}) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(v)
}
