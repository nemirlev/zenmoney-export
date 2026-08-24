package postgres

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/nemirlev/zenmoney-go-sdk/v3/models"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
)

func TestGetTransaction_Success(t *testing.T) {
	db, mock := newTestDB(t)

	expectedTransaction := testTransaction()

	rows := mock.NewRows([]string{
		"id", "user", "date", "income", "outcome", "changed", "income_instrument",
		"outcome_instrument", "created", "original_payee", "deleted", "viewed",
		"hold", "qr_code", "source", "income_account", "outcome_account", "tag",
		"comment", "payee", "op_income", "op_outcome", "op_income_instrument",
		"op_outcome_instrument", "latitude", "longitude", "merchant",
		"income_bank_id", "outcome_bank_id", "reminder_marker",
	}).AddRow(
		expectedTransaction.ID, expectedTransaction.User, expectedTransaction.Date, expectedTransaction.Income, expectedTransaction.Outcome,
		expectedTransaction.Changed, expectedTransaction.IncomeInstrument, expectedTransaction.OutcomeInstrument, expectedTransaction.Created,
		expectedTransaction.OriginalPayee, expectedTransaction.Deleted, expectedTransaction.Viewed, expectedTransaction.Hold, expectedTransaction.QRCode,
		expectedTransaction.Source, expectedTransaction.IncomeAccount, expectedTransaction.OutcomeAccount, expectedTransaction.Tag, expectedTransaction.Comment,
		expectedTransaction.Payee, expectedTransaction.OpIncome, expectedTransaction.OpOutcome, expectedTransaction.OpIncomeInstrument,
		expectedTransaction.OpOutcomeInstrument, expectedTransaction.Latitude, expectedTransaction.Longitude, expectedTransaction.Merchant,
		expectedTransaction.IncomeBankID, expectedTransaction.OutcomeBankID, expectedTransaction.ReminderMarker,
	)

	mock.ExpectQuery(`SELECT id, "user", to_char[(]date, 'YYYY-MM-DD'[)] AS date, income, outcome, changed, income_instrument, outcome_instrument, created, original_payee, deleted, viewed, hold, qr_code, source, income_account, outcome_account, tag, comment, payee, op_income, op_outcome, op_income_instrument, op_outcome_instrument, latitude, longitude, merchant, income_bank_id, outcome_bank_id, reminder_marker FROM transaction WHERE id = \$1`).
		WithArgs("test-id").
		WillReturnRows(rows)

	result, err := db.GetTransaction(context.Background(), "test-id")
	assert.NoError(t, err)
	assert.Equal(t, expectedTransaction, result)
}

func TestGetTransaction_NotFound(t *testing.T) {
	db, mock := newTestDB(t)

	mock.ExpectQuery(`SELECT id, "user", to_char[(]date, 'YYYY-MM-DD'[)] AS date, income, outcome, changed, income_instrument, outcome_instrument, created, original_payee, deleted, viewed, hold, qr_code, source, income_account, outcome_account, tag, comment, payee, op_income, op_outcome, op_income_instrument, op_outcome_instrument, latitude, longitude, merchant, income_bank_id, outcome_bank_id, reminder_marker FROM transaction WHERE id = \$1`).
		WithArgs("non-existing-id").
		WillReturnError(pgx.ErrNoRows)

	result, err := db.GetTransaction(context.Background(), "non-existing-id")
	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "transaction not found")
}

func TestGetTransaction_QueryError(t *testing.T) {
	db, mock := newTestDB(t)

	mock.ExpectQuery(`SELECT id, "user", to_char[(]date, 'YYYY-MM-DD'[)] AS date, income, outcome, changed, income_instrument, outcome_instrument, created, original_payee, deleted, viewed, hold, qr_code, source, income_account, outcome_account, tag, comment, payee, op_income, op_outcome, op_income_instrument, op_outcome_instrument, latitude, longitude, merchant, income_bank_id, outcome_bank_id, reminder_marker FROM transaction WHERE id = \$1`).
		WithArgs("test-id").
		WillReturnError(errors.New("query error"))

	result, err := db.GetTransaction(context.Background(), "test-id")
	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get transaction")
}

