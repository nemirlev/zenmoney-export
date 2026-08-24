package postgres

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
)

func TestGetReminderMarker_Success(t *testing.T) {
	db, mock := newTestDB(t)

	expectedMarker := testReminderMarker("test-id", "2025-02-01")

	rows := mock.NewRows([]string{
		"id", "user", "date", "income", "outcome", "changed",
		"income_instrument", "outcome_instrument", "state", "is_forecast",
		"reminder", "income_account", "outcome_account", "comment",
		"payee", "merchant", "notify", "tag",
	}).AddRow(
		expectedMarker.ID,
		expectedMarker.User,
		expectedMarker.Date,
		expectedMarker.Income,
		expectedMarker.Outcome,
		expectedMarker.Changed,
		expectedMarker.IncomeInstrument,
		expectedMarker.OutcomeInstrument,
		expectedMarker.State,
		expectedMarker.IsForecast,
		expectedMarker.Reminder,
		expectedMarker.IncomeAccount,
		expectedMarker.OutcomeAccount,
		expectedMarker.Comment,
		expectedMarker.Payee,
		expectedMarker.Merchant,
		expectedMarker.Notify,
		expectedMarker.Tag,
	)

	mock.ExpectQuery(`SELECT id, "user", to_char[(]date, 'YYYY-MM-DD'[)] AS date, income, outcome, changed, income_instrument, outcome_instrument, state, is_forecast, reminder, income_account, outcome_account, comment, payee, merchant, notify, tag FROM reminder_marker WHERE id = \$1`).
		WithArgs("test-id").
		WillReturnRows(rows)

	result, err := db.GetReminderMarker(context.Background(), "test-id")
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, expectedMarker, result)
}

func TestGetReminderMarker_NotFound(t *testing.T) {
	db, mock := newTestDB(t)

	mock.ExpectQuery(`SELECT id, "user", to_char[(]date, 'YYYY-MM-DD'[)] AS date, income, outcome, changed, income_instrument, outcome_instrument, state, is_forecast, reminder, income_account, outcome_account, comment, payee, merchant, notify, tag FROM reminder_marker WHERE id = \$1`).
		WithArgs("test-id").
		WillReturnError(pgx.ErrNoRows)

	result, err := db.GetReminderMarker(context.Background(), "test-id")
	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "reminder marker not found")
}

func TestGetReminderMarker_QueryError(t *testing.T) {
	db, mock := newTestDB(t)

	mock.ExpectQuery(`SELECT id, "user", to_char[(]date, 'YYYY-MM-DD'[)] AS date, income, outcome, changed, income_instrument, outcome_instrument, state, is_forecast, reminder, income_account, outcome_account, comment, payee, merchant, notify, tag FROM reminder_marker WHERE id = \$1`).
		WithArgs("test-id").
		WillReturnError(errors.New("database error"))

	result, err := db.GetReminderMarker(context.Background(), "test-id")
	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get reminder marker")
}

func TestListReminderMarkers_Success(t *testing.T) {
	db, mock := newTestDB(t)

	filter := testDateRangeFilter()

	expectedMarker := *testReminderMarker("test-id", "2025-01-15")

	rows := mock.NewRows([]string{
		"id", "user", "date", "income", "outcome", "changed",
		"income_instrument", "outcome_instrument", "state", "is_forecast",
		"reminder", "income_account", "outcome_account", "comment",
		"payee", "merchant", "notify", "tag",
	}).AddRow(
		expectedMarker.ID,
		expectedMarker.User,
		expectedMarker.Date,
		expectedMarker.Income,
		expectedMarker.Outcome,
		expectedMarker.Changed,
		expectedMarker.IncomeInstrument,
		expectedMarker.OutcomeInstrument,
		expectedMarker.State,
		expectedMarker.IsForecast,
		expectedMarker.Reminder,
		expectedMarker.IncomeAccount,
		expectedMarker.OutcomeAccount,
		expectedMarker.Comment,
		expectedMarker.Payee,
		expectedMarker.Merchant,
		expectedMarker.Notify,
		expectedMarker.Tag,
	)

	mock.ExpectQuery(`SELECT id, "user", to_char[(]date, 'YYYY-MM-DD'[)] AS date, income, outcome, changed, income_instrument, outcome_instrument, state, is_forecast, reminder, income_account, outcome_account, comment, payee, merchant, notify, tag FROM reminder_marker WHERE "user" = \$1 AND date >= \$2 AND date <= \$3 LIMIT \$4 OFFSET \$5`).
		WithArgs(1, *filter.StartDate, *filter.EndDate, 10, 0).
		WillReturnRows(rows)

	markers, err := db.ListReminderMarkers(context.Background(), filter)
	assert.NoError(t, err)
	assert.Len(t, markers, 1)
	assert.Equal(t, expectedMarker, markers[0])
}

