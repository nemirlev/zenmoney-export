package postgres

import (
	"context"
	"errors"
	"github.com/nemirlev/zenmoney-export/v2/internal/interfaces"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/nemirlev/zenmoney-go-sdk/v2/models"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
)

func TestGetReminderMarker_Success(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	expectedMarker := &models.ReminderMarker{
		ID:                "test-id",
		User:              1,
		Date:              "2025-02-01",
		Income:            1000.0,
		Outcome:           500.0,
		Changed:           1234567890,
		IncomeInstrument:  1,
		OutcomeInstrument: 2,
		State:             "active",
		IsForecast:        true,
		Reminder:          "reminder-id",
		IncomeAccount:     "income-account-id",
		OutcomeAccount:    "outcome-account-id",
		Comment:           "test comment",
		Payee:             ptr("payee-id"),
		Merchant:          ptr("merchant-id"),
		Notify:            true,
		Tag:               []string{"tag1", "tag2"},
	}

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

	mock.ExpectQuery(`SELECT id, "user", date, income, outcome, changed, income_instrument, outcome_instrument, state, is_forecast, reminder, income_account, outcome_account, comment, payee, merchant, notify, tag FROM reminder_marker WHERE id = \$1`).
		WithArgs("test-id").
		WillReturnRows(rows)

	result, err := db.GetReminderMarker(context.Background(), "test-id")
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, expectedMarker, result)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetReminderMarker_NotFound(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	mock.ExpectQuery(`SELECT id, "user", date, income, outcome, changed, income_instrument, outcome_instrument, state, is_forecast, reminder, income_account, outcome_account, comment, payee, merchant, notify, tag FROM reminder_marker WHERE id = \$1`).
		WithArgs("test-id").
		WillReturnError(pgx.ErrNoRows)

	result, err := db.GetReminderMarker(context.Background(), "test-id")
	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "reminder marker not found")

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetReminderMarker_QueryError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	mock.ExpectQuery(`SELECT id, "user", date, income, outcome, changed, income_instrument, outcome_instrument, state, is_forecast, reminder, income_account, outcome_account, comment, payee, merchant, notify, tag FROM reminder_marker WHERE id = \$1`).
		WithArgs("test-id").
		WillReturnError(errors.New("database error"))

	result, err := db.GetReminderMarker(context.Background(), "test-id")
	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get reminder marker")

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestListReminderMarkers_Success(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	filter := interfaces.Filter{
		UserID:    ptr(1),
		StartDate: ptr(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)),
		EndDate:   ptr(time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC)),
		Limit:     10,
		Page:      1,
	}

	expectedMarker := models.ReminderMarker{
		ID:                "test-id",
		User:              1,
		Date:              "2025-01-15",
		Income:            1000.0,
		Outcome:           500.0,
		Changed:           1234567890,
		IncomeInstrument:  1,
		OutcomeInstrument: 2,
		State:             "active",
		IsForecast:        true,
		Reminder:          "reminder-id",
		IncomeAccount:     "income-account-id",
		OutcomeAccount:    "outcome-account-id",
		Comment:           "test comment",
		Payee:             ptr("payee-id"),
		Merchant:          ptr("merchant-id"),
		Notify:            true,
		Tag:               []string{"tag1", "tag2"},
	}

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

	mock.ExpectQuery(`SELECT id, "user", date, income, outcome, changed, income_instrument, outcome_instrument, state, is_forecast, reminder, income_account, outcome_account, comment, payee, merchant, notify, tag FROM reminder_marker WHERE "user" = \$1 AND date >= \$2 AND date <= \$3 LIMIT \$4 OFFSET \$5`).
		WithArgs(1, "2025-01-01", "2025-02-01", 10, 0).
		WillReturnRows(rows)

	markers, err := db.ListReminderMarkers(context.Background(), filter)
	assert.NoError(t, err)
	assert.Len(t, markers, 1)
	assert.Equal(t, expectedMarker, markers[0])

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestListReminderMarkers_QueryError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	filter := interfaces.Filter{
		UserID:    ptr(1),
		StartDate: ptr(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)),
		EndDate:   ptr(time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC)),
		Limit:     10,
		Page:      1,
	}

	mock.ExpectQuery(`SELECT id, "user", date, income, outcome, changed, income_instrument, outcome_instrument, state, is_forecast, reminder, income_account, outcome_account, comment, payee, merchant, notify, tag FROM reminder_marker WHERE "user" = \$1 AND date >= \$2 AND date <= \$3 LIMIT \$4 OFFSET \$5`).
		WithArgs(1, "2025-01-01", "2025-02-01", 10, 0).
		WillReturnError(errors.New("database error"))

	markers, err := db.ListReminderMarkers(context.Background(), filter)
	assert.Error(t, err)
	assert.Nil(t, markers)
	assert.Contains(t, err.Error(), "failed to list reminder markers")

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateReminderMarker_Success(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	marker := &models.ReminderMarker{
		ID:                "test-id",
		User:              1,
		Date:              "2025-02-01",
		Income:            1000.0,
		Outcome:           500.0,
		Changed:           1234567890,
		IncomeInstrument:  1,
		OutcomeInstrument: 2,
		State:             "active",
		IsForecast:        true,
		Reminder:          "reminder-id",
		IncomeAccount:     "income-account-id",
		OutcomeAccount:    "outcome-account-id",
		Comment:           "test comment",
		Payee:             ptr("payee-id"),
		Merchant:          ptr("merchant-id"),
		Notify:            true,
		Tag:               []string{"tag1", "tag2"},
	}

	mock.ExpectExec(`INSERT INTO reminder_marker \(
		id, "user", date, income, outcome, changed,
		income_instrument, outcome_instrument, state, is_forecast,
		reminder, income_account, outcome_account, comment,
		payee, merchant, notify, tag
	\) VALUES \(\$1, \$2, \$3, \$4, \$5, \$6, \$7, \$8, \$9, \$10, \$11, \$12, \$13, \$14, \$15, \$16, \$17, \$18\)`).
		WithArgs(
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
		).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	err = db.CreateReminderMarker(context.Background(), marker)
	assert.NoError(t, err)
}

