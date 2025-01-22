package postgres

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/nemirlev/zenmoney-export/internal/interfaces"
	"github.com/nemirlev/zenmoney-go-sdk/v2/models"
	"strings"
)

// GetUser retrieves a specific user by their ID
func (s *DB) GetUser(ctx context.Context, id int) (*models.User, error) {
	query := `
        SELECT id, country, login, parent, country_code, email,
               changed, currency, paid_till, month_start_day,
               is_forecast_enabled, plan_balance_mode, plan_settings,
               subscription, subscription_renewal_date
        FROM "user"
        WHERE id = $1`

	user := &models.User{}
	err := s.pool.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.Country,
		&user.Login,
		&user.Parent,
		&user.CountryCode,
		&user.Email,
		&user.Changed,
		&user.Currency,
		&user.PaidTill,
		&user.MonthStartDay,
		&user.IsForecastEnabled,
		&user.PlanBalanceMode,
		&user.PlanSettings,
		&user.Subscription,
		&user.SubscriptionRenewalDate,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("user not found: %d", id)
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

// ListUsers retrieves a list of users based on the provided filter
func (s *DB) ListUsers(ctx context.Context, filter interfaces.Filter) ([]models.User, error) {
	var conditions []string
	var args []interface{}
	argNum := 1

	// Build the WHERE clause based on filter
	if filter.UserID != nil {
		conditions = append(conditions, fmt.Sprintf("id = $%d", argNum))
		args = append(args, *filter.UserID)
		argNum++
	}

	query := `
        SELECT id, country, login, parent, country_code, email,
               changed, currency, paid_till, month_start_day,
               is_forecast_enabled, plan_balance_mode, plan_settings,
               subscription, subscription_renewal_date
        FROM "user"`

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	// Add pagination
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argNum, argNum+1)
	args = append(args, filter.Limit, (filter.Page-1)*filter.Limit)

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var user models.User
		err := rows.Scan(
			&user.ID,
			&user.Country,
			&user.Login,
			&user.Parent,
			&user.CountryCode,
			&user.Email,
			&user.Changed,
			&user.Currency,
			&user.PaidTill,
			&user.MonthStartDay,
			&user.IsForecastEnabled,
			&user.PlanBalanceMode,
			&user.PlanSettings,
			&user.Subscription,
			&user.SubscriptionRenewalDate,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, user)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating users: %w", err)
	}

	return users, nil
}

// CreateUser creates a new user record
func (s *DB) CreateUser(ctx context.Context, user *models.User) error {
	query := `
        INSERT INTO "user" (
            id, country, login, parent, country_code, email,
            changed, currency, paid_till, month_start_day,
            is_forecast_enabled, plan_balance_mode, plan_settings,
            subscription, subscription_renewal_date
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)`

	_, err := s.pool.Exec(ctx, query,
		user.ID,
		user.Country,
		user.Login,
		user.Parent,
		user.CountryCode,
		user.Email,
		user.Changed,
		user.Currency,
		user.PaidTill,
		user.MonthStartDay,
		user.IsForecastEnabled,
		user.PlanBalanceMode,
		user.PlanSettings,
		user.Subscription,
		user.SubscriptionRenewalDate,
	)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

// UpdateUser updates an existing user record
func (s *DB) UpdateUser(ctx context.Context, user *models.User) error {
	query := `
        UPDATE "user"
        SET country = $2, login = $3, parent = $4, country_code = $5,
            email = $6, changed = $7, currency = $8, paid_till = $9,
            month_start_day = $10, is_forecast_enabled = $11,
            plan_balance_mode = $12, plan_settings = $13,
            subscription = $14, subscription_renewal_date = $15
        WHERE id = $1`

	commandTag, err := s.pool.Exec(ctx, query,
		user.ID,
		user.Country,
		user.Login,
		user.Parent,
		user.CountryCode,
		user.Email,
		user.Changed,
		user.Currency,
		user.PaidTill,
		user.MonthStartDay,
		user.IsForecastEnabled,
		user.PlanBalanceMode,
		user.PlanSettings,
		user.Subscription,
		user.SubscriptionRenewalDate,
	)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	if commandTag.RowsAffected() == 0 {
		return fmt.Errorf("user not found: %d", user.ID)
	}

	return nil
}

// DeleteUser deletes a user by their ID
func (s *DB) DeleteUser(ctx context.Context, id int) error {
	query := `DELETE FROM "user" WHERE id = $1`

	commandTag, err := s.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	if commandTag.RowsAffected() == 0 {
		return fmt.Errorf("user not found: %d", id)
	}

	return nil
}
