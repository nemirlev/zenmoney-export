package db_test

import (
	"context"
	"github.com/nemirlev/zenmoney-export/v2/internal/interfaces"
	"testing"

	"github.com/nemirlev/zenmoney-export/v2/internal/db"
	"github.com/nemirlev/zenmoney-export/v2/mocks"
	"github.com/stretchr/testify/assert"
)

func TestNewStorage(t *testing.T) {
	ctx := context.Background()

	// Create a mock Storage
	mockStorage := mocks.NewStorage(t)

	// Create a new storage
	storage, err := db.NewStorage(ctx, interfaces.PostgresStorage, "postgres://user:pass@localhost:5432/dbname")
	assert.NoError(t, err)
	assert.NotNil(t, storage)

	// Test invalid storage type
	storage, err = db.NewStorage(ctx, "InvalidStorage", "")
	assert.Error(t, err)
	assert.Nil(t, storage)

	// Verify mock expectations
	mockStorage.AssertExpectations(t)
}
