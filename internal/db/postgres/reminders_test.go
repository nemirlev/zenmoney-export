package postgres

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
)

func TestGetReminder_Success(t *testing.T) {
	db, mock := newTestDB(t)

	id := "test-id"
	expectedReminder := testReminder(id)

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

	mock.ExpectQuery(`SELECT id, "user", income, outcome, changed, income_instrument, outcome_instrument, step, points, tag, to_char[(]start_date, 'YYYY-MM-DD'[)] AS start_date, to_char[(]end_date, 'YYYY-MM-DD'[)] AS end_date, notify, interval, income_account, outcome_account, comment, payee, merchant FROM reminder WHERE id = \$1`).
		WithArgs(id).
		WillReturnRows(rows)

	result, err := db.GetReminder(context.Background(), id)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, expectedReminder, result)
}

func TestGetReminder_NotFound(t *testing.T) {
	db, mock := newTestDB(t)

	id := "non-existent-id"

	mock.ExpectQuery(`SELECT id, "user", income, outcome, changed, income_instrument, outcome_instrument, step, points, tag, to_char[(]start_date, 'YYYY-MM-DD'[)] AS start_date, to_char[(]end_date, 'YYYY-MM-DD'[)] AS end_date, notify, interval, income_account, outcome_account, comment, payee, merchant FROM reminder WHERE id = \$1`).
		WithArgs(id).
		WillReturnError(pgx.ErrNoRows)

	result, err := db.GetReminder(context.Background(), id)
	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), fmt.Sprintf("reminder not found: %s", id))
}

func TestGetReminder_QueryError(t *testing.T) {
	db, mock := newTestDB(t)

	id := "test-id"

	mock.ExpectQuery(`SELECT id, "user", income, outcome, changed, income_instrument, outcome_instrument, step, points, tag, to_char[(]start_date, 'YYYY-MM-DD'[)] AS start_date, to_char[(]end_date, 'YYYY-MM-DD'[)] AS end_date, notify, interval, income_account, outcome_account, comment, payee, merchant FROM reminder WHERE id = \$1`).
		WithArgs(id).
		WillReturnError(errors.New("database error"))

	result, err := db.GetReminder(context.Background(), id)
	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get reminder")
}

func TestListReminders_Success(t *testing.T) {
	db, mock := newTestDB(t)

	filter := testPageFilter()

	expectedReminder := *testReminder("test-id")

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

	mock.ExpectQuery(`SELECT id, "user", income, outcome, changed, income_instrument, outcome_instrument, step, points, tag, to_char[(]start_date, 'YYYY-MM-DD'[)] AS start_date, to_char[(]end_date, 'YYYY-MM-DD'[)] AS end_date, notify, interval, income_account, outcome_account, comment, payee, merchant FROM reminder WHERE "user" = \$1 LIMIT \$2 OFFSET \$3`).
		WithArgs(1, 10, 0).
		WillReturnRows(rows)

	reminders, err := db.ListReminders(context.Background(), filter)
	assert.NoError(t, err)
	assert.Len(t, reminders, 1)
	assert.Equal(t, expectedReminder, reminders[0])
}

func TestListReminders_QueryError(t *testing.T) {
	db, mock := newTestDB(t)

	filter := testPageFilter()

	mock.ExpectQuery(`SELECT id, "user", income, outcome, changed, income_instrument, outcome_instrument, step, points, tag, to_char[(]start_date, 'YYYY-MM-DD'[)] AS start_date, to_char[(]end_date, 'YYYY-MM-DD'[)] AS end_date, notify, interval, income_account, outcome_account, comment, payee, merchant FROM reminder WHERE "user" = \$1 LIMIT \$2 OFFSET \$3`).
		WithArgs(1, 10, 0).
		WillReturnError(errors.New("database error"))

	reminders, err := db.ListReminders(context.Background(), filter)
	assert.Error(t, err)
	assert.Nil(t, reminders)
	assert.Contains(t, err.Error(), "failed to list reminders")
}

func TestCreateReminder_Success(t *testing.T) {
	db, mock := newTestDB(t)

	reminder := testReminder("test-id")

	mock.ExpectExec(`INSERT INTO reminder`).
		WithArgs(reminderArgs(reminder)...).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	err := db.CreateReminder(context.Background(), reminder)
	assert.NoError(t, err)
}

func TestCreateReminder_QueryError(t *testing.T) {
	db, mock := newTestDB(t)

	reminder := testReminder("test-id")

	mock.ExpectExec(`INSERT INTO reminder`).
		WithArgs(reminderArgs(reminder)...).
		WillReturnError(errors.New("insert error"))

	err := db.CreateReminder(context.Background(), reminder)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create reminder")
}

func TestUpdateReminder_Success(t *testing.T) {
	db, mock := newTestDB(t)

	reminder := testReminder("test-id")

	mock.ExpectExec(`UPDATE reminder SET`).
		WithArgs(reminderArgs(reminder)...).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	err := db.UpdateReminder(context.Background(), reminder)
	assert.NoError(t, err)
}

func TestUpdateReminder_NotFound(t *testing.T) {
	db, mock := newTestDB(t)

	reminder := testReminder("non-existent-id")

	mock.ExpectExec(`UPDATE reminder SET`).
		WithArgs(reminderArgs(reminder)...).
		WillReturnResult(pgxmock.NewResult("UPDATE", 0))

	err := db.UpdateReminder(context.Background(), reminder)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "reminder not found")
}

func TestUpdateReminder_QueryError(t *testing.T) {
	db, mock := newTestDB(t)

	reminder := testReminder("test-id")

	mock.ExpectExec(`UPDATE reminder SET`).
		WithArgs(reminderArgs(reminder)...).
		WillReturnError(errors.New("update error"))

	err := db.UpdateReminder(context.Background(), reminder)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to update reminder")
}

func TestDeleteReminder_Success(t *testing.T) {
	db, mock := newTestDB(t)

	id := "test-id"

	mock.ExpectExec(`DELETE FROM reminder WHERE id = \$1`).
		WithArgs(id).
		WillReturnResult(pgxmock.NewResult("DELETE", 1))

	err := db.DeleteReminder(context.Background(), id)
	assert.NoError(t, err)
}

func TestDeleteReminder_NotFound(t *testing.T) {
	db, mock := newTestDB(t)

	id := "non-existent-id"

	mock.ExpectExec(`DELETE FROM reminder WHERE id = \$1`).
		WithArgs(id).
		WillReturnResult(pgxmock.NewResult("DELETE", 0))

	err := db.DeleteReminder(context.Background(), id)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "reminder not found")
}

func TestDeleteReminder_QueryError(t *testing.T) {
	db, mock := newTestDB(t)

	id := "test-id"

	mock.ExpectExec(`DELETE FROM reminder WHERE id = \$1`).
		WithArgs(id).
		WillReturnError(errors.New("delete error"))

	err := db.DeleteReminder(context.Background(), id)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to delete reminder")
}
