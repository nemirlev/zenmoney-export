package cmd

import (
	"github.com/golang-migrate/migrate/v4"
	"github.com/spf13/cobra"
	"log"
)

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Apply database migrations",
	Run: func(cmd *cobra.Command, args []string) {
		m, err := migrate.New(
			"file://migrations",
			"postgres://your-db-url")
		if err != nil {
			log.Fatalf("Migration init failed: %v", err)
		}

		if err := m.Up(); err != nil && err != migrate.ErrNoChange {
			log.Fatalf("Migration failed: %v", err)
		}
		log.Println("Migrations applied successfully")
	},
}

func init() {
	rootCmd.AddCommand(migrateCmd)
}
