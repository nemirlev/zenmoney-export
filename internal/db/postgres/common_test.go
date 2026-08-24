package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/nemirlev/zenmoney-export/v2/internal/interfaces"
	"github.com/nemirlev/zenmoney-go-sdk/v3/models"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestDB(t *testing.T) (*DB, pgxmock.PgxPoolIface) {
	t.Helper()

	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, mock.ExpectationsWereMet())
		mock.Close()
	})

	return &DB{pool: mock}, mock
}

func testPageFilter() interfaces.Filter {
	return interfaces.Filter{UserID: new(1), Limit: 10, Page: 1}
}

func testDateRangeFilter() interfaces.Filter {
	return interfaces.Filter{
		UserID:    new(1),
		StartDate: new(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)),
		EndDate:   new(time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC)),
		Limit:     10,
		Page:      1,
	}
}

func testReminderMarker(id, date string) *models.ReminderMarker {
	return &models.ReminderMarker{
		ID: id, User: 1, Date: date, Income: 1000, Outcome: 500, Changed: 1234567890,
		IncomeInstrument: 1, OutcomeInstrument: 2, State: "active", IsForecast: true,
		Reminder: "reminder-id", IncomeAccount: "income-account-id", OutcomeAccount: "outcome-account-id",
		Comment: "test comment", Payee: new("payee-id"), Merchant: new("merchant-id"), Notify: true,
		Tag: []string{"tag1", "tag2"},
	}
}

func reminderMarkerArgs(marker *models.ReminderMarker) []any {
	return []any{
		marker.ID, marker.User, marker.Date, decimalString(marker.Income), decimalString(marker.Outcome),
		marker.Changed, marker.IncomeInstrument, marker.OutcomeInstrument, marker.State, marker.IsForecast,
		marker.Reminder, marker.IncomeAccount, marker.OutcomeAccount, marker.Comment, marker.Payee,
		marker.Merchant, marker.Notify, marker.Tag,
	}
}

func testReminder(id string) *models.Reminder {
	return &models.Reminder{
		ID: id, User: 1, Income: 100, Outcome: 50, Changed: 1234567890,
		IncomeInstrument: 1, OutcomeInstrument: 2, Step: 7, Points: []int{0, 2, 4},
		Tag: []string{"tag1", "tag2"}, StartDate: "2025-01-01", Notify: true, Interval: new("week"),
		IncomeAccount: "income-account", OutcomeAccount: "outcome-account", Comment: new("test comment"),
		Payee: new("payee-id"), Merchant: new("merchant-id"),
	}
}

func reminderArgs(reminder *models.Reminder) []any {
	return []any{
		reminder.ID, reminder.User, decimalString(reminder.Income), decimalString(reminder.Outcome),
		reminder.Changed, reminder.IncomeInstrument, reminder.OutcomeInstrument, reminder.Step,
		reminder.Points, reminder.Tag, reminder.StartDate, reminder.EndDate, reminder.Notify,
		reminder.Interval, reminder.IncomeAccount, reminder.OutcomeAccount, reminder.Comment,
		reminder.Payee, reminder.Merchant,
	}
}

func testTransaction() *models.Transaction {
	return &models.Transaction{
		ID: "test-id", User: 1, Date: "2023-01-01", Income: 1000, Outcome: 500, Changed: 1234567890,
		IncomeInstrument: 1, OutcomeInstrument: 2, Created: 1234567890, OriginalPayee: "Original Payee",
		Viewed: true, QRCode: new("QRCode"), Source: "Source", IncomeAccount: "IncomeAccount",
		OutcomeAccount: new("OutcomeAccount"), Tag: []string{"tag1", "tag2"}, Comment: new("Comment"),
		Payee: "Payee", OpIncome: 100, OpOutcome: 50, OpIncomeInstrument: new(3), OpOutcomeInstrument: new(4),
		Latitude: new(55.7558), Longitude: new(37.6176), Merchant: new("Merchant"),
		IncomeBankID: new("IncomeBankID"), OutcomeBankID: new("OutcomeBankID"), ReminderMarker: new("ReminderMarker"),
	}
}

