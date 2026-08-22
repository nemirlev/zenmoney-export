package postgres

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/nemirlev/zenmoney-export/v2/internal/interfaces"
	"github.com/nemirlev/zenmoney-go-sdk/v3/models"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
)

func TestGetTransaction_Success(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	expectedTransaction := &models.Transaction{
		ID:                  "test-id",
		User:                1,
		Date:                "2023-01-01",
		Income:              1000.0,
		Outcome:             500.0,
		Changed:             1234567890,
		IncomeInstrument:    1,
		OutcomeInstrument:   2,
		Created:             1234567890,
		OriginalPayee:       "Original Payee",
		Deleted:             false,
		Viewed:              true,
		Hold:                false,
		QRCode:              new("QRCode"),
		Source:              "Source",
		IncomeAccount:       "IncomeAccount",
		OutcomeAccount:      new("OutcomeAccount"),
		Tag:                 []string{"tag1", "tag2"},
		Comment:             new("Comment"),
		Payee:               "Payee",
		OpIncome:            100.0,
		OpOutcome:           50.0,
		OpIncomeInstrument:  new(3),
		OpOutcomeInstrument: new(4),
		Latitude:            new(55.7558),
		Longitude:           new(37.6176),
		Merchant:            new("Merchant"),
		IncomeBankID:        new("IncomeBankID"),
		OutcomeBankID:       new("OutcomeBankID"),
		ReminderMarker:      new("ReminderMarker"),
	}

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

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetTransaction_NotFound(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	mock.ExpectQuery(`SELECT id, "user", to_char[(]date, 'YYYY-MM-DD'[)] AS date, income, outcome, changed, income_instrument, outcome_instrument, created, original_payee, deleted, viewed, hold, qr_code, source, income_account, outcome_account, tag, comment, payee, op_income, op_outcome, op_income_instrument, op_outcome_instrument, latitude, longitude, merchant, income_bank_id, outcome_bank_id, reminder_marker FROM transaction WHERE id = \$1`).
		WithArgs("non-existing-id").
		WillReturnError(pgx.ErrNoRows)

	result, err := db.GetTransaction(context.Background(), "non-existing-id")
	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "transaction not found")

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetTransaction_QueryError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	mock.ExpectQuery(`SELECT id, "user", to_char[(]date, 'YYYY-MM-DD'[)] AS date, income, outcome, changed, income_instrument, outcome_instrument, created, original_payee, deleted, viewed, hold, qr_code, source, income_account, outcome_account, tag, comment, payee, op_income, op_outcome, op_income_instrument, op_outcome_instrument, latitude, longitude, merchant, income_bank_id, outcome_bank_id, reminder_marker FROM transaction WHERE id = \$1`).
		WithArgs("test-id").
		WillReturnError(errors.New("query error"))

	result, err := db.GetTransaction(context.Background(), "test-id")
	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get transaction")

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestListTransactions_Success(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	filter := interfaces.Filter{
		UserID: new(1),
		Limit:  10,
		Page:   1,
	}

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

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestListTransactions_QueryError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	filter := interfaces.Filter{
		UserID: new(1),
		Limit:  10,
		Page:   1,
	}

	mock.ExpectQuery(`SELECT id, "user", to_char[(]date, 'YYYY-MM-DD'[)] AS date, income, outcome, changed, income_instrument, outcome_instrument, created, original_payee, deleted, viewed, hold, qr_code, source, income_account, outcome_account, tag, comment, payee, op_income, op_outcome, op_income_instrument, op_outcome_instrument, latitude, longitude, merchant, income_bank_id, outcome_bank_id, reminder_marker FROM transaction WHERE "user" = \$1 ORDER BY date DESC, created DESC LIMIT \$2 OFFSET \$3`).
		WithArgs(1, 10, 0).
		WillReturnError(errors.New("query error"))

	transactions, err := db.ListTransactions(context.Background(), filter)
	assert.Error(t, err)
	assert.Nil(t, transactions)
	assert.Contains(t, err.Error(), "failed to list transactions")

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateTransaction_Success(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	transaction := &models.Transaction{
		ID:                  "test-id",
		User:                1,
		Date:                "2023-01-01",
		Income:              1000.0,
		Outcome:             500.0,
		Changed:             1234567890,
		IncomeInstrument:    1,
		OutcomeInstrument:   2,
		Created:             1234567890,
		OriginalPayee:       "Original Payee",
		Deleted:             false,
		Viewed:              true,
		Hold:                false,
		QRCode:              new("QRCode"),
		Source:              "Source",
		IncomeAccount:       "IncomeAccount",
		OutcomeAccount:      new("OutcomeAccount"),
		Tag:                 []string{"tag1", "tag2"},
		Comment:             new("Comment"),
		Payee:               "Payee",
		OpIncome:            100.0,
		OpOutcome:           50.0,
		OpIncomeInstrument:  new(3),
		OpOutcomeInstrument: new(4),
		Latitude:            new(55.7558),
		Longitude:           new(37.6176),
		Merchant:            new("Merchant"),
		IncomeBankID:        new("IncomeBankID"),
		OutcomeBankID:       new("OutcomeBankID"),
		ReminderMarker:      new("ReminderMarker"),
	}

	mock.ExpectExec(`INSERT INTO transaction`).
		WithArgs(
			transaction.ID, transaction.User, transaction.Date, decimalString(transaction.Income), decimalString(transaction.Outcome),
			transaction.Changed, transaction.IncomeInstrument, transaction.OutcomeInstrument, transaction.Created,
			transaction.OriginalPayee, transaction.Deleted, transaction.Viewed, transaction.Hold, transaction.QRCode,
			transaction.Source, transaction.IncomeAccount, transaction.OutcomeAccount, transaction.Tag, transaction.Comment,
			transaction.Payee, decimalString(transaction.OpIncome), decimalString(transaction.OpOutcome), transaction.OpIncomeInstrument,
			transaction.OpOutcomeInstrument, transaction.Latitude, transaction.Longitude, transaction.Merchant,
			transaction.IncomeBankID, transaction.OutcomeBankID, transaction.ReminderMarker,
		).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	err = db.CreateTransaction(context.Background(), transaction)
	assert.NoError(t, err)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateTransaction_QueryError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	transaction := &models.Transaction{
		ID:                  "test-id",
		User:                1,
		Date:                "2023-01-01",
		Income:              1000.0,
		Outcome:             500.0,
		Changed:             1234567890,
		IncomeInstrument:    1,
		OutcomeInstrument:   2,
		Created:             1234567890,
		OriginalPayee:       "Original Payee",
		Deleted:             false,
		Viewed:              true,
		Hold:                false,
		QRCode:              new("QRCode"),
		Source:              "Source",
		IncomeAccount:       "IncomeAccount",
		OutcomeAccount:      new("OutcomeAccount"),
		Tag:                 []string{"tag1", "tag2"},
		Comment:             new("Comment"),
		Payee:               "Payee",
		OpIncome:            100.0,
		OpOutcome:           50.0,
		OpIncomeInstrument:  new(3),
		OpOutcomeInstrument: new(4),
		Latitude:            new(55.7558),
		Longitude:           new(37.6176),
		Merchant:            new("Merchant"),
		IncomeBankID:        new("IncomeBankID"),
		OutcomeBankID:       new("OutcomeBankID"),
		ReminderMarker:      new("ReminderMarker"),
	}

	mock.ExpectExec(`INSERT INTO transaction`).
		WithArgs(
			transaction.ID, transaction.User, transaction.Date, decimalString(transaction.Income), decimalString(transaction.Outcome),
			transaction.Changed, transaction.IncomeInstrument, transaction.OutcomeInstrument, transaction.Created,
			transaction.OriginalPayee, transaction.Deleted, transaction.Viewed, transaction.Hold, transaction.QRCode,
			transaction.Source, transaction.IncomeAccount, transaction.OutcomeAccount, transaction.Tag, transaction.Comment,
			transaction.Payee, decimalString(transaction.OpIncome), decimalString(transaction.OpOutcome), transaction.OpIncomeInstrument,
			transaction.OpOutcomeInstrument, transaction.Latitude, transaction.Longitude, transaction.Merchant,
			transaction.IncomeBankID, transaction.OutcomeBankID, transaction.ReminderMarker,
		).
		WillReturnError(errors.New("insert error"))

	err = db.CreateTransaction(context.Background(), transaction)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create transaction")

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateTransaction_Success(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	transaction := &models.Transaction{
		ID:                  "test-id",
		User:                1,
		Date:                "2023-01-01",
		Income:              1000.0,
		Outcome:             500.0,
		Changed:             1234567890,
		IncomeInstrument:    1,
		OutcomeInstrument:   2,
		Created:             1234567890,
		OriginalPayee:       "Original Payee",
		Deleted:             false,
		Viewed:              true,
		Hold:                false,
		QRCode:              new("QRCode"),
		Source:              "Source",
		IncomeAccount:       "IncomeAccount",
		OutcomeAccount:      new("OutcomeAccount"),
		Tag:                 []string{"tag1", "tag2"},
		Comment:             new("Comment"),
		Payee:               "Payee",
		OpIncome:            100.0,
		OpOutcome:           50.0,
		OpIncomeInstrument:  new(3),
		OpOutcomeInstrument: new(4),
		Latitude:            new(55.7558),
		Longitude:           new(37.6176),
		Merchant:            new("Merchant"),
		IncomeBankID:        new("IncomeBankID"),
		OutcomeBankID:       new("OutcomeBankID"),
		ReminderMarker:      new("ReminderMarker"),
	}

	mock.ExpectExec(`UPDATE transaction SET`).
		WithArgs(
			transaction.ID, transaction.User, transaction.Date, decimalString(transaction.Income), decimalString(transaction.Outcome), transaction.Changed,
			transaction.IncomeInstrument, transaction.OutcomeInstrument, transaction.Created, transaction.OriginalPayee,
			transaction.Deleted, transaction.Viewed, transaction.Hold, transaction.QRCode, transaction.Source,
			transaction.IncomeAccount, transaction.OutcomeAccount, transaction.Tag, transaction.Comment, transaction.Payee,
			decimalString(transaction.OpIncome), decimalString(transaction.OpOutcome), transaction.OpIncomeInstrument, transaction.OpOutcomeInstrument,
			transaction.Latitude, transaction.Longitude, transaction.Merchant, transaction.IncomeBankID, transaction.OutcomeBankID,
			transaction.ReminderMarker,
		).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	err = db.UpdateTransaction(context.Background(), transaction)
	assert.NoError(t, err)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateTransaction_QueryError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	transaction := &models.Transaction{
		ID:            "test-id",
		OriginalPayee: "Updated Transaction",
	}

	mock.ExpectExec(`UPDATE transaction SET`).
		WithArgs(
			transaction.ID, transaction.User, transaction.Date, decimalString(transaction.Income), decimalString(transaction.Outcome), transaction.Changed,
			transaction.IncomeInstrument, transaction.OutcomeInstrument, transaction.Created, transaction.OriginalPayee,
			transaction.Deleted, transaction.Viewed, transaction.Hold, transaction.QRCode, transaction.Source,
			transaction.IncomeAccount, transaction.OutcomeAccount, transaction.Tag, transaction.Comment, transaction.Payee,
			decimalString(transaction.OpIncome), decimalString(transaction.OpOutcome), transaction.OpIncomeInstrument, transaction.OpOutcomeInstrument,
			transaction.Latitude, transaction.Longitude, transaction.Merchant, transaction.IncomeBankID, transaction.OutcomeBankID,
			transaction.ReminderMarker,
		).
		WillReturnError(errors.New("update error"))

	err = db.UpdateTransaction(context.Background(), transaction)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to update transaction")

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDeleteTransaction_Success(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	transactionID := "test-id"

	mock.ExpectExec(`DELETE FROM transaction WHERE id = \$1`).
		WithArgs(transactionID).
		WillReturnResult(pgxmock.NewResult("DELETE", 1))

	err = db.DeleteTransaction(context.Background(), transactionID)
	assert.NoError(t, err)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDeleteTransaction_NotFound(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	transactionID := "non-existing-id"

	mock.ExpectExec(`DELETE FROM transaction WHERE id = \$1`).
		WithArgs(transactionID).
		WillReturnResult(pgxmock.NewResult("DELETE", 0))

	err = db.DeleteTransaction(context.Background(), transactionID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "transaction not found")

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDeleteTransaction_QueryError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	transactionID := "test-id"

	mock.ExpectExec(`DELETE FROM transaction WHERE id = \$1`).
		WithArgs(transactionID).
		WillReturnError(errors.New("delete error"))

	err = db.DeleteTransaction(context.Background(), transactionID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to delete transaction")

	assert.NoError(t, mock.ExpectationsWereMet())
}
