package postgres

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/nemirlev/zenmoney-export/v2/internal/interfaces"
	"github.com/nemirlev/zenmoney-go-sdk/v3/models"
	"github.com/stretchr/testify/require"
)

func TestPostgresCompatibility(t *testing.T) {
	dsn := os.Getenv("POSTGRES_TEST_DSN")
	if dsn == "" {
		t.Skip("POSTGRES_TEST_DSN is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	storage, err := NewPostgresStorage(dsn)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, storage.Close(context.Background()))
	})
	require.NoError(t, storage.Ping(ctx))

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
}
