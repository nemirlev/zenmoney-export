package cmd

import (
	"context"
	"github.com/nemirlev/zenmoney-export/config"
	"github.com/nemirlev/zenmoney-export/internal/db"
	"github.com/nemirlev/zenmoney-export/internal/interfaces"
	"github.com/nemirlev/zenmoney-go-sdk/v2/api"
	"github.com/nemirlev/zenmoney-go-sdk/v2/models"
	"log"
	"log/slog"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// syncCmd represents the sync command
var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync data from ZenMoney",
	Long: `Synchronizes data from ZenMoney to your local database.
You can specify which entities to sync, date range, and whether to run
in daemon mode with periodic updates.

Example:
  zenexport sync --entities=transactions,accounts --from=2024-01-01
  zenexport sync -d --interval=30 --db-type=clickhouse`,
	RunE: runSync,
}

func init() {
	rootCmd.AddCommand(syncCmd)

	syncCmd.Flags().String("db-type", "clickhouse", "database type (clickhouse, postgres, mysql)")
	syncCmd.Flags().String("db-url", "", "database connection URL")
	syncCmd.Flags().BoolP("daemon", "d", false, "run in daemon mode")
	syncCmd.Flags().Int("interval", 30, "sync interval in minutes (for daemon mode)")
	syncCmd.Flags().String("from", "", "start date for sync (format: YYYY-MM-DD)")
	syncCmd.Flags().String("to", "", "end date for sync (format: YYYY-MM-DD)")
	syncCmd.Flags().String("entities", "transactions,accounts,tags,merchants",
		"comma-separated list of entities to sync")
	syncCmd.Flags().Int("batch-size", 1000, "number of records to process in one batch")
	syncCmd.Flags().Bool("force", false, "force full sync ignoring last sync timestamp")
	syncCmd.Flags().Bool("dry-run", false, "show what would be synced without actually syncing")

	viper.BindPFlag("db_type", syncCmd.Flags().Lookup("db-type"))
	viper.BindPFlag("db_url", syncCmd.Flags().Lookup("db-url"))
}

func runSync(cmd *cobra.Command, args []string) error {
	isDaemon, _ := cmd.Flags().GetBool("daemon")
	if isDaemon {
		return runDaemonMode(cmd)
	}
	return runOneTimeSync(cmd)
}

func runDaemonMode(cmd *cobra.Command) error {
	interval, _ := cmd.Flags().GetInt("interval")

	slog.Info("starting sync in daemon mode", "interval_minutes", interval)

	for {
		if err := performSync(cmd); err != nil {
			slog.Error("sync failed", "error", err)
		}

		slog.Info("waiting for next sync", "minutes", interval)
		time.Sleep(time.Duration(interval) * time.Minute)
	}
}

func runOneTimeSync(cmd *cobra.Command) error {
	slog.Info("starting one-time sync")
	return performSync(cmd)
}

func performSync(cmd *cobra.Command) error {
	entities, _ := cmd.Flags().GetString("entities")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	force, _ := cmd.Flags().GetBool("force")
	batchSize, _ := cmd.Flags().GetInt("batch-size")

	slog.Info("Starting sync process",
		"entities", entities,
		"dry_run", dryRun,
		"force", force,
		"batch_size", batchSize)

	slog.Info("Initializing configuration")
	cfg, err := config.LoadConfig()
	if err != nil {
		slog.Error("Failed to load configuration", "error", err)
		os.Exit(1)
	}

	slog.Info("Initializing database connection")
	ctx := context.Background()
	storage, err := db.NewStorage(ctx, interfaces.PostgresStorage, cfg.DBConfig)
	if err != nil {
		slog.Error("Failed to initialize database connection", "error", err)
		os.Exit(1)
	}

	slog.Info("Initializing ZenMoney API client")
	zen, err := api.NewClient(cfg.ZenMoneyToken)
	if err != nil {
		slog.Error("Failed to initialize ZenMoney API client", "error", err)
		os.Exit(1)
	}

	slog.Info("Getting last sync timestamp")
	lastSync, err := storage.GetLastSyncStatus(ctx)
	if err != nil {
		slog.Error("Failed to get last sync status", "error", err)
		os.Exit(1)
	}

	var zenLastSync models.Response

	if lastSync == (interfaces.SyncStatus{}) || force {
		slog.Info("No previous sync found or force sync enabled - starting full sync")
		zenLastSync, err = zen.FullSync(ctx)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		slog.Info("Starting incremental sync", "last_sync", lastSync.FinishedAt)
		zenLastSync, err = zen.SyncSince(ctx, *lastSync.FinishedAt)
		if err != nil {
			log.Fatal(err)
		}
	}

	slog.Info("Saving all entities to database")
	if !dryRun {
		err = storage.Save(ctx, &zenLastSync)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		slog.Info("Dry run - skipping database save")
	}

	slog.Info("Sync process completed successfully")
	return nil
}
