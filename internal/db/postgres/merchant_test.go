package postgres

import (
	"context"
	"errors"
	"github.com/nemirlev/zenmoney-export/v2/internal/interfaces"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/nemirlev/zenmoney-go-sdk/v2/models"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
)

func TestGetMerchant_Success(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	merchantID := "test-id"
	expectedMerchant := &models.Merchant{
		ID:      merchantID,
		User:    1,
		Title:   "Test Merchant",
		Changed: 1234567890,
	}

	rows := mock.NewRows([]string{"id", "user", "title", "changed"}).
		AddRow(expectedMerchant.ID, expectedMerchant.User, expectedMerchant.Title, expectedMerchant.Changed)

	mock.ExpectQuery(`SELECT id, "user", title, changed FROM merchant WHERE id = \$1`).
		WithArgs(merchantID).
		WillReturnRows(rows)

	result, err := db.GetMerchant(context.Background(), merchantID)
	assert.NoError(t, err)
	assert.Equal(t, expectedMerchant, result)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetMerchant_NotFound(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	merchantID := "non-existing-id"

	mock.ExpectQuery(`SELECT id, "user", title, changed FROM merchant WHERE id = \$1`).
		WithArgs(merchantID).
		WillReturnError(pgx.ErrNoRows)

	result, err := db.GetMerchant(context.Background(), merchantID)
	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "merchant not found")

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetMerchant_QueryError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	merchantID := "test-id"

	mock.ExpectQuery(`SELECT id, "user", title, changed FROM merchant WHERE id = \$1`).
		WithArgs(merchantID).
		WillReturnError(errors.New("query error"))

	result, err := db.GetMerchant(context.Background(), merchantID)
	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get merchant")

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestListMerchants_Success(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	filter := interfaces.Filter{
		UserID: ptr(1),
		Limit:  10,
		Page:   1,
	}

	rows := mock.NewRows([]string{"id", "user", "title", "changed"}).
		AddRow("test-id", 1, "Test Merchant", 1234567890)

	mock.ExpectQuery(`SELECT id, "user", title, changed FROM merchant WHERE "user" = \$1 LIMIT \$2 OFFSET \$3`).
		WithArgs(1, 10, 0).
		WillReturnRows(rows)

	merchants, err := db.ListMerchants(context.Background(), filter)
	assert.NoError(t, err)
	assert.Len(t, merchants, 1)
	assert.Equal(t, "test-id", merchants[0].ID)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestListMerchants_QueryError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	filter := interfaces.Filter{
		UserID: ptr(1),
		Limit:  10,
		Page:   1,
	}

	mock.ExpectQuery(`SELECT id, "user", title, changed FROM merchant WHERE "user" = \$1 LIMIT \$2 OFFSET \$3`).
		WithArgs(1, 10, 0).
		WillReturnError(errors.New("query error"))

	merchants, err := db.ListMerchants(context.Background(), filter)
	assert.Error(t, err)
	assert.Nil(t, merchants)
	assert.Contains(t, err.Error(), "failed to list merchants")

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestListMerchants_NoResults(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	filter := interfaces.Filter{
		UserID: ptr(1),
		Limit:  10,
		Page:   1,
	}

	mock.ExpectQuery(`SELECT id, "user", title, changed FROM merchant WHERE "user" = \$1 LIMIT \$2 OFFSET \$3`).
		WithArgs(1, 10, 0).
		WillReturnRows(pgxmock.NewRows([]string{"id", "user", "title", "changed"}))

	merchants, err := db.ListMerchants(context.Background(), filter)
	assert.NoError(t, err)
	assert.Empty(t, merchants)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateMerchant_Success(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	merchant := &models.Merchant{
		ID:      "test-id",
		User:    1,
		Title:   "Test Merchant",
		Changed: 1234567890,
	}

	mock.ExpectExec(`INSERT INTO merchant \(id, "user", title, changed\) VALUES \(\$1, \$2, \$3, \$4\)`).
		WithArgs(merchant.ID, merchant.User, merchant.Title, merchant.Changed).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	err = db.CreateMerchant(context.Background(), merchant)
	assert.NoError(t, err)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateMerchant_QueryError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	merchant := &models.Merchant{
		ID:      "test-id",
		User:    1,
		Title:   "Test Merchant",
		Changed: 1234567890,
	}

	mock.ExpectExec(`INSERT INTO merchant \(id, "user", title, changed\) VALUES \(\$1, \$2, \$3, \$4\)`).
		WithArgs(merchant.ID, merchant.User, merchant.Title, merchant.Changed).
		WillReturnError(errors.New("query error"))

	err = db.CreateMerchant(context.Background(), merchant)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create merchant")

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateMerchant_Success(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	merchant := &models.Merchant{
		ID:      "test-id",
		User:    1,
		Title:   "Test Merchant",
		Changed: 1234567890,
	}

	mock.ExpectExec(`UPDATE merchant SET "user" = \$2, title = \$3, changed = \$4 WHERE id = \$1`).
		WithArgs(merchant.ID, merchant.User, merchant.Title, merchant.Changed).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	err = db.UpdateMerchant(context.Background(), merchant)
	assert.NoError(t, err)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateMerchant_QueryError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	merchant := &models.Merchant{
		ID:      "test-id",
		User:    1,
		Title:   "Test Merchant",
		Changed: 1234567890,
	}

	mock.ExpectExec(`UPDATE merchant SET "user" = \$2, title = \$3, changed = \$4 WHERE id = \$1`).
		WithArgs(merchant.ID, merchant.User, merchant.Title, merchant.Changed).
		WillReturnError(errors.New("query error"))

	err = db.UpdateMerchant(context.Background(), merchant)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to update merchant")

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDeleteMerchant_Success(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	merchantID := "test-id"

	mock.ExpectExec(`DELETE FROM merchant WHERE id = \$1`).
		WithArgs(merchantID).
		WillReturnResult(pgxmock.NewResult("DELETE", 1))

	err = db.DeleteMerchant(context.Background(), merchantID)
	assert.NoError(t, err)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDeleteMerchant_NotFound(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	merchantID := "non-existing-id"

	mock.ExpectExec(`DELETE FROM merchant WHERE id = \$1`).
		WithArgs(merchantID).
		WillReturnResult(pgxmock.NewResult("DELETE", 0))

	err = db.DeleteMerchant(context.Background(), merchantID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "merchant not found")

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDeleteMerchant_QueryError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	merchantID := "test-id"

	mock.ExpectExec(`DELETE FROM merchant WHERE id = \$1`).
		WithArgs(merchantID).
		WillReturnError(errors.New("query error"))

	err = db.DeleteMerchant(context.Background(), merchantID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to delete merchant")

	assert.NoError(t, mock.ExpectationsWereMet())
}
