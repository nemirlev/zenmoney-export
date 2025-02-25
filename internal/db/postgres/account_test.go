package postgres

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/nemirlev/zenmoney-export/v2/internal/interfaces"
	"testing"

	"github.com/nemirlev/zenmoney-go-sdk/v2/models"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
)

// Тест успешного получения аккаунта
func TestGetAccount_Success(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create mock pool: %v", err)
	}
	defer mock.Close()

	db := &DB{pool: mock}

	accountID := "test-id"
	expectedAccount := &models.Account{
		ID:      accountID,
		User:    1,
		Title:   "Test Account",
		Type:    "checking",
		Private: true,
	}

	rows := mock.NewRows([]string{
		"id", "user", "instrument", "type", "role", "private", "savings",
		"title", "in_balance", "credit_limit", "start_balance", "balance",
		"company", "archive", "enable_correction", "balance_correction_type",
		"start_date", "capitalization", "percent", "changed", "sync_id",
		"enable_sms", "end_date_offset", "end_date_offset_interval",
		"payoff_step", "payoff_interval",
	}).AddRow(
		expectedAccount.ID, expectedAccount.User, nil, expectedAccount.Type, nil, expectedAccount.Private, nil,
		expectedAccount.Title, true, nil, nil, nil,
		nil, false, false, "", nil, nil, nil, 1234567890, nil,
		false, nil, nil, nil, nil,
	)

	mock.ExpectQuery(`SELECT id, "user", instrument, type, role, private, savings,`).
		WithArgs(accountID).
		WillReturnRows(rows)

	result, err := db.GetAccount(context.Background(), accountID)
	assert.NoError(t, err)
	assert.Equal(t, expectedAccount.ID, result.ID)
	assert.Equal(t, expectedAccount.User, result.User)
	assert.Equal(t, expectedAccount.Title, result.Title)
	assert.Equal(t, expectedAccount.Type, result.Type)
	assert.Equal(t, expectedAccount.Private, result.Private)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations were not met: %v", err)
	}
}

// Тест случая, когда аккаунт не найден
func TestGetAccount_NotFound(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create mock pool: %v", err)
	}
	defer mock.Close()

	db := &DB{pool: mock}

	accountID := "non-existing-id"

	mock.ExpectQuery(`SELECT id, "user", instrument, type, role, private, savings,`).
		WithArgs(accountID).
		WillReturnError(pgx.ErrNoRows)

	result, err := db.GetAccount(context.Background(), accountID)
	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "account not found")

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations were not met: %v", err)
	}
}

