package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/nemirlev/zenmoney-export/v2/internal/interfaces"
	"github.com/nemirlev/zenmoney-go-sdk/v3/models"
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

	mock.ExpectQuery(`SELECT "user", changed, to_char[(]date, 'YYYY-MM-DD'[)] AS date, tag, income, outcome, income_lock, outcome_lock, is_income_forecast, is_outcome_forecast FROM budget`).
		WithArgs(userID, tagID, date).
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

	mock.ExpectQuery(`SELECT "user", changed, to_char[(]date, 'YYYY-MM-DD'[)] AS date, tag, income, outcome, income_lock, outcome_lock, is_income_forecast, is_outcome_forecast FROM budget`).
		WithArgs(userID, tagID, date).
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

	mock.ExpectQuery(`SELECT "user", changed, to_char[(]date, 'YYYY-MM-DD'[)] AS date, tag, income, outcome, income_lock, outcome_lock, is_income_forecast, is_outcome_forecast FROM budget`).
		WithArgs(userID, tagID, date).
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
		UserID:    new(1),
		StartDate: new(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)),
		EndDate:   new(time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC)),
		Limit:     10,
		Page:      1,
	}

	rows := mock.NewRows([]string{
		"user", "changed", "date", "tag", "income", "outcome",
		"income_lock", "outcome_lock", "is_income_forecast", "is_outcome_forecast",
	}).AddRow(
		1, 1234567890, "2025-01-15", new("test-tag"), 1000.0, 500.0, true, false, true, false,
	)

	mock.ExpectQuery(`SELECT "user", changed, to_char[(]date, 'YYYY-MM-DD'[)] AS date, tag, income, outcome, income_lock, outcome_lock, is_income_forecast, is_outcome_forecast FROM budget WHERE "user" = \$1 AND date >= \$2 AND date <= \$3 LIMIT \$4 OFFSET \$5`).
		WithArgs(1, *filter.StartDate, *filter.EndDate, 10, 0).
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
		UserID:    new(1),
		StartDate: new(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)),
		EndDate:   new(time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC)),
		Limit:     10,
		Page:      1,
	}

	mock.ExpectQuery(`SELECT "user", changed, to_char[(]date, 'YYYY-MM-DD'[)] AS date, tag, income, outcome, income_lock, outcome_lock, is_income_forecast, is_outcome_forecast FROM budget WHERE "user" = \$1 AND date >= \$2 AND date <= \$3 LIMIT \$4 OFFSET \$5`).
		WithArgs(1, *filter.StartDate, *filter.EndDate, 10, 0).
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
		Tag:               new("test-tag"),
		Income:            1000.0,
		Outcome:           500.0,
		IncomeLock:        true,
		OutcomeLock:       false,
		IsIncomeForecast:  true,
		IsOutcomeForecast: false,
	}

	mock.ExpectExec(`INSERT INTO budget`).
		WithArgs(
			budget.User, budget.Changed, budget.Date, budget.Tag, decimalString(budget.Income),
			decimalString(budget.Outcome), budget.IncomeLock, budget.OutcomeLock,
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
		Tag:               new("test-tag"),
		Income:            1000.0,
		Outcome:           500.0,
		IncomeLock:        true,
		OutcomeLock:       false,
		IsIncomeForecast:  true,
		IsOutcomeForecast: false,
	}

	mock.ExpectExec(`INSERT INTO budget`).
		WithArgs(
			budget.User, budget.Changed, budget.Date, budget.Tag, decimalString(budget.Income),
			decimalString(budget.Outcome), budget.IncomeLock, budget.OutcomeLock,
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

	mock.ExpectExec(`DELETE FROM budget WHERE "user" = \$1 AND tag = \$2 AND date = \$3`).
		WithArgs(userID, tagID, time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC)).
		WillReturnResult(pgxmock.NewResult("DELETE", 1))

	err = db.DeleteBudget(
		context.Background(),
		userID,
		tagID,
		time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC),
	)
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

	mock.ExpectExec(`DELETE FROM budget WHERE "user" = \$1 AND tag = \$2 AND date = \$3`).
		WithArgs(userID, tagID, time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC)).
		WillReturnResult(pgxmock.NewResult("DELETE", 0))

	err = db.DeleteBudget(
		context.Background(),
		userID,
		tagID,
		time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC),
	)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "budget not found")
	assert.NoError(t, mock.ExpectationsWereMet())
}