func TestListReminderMarkers_QueryError(t *testing.T) {
	db, mock := newTestDB(t)

	filter := testDateRangeFilter()

	mock.ExpectQuery(`SELECT id, "user", to_char[(]date, 'YYYY-MM-DD'[)] AS date, income, outcome, changed, income_instrument, outcome_instrument, state, is_forecast, reminder, income_account, outcome_account, comment, payee, merchant, notify, tag FROM reminder_marker WHERE "user" = \$1 AND date >= \$2 AND date <= \$3 LIMIT \$4 OFFSET \$5`).
		WithArgs(1, *filter.StartDate, *filter.EndDate, 10, 0).
		WillReturnError(errors.New("database error"))

	markers, err := db.ListReminderMarkers(context.Background(), filter)
	assert.Error(t, err)
	assert.Nil(t, markers)
	assert.Contains(t, err.Error(), "failed to list reminder markers")
}

func TestCreateReminderMarker_Success(t *testing.T) {
	db, mock := newTestDB(t)

	marker := testReminderMarker("test-id", "2025-02-01")

	mock.ExpectExec(`INSERT INTO reminder_marker`).
		WithArgs(reminderMarkerArgs(marker)...).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	err := db.CreateReminderMarker(context.Background(), marker)
	assert.NoError(t, err)
}

func TestCreateReminderMarker_QueryError(t *testing.T) {
	db, mock := newTestDB(t)

	marker := testReminderMarker("test-id", "2025-02-01")

	mock.ExpectExec(`INSERT INTO reminder_marker`).
		WithArgs(reminderMarkerArgs(marker)...).
		WillReturnError(errors.New("insert error"))

	err := db.CreateReminderMarker(context.Background(), marker)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create reminder marker")
}

func TestUpdateReminderMarker_Success(t *testing.T) {
	db, mock := newTestDB(t)

	marker := testReminderMarker("test-id", "2025-02-01")

	mock.ExpectExec(`UPDATE reminder_marker SET`).
		WithArgs(reminderMarkerArgs(marker)...).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	err := db.UpdateReminderMarker(context.Background(), marker)
	assert.NoError(t, err)
}

func TestUpdateReminderMarker_QueryError(t *testing.T) {
	db, mock := newTestDB(t)

	marker := testReminderMarker("test-id", "2025-02-01")

	mock.ExpectExec(`UPDATE reminder_marker SET`).
		WithArgs(reminderMarkerArgs(marker)...).
		WillReturnError(errors.New("update error"))

	err := db.UpdateReminderMarker(context.Background(), marker)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to update reminder marker")
}

func TestDeleteReminderMarker_Success(t *testing.T) {
	db, mock := newTestDB(t)

	mock.ExpectExec(`DELETE FROM reminder_marker WHERE id = \$1`).
		WithArgs("test-id").
		WillReturnResult(pgxmock.NewResult("DELETE", 1))

	err := db.DeleteReminderMarker(context.Background(), "test-id")
	assert.NoError(t, err)
}

func TestDeleteReminderMarker_NotFound(t *testing.T) {
	db, mock := newTestDB(t)

	mock.ExpectExec(`DELETE FROM reminder_marker WHERE id = \$1`).
		WithArgs("test-id").
		WillReturnResult(pgxmock.NewResult("DELETE", 0))

	err := db.DeleteReminderMarker(context.Background(), "test-id")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "reminder marker not found")
}

func TestDeleteReminderMarker_QueryError(t *testing.T) {
	db, mock := newTestDB(t)

	mock.ExpectExec(`DELETE FROM reminder_marker WHERE id = \$1`).
		WithArgs("test-id").
		WillReturnError(errors.New("database error"))

	err := db.DeleteReminderMarker(context.Background(), "test-id")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to delete reminder marker")
}