func TestCreateReminderMarker_QueryError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	marker := &models.ReminderMarker{
		ID:                "test-id",
		User:              1,
		Date:              "2025-02-01",
		Income:            1000.0,
		Outcome:           500.0,
		Changed:           1234567890,
		IncomeInstrument:  1,
		OutcomeInstrument: 2,
		State:             "active",
		IsForecast:        true,
		Reminder:          "reminder-id",
		IncomeAccount:     "income-account-id",
		OutcomeAccount:    "outcome-account-id",
		Comment:           "test comment",
		Payee:             ptr("payee-id"),
		Merchant:          ptr("merchant-id"),
		Notify:            true,
		Tag:               []string{"tag1", "tag2"},
	}

	mock.ExpectExec(`INSERT INTO reminder_marker \(
		id, "user", date, income, outcome, changed,
		income_instrument, outcome_instrument, state, is_forecast,
		reminder, income_account, outcome_account, comment,
		payee, merchant, notify, tag
	\) VALUES \(\$1, \$2, \$3, \$4, \$5, \$6, \$7, \$8, \$9, \$10, \$11, \$12, \$13, \$14, \$15, \$16, \$17, \$18\)`).
		WithArgs(
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
		).
		WillReturnError(errors.New("insert error"))

	err = db.CreateReminderMarker(context.Background(), marker)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create reminder marker")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateReminderMarker_Success(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	marker := &models.ReminderMarker{
		ID:                "test-id",
		User:              1,
		Date:              "2025-02-01",
		Income:            1000.0,
		Outcome:           500.0,
		Changed:           1234567890,
		IncomeInstrument:  1,
		OutcomeInstrument: 2,
		State:             "active",
		IsForecast:        true,
		Reminder:          "reminder-id",
		IncomeAccount:     "income-account-id",
		OutcomeAccount:    "outcome-account-id",
		Comment:           "test comment",
		Payee:             ptr("payee-id"),
		Merchant:          ptr("merchant-id"),
		Notify:            true,
		Tag:               []string{"tag1", "tag2"},
	}

	mock.ExpectExec(`UPDATE reminder_marker SET
		"user" = \$2,
		date = \$3,
		income = \$4,
		outcome = \$5,
		changed = \$6,
		income_instrument = \$7,
		outcome_instrument = \$8,
		state = \$9,
		is_forecast = \$10,
		reminder = \$11,
		income_account = \$12,
		outcome_account = \$13,
		comment = \$14,
		payee = \$15,
		merchant = \$16,
		notify = \$17,
		tag = \$18
		WHERE id = \$1`).
		WithArgs(
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
		).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	err = db.UpdateReminderMarker(context.Background(), marker)
	assert.NoError(t, err)
}

func TestUpdateReminderMarker_QueryError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	marker := &models.ReminderMarker{
		ID:                "test-id",
		User:              1,
		Date:              "2025-02-01",
		Income:            1000.0,
		Outcome:           500.0,
		Changed:           1234567890,
		IncomeInstrument:  1,
		OutcomeInstrument: 2,
		State:             "active",
		IsForecast:        true,
		Reminder:          "reminder-id",
		IncomeAccount:     "income-account-id",
		OutcomeAccount:    "outcome-account-id",
		Comment:           "test comment",
		Payee:             ptr("payee-id"),
		Merchant:          ptr("merchant-id"),
		Notify:            true,
		Tag:               []string{"tag1", "tag2"},
	}

	mock.ExpectExec(`UPDATE reminder_marker SET
		"user" = \$2,
		date = \$3,
		income = \$4,
		outcome = \$5,
		changed = \$6,
		income_instrument = \$7,
		outcome_instrument = \$8,
		state = \$9,
		is_forecast = \$10,
		reminder = \$11,
		income_account = \$12,
		outcome_account = \$13,
		comment = \$14,
		payee = \$15,
		merchant = \$16,
		notify = \$17,
		tag = \$18
		WHERE id = \$1`).
		WithArgs(
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
		).
		WillReturnError(errors.New("update error"))

	err = db.UpdateReminderMarker(context.Background(), marker)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to update reminder marker")
}

func TestDeleteReminderMarker_Success(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	mock.ExpectExec(`DELETE FROM reminder_marker WHERE id = \$1`).
		WithArgs("test-id").
		WillReturnResult(pgxmock.NewResult("DELETE", 1))

	err = db.DeleteReminderMarker(context.Background(), "test-id")
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDeleteReminderMarker_NotFound(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	mock.ExpectExec(`DELETE FROM reminder_marker WHERE id = \$1`).
		WithArgs("test-id").
		WillReturnResult(pgxmock.NewResult("DELETE", 0))

	err = db.DeleteReminderMarker(context.Background(), "test-id")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "reminder marker not found")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDeleteReminderMarker_QueryError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	mock.ExpectExec(`DELETE FROM reminder_marker WHERE id = \$1`).
		WithArgs("test-id").
		WillReturnError(errors.New("database error"))

	err = db.DeleteReminderMarker(context.Background(), "test-id")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to delete reminder marker")
	assert.NoError(t, mock.ExpectationsWereMet())
}
