package postgres

import (
	"context"
	"fmt"
	"github.com/nemirlev/zenmoney-export/internal/interfaces"
	"github.com/nemirlev/zenmoney-go-sdk/v2/models"
	"strings"
)

// ListReminders retrieves a list of reminders based on the provided filter
func (s *DB) ListReminders(ctx context.Context, filter interfaces.Filter) ([]models.Reminder, error) {
	var conditions []string
	var args []interface{}
	argNum := 1

	if filter.UserID != nil {
		conditions = append(conditions, fmt.Sprintf(`"user" = $%d`, argNum))
		args = append(args, *filter.UserID)
		argNum++
	}

	query := `
        SELECT id, "user", income, outcome, changed, income_instrument,
               outcome_instrument, step, points, tag, start_date, end_date,
               notify, interval, income_account, outcome_account, comment,
               payee, merchant
        FROM reminder`

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argNum, argNum+1)
	args = append(args, filter.Limit, (filter.Page-1)*filter.Limit)

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list reminders: %w", err)
	}
	defer rows.Close()

	var reminders []models.Reminder
	for rows.Next() {
		var reminder models.Reminder
		err := rows.Scan(
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
			return nil, fmt.Errorf("failed to scan reminder: %w", err)
		}
		reminders = append(reminders, reminder)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating reminders: %w", err)
	}

	return reminders, nil
}

// CreateReminder creates a new reminder record
func (s *DB) CreateReminder(ctx context.Context, reminder *models.Reminder) error {
	query := `
        INSERT INTO reminder (
            id, "user", income, outcome, changed, income_instrument,
            outcome_instrument, step, points, tag, start_date, end_date,
            notify, interval, income_account, outcome_account, comment,
            payee, merchant
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, 
                  $14, $15, $16, $17, $18, $19)`

	_, err := s.pool.Exec(ctx, query,
		reminder.ID,
		reminder.User,
		reminder.Income,
		reminder.Outcome,
		reminder.Changed,
		reminder.IncomeInstrument,
		reminder.OutcomeInstrument,
		reminder.Step,
		reminder.Points,
		reminder.Tag,
		reminder.StartDate,
		reminder.EndDate,
		reminder.Notify,
		reminder.Interval,
		reminder.IncomeAccount,
		reminder.OutcomeAccount,
		reminder.Comment,
		reminder.Payee,
		reminder.Merchant,
	)

	if err != nil {
		return fmt.Errorf("failed to create reminder: %w", err)
	}

	return nil
}

// UpdateReminder updates an existing reminder record
func (s *DB) UpdateReminder(ctx context.Context, reminder *models.Reminder) error {
	query := `
        UPDATE reminder SET
            "user" = $2,
            income = $3,
            outcome = $4,
            changed = $5,
            income_instrument = $6,
            outcome_instrument = $7,
            step = $8,
            points = $9,
            tag = $10,
            start_date = $11,
            end_date = $12,
            notify = $13,
            interval = $14,
            income_account = $15,
            outcome_account = $16,
            comment = $17,
            payee = $18,
            merchant = $19
        WHERE id = $1`

	commandTag, err := s.pool.Exec(ctx, query,
		reminder.ID,
		reminder.User,
		reminder.Income,
		reminder.Outcome,
		reminder.Changed,
		reminder.IncomeInstrument,
		reminder.OutcomeInstrument,
		reminder.Step,
		reminder.Points,
		reminder.Tag,
		reminder.StartDate,
		reminder.EndDate,
		reminder.Notify,
		reminder.Interval,
		reminder.IncomeAccount,
		reminder.OutcomeAccount,
		reminder.Comment,
		reminder.Payee,
		reminder.Merchant,
	)

	if err != nil {
		return fmt.Errorf("failed to update reminder: %w", err)
	}

	if commandTag.RowsAffected() == 0 {
		return fmt.Errorf("reminder not found: %s", reminder.ID)
	}

	return nil
}

// DeleteReminder deletes a reminder by its ID
func (s *DB) DeleteReminder(ctx context.Context, id string) error {
	query := `DELETE FROM reminder WHERE id = $1`

	commandTag, err := s.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete reminder: %w", err)
	}

	if commandTag.RowsAffected() == 0 {
		return fmt.Errorf("reminder not found: %s", id)
	}

	return nil
}
