package postgres

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/nemirlev/zenmoney-export/v2/internal/interfaces"
	"github.com/nemirlev/zenmoney-go-sdk/v2/models"
	"strings"
)

// GetReminderMarker retrieves a specific reminder marker by its ID
func (s *DB) GetReminderMarker(ctx context.Context, id string) (*models.ReminderMarker, error) {
	query := `
        SELECT id, "user", date, income, outcome, changed,
               income_instrument, outcome_instrument, state, is_forecast,
               reminder, income_account, outcome_account, comment,
               payee, merchant, notify, tag
        FROM reminder_marker
        WHERE id = $1`

	marker := &models.ReminderMarker{}
	err := s.pool.QueryRow(ctx, query, id).Scan(
		&marker.ID,
		&marker.User,
		&marker.Date,
		&marker.Income,
		&marker.Outcome,
		&marker.Changed,
		&marker.IncomeInstrument,
		&marker.OutcomeInstrument,
		&marker.State,
		&marker.IsForecast,
		&marker.Reminder,
		&marker.IncomeAccount,
		&marker.OutcomeAccount,
		&marker.Comment,
		&marker.Payee,
		&marker.Merchant,
		&marker.Notify,
		&marker.Tag,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("reminder marker not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get reminder marker: %w", err)
	}

	return marker, nil
}

// ListReminderMarkers retrieves a list of reminder markers based on the provided filter
func (s *DB) ListReminderMarkers(ctx context.Context, filter interfaces.Filter) ([]models.ReminderMarker, error) {
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
        SELECT id, "user", date, income, outcome, changed,
               income_instrument, outcome_instrument, state, is_forecast,
               reminder, income_account, outcome_account, comment,
               payee, merchant, notify, tag
        FROM reminder_marker`

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argNum, argNum+1)
	args = append(args, filter.Limit, (filter.Page-1)*filter.Limit)

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list reminder markers: %w", err)
	}
	defer rows.Close()

	var markers []models.ReminderMarker
	for rows.Next() {
		var marker models.ReminderMarker
		err := rows.Scan(
			&marker.ID,
			&marker.User,
			&marker.Date,
			&marker.Income,
			&marker.Outcome,
			&marker.Changed,
			&marker.IncomeInstrument,
			&marker.OutcomeInstrument,
			&marker.State,
			&marker.IsForecast,
			&marker.Reminder,
			&marker.IncomeAccount,
			&marker.OutcomeAccount,
			&marker.Comment,
			&marker.Payee,
			&marker.Merchant,
			&marker.Notify,
			&marker.Tag,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan reminder marker: %w", err)
		}
		markers = append(markers, marker)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating reminder markers: %w", err)
	}

	return markers, nil
}

// CreateReminderMarker creates a new reminder marker record
func (s *DB) CreateReminderMarker(ctx context.Context, marker *models.ReminderMarker) error {
	query := `
        INSERT INTO reminder_marker (
            id, "user", date, income, outcome, changed,
            income_instrument, outcome_instrument, state, is_forecast,
            reminder, income_account, outcome_account, comment,
            payee, merchant, notify, tag
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, 
                  $13, $14, $15, $16, $17, $18)`

	_, err := s.pool.Exec(ctx, query,
		marker.ID,
		marker.User,
		marker.Date,
		marker.Income,
		marker.Outcome,
		marker.Changed,
		marker.IncomeInstrument,
		marker.OutcomeInstrument,
		marker.State,
		marker.IsForecast,
		marker.Reminder,
		marker.IncomeAccount,
		marker.OutcomeAccount,
		marker.Comment,
		marker.Payee,
		marker.Merchant,
		marker.Notify,
		marker.Tag,
	)

	if err != nil {
		return fmt.Errorf("failed to create reminder marker: %w", err)
	}

	return nil
}

// UpdateReminderMarker updates an existing reminder marker record
func (s *DB) UpdateReminderMarker(ctx context.Context, marker *models.ReminderMarker) error {
	query := `
        UPDATE reminder_marker SET
            "user" = $2,
            date = $3,
            income = $4,
            outcome = $5,
            changed = $6,
            income_instrument = $7,
            outcome_instrument = $8,
            state = $9,
            is_forecast = $10,
            reminder = $11,
            income_account = $12,
            outcome_account = $13,
            comment = $14,
            payee = $15,
            merchant = $16,
            notify = $17,
            tag = $18
        WHERE id = $1`

	commandTag, err := s.pool.Exec(ctx, query,
		marker.ID,
		marker.User,
		marker.Date,
		marker.Income,
		marker.Outcome,
		marker.Changed,
		marker.IncomeInstrument,
		marker.OutcomeInstrument,
		marker.State,
		marker.IsForecast,
		marker.Reminder,
		marker.IncomeAccount,
		marker.OutcomeAccount,
		marker.Comment,
		marker.Payee,
		marker.Merchant,
		marker.Notify,
		marker.Tag,
	)

	if err != nil {
		return fmt.Errorf("failed to update reminder marker: %w", err)
	}

	if commandTag.RowsAffected() == 0 {
		return fmt.Errorf("reminder marker not found: %s", marker.ID)
	}

	return nil
}

// DeleteReminderMarker deletes a reminder marker by its ID
func (s *DB) DeleteReminderMarker(ctx context.Context, id string) error {
	query := `DELETE FROM reminder_marker WHERE id = $1`

	commandTag, err := s.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete reminder marker: %w", err)
	}

	if commandTag.RowsAffected() == 0 {
		return fmt.Errorf("reminder marker not found: %s", id)
	}

	return nil
}
