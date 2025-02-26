package postgres

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/nemirlev/zenmoney-export/v2/internal/interfaces"
	"testing"
	"time"

	"github.com/nemirlev/zenmoney-go-sdk/v2/models"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
)

func TestGetBudget_Success(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create mock pool: %v", err)
	}
	defer mock.Close()

	db := &DB{pool: mock}

	userID := 1
	tagID := "test-tag"
	date := time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC)
	expectedBudget := &models.Budget{
		User:              userID,
		Changed:           1234567890,
		Date:              date.Format("2006-01-02"),
		Tag:               &tagID,
		Income:            1000.0,
		Outcome:           500.0,
		IncomeLock:        true,
		OutcomeLock:       false,
		IsIncomeForecast:  true,
		IsOutcomeForecast: false,
	}

	rows := mock.NewRows([]string{
		"user", "changed", "date", "tag", "income", "outcome",
		"income_lock", "outcome_lock", "is_income_forecast", "is_outcome_forecast",
	}).AddRow(
		expectedBudget.User,
		expectedBudget.Changed,
		expectedBudget.Date,
		expectedBudget.Tag,
		expectedBudget.Income,
		expectedBudget.Outcome,
		expectedBudget.IncomeLock,
		expectedBudget.OutcomeLock,
		expectedBudget.IsIncomeForecast,
		expectedBudget.IsOutcomeForecast,
	)

	mock.ExpectQuery(`SELECT "user", changed, date, tag, income, outcome, income_lock, outcome_lock, is_income_forecast, is_outcome_forecast FROM budget`).
		WithArgs(userID, tagID, date.Format("2006-01-02")).
		WillReturnRows(rows)

	result, err := db.GetBudget(context.Background(), userID, tagID, date)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, expectedBudget, result)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetBudget_NotFound(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create mock pool: %v", err)
	}
	defer mock.Close()

	db := &DB{pool: mock}

	userID := 1
	tagID := "test-tag"
	date := time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC)

	mock.ExpectQuery(`SELECT "user", changed, date, tag, income, outcome, income_lock, outcome_lock, is_income_forecast, is_outcome_forecast FROM budget`).
		WithArgs(userID, tagID, date.Format("2006-01-02")).
		WillReturnError(pgx.ErrNoRows)

	result, err := db.GetBudget(context.Background(), userID, tagID, date)
	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "budget not found")

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetBudget_QueryError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create mock pool: %v", err)
	}
	defer mock.Close()

	db := &DB{pool: mock}

	userID := 1
	tagID := "test-tag"
	date := time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC)

	mock.ExpectQuery(`SELECT "user", changed, date, tag, income, outcome, income_lock, outcome_lock, is_income_forecast, is_outcome_forecast FROM budget`).
		WithArgs(userID, tagID, date.Format("2006-01-02")).
		WillReturnError(errors.New("database error"))

	result, err := db.GetBudget(context.Background(), userID, tagID, date)
	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get budget")

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestListBudgets_Success(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create mock pool: %v", err)
	}
	defer mock.Close()

	db := &DB{pool: mock}

	filter := interfaces.Filter{
		UserID:    ptr(1),
		StartDate: ptr(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)),
		EndDate:   ptr(time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC)),
		Limit:     10,
		Page:      1,
	}

	tagID := "test-tag"
	rows := mock.NewRows([]string{
		"user", "changed", "date", "tag", "income", "outcome",
		"income_lock", "outcome_lock", "is_income_forecast", "is_outcome_forecast",
	}).AddRow(
		1, 1234567890, "2025-01-15", &tagID, 1000.0, 500.0, true, false, true, false,
	)

	mock.ExpectQuery(`SELECT "user", changed, date, tag, income, outcome, income_lock, outcome_lock, is_income_forecast, is_outcome_forecast FROM budget WHERE "user" = \$1 AND date >= \$2 AND date <= \$3 LIMIT \$4 OFFSET \$5`).
		WithArgs(1, "2025-01-01", "2025-02-01", 10, 0).
		WillReturnRows(rows)

	budgets, err := db.ListBudgets(context.Background(), filter)
	assert.NoError(t, err)
	assert.Len(t, budgets, 1)
	assert.Equal(t, 1, budgets[0].User)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestListBudgets_QueryError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create mock pool: %v", err)
	}
	defer mock.Close()

	db := &DB{pool: mock}

	filter := interfaces.Filter{
		UserID:    ptr(1),
		StartDate: ptr(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)),
		EndDate:   ptr(time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC)),
		Limit:     10,
		Page:      1,
	}

	mock.ExpectQuery(`SELECT "user", changed, date, tag, income, outcome, income_lock, outcome_lock, is_income_forecast, is_outcome_forecast FROM budget WHERE "user" = \$1 AND date >= \$2 AND date <= \$3 LIMIT \$4 OFFSET \$5`).
		WithArgs(1, "2025-01-01", "2025-02-01", 10, 0).
		WillReturnError(errors.New("database error"))

	budgets, err := db.ListBudgets(context.Background(), filter)
	assert.Error(t, err)
	assert.Nil(t, budgets)
	assert.Contains(t, err.Error(), "failed to list budgets")

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateBudget_Success(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	budget := &models.Budget{
		User:              1,
		Changed:           1234567890,
		Date:              "2025-02-01",
		Tag:               ptr("test-tag"),
		Income:            1000.0,
		Outcome:           500.0,
		IncomeLock:        true,
		OutcomeLock:       false,
		IsIncomeForecast:  true,
		IsOutcomeForecast: false,
	}

	mock.ExpectExec(`(?i)INSERT INTO budget \(\s*"user",\s*changed,\s*date,\s*tag,\s*income,\s*outcome,\s*income_lock,\s*outcome_lock,\s*is_income_forecast,\s*is_outcome_forecast\s*\) VALUES \(\s*\$1,\s*\$2,\s*\$3,\s*\$4,\s*\$5,\s*\$6,\s*\$7,\s*\$8,\s*\$9,\s*\$10\s*\)`).
		WithArgs(
			budget.User, budget.Changed, budget.Date, budget.Tag, budget.Income,
			budget.Outcome, budget.IncomeLock, budget.OutcomeLock,
			budget.IsIncomeForecast, budget.IsOutcomeForecast,
		).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	err = db.CreateBudget(context.Background(), budget)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateBudget_QueryError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	budget := &models.Budget{
		User:              1,
		Changed:           1234567890,
		Date:              "2025-02-01",
		Tag:               ptr("test-tag"),
		Income:            1000.0,
		Outcome:           500.0,
		IncomeLock:        true,
		OutcomeLock:       false,
		IsIncomeForecast:  true,
		IsOutcomeForecast: false,
	}

	mock.ExpectExec(`(?i)INSERT INTO budget \(\s*"user",\s*changed,\s*date,\s*tag,\s*income,\s*outcome,\s*income_lock,\s*outcome_lock,\s*is_income_forecast,\s*is_outcome_forecast\s*\) VALUES \(\s*\$1,\s*\$2,\s*\$3,\s*\$4,\s*\$5,\s*\$6,\s*\$7,\s*\$8,\s*\$9,\s*\$10\s*\)`).
		WithArgs(
			budget.User, budget.Changed, budget.Date, budget.Tag, budget.Income,
			budget.Outcome, budget.IncomeLock, budget.OutcomeLock,
			budget.IsIncomeForecast, budget.IsOutcomeForecast,
		).
		WillReturnError(errors.New("insert error"))

	err = db.CreateBudget(context.Background(), budget)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create budget")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDeleteBudget_Success(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	userID := 1
	tagID := "test-tag"
	date := "2025-02-01"

	mock.ExpectExec(`DELETE FROM budget WHERE "user" = \$1 AND tag = \$2 AND date = \$3`).
		WithArgs(userID, tagID, date).
		WillReturnResult(pgxmock.NewResult("DELETE", 1))

	err = db.DeleteBudget(context.Background(), userID, tagID, time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC))
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDeleteBudget_NotFound(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	userID := 1
	tagID := "test-tag"
	date := "2025-02-01"

	mock.ExpectExec(`DELETE FROM budget WHERE "user" = \$1 AND tag = \$2 AND date = \$3`).
		WithArgs(userID, tagID, date).
		WillReturnResult(pgxmock.NewResult("DELETE", 0))

	err = db.DeleteBudget(context.Background(), userID, tagID, time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "budget not found")
	assert.NoError(t, mock.ExpectationsWereMet())
}
