package cmd

import (
	"github.com/nemirlev/zenmoney-export/config"
	"github.com/nemirlev/zenmoney-export/internal/app"
	"github.com/spf13/cobra"
)

func NewSyncCommand(root *RootCommand) *cobra.Command {
	opts := &config.SyncOptions{}

	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Sync data from ZenMoney",
		Long:  `Synchronizes data from ZenMoney to your local database.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			params := &app.SyncParams{
				Entities: opts.Entities,
				DryRun:   opts.DryRun,
				Force:    opts.Force,
				FromDate: opts.FromDate,
				ToDate:   opts.ToDate,
			}

			if opts.IsDaemon {
				return root.app.SyncService.DaemonSync(cmd.Context(), params, opts.Interval)
			}
			return root.app.SyncService.Sync(cmd.Context(), params)
		},
	}

	addSyncFlags(cmd, opts)
	return cmd
}

func addSyncFlags(cmd *cobra.Command, opts *config.SyncOptions) {
	flags := cmd.Flags()
	flags.BoolVarP(&opts.IsDaemon, "daemon", "d", false, "run in daemon mode")
	flags.IntVar(&opts.Interval, "interval", 30, "sync interval in minutes")
	flags.StringVar(&opts.FromDate, "from", "", "start date")
	flags.StringVar(&opts.ToDate, "to", "", "end date")
	flags.StringVar(&opts.Entities, "entities", "all", "entities to sync")
	flags.BoolVar(&opts.Force, "force", false, "force full sync")
	flags.BoolVar(&opts.DryRun, "dry-run", false, "dry run mode")
}
