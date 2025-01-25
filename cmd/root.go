package cmd

import (
	"context"
	"fmt"
	"github.com/nemirlev/zenmoney-export/config"
	"github.com/nemirlev/zenmoney-export/internal/app"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
)

type RootCommand struct {
	cmd  *cobra.Command
	app  *app.Application
	opts *config.CommandOptions
}

func NewRootCommand() *cobra.Command {
	root := &RootCommand{
		opts: &config.CommandOptions{},
	}

	cmd := &cobra.Command{
		Use:               "zenexport",
		Short:             "A tool for exporting and syncing ZenMoney data",
		PersistentPreRunE: root.preRun,
	}

	root.cmd = cmd
	root.addFlags()
	root.addCommands()

	return cmd
}

func (r *RootCommand) addFlags() {
	flags := r.cmd.PersistentFlags()
	flags.StringVar(&r.opts.ConfigFile, "config", "", "config file path")
	flags.StringVar(&r.opts.Token, "token", "", "ZenMoney API token")
	flags.StringVar(&r.opts.LogLevel, "log-level", "info", "log level (debug, info, warn, error)")
	flags.StringVar(&r.opts.Format, "format", "json", "output format (text, json)")

	viper.BindPFlag("token", flags.Lookup("token"))
	viper.BindPFlag("log_level", flags.Lookup("log-level"))
	viper.BindPFlag("format", flags.Lookup("format"))
}

func (r *RootCommand) addCommands() {
	r.cmd.AddCommand(NewSyncCommand(r))
}

func (r *RootCommand) preRun(cmd *cobra.Command, args []string) error {
	if cmd.Name() == "help" || cmd.Name() == "version" {
		return nil
	}

	cfg, err := config.NewConfigFromViper()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	ctx := context.Background()
	application, err := app.NewApplication(ctx, cfg)
	if err != nil {
		return fmt.Errorf("failed to initialize application: %w", err)
	}

	r.app = application
	return nil
}

func Execute() {
	root := NewRootCommand()
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
