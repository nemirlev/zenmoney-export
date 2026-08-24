package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/nemirlev/zenmoney-export/v2/internal/interfaces"
	"github.com/nemirlev/zenmoney-go-sdk/v3/models"
)

const budgetSelectQuery = `
        SELECT "user", changed, to_char(date, 'YYYY-MM-DD') AS date,
               tag, income, outcome,
               income_lock, outcome_lock, is_income_forecast, is_outcome_forecast
        FROM budget`

const budgetInsertQuery = `
        INSERT INTO budget (
            "user", changed, date, tag, income, outcome,
            income_lock, outcome_lock, is_income_forecast, is_outcome_forecast
        ) VALUES ($1, $2, $3::text::date, $4, $5::text::numeric,
                  $6::text::numeric, $7, $8, $9, $10)`

func scanBudget(row rowScanner) (models.Budget, error) {
	var budget models.Budget
	err := row.Scan(
		&budget.User,
		&budget.Changed,
		&budget.Date,
		&budget.Tag,
		&budget.Income,
		&budget.Outcome,
		&budget.IncomeLock,
		&budget.OutcomeLock,
		&budget.IsIncomeForecast,
		&budget.IsOutcomeForecast,
	)
	return budget, err
}

func budgetValues(budget *models.Budget) []any {
	return []any{
		budget.User,
		budget.Changed,
		budget.Date,
		budget.Tag,
		decimalString(budget.Income),
		decimalString(budget.Outcome),
		budget.IncomeLock,
		budget.OutcomeLock,
		budget.IsIncomeForecast,
		budget.IsOutcomeForecast,
	}
}

// GetBudget retrieves a specific budget by user ID, tag ID and date
func (s *DB) GetBudget(
	ctx context.Context,
	userID int,
	tagID string,
	date time.Time,
) (*models.Budget, error) {
	budget, err := scanBudget(
		s.pool.QueryRow(
			ctx,
			budgetSelectQuery+` WHERE "user" = $1 AND tag = $2 AND date = $3`,
			userID,
			tagID,
			date,
		),
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("budget not found for user %d, tag %s, date %s",
				userID, tagID, date.Format("2006-01-02"))
		}
		return nil, fmt.Errorf("failed to get budget: %w", err)
	}

	return &budget, nil
}

// ListBudgets retrieves a list of budgets based on the provided filter
func (s *DB) ListBudgets(ctx context.Context, filter interfaces.Filter) ([]models.Budget, error) {
	query, args := buildListQuery(budgetSelectQuery, filter, true, "")

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list budgets: %w", err)
	}
	defer rows.Close()

	return collectRows(rows, scanBudget, "failed to scan budget", "error iterating budgets")
}

// CreateBudget creates a new budget record
func (s *DB) CreateBudget(ctx context.Context, budget *models.Budget) error {
	_, err := execCommand(
		ctx,
		s.pool,
		budgetInsertQuery,
		"failed to create budget",
		budgetValues(budget)...,
	)
	return err
}

// UpdateBudget updates an existing budget record
func (s *DB) UpdateBudget(ctx context.Context, budget *models.Budget) error {
	query := `
        UPDATE budget 
        SET changed = $4,
            income = $5::text::numeric,
            outcome = $6::text::numeric,
            income_lock = $7,
            outcome_lock = $8,
            is_income_forecast = $9,
            is_outcome_forecast = $10
        WHERE "user" = $1 AND tag = $2 AND date = $3::text::date`

	commandTag, err := execCommand(ctx, s.pool, query, "failed to update budget",
		budget.User,
		budget.Tag,
		budget.Date,
		budget.Changed,
		decimalString(budget.Income),
		decimalString(budget.Outcome),
		budget.IncomeLock,
		budget.OutcomeLock,
		budget.IsIncomeForecast,
		budget.IsOutcomeForecast,
	)
	if err != nil {
		return err
	}

	if commandTag.RowsAffected() == 0 {
		return fmt.Errorf("budget not found for user %d, tag %s, date %s",
			budget.User, *budget.Tag, budget.Date)
	}

	return nil
}

// DeleteBudget deletes a budget by user ID, tag ID and date
func (s *DB) DeleteBudget(ctx context.Context, userID int, tagID string, date time.Time) error {
	query := `DELETE FROM budget WHERE "user" = $1 AND tag = $2 AND date = $3`

	commandTag, err := execCommand(
		ctx,
		s.pool,
		query,
		"failed to delete budget",
		userID,
		tagID,
		date,
	)
	if err != nil {
		return err
	}

	if commandTag.RowsAffected() == 0 {
		return fmt.Errorf("budget not found for user %d, tag %s, date %s",
			userID, tagID, date.Format("2006-01-02"))
	}

	return nil
}
