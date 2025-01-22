package postgres

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/nemirlev/zenmoney-export/internal/interfaces"
	"github.com/nemirlev/zenmoney-go-sdk/v2/models"
	"strings"
	"time"
)

// GetBudget retrieves a specific budget by user ID, tag ID and date
func (s *DB) GetBudget(ctx context.Context, userID int, tagID string, date time.Time) (*models.Budget, error) {
	query := `
        SELECT "user", changed, date, tag, income, outcome, 
               income_lock, outcome_lock, is_income_forecast, is_outcome_forecast
        FROM budget
        WHERE "user" = $1 AND tag = $2 AND date = $3`

	budget := &models.Budget{}
	err := s.pool.QueryRow(ctx, query, userID, tagID, date.Format("2006-01-02")).Scan(
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

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("budget not found for user %d, tag %s, date %s",
				userID, tagID, date.Format("2006-01-02"))
		}
		return nil, fmt.Errorf("failed to get budget: %w", err)
	}

	return budget, nil
}

// ListBudgets retrieves a list of budgets based on the provided filter
func (s *DB) ListBudgets(ctx context.Context, filter interfaces.Filter) ([]models.Budget, error) {
	var conditions []string
	var args []interface{}
	argNum := 1

	if filter.UserID != nil {
		conditions = append(conditions, fmt.Sprintf(`"user" = $%d`, argNum))
		args = append(args, *filter.UserID)
		argNum++
	}

	if filter.StartDate != nil {
		conditions = append(conditions, fmt.Sprintf(`date >= $%d`, argNum))
		args = append(args, filter.StartDate.Format("2006-01-02"))
		argNum++
	}

	if filter.EndDate != nil {
		conditions = append(conditions, fmt.Sprintf(`date <= $%d`, argNum))
		args = append(args, filter.EndDate.Format("2006-01-02"))
		argNum++
	}

	query := `
        SELECT "user", changed, date, tag, income, outcome,
               income_lock, outcome_lock, is_income_forecast, is_outcome_forecast
        FROM budget`

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argNum, argNum+1)
	args = append(args, filter.Limit, (filter.Page-1)*filter.Limit)

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list budgets: %w", err)
	}
	defer rows.Close()

	var budgets []models.Budget
	for rows.Next() {
		var budget models.Budget
		err := rows.Scan(
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
		if err != nil {
			return nil, fmt.Errorf("failed to scan budget: %w", err)
		}
		budgets = append(budgets, budget)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating budgets: %w", err)
	}

	return budgets, nil
}

// CreateBudget creates a new budget record
func (s *DB) CreateBudget(ctx context.Context, budget *models.Budget) error {
	query := `
        INSERT INTO budget (
            "user", changed, date, tag, income, outcome,
            income_lock, outcome_lock, is_income_forecast, is_outcome_forecast
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`

	_, err := s.pool.Exec(ctx, query,
		budget.User,
		budget.Changed,
		budget.Date,
		budget.Tag,
		budget.Income,
		budget.Outcome,
		budget.IncomeLock,
		budget.OutcomeLock,
		budget.IsIncomeForecast,
		budget.IsOutcomeForecast,
	)

	if err != nil {
		return fmt.Errorf("failed to create budget: %w", err)
	}

	return nil
}

// UpdateBudget updates an existing budget record
func (s *DB) UpdateBudget(ctx context.Context, budget *models.Budget) error {
	query := `
        UPDATE budget 
        SET changed = $4,
            income = $5,
            outcome = $6,
            income_lock = $7,
            outcome_lock = $8,
            is_income_forecast = $9,
            is_outcome_forecast = $10
        WHERE "user" = $1 AND tag = $2 AND date = $3`

	commandTag, err := s.pool.Exec(ctx, query,
		budget.User,
		budget.Tag,
		budget.Date,
		budget.Changed,
		budget.Income,
		budget.Outcome,
		budget.IncomeLock,
		budget.OutcomeLock,
		budget.IsIncomeForecast,
		budget.IsOutcomeForecast,
	)

	if err != nil {
		return fmt.Errorf("failed to update budget: %w", err)
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

	commandTag, err := s.pool.Exec(ctx, query, userID, tagID, date.Format("2006-01-02"))
	if err != nil {
		return fmt.Errorf("failed to delete budget: %w", err)
	}

	if commandTag.RowsAffected() == 0 {
		return fmt.Errorf("budget not found for user %d, tag %s, date %s",
			userID, tagID, date.Format("2006-01-02"))
	}

	return nil
}

// GetReminder retrieves a specific reminder by its ID
func (s *DB) GetReminder(ctx context.Context, id string) (*models.Reminder, error) {
	query := `
        SELECT id, "user", income, outcome, changed, income_instrument,
               outcome_instrument, step, points, tag, start_date, end_date,
               notify, interval, income_account, outcome_account, comment,
               payee, merchant
        FROM reminder
        WHERE id = $1`

	reminder := &models.Reminder{}
	err := s.pool.QueryRow(ctx, query, id).Scan(
		&reminder.ID,
		&reminder.User,
		&reminder.Income,
		&reminder.Outcome,
		&reminder.Changed,
		&reminder.IncomeInstrument,
		&reminder.OutcomeInstrument,
		&reminder.Step,
		&reminder.Points,
		&reminder.Tag,
		&reminder.StartDate,
		&reminder.EndDate,
		&reminder.Notify,
		&reminder.Interval,
		&reminder.IncomeAccount,
		&reminder.OutcomeAccount,
		&reminder.Comment,
		&reminder.Payee,
		&reminder.Merchant,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("reminder not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get reminder: %w", err)
	}

	return reminder, nil
}
