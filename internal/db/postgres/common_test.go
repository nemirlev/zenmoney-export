package postgres

import (
	"context"
	"errors"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestNewPostgresStorage(t *testing.T) {
	t.Run("invalid connection string", func(t *testing.T) {
		_, err := NewPostgresStorage("invalid_connection_string")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse postgres config")
	})

}

// Тест для метода Close
func TestDB_Close(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)

	db := &DB{pool: mock}

	err = db.Close(context.Background())
	assert.NoError(t, err)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

// Тест для метода Ping
func TestDB_Ping(t *testing.T) {
	t.Run("successful ping", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)

		mock.ExpectPing()

		db := &DB{pool: mock}

		err = db.Ping(context.Background())
		assert.NoError(t, err)

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("failed ping", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)

		expectedErr := errors.New("ping error")
		mock.ExpectPing().WillReturnError(expectedErr)

		db := &DB{pool: mock}

		err = db.Ping(context.Background())
		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})
}