func transactionArgs(transaction *models.Transaction) []any {
	return []any{
		transaction.ID, transaction.User, transaction.Date, decimalString(transaction.Income), decimalString(transaction.Outcome),
		transaction.Changed, transaction.IncomeInstrument, transaction.OutcomeInstrument, transaction.Created,
		transaction.OriginalPayee, transaction.Deleted, transaction.Viewed, transaction.Hold, transaction.QRCode,
		transaction.Source, transaction.IncomeAccount, transaction.OutcomeAccount, transaction.Tag, transaction.Comment,
		transaction.Payee, decimalString(transaction.OpIncome), decimalString(transaction.OpOutcome), transaction.OpIncomeInstrument,
		transaction.OpOutcomeInstrument, transaction.Latitude, transaction.Longitude, transaction.Merchant,
		transaction.IncomeBankID, transaction.OutcomeBankID, transaction.ReminderMarker,
	}
}

func testUser(login, email string) *models.User {
	return &models.User{
		ID: 1, Country: 1, Login: login, CountryCode: "US", Email: email, Changed: 1234567890,
		Currency: 1, PaidTill: 1234567890, MonthStartDay: 1, IsForecastEnabled: true,
		PlanBalanceMode: "balance", PlanSettings: "settings", Subscription: "subscription",
	}
}

func userArgs(user *models.User) []any {
	return []any{
		user.ID, user.Country, user.Login, user.Parent, user.CountryCode, user.Email,
		user.Changed, user.Currency, user.PaidTill, user.MonthStartDay, user.IsForecastEnabled,
		user.PlanBalanceMode, user.PlanSettings, user.Subscription, user.SubscriptionRenewalDate,
	}
}

func testTag(id, title string) *models.Tag {
	return &models.Tag{
		ID: id, User: 1, Changed: 1234567890, Icon: new("icon"), BudgetIncome: true,
		Required: new(true), Archive: true, Color: new(int64(123456)), Picture: new("picture"),
		Title: title, ShowIncome: true, Parent: new("parent-id"), StaticID: "static-id",
	}
}

func tagArgs(tag *models.Tag) []any {
	return []any{
		tag.ID, tag.User, tag.Changed, tag.Icon, tag.BudgetIncome, tag.BudgetOutcome,
		tag.Required, tag.Archive, tag.Color, tag.Picture, tag.Title, tag.ShowIncome,
		tag.ShowOutcome, tag.Parent, tag.StaticID,
	}
}

func TestNewPostgresStorage(t *testing.T) {
	t.Run("invalid connection string", func(t *testing.T) {
		_, err := NewPostgresStorage("invalid_connection_string")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse postgres config")
	})
}

func TestDecimalStringUsesCanonicalNonExponentialRepresentation(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		value float64
		want  string
	}{
		"fraction":      {value: 0.1, want: "0.1"},
		"trailing zero": {value: 12.50, want: "12.5"},
		"small amount":  {value: 0.00000001, want: "0.00000001"},
		"large amount":  {value: 1e20, want: "100000000000000000000"},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, decimalString(tt.value))
		})
	}
}

func TestOptionalDecimalStringPreservesNull(t *testing.T) {
	t.Parallel()

	var value *float64
	assert.Equal(t, value, optionalDecimalString(value))

	amount := 42.125
	assert.Equal(t, "42.125", optionalDecimalString(&amount))
}

// Тест для метода Close
func TestDB_Close(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)

	db := &DB{pool: mock}

	err = db.Close(context.Background())
	assert.NoError(t, err)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

// Тест для метода Ping
func TestDB_Ping(t *testing.T) {
	t.Run("successful ping", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)

		mock.ExpectPing()

		db := &DB{pool: mock}

		err = db.Ping(context.Background())
		assert.NoError(t, err)

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("failed ping", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)

		expectedErr := errors.New("ping error")
		mock.ExpectPing().WillReturnError(expectedErr)

		db := &DB{pool: mock}

		err = db.Ping(context.Background())
		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})
}
