package postgres

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/nemirlev/zenmoney-export/v2/internal/interfaces"
	"github.com/nemirlev/zenmoney-go-sdk/v3/models"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
)

func TestPostgresCompatibility(t *testing.T) {
	postgresVersion := os.Getenv("POSTGRES_VERSION")
	if postgresVersion == "" {
		postgresVersion = "18"
	}

	migrationFiles, err := filepath.Glob(
		filepath.Join("..", "..", "..", "migrations", "postgres", "*.up.sql"),
	)
	require.NoError(t, err)
	require.NotEmpty(t, migrationFiles)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	container, err := tcpostgres.Run(
		ctx,
		"postgres:"+postgresVersion+"-alpine",
		tcpostgres.WithDatabase("zenexport"),
		tcpostgres.WithUsername("zenexport"),
		tcpostgres.WithPassword("zenexport"),
		tcpostgres.BasicWaitStrategies(),
	)
	testcontainers.CleanupContainer(t, container)
	require.NoError(t, err)

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)
	applyMigrations(t, ctx, dsn, migrationFiles)

	storage, err := NewPostgresStorage(dsn)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, storage.Close(context.Background()))
	})
	require.NoError(t, storage.Ping(ctx))
	t.Logf("testing PostgreSQL %s", postgresVersion)

	transactions := []struct {
		id   string
		mode interfaces.WriteMode
	}{
		{id: "00000000-0000-0000-0000-000000000015", mode: interfaces.WriteModeBatch},
		{id: "00000000-0000-0000-0000-000000000018", mode: interfaces.WriteModeCopy},
	}

	for index, test := range transactions {
		transaction := models.Transaction{
			ID:            test.id,
			User:          1,
			Date:          "2026-08-24",
			Income:        float64(index) + 10.25,
			Outcome:       float64(index) + 1.5,
			Changed:       int64(index + 1),
			Created:       int64(index + 1),
			IncomeAccount: "00000000-0000-0000-0000-000000000001",
		}

		err = storage.Save(ctx, &models.Response{
			ServerTimestamp: int64(index + 1),
			Transaction:     []models.Transaction{transaction},
		}, interfaces.SaveOptions{WriteMode: test.mode})
		require.NoError(t, err)

		stored, getErr := storage.GetTransaction(ctx, test.id)
		require.NoError(t, getErr)
		require.Equal(t, transaction.ID, stored.ID)
		require.Equal(t, transaction.Date, stored.Date)
		require.Equal(t, transaction.Income, stored.Income)
		require.Equal(t, transaction.Outcome, stored.Outcome)
	}

	tags := []models.Tag{
		{
			ID:       "00000000-0000-0000-0000-000000000021",
			User:     1,
			Title:    "User category",
			StaticID: nil,
		},
		{
			ID:       "00000000-0000-0000-0000-000000000022",
			User:     1,
			Title:    "System category",
			StaticID: new("69"),
		},
	}
	require.NoError(t, storage.Save(ctx, &models.Response{
		ServerTimestamp: 3,
		Tag:             tags,
	}, interfaces.SaveOptions{}))

	for _, tag := range tags {
		stored, getErr := storage.GetTag(ctx, tag.ID)
		require.NoError(t, getErr)
		require.Equal(t, tag.StaticID, stored.StaticID)
	}
}

func applyMigrations(t *testing.T, ctx context.Context, dsn string, files []string) {
	t.Helper()

	config, err := pgx.ParseConfig(dsn)
	require.NoError(t, err)
	config.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol

	connection, err := pgx.ConnectConfig(ctx, config)
	require.NoError(t, err)
	defer func() { require.NoError(t, connection.Close(context.Background())) }()

	for _, migrationFile := range files {
		migration, readErr := os.ReadFile(migrationFile)
		require.NoError(t, readErr)

		_, execErr := connection.Exec(ctx, string(migration))
		require.NoErrorf(t, execErr, "apply migration %s", filepath.Base(migrationFile))
	}
}
