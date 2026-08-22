package cmd

import (
	"context"
	"errors"
	"fmt"

	"github.com/nemirlev/zenmoney-export/v2/config"
	"github.com/nemirlev/zenmoney-export/v2/internal/app"
	"github.com/spf13/cobra"
)

type syncRunner interface {
	Sync(ctx context.Context, params *app.SyncParams) error
	DaemonSync(ctx context.Context, params *app.SyncParams, interval int) error
}

type applicationCloser interface {
	Close(ctx context.Context) error
}

func NewSyncCommand(root *RootCommand) *cobra.Command {
	opts := &config.SyncOptions{}

	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Sync data from ZenMoney",
		Long:  `Synchronizes data from ZenMoney to your local database.`,
		Args: func(cmd *cobra.Command, args []string) error {
			return validateSyncOptions(opts)
		},
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return validateSyncOptions(opts)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			params := &app.SyncParams{
				Entities: opts.Entities,
				DryRun:   opts.DryRun,
				Force:    opts.Force,
			}

			return runSyncAndClose(
				cmd.Context(),
				root.app.SyncService,
				root.app,
				params,
				opts.IsDaemon,
				opts.Interval,
			)
		},
	}

	addSyncFlags(cmd, opts)
	return cmd
}

func addSyncFlags(cmd *cobra.Command, opts *config.SyncOptions) {
	flags := cmd.Flags()
	flags.BoolVarP(&opts.IsDaemon, "daemon", "d", false, "run in daemon mode")
	flags.IntVar(&opts.Interval, "interval", 30, "sync interval in minutes")
	flags.StringVar(&opts.Entities, "entities", "all", "comma-separated entities to force-fetch, or all")
	flags.BoolVar(&opts.Force, "force", false, "force full sync")
	flags.BoolVar(&opts.DryRun, "dry-run", false, "dry run mode")
}

func validateSyncOptions(opts *config.SyncOptions) error {
	if opts.IsDaemon && opts.Interval <= 0 {
		return fmt.Errorf("--interval must be greater than zero in daemon mode")
	}
	_, _, err := app.ParseSyncEntities(opts.Entities)
	return err
}

func runSyncAndClose(
	ctx context.Context,
	runner syncRunner,
	closer applicationCloser,
	params *app.SyncParams,
	isDaemon bool,
	interval int,
) (err error) {
	defer func() {
		if closeErr := closer.Close(context.WithoutCancel(ctx)); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("close database: %w", closeErr))
		}
	}()

	if isDaemon {
		err = runner.DaemonSync(ctx, params, interval)
	} else {
		err = runner.Sync(ctx, params)
	}

	if ctxErr := ctx.Err(); ctxErr != nil && errors.Is(err, ctxErr) {
		return nil
	}
	return err
}
