package postgres

import (
	"context"
	"errors"
	"fmt"
	"github.com/nemirlev/zenmoney-export/v2/internal/interfaces"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/nemirlev/zenmoney-go-sdk/v2/models"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
)

func TestGetReminder_Success(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	id := "test-id"
	expectedReminder := &models.Reminder{
		ID:                id,
		User:              1,
		Income:            100.0,
		Outcome:           50.0,
		Changed:           1234567890,
		IncomeInstrument:  1,
		OutcomeInstrument: 2,
		Step:              7,
		Points:            []int{0, 2, 4},
		Tag:               []string{"tag1", "tag2"},
		StartDate:         "2025-01-01",
		EndDate:           nil,
		Notify:            true,
		Interval:          ptr("week"),
		IncomeAccount:     "income-account",
		OutcomeAccount:    "outcome-account",
		Comment:           "test comment",
		Payee:             ptr("payee-id"),
		Merchant:          ptr("merchant-id"),
	}

	rows := mock.NewRows([]string{
		"id", "user", "income", "outcome", "changed", "income_instrument",
		"outcome_instrument", "step", "points", "tag", "start_date", "end_date",
		"notify", "interval", "income_account", "outcome_account", "comment",
		"payee", "merchant",
	}).AddRow(
		expectedReminder.ID,
		expectedReminder.User,
		expectedReminder.Income,
		expectedReminder.Outcome,
		expectedReminder.Changed,
		expectedReminder.IncomeInstrument,
		expectedReminder.OutcomeInstrument,
		expectedReminder.Step,
		expectedReminder.Points,
		expectedReminder.Tag,
		expectedReminder.StartDate,
		expectedReminder.EndDate,
		expectedReminder.Notify,
		expectedReminder.Interval,
		expectedReminder.IncomeAccount,
		expectedReminder.OutcomeAccount,
		expectedReminder.Comment,
		expectedReminder.Payee,
		expectedReminder.Merchant,
	)

	mock.ExpectQuery(`SELECT id, "user", income, outcome, changed, income_instrument, outcome_instrument, step, points, tag, start_date, end_date, notify, interval, income_account, outcome_account, comment, payee, merchant FROM reminder WHERE id = \$1`).
		WithArgs(id).
		WillReturnRows(rows)

	result, err := db.GetReminder(context.Background(), id)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, expectedReminder, result)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetReminder_NotFound(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	id := "non-existent-id"

	mock.ExpectQuery(`SELECT id, "user", income, outcome, changed, income_instrument, outcome_instrument, step, points, tag, start_date, end_date, notify, interval, income_account, outcome_account, comment, payee, merchant FROM reminder WHERE id = \$1`).
		WithArgs(id).
		WillReturnError(pgx.ErrNoRows)

	result, err := db.GetReminder(context.Background(), id)
	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), fmt.Sprintf("reminder not found: %s", id))

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetReminder_QueryError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	id := "test-id"

	mock.ExpectQuery(`SELECT id, "user", income, outcome, changed, income_instrument, outcome_instrument, step, points, tag, start_date, end_date, notify, interval, income_account, outcome_account, comment, payee, merchant FROM reminder WHERE id = \$1`).
		WithArgs(id).
		WillReturnError(errors.New("database error"))

	result, err := db.GetReminder(context.Background(), id)
	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get reminder")

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestListReminders_Success(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	filter := interfaces.Filter{
		UserID: ptr(1),
		Limit:  10,
		Page:   1,
	}

	expectedReminder := models.Reminder{
		ID:                "test-id",
		User:              1,
		Income:            100.0,
		Outcome:           50.0,
		Changed:           1234567890,
		IncomeInstrument:  1,
		OutcomeInstrument: 2,
		Step:              7,
		Points:            []int{0, 2, 4},
		Tag:               []string{"tag1", "tag2"},
		StartDate:         "2025-01-01",
		EndDate:           nil,
		Notify:            true,
		Interval:          ptr("week"),
		IncomeAccount:     "income-account",
		OutcomeAccount:    "outcome-account",
		Comment:           "test comment",
		Payee:             ptr("payee-id"),
		Merchant:          ptr("merchant-id"),
	}

	rows := mock.NewRows([]string{
		"id", "user", "income", "outcome", "changed", "income_instrument",
		"outcome_instrument", "step", "points", "tag", "start_date", "end_date",
		"notify", "interval", "income_account", "outcome_account", "comment",
		"payee", "merchant",
	}).AddRow(
		expectedReminder.ID,
		expectedReminder.User,
		expectedReminder.Income,
		expectedReminder.Outcome,
		expectedReminder.Changed,
		expectedReminder.IncomeInstrument,
		expectedReminder.OutcomeInstrument,
		expectedReminder.Step,
		expectedReminder.Points,
		expectedReminder.Tag,
		expectedReminder.StartDate,
		expectedReminder.EndDate,
		expectedReminder.Notify,
		expectedReminder.Interval,
		expectedReminder.IncomeAccount,
		expectedReminder.OutcomeAccount,
		expectedReminder.Comment,
		expectedReminder.Payee,
		expectedReminder.Merchant,
	)

	mock.ExpectQuery(`SELECT id, "user", income, outcome, changed, income_instrument, outcome_instrument, step, points, tag, start_date, end_date, notify, interval, income_account, outcome_account, comment, payee, merchant FROM reminder WHERE "user" = \$1 LIMIT \$2 OFFSET \$3`).
		WithArgs(1, 10, 0).
		WillReturnRows(rows)

	reminders, err := db.ListReminders(context.Background(), filter)
	assert.NoError(t, err)
	assert.Len(t, reminders, 1)
	assert.Equal(t, expectedReminder, reminders[0])

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestListReminders_QueryError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	filter := interfaces.Filter{
		UserID: ptr(1),
		Limit:  10,
		Page:   1,
	}

	mock.ExpectQuery(`SELECT id, "user", income, outcome, changed, income_instrument, outcome_instrument, step, points, tag, start_date, end_date, notify, interval, income_account, outcome_account, comment, payee, merchant FROM reminder WHERE "user" = \$1 LIMIT \$2 OFFSET \$3`).
		WithArgs(1, 10, 0).
		WillReturnError(errors.New("database error"))

	reminders, err := db.ListReminders(context.Background(), filter)
	assert.Error(t, err)
	assert.Nil(t, reminders)
	assert.Contains(t, err.Error(), "failed to list reminders")

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateReminder_Success(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	reminder := &models.Reminder{
		ID:                "test-id",
		User:              1,
		Income:            100.0,
		Outcome:           50.0,
		Changed:           1234567890,
		IncomeInstrument:  1,
		OutcomeInstrument: 2,
		Step:              7,
		Points:            []int{0, 2, 4},
		Tag:               []string{"tag1", "tag2"},
		StartDate:         "2025-01-01",
		EndDate:           nil,
		Notify:            true,
		Interval:          ptr("week"),
		IncomeAccount:     "income-account",
		OutcomeAccount:    "outcome-account",
		Comment:           "test comment",
		Payee:             ptr("payee-id"),
		Merchant:          ptr("merchant-id"),
	}

	mock.ExpectExec(`INSERT INTO reminder \(
  id, "user", income, outcome, changed, income_instrument,
  outcome_instrument, step, points, tag, start_date, end_date,
  notify, interval, income_account, outcome_account, comment,
  payee, merchant
 \) VALUES \(\$1, \$2, \$3, \$4, \$5, \$6, \$7, \$8, \$9, \$10, \$11, \$12, \$13, \$14, \$15, \$16, \$17, \$18, \$19\)`).
		WithArgs(
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
		).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	err = db.CreateReminder(context.Background(), reminder)
	assert.NoError(t, err)
}

func TestCreateReminder_QueryError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	reminder := &models.Reminder{
		ID:                "test-id",
		User:              1,
		Income:            100.0,
		Outcome:           50.0,
		Changed:           1234567890,
		IncomeInstrument:  1,
		OutcomeInstrument: 2,
		Step:              7,
		Points:            []int{0, 2, 4},
		Tag:               []string{"tag1", "tag2"},
		StartDate:         "2025-01-01",
		EndDate:           nil,
		Notify:            true,
		Interval:          ptr("week"),
		IncomeAccount:     "income-account",
		OutcomeAccount:    "outcome-account",
		Comment:           "test comment",
		Payee:             ptr("payee-id"),
		Merchant:          ptr("merchant-id"),
	}

	mock.ExpectExec(`INSERT INTO reminder \(
  id, "user", income, outcome, changed, income_instrument,
  outcome_instrument, step, points, tag, start_date, end_date,
  notify, interval, income_account, outcome_account, comment,
  payee, merchant
 \) VALUES \(\$1, \$2, \$3, \$4, \$5, \$6, \$7, \$8, \$9, \$10, \$11, \$12, \$13, \$14, \$15, \$16, \$17, \$18, \$19\)`).
		WithArgs(
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
		).
		WillReturnError(errors.New("insert error"))

	err = db.CreateReminder(context.Background(), reminder)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create reminder")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateReminder_Success(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	reminder := &models.Reminder{
		ID:                "test-id",
		User:              1,
		Income:            100.0,
		Outcome:           50.0,
		Changed:           1234567890,
		IncomeInstrument:  1,
		OutcomeInstrument: 2,
		Step:              7,
		Points:            []int{0, 2, 4},
		Tag:               []string{"tag1", "tag2"},
		StartDate:         "2025-01-01",
		EndDate:           nil,
		Notify:            true,
		Interval:          ptr("week"),
		IncomeAccount:     "income-account",
		OutcomeAccount:    "outcome-account",
		Comment:           "test comment",
		Payee:             ptr("payee-id"),
		Merchant:          ptr("merchant-id"),
	}

	mock.ExpectExec(`UPDATE reminder SET
  "user" = \$2,
  income = \$3,
  outcome = \$4,
  changed = \$5,
  income_instrument = \$6,
  outcome_instrument = \$7,
  step = \$8,
  points = \$9,
  tag = \$10,
  start_date = \$11,
  end_date = \$12,
  notify = \$13,
  interval = \$14,
  income_account = \$15,
  outcome_account = \$16,
  comment = \$17,
  payee = \$18,
  merchant = \$19
 WHERE id = \$1`).
		WithArgs(
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
		).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	err = db.UpdateReminder(context.Background(), reminder)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateReminder_NotFound(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	reminder := &models.Reminder{
		ID:                "non-existent-id",
		User:              1,
		Income:            100.0,
		Outcome:           50.0,
		Changed:           1234567890,
		IncomeInstrument:  1,
		OutcomeInstrument: 2,
		Step:              7,
		Points:            []int{0, 2, 4},
		Tag:               []string{"tag1", "tag2"},
		StartDate:         "2025-01-01",
		EndDate:           nil,
		Notify:            true,
		Interval:          ptr("week"),
		IncomeAccount:     "income-account",
		OutcomeAccount:    "outcome-account",
		Comment:           "test comment",
		Payee:             ptr("payee-id"),
		Merchant:          ptr("merchant-id"),
	}

	mock.ExpectExec(`UPDATE reminder SET
  "user" = \$2,
  income = \$3,
  outcome = \$4,
  changed = \$5,
  income_instrument = \$6,
  outcome_instrument = \$7,
  step = \$8,
  points = \$9,
  tag = \$10,
  start_date = \$11,
  end_date = \$12,
  notify = \$13,
  interval = \$14,
  income_account = \$15,
  outcome_account = \$16,
  comment = \$17,
  payee = \$18,
  merchant = \$19
 WHERE id = \$1`).
		WithArgs(
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
		).
		WillReturnResult(pgxmock.NewResult("UPDATE", 0))

	err = db.UpdateReminder(context.Background(), reminder)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "reminder not found")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateReminder_QueryError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	reminder := &models.Reminder{
		ID:                "test-id",
		User:              1,
		Income:            100.0,
		Outcome:           50.0,
		Changed:           1234567890,
		IncomeInstrument:  1,
		OutcomeInstrument: 2,
		Step:              7,
		Points:            []int{0, 2, 4},
		Tag:               []string{"tag1", "tag2"},
		StartDate:         "2025-01-01",
		EndDate:           nil,
		Notify:            true,
		Interval:          ptr("week"),
		IncomeAccount:     "income-account",
		OutcomeAccount:    "outcome-account",
		Comment:           "test comment",
		Payee:             ptr("payee-id"),
		Merchant:          ptr("merchant-id"),
	}

	mock.ExpectExec(`UPDATE reminder SET
  "user" = \$2,
  income = \$3,
  outcome = \$4,
  changed = \$5,
  income_instrument = \$6,
  outcome_instrument = \$7,
  step = \$8,
  points = \$9,
  tag = \$10,
  start_date = \$11,
  end_date = \$12,
  notify = \$13,
  interval = \$14,
  income_account = \$15,
  outcome_account = \$16,
  comment = \$17,
  payee = \$18,
  merchant = \$19
 WHERE id = \$1`).
		WithArgs(
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
		).
		WillReturnError(errors.New("update error"))

	err = db.UpdateReminder(context.Background(), reminder)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to update reminder")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDeleteReminder_Success(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	id := "test-id"

	mock.ExpectExec(`DELETE FROM reminder WHERE id = \$1`).
		WithArgs(id).
		WillReturnResult(pgxmock.NewResult("DELETE", 1))

	err = db.DeleteReminder(context.Background(), id)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDeleteReminder_NotFound(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	id := "non-existent-id"

	mock.ExpectExec(`DELETE FROM reminder WHERE id = \$1`).
		WithArgs(id).
		WillReturnResult(pgxmock.NewResult("DELETE", 0))

	err = db.DeleteReminder(context.Background(), id)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "reminder not found")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDeleteReminder_QueryError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	id := "test-id"

	mock.ExpectExec(`DELETE FROM reminder WHERE id = \$1`).
		WithArgs(id).
		WillReturnError(errors.New("delete error"))

	err = db.DeleteReminder(context.Background(), id)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to delete reminder")
	assert.NoError(t, mock.ExpectationsWereMet())
}
