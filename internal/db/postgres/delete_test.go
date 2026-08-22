package postgres

import (
	"context"
	"errors"
	"testing"

	"github.com/nemirlev/zenmoney-go-sdk/v3/models"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
)

func TestDeleteObjects_Success(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	deletions := []models.Deletion{
		{ID: "1", Object: "account", User: 1, Stamp: 1234567890},
		{ID: "2", Object: "tag", User: 1, Stamp: 1234567890},
		{ID: "3", Object: "country", User: 1, Stamp: 1234567890},
	}

	mock.ExpectBegin()

	mock.ExpectExec(`DELETE FROM account WHERE id = \$1 AND "user" = \$2`).
		WithArgs("1", 1).
		WillReturnResult(pgxmock.NewResult("DELETE", 1))

	mock.ExpectExec(`INSERT INTO deletion_history \(.+\) VALUES \(.+\)`).
		WithArgs("1", "account", 1, int64(1234567890)).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	mock.ExpectExec(`DELETE FROM tag WHERE id = \$1 AND "user" = \$2`).
		WithArgs("2", 1).
		WillReturnResult(pgxmock.NewResult("DELETE", 1))

	mock.ExpectExec(`INSERT INTO deletion_history \(.+\) VALUES \(.+\)`).
		WithArgs("2", "tag", 1, int64(1234567890)).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	mock.ExpectExec(`DELETE FROM country WHERE id = \$1`).
		WithArgs(3).
		WillReturnResult(pgxmock.NewResult("DELETE", 1))

	mock.ExpectExec(`INSERT INTO deletion_history \(.+\) VALUES \(.+\)`).
		WithArgs("3", "country", 1, int64(1234567890)).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	mock.ExpectCommit()

	err = db.DeleteObjects(context.Background(), deletions)
	assert.NoError(t, err)

	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestDeleteObjects_QueryError tests the case when there is a query error
func TestDeleteObjects_QueryError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	deletions := []models.Deletion{
		{ID: "1", Object: "account", User: 1, Stamp: 1234567890},
	}

	mock.ExpectBegin()

	mock.ExpectExec(`DELETE FROM account WHERE id = \$1 AND "user" = \$2`).
		WithArgs("1", 1).
		WillReturnError(errors.New("delete error"))

	mock.ExpectRollback()

	err = db.DeleteObjects(context.Background(), deletions)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to delete account with ID 1")

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDeleteObjects_InvalidCountryID(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}
	deletions := []models.Deletion{
		{ID: "not-a-number", Object: string(models.EntityTypeCountry), User: 1, Stamp: 1234567890},
	}

	mock.ExpectBegin()
	mock.ExpectRollback()

	err = db.DeleteObjects(context.Background(), deletions)
	assert.ErrorContains(t, err, `invalid country ID "not-a-number"`)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestDeleteObjects_UnsupportedObject tests the case when an unsupported object type is provided
func TestDeleteObjects_UnsupportedObject(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	deletions := []models.Deletion{
		{ID: "1", Object: "unsupported", User: 1, Stamp: 1234567890},
	}

	mock.ExpectBegin()

	err = db.DeleteObjects(context.Background(), deletions)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported object type for deletion")

	assert.NoError(t, mock.ExpectationsWereMet())
}
