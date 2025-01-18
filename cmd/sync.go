package cmd

import (
	"log/slog"
	"strings"
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

	// 1. Подготовка
	slog.Info("Initializing database connection")
	// TODO: Инициализация подключения к БД

	slog.Info("Initializing ZenMoney API client")
	// TODO: Инициализация клиента API

	// 2. Получение последней синхронизации
	slog.Info("Getting last sync timestamp")
	// TODO: Получение timestamp последней синхронизации

	// 3. Синхронизация по сущностям
	for _, entity := range strings.Split(entities, ",") {
		entity = strings.TrimSpace(entity)
		slog.Info("Starting entity sync", "entity", entity)

		// 3.1. Получение данных из API
		slog.Info("Fetching data from API", "entity", entity)
		// TODO: Получение данных из API

		// 3.2. Сохранение в БД
		if !dryRun {
			slog.Info("Saving data to database", "entity", entity)
			// TODO: Сохранение данных в БД
		} else {
			slog.Info("Dry run - skipping database save", "entity", entity)
		}

		slog.Info("Entity sync completed", "entity", entity)
	}

	// 4. Обновление статуса синхронизации
	if !dryRun {
		slog.Info("Updating sync status")
		// TODO: Обновление статуса синхронизации
	}

	slog.Info("Sync process completed successfully")
	return nil
}
