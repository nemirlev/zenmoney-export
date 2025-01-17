package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "zenexport",
	Short: "A tool for exporting and syncing ZenMoney data",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		panic(err)
	}
}
