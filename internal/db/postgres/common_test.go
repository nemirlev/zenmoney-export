package postgres

import (
	"context"
	"errors"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPostgresStorage(t *testing.T) {
	t.Run("invalid connection string", func(t *testing.T) {
		_, err := NewPostgresStorage("invalid_connection_string")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse postgres config")
	})
}

func TestDecimalStringUsesCanonicalNonExponentialRepresentation(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		value float64
		want  string
	}{
		"fraction":      {value: 0.1, want: "0.1"},
		"trailing zero": {value: 12.50, want: "12.5"},
		"small amount":  {value: 0.00000001, want: "0.00000001"},
		"large amount":  {value: 1e20, want: "100000000000000000000"},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, decimalString(tt.value))
		})
	}
}

func TestOptionalDecimalStringPreservesNull(t *testing.T) {
	t.Parallel()

	var value *float64
	assert.Equal(t, value, optionalDecimalString(value))

	amount := 42.125
	assert.Equal(t, "42.125", optionalDecimalString(&amount))
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
