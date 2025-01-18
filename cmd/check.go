package cmd

import (
	"github.com/spf13/cobra"
	"log/slog"
)

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Perform system checks",
	Long: `Performs various checks and validations of the system:
- Database connection
- API token validity
- Migrations status

Example:
  zenexport check --db-connection --api-token
  zenexport check --migrations`,
	RunE: runCheck,
}

func init() {
	rootCmd.AddCommand(checkCmd)

	checkCmd.Flags().Bool("db-connection", false, "check database connection")
	checkCmd.Flags().Bool("api-token", false, "validate API token")
	checkCmd.Flags().Bool("migrations", false, "check if migrations are up to date")

	// Если ни один флаг не указан, проверяем всё
	checkCmd.Flags().Bool("all", false, "run all checks")
}

func runCheck(cmd *cobra.Command, args []string) error {
	checkDB, _ := cmd.Flags().GetBool("db-connection")
	checkToken, _ := cmd.Flags().GetBool("api-token")
	checkMigrations, _ := cmd.Flags().GetBool("migrations")
	checkAll, _ := cmd.Flags().GetBool("all")

	// Если ни один флаг не указан или указан --all, проверяем всё
	if checkAll || (!checkDB && !checkToken && !checkMigrations) {
		checkDB = true
		checkToken = true
		checkMigrations = true
	}

	slog.Info("Starting system checks")

	// 1. Проверка подключения к БД
	if checkDB {
		slog.Info("Checking database connection")
		if err := checkDatabase(); err != nil {
			slog.Error("Database check failed", "error", err)
		} else {
			slog.Info("Database connection check passed")
		}
	}

	// 2. Проверка токена API
	if checkToken {
		slog.Info("Checking API token")
		if err := checkAPIToken(); err != nil {
			slog.Error("API token check failed", "error", err)
		} else {
			slog.Info("API token check passed")
		}
	}

	// 3. Проверка миграций
	if checkMigrations {
		slog.Info("Checking migrations status")
		if err := checkMigrationsStatus(); err != nil {
			slog.Error("Migrations check failed", "error", err)
		} else {
			slog.Info("Migrations check passed")
		}
	}

	slog.Info("System checks completed")
	return nil
}

func checkDatabase() error {
	// TODO: Подключение к БД и проверка соединения
	return nil
}

func checkAPIToken() error {
	// TODO: Проверка валидности токена через API
	return nil
}

func checkMigrationsStatus() error {
	// TODO: Проверка статуса миграций
	return nil
}
