package db_test

import (
	"context"
	"testing"

	"github.com/nemirlev/zenmoney-export/v2/internal/db"
	"github.com/nemirlev/zenmoney-export/v2/internal/interfaces"
	"github.com/stretchr/testify/require"
)

func TestNewStorage(t *testing.T) {
	ctx := context.Background()

	storage, err := db.NewStorage(
		ctx,
		interfaces.PostgresStorage,
		"postgres://user:pass@localhost:5432/dbname",
	)
	require.NoError(t, err)
	require.NotNil(t, storage)
	postgresStorage := storage
	t.Cleanup(func() { require.NoError(t, postgresStorage.Close(context.Background())) })

	storage, err = db.NewStorage(ctx, "InvalidStorage", "")
	require.ErrorContains(t, err, "unsupported storage type")
	require.Nil(t, storage)
}
