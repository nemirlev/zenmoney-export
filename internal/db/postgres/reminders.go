package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/nemirlev/zenmoney-export/v2/internal/interfaces"
	"github.com/nemirlev/zenmoney-go-sdk/v3/models"
)

const reminderSelectQuery = `
        SELECT id, "user", income, outcome, changed, income_instrument,
               outcome_instrument, step, points, tag,
               to_char(start_date, 'YYYY-MM-DD') AS start_date,
               to_char(end_date, 'YYYY-MM-DD') AS end_date,
               notify, interval, income_account, outcome_account, comment,
               payee, merchant
        FROM reminder`

const reminderInsertQuery = `
        INSERT INTO reminder (
            id, "user", income, outcome, changed, income_instrument,
            outcome_instrument, step, points, tag, start_date, end_date,
            notify, interval, income_account, outcome_account, comment,
            payee, merchant
        ) VALUES ($1, $2, $3::text::numeric, $4::text::numeric, $5, $6, $7,
                  $8, $9, $10, $11::text::date,
                  NULLIF(BTRIM($12::text), '')::date, $13, $14,
                  $15::text::uuid, $16::text::uuid, $17, $18, $19)`

func scanReminder(row rowScanner) (models.Reminder, error) {
	var reminder models.Reminder
	err := row.Scan(
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
	return reminder, err
}

func reminderValues(reminder *models.Reminder) []any {
	return []any{
		reminder.ID,
		reminder.User,
		decimalString(reminder.Income),
		decimalString(reminder.Outcome),
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
	}
}

// GetReminder retrieves a specific reminder by its ID
func (s *DB) GetReminder(ctx context.Context, id string) (*models.Reminder, error) {
	reminder, err := scanReminder(s.pool.QueryRow(ctx, reminderSelectQuery+` WHERE id = $1`, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("reminder not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get reminder: %w", err)
	}

	return &reminder, nil
}

// ListReminders retrieves a list of reminders based on the provided filter
func (s *DB) ListReminders(
	ctx context.Context,
	filter interfaces.Filter,
) ([]models.Reminder, error) {
	query, args := buildListQuery(reminderSelectQuery, filter, false, "")

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list reminders: %w", err)
	}
	defer rows.Close()

	return collectRows(rows, scanReminder, "failed to scan reminder", "error iterating reminders")
}

// CreateReminder creates a new reminder record
func (s *DB) CreateReminder(ctx context.Context, reminder *models.Reminder) error {
	_, err := execCommand(
		ctx,
		s.pool,
		reminderInsertQuery,
		"failed to create reminder",
		reminderValues(reminder)...,
	)
	return err
}

// UpdateReminder updates an existing reminder record
func (s *DB) UpdateReminder(ctx context.Context, reminder *models.Reminder) error {
	query := `
        UPDATE reminder SET
            "user" = $2,
            income = $3::text::numeric,
            outcome = $4::text::numeric,
            changed = $5,
            income_instrument = $6,
            outcome_instrument = $7,
            step = $8,
            points = $9,
            tag = $10,
            start_date = $11::text::date,
            end_date = NULLIF(BTRIM($12::text), '')::date,
            notify = $13,
            interval = $14,
            income_account = $15::text::uuid,
            outcome_account = $16::text::uuid,
            comment = $17,
            payee = $18,
            merchant = $19
        WHERE id = $1`

	commandTag, err := execCommand(
		ctx,
		s.pool,
		query,
		"failed to update reminder",
		reminderValues(reminder)...,
	)
	if err != nil {
		return err
	}

	return requireRowsAffected(commandTag, fmt.Sprintf("reminder not found: %s", reminder.ID))
}

// DeleteReminder deletes a reminder by its ID
func (s *DB) DeleteReminder(ctx context.Context, id string) error {
	query := `DELETE FROM reminder WHERE id = $1`

	commandTag, err := execCommand(ctx, s.pool, query, "failed to delete reminder", id)
	if err != nil {
		return err
	}

	return requireRowsAffected(commandTag, fmt.Sprintf("reminder not found: %s", id))
}