func TestListAccounts_Success(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	filter := interfaces.Filter{
		UserID: ptr(1), // ptr — вспомогательная функция, объявленная ниже
		Limit:  10,
		Page:   1,
	}

	rows := mock.NewRows([]string{
		"id", "user", "instrument", "type", "role", "private", "savings",
		"title", "in_balance", "credit_limit", "start_balance", "balance",
		"company", "archive", "enable_correction", "balance_correction_type",
		"start_date", "capitalization", "percent", "changed", "sync_id",
		"enable_sms", "end_date_offset", "end_date_offset_interval",
		"payoff_step", "payoff_interval",
	}).AddRow(
		"test-id", 1, nil, "checking", nil, true, nil,
		"Test Account", true, nil, nil, nil,
		nil, false, false, "", nil, nil, nil, 1234567890, nil,
		false, nil, nil, nil, nil,
	)

	mock.ExpectQuery(`SELECT id, "user", instrument, type, role, private, savings,`).
		WithArgs(1, 10, 0).
		WillReturnRows(rows)

	accounts, err := db.ListAccounts(context.Background(), filter)
	assert.NoError(t, err)
	assert.Len(t, accounts, 1)
	assert.Equal(t, "test-id", accounts[0].ID)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestListAccounts_QueryError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	filter := interfaces.Filter{
		UserID: ptr(1),
		Limit:  10,
		Page:   1,
	}

	mock.ExpectQuery(`SELECT id, "user", instrument, type, role, private, savings,`).
		WithArgs(1, 10, 0).
		WillReturnError(errors.New("query error"))

	accounts, err := db.ListAccounts(context.Background(), filter)
	assert.Error(t, err)
	assert.Nil(t, accounts)
	assert.Contains(t, err.Error(), "failed to list accounts")

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateAccount_Success(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	account := &models.Account{
		ID:      "test-id",
		User:    1,
		Title:   "Test Account",
		Type:    "checking",
		Private: true,
	}

	mock.ExpectExec(`INSERT INTO account`).
		WithArgs(
			account.ID, account.User, account.Instrument, account.Type,
			account.Role, account.Private, account.Savings, account.Title,
			account.InBalance, account.CreditLimit, account.StartBalance,
			account.Balance, account.Company, account.Archive,
			account.EnableCorrection, account.BalanceCorrectionType,
			account.StartDate, account.Capitalization, account.Percent,
			account.Changed, account.SyncID, account.EnableSMS,
			account.EndDateOffset, account.EndDateOffsetInterval,
			account.PayoffStep, account.PayoffInterval,
		).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	err = db.CreateAccount(context.Background(), account)
	assert.NoError(t, err)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateAccount_QueryError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	account := &models.Account{
		ID:      "test-id",
		User:    1,
		Title:   "Test Account",
		Type:    "checking",
		Private: true,
	}

	mock.ExpectExec(`INSERT INTO account`).
		WithArgs(
			account.ID, account.User, account.Instrument, account.Type,
			account.Role, account.Private, account.Savings, account.Title,
			account.InBalance, account.CreditLimit, account.StartBalance,
			account.Balance, account.Company, account.Archive,
			account.EnableCorrection, account.BalanceCorrectionType,
			account.StartDate, account.Capitalization, account.Percent,
			account.Changed, account.SyncID, account.EnableSMS,
			account.EndDateOffset, account.EndDateOffsetInterval,
			account.PayoffStep, account.PayoffInterval,
		).
		WillReturnError(errors.New("insert error"))

	err = db.CreateAccount(context.Background(), account)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create account")

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateAccount_Success(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	account := &models.Account{
		ID:    "test-id",
		Title: "Updated Account",
	}

	mock.ExpectExec(`UPDATE account SET`).
		WithArgs(
			account.ID, account.User, account.Instrument, account.Type,
			account.Role, account.Private, account.Savings, account.Title,
			account.InBalance, account.CreditLimit, account.StartBalance,
			account.Balance, account.Company, account.Archive,
			account.EnableCorrection, account.BalanceCorrectionType,
			account.StartDate, account.Capitalization, account.Percent,
			account.Changed, account.SyncID, account.EnableSMS,
			account.EndDateOffset, account.EndDateOffsetInterval,
			account.PayoffStep, account.PayoffInterval,
		).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	err = db.UpdateAccount(context.Background(), account)
	assert.NoError(t, err)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateAccount_NotFound(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	account := &models.Account{
		ID:    "test-id",
		Title: "Updated Account",
	}

	mock.ExpectExec(`UPDATE account SET`).
		WithArgs(
			account.ID, account.User, account.Instrument, account.Type,
			account.Role, account.Private, account.Savings, account.Title,
			account.InBalance, account.CreditLimit, account.StartBalance,
			account.Balance, account.Company, account.Archive,
			account.EnableCorrection, account.BalanceCorrectionType,
			account.StartDate, account.Capitalization, account.Percent,
			account.Changed, account.SyncID, account.EnableSMS,
			account.EndDateOffset, account.EndDateOffsetInterval,
			account.PayoffStep, account.PayoffInterval,
		).
		WillReturnResult(pgxmock.NewResult("UPDATE", 0))

	err = db.UpdateAccount(context.Background(), account)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "account not found") // <-- теперь содержит "account not found"

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDeleteAccount_Success(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	accountID := "test-id"

	mock.ExpectExec(`DELETE FROM account WHERE id = \$1`).
		WithArgs(accountID).
		WillReturnResult(pgxmock.NewResult("DELETE", 1))

	err = db.DeleteAccount(context.Background(), accountID)
	assert.NoError(t, err)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDeleteAccount_NotFound(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	accountID := "test-id"

	mock.ExpectExec(`DELETE FROM account WHERE id = \$1`).
		WithArgs(accountID).
		WillReturnResult(pgxmock.NewResult("DELETE", 0))

	err = db.DeleteAccount(context.Background(), accountID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "account not found")

	assert.NoError(t, mock.ExpectationsWereMet())
}

func ptr[T any](v T) *T {
	return &v
}