func TestListTransactions_Success(t *testing.T) {
	db, mock := newTestDB(t)

	filter := testPageFilter()

	rows := mock.NewRows([]string{
		"id", "user", "date", "income", "outcome", "changed", "income_instrument",
		"outcome_instrument", "created", "original_payee", "deleted", "viewed",
		"hold", "qr_code", "source", "income_account", "outcome_account", "tag",
		"comment", "payee", "op_income", "op_outcome", "op_income_instrument",
		"op_outcome_instrument", "latitude", "longitude", "merchant",
		"income_bank_id", "outcome_bank_id", "reminder_marker",
	}).AddRow(
		"test-id", 1, "2023-01-01", 1000.0, 500.0, 1234567890, 1, 2, 1234567890,
		"Original Payee", false, true, false, new("QRCode"), "Source", "IncomeAccount",
		new("OutcomeAccount"), []string{"tag1", "tag2"}, new("Comment"), "Payee", 100.0, 50.0,
		new(
			3,
		), new(4), new(55.7558), new(37.6176), new("Merchant"), new("IncomeBankID"), new("OutcomeBankID"), new("ReminderMarker"),
	)

	mock.ExpectQuery(`SELECT id, "user", to_char[(]date, 'YYYY-MM-DD'[)] AS date, income, outcome, changed, income_instrument, outcome_instrument, created, original_payee, deleted, viewed, hold, qr_code, source, income_account, outcome_account, tag, comment, payee, op_income, op_outcome, op_income_instrument, op_outcome_instrument, latitude, longitude, merchant, income_bank_id, outcome_bank_id, reminder_marker FROM transaction WHERE "user" = \$1 ORDER BY date DESC, created DESC LIMIT \$2 OFFSET \$3`).
		WithArgs(1, 10, 0).
		WillReturnRows(rows)

	transactions, err := db.ListTransactions(context.Background(), filter)
	assert.NoError(t, err)
	assert.Len(t, transactions, 1)
	assert.Equal(t, "test-id", transactions[0].ID)
}

func TestListTransactions_QueryError(t *testing.T) {
	db, mock := newTestDB(t)

	filter := testPageFilter()

	mock.ExpectQuery(`SELECT id, "user", to_char[(]date, 'YYYY-MM-DD'[)] AS date, income, outcome, changed, income_instrument, outcome_instrument, created, original_payee, deleted, viewed, hold, qr_code, source, income_account, outcome_account, tag, comment, payee, op_income, op_outcome, op_income_instrument, op_outcome_instrument, latitude, longitude, merchant, income_bank_id, outcome_bank_id, reminder_marker FROM transaction WHERE "user" = \$1 ORDER BY date DESC, created DESC LIMIT \$2 OFFSET \$3`).
		WithArgs(1, 10, 0).
		WillReturnError(errors.New("query error"))

	transactions, err := db.ListTransactions(context.Background(), filter)
	assert.Error(t, err)
	assert.Nil(t, transactions)
	assert.Contains(t, err.Error(), "failed to list transactions")
}

func TestCreateTransaction_Success(t *testing.T) {
	db, mock := newTestDB(t)

	transaction := testTransaction()

	mock.ExpectExec(`INSERT INTO transaction`).
		WithArgs(transactionArgs(transaction)...).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	err := db.CreateTransaction(context.Background(), transaction)
	assert.NoError(t, err)
}

func TestCreateTransaction_QueryError(t *testing.T) {
	db, mock := newTestDB(t)

	transaction := testTransaction()

	mock.ExpectExec(`INSERT INTO transaction`).
		WithArgs(transactionArgs(transaction)...).
		WillReturnError(errors.New("insert error"))

	err := db.CreateTransaction(context.Background(), transaction)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create transaction")
}

func TestUpdateTransaction_Success(t *testing.T) {
	db, mock := newTestDB(t)

	transaction := testTransaction()

	mock.ExpectExec(`UPDATE transaction SET`).
		WithArgs(transactionArgs(transaction)...).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	err := db.UpdateTransaction(context.Background(), transaction)
	assert.NoError(t, err)
}

func TestUpdateTransaction_QueryError(t *testing.T) {
	db, mock := newTestDB(t)

	transaction := &models.Transaction{
		ID:            "test-id",
		OriginalPayee: "Updated Transaction",
	}

	mock.ExpectExec(`UPDATE transaction SET`).
		WithArgs(transactionArgs(transaction)...).
		WillReturnError(errors.New("update error"))

	err := db.UpdateTransaction(context.Background(), transaction)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to update transaction")
}

func TestDeleteTransaction_Success(t *testing.T) {
	db, mock := newTestDB(t)

	transactionID := "test-id"

	mock.ExpectExec(`DELETE FROM transaction WHERE id = \$1`).
		WithArgs(transactionID).
		WillReturnResult(pgxmock.NewResult("DELETE", 1))

	err := db.DeleteTransaction(context.Background(), transactionID)
	assert.NoError(t, err)
}

func TestDeleteTransaction_NotFound(t *testing.T) {
	db, mock := newTestDB(t)

	transactionID := "non-existing-id"

	mock.ExpectExec(`DELETE FROM transaction WHERE id = \$1`).
		WithArgs(transactionID).
		WillReturnResult(pgxmock.NewResult("DELETE", 0))

	err := db.DeleteTransaction(context.Background(), transactionID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "transaction not found")
}

func TestDeleteTransaction_QueryError(t *testing.T) {
	db, mock := newTestDB(t)

	transactionID := "test-id"

	mock.ExpectExec(`DELETE FROM transaction WHERE id = \$1`).
		WithArgs(transactionID).
		WillReturnError(errors.New("delete error"))

	err := db.DeleteTransaction(context.Background(), transactionID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to delete transaction")
}
