package postgres

import (
	"context"
	"testing"

	"github.com/nemirlev/zenmoney-go-sdk/v2/models"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
)

func TestSaveInstruments_Success(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	instruments := []models.Instrument{
		{ID: 1, Title: "USD", ShortTitle: "USD", Symbol: "$", Rate: 1.0, Changed: 123456},
	}

	batch := mock.ExpectBatch()
	batch.ExpectExec("INSERT INTO instrument").WithArgs(instruments[0].ID, instruments[0].Title, instruments[0].ShortTitle, instruments[0].Symbol, instruments[0].Rate, instruments[0].Changed).WillReturnResult(pgxmock.NewResult("INSERT", 1))

	err = db.SaveInstruments(context.Background(), instruments)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSaveCountries_Success(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	countries := []models.Country{
		{ID: 1, Title: "USA", Currency: 1, Domain: "us"},
	}

	batch := mock.ExpectBatch()
	batch.ExpectExec("INSERT INTO country").WithArgs(countries[0].ID, countries[0].Title, countries[0].Currency, countries[0].Domain).WillReturnResult(pgxmock.NewResult("INSERT", 1))

	err = db.SaveCountries(context.Background(), countries)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSaveCompanies_Success(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	companies := []models.Company{
		{ID: 1, Title: "Bank", FullTitle: "Bank Inc.", Www: "www.bank.com", Country: 1, Deleted: false, CountryCode: "US", Changed: 123456},
	}

	batch := mock.ExpectBatch()
	batch.ExpectExec("INSERT INTO company").WithArgs(companies[0].ID, companies[0].Title, companies[0].FullTitle, companies[0].Www, companies[0].Country, companies[0].Deleted, companies[0].CountryCode, companies[0].Changed).WillReturnResult(pgxmock.NewResult("INSERT", 1))

	err = db.SaveCompanies(context.Background(), companies)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSaveUsers_Success(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	users := []models.User{
		{ID: 1, Country: 1, Login: "testuser", Email: "test@example.com", Changed: 123456},
	}

	batch := mock.ExpectBatch()
	batch.ExpectExec("INSERT INTO \"user\"").WithArgs(users[0].ID, users[0].Country, users[0].Login, users[0].Parent, users[0].CountryCode, users[0].Email, users[0].Changed, users[0].Currency, users[0].PaidTill, users[0].MonthStartDay, users[0].IsForecastEnabled, users[0].PlanBalanceMode, users[0].PlanSettings, users[0].Subscription, users[0].SubscriptionRenewalDate).WillReturnResult(pgxmock.NewResult("INSERT", 1))

	err = db.SaveUsers(context.Background(), users)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSaveAccounts_Success(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	accounts := []models.Account{
		{ID: "acc-1", User: 1, Title: "Main Account", Type: "checking", Private: true},
	}

	batch := mock.ExpectBatch()
	batch.ExpectExec("INSERT INTO account").WithArgs(accounts[0].ID, accounts[0].User, accounts[0].Instrument, accounts[0].Type, accounts[0].Role, accounts[0].Private, accounts[0].Savings, accounts[0].Title, accounts[0].InBalance, accounts[0].CreditLimit, accounts[0].StartBalance, accounts[0].Balance, accounts[0].Company, accounts[0].Archive, accounts[0].EnableCorrection, accounts[0].BalanceCorrectionType, accounts[0].StartDate, accounts[0].Capitalization, accounts[0].Percent, accounts[0].Changed, accounts[0].SyncID, accounts[0].EnableSMS, accounts[0].EndDateOffset, accounts[0].EndDateOffsetInterval, accounts[0].PayoffStep, accounts[0].PayoffInterval).WillReturnResult(pgxmock.NewResult("INSERT", 1))

	err = db.SaveAccounts(context.Background(), accounts)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSaveTags_Success(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	tags := []models.Tag{
		{ID: "tag-1", User: 1, Title: "Groceries", ShowIncome: false, ShowOutcome: true},
	}

	batch := mock.ExpectBatch()
	batch.ExpectExec("INSERT INTO tag").WithArgs(tags[0].ID, tags[0].User, tags[0].Changed, tags[0].Icon, tags[0].BudgetIncome, tags[0].BudgetOutcome, tags[0].Required, tags[0].Color, tags[0].Picture, tags[0].Title, tags[0].ShowIncome, tags[0].ShowOutcome, tags[0].Parent, tags[0].StaticID).WillReturnResult(pgxmock.NewResult("INSERT", 1))

	err = db.SaveTags(context.Background(), tags)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSaveMerchants_Success(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	merchants := []models.Merchant{
		{ID: "merchant-1", User: 1, Title: "Amazon", Changed: 123456},
	}

	batch := mock.ExpectBatch()
	batch.ExpectExec("INSERT INTO merchant").WithArgs(merchants[0].ID, merchants[0].User, merchants[0].Title, merchants[0].Changed).WillReturnResult(pgxmock.NewResult("INSERT", 1))

	err = db.SaveMerchants(context.Background(), merchants)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSaveBudgets_Success(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	budgets := []models.Budget{
		{User: 1, Changed: 123456, Date: "2024-02-01", Income: 5000.0, Outcome: 2000.0, IncomeLock: false, OutcomeLock: true, IsIncomeForecast: false, IsOutcomeForecast: true},
	}

	batch := mock.ExpectBatch()
	batch.ExpectExec("INSERT INTO budget").WithArgs(budgets[0].User, budgets[0].Changed, budgets[0].Date, budgets[0].Tag, budgets[0].Income, budgets[0].Outcome, budgets[0].IncomeLock, budgets[0].OutcomeLock, budgets[0].IsIncomeForecast, budgets[0].IsOutcomeForecast).WillReturnResult(pgxmock.NewResult("INSERT", 1))

	err = db.SaveBudgets(context.Background(), budgets)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSaveReminders_Success(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	reminders := []models.Reminder{
		{ID: "reminder-1", User: 1, Income: 1000.0, Outcome: 0.0, Changed: 123456, IncomeInstrument: 1, OutcomeInstrument: 2, Step: 1, Points: []int{0}, Tag: []string{"tag-1"}, StartDate: "2024-02-01", EndDate: nil, Notify: true, Interval: nil, IncomeAccount: "acc-1", OutcomeAccount: "acc-2", Comment: "Test Reminder", Payee: nil, Merchant: nil},
	}

	batch := mock.ExpectBatch()
	batch.ExpectExec("INSERT INTO reminder").WithArgs(reminders[0].ID, reminders[0].User, reminders[0].Income, reminders[0].Outcome, reminders[0].Changed, reminders[0].IncomeInstrument, reminders[0].OutcomeInstrument, reminders[0].Step, reminders[0].Points, reminders[0].Tag, reminders[0].StartDate, reminders[0].EndDate, reminders[0].Notify, reminders[0].Interval, reminders[0].IncomeAccount, reminders[0].OutcomeAccount, reminders[0].Comment, reminders[0].Payee, reminders[0].Merchant).WillReturnResult(pgxmock.NewResult("INSERT", 1))

	err = db.SaveReminders(context.Background(), reminders)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSaveReminderMarkers_Success(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	markers := []models.ReminderMarker{
		{ID: "marker-1", User: 1, Date: "2024-02-01", Income: 1000.0, Outcome: 500.0, Changed: 123456, IncomeInstrument: 1, OutcomeInstrument: 2, State: "active", IsForecast: false, Reminder: "reminder-1", IncomeAccount: "acc-1", OutcomeAccount: "acc-2", Comment: "Test Marker", Payee: nil, Merchant: nil, Notify: true, Tag: []string{"tag-1"}},
	}

	batch := mock.ExpectBatch()
	batch.ExpectExec("INSERT INTO reminder_marker").WithArgs(markers[0].ID, markers[0].User, markers[0].Date, markers[0].Income, markers[0].Outcome, markers[0].Changed, markers[0].IncomeInstrument, markers[0].OutcomeInstrument, markers[0].State, markers[0].IsForecast, markers[0].Reminder, markers[0].IncomeAccount, markers[0].OutcomeAccount, markers[0].Comment, markers[0].Payee, markers[0].Merchant, markers[0].Notify, markers[0].Tag).WillReturnResult(pgxmock.NewResult("INSERT", 1))

	err = db.SaveReminderMarkers(context.Background(), markers)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSaveTransactions_Success(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	transactions := []models.Transaction{
		{ID: "txn-1", User: 1, Date: "2024-02-25", Income: 100.0, Outcome: 50.0, Changed: 123456, IncomeInstrument: 1, OutcomeInstrument: 2, Created: 123456789, OriginalPayee: "Store", Deleted: false, Viewed: true, Hold: false, QRCode: nil, Source: "bank", IncomeAccount: "acc-1", OutcomeAccount: ptr("acc-2"), Tag: []string{"tag-1"}, Comment: nil, Payee: "Payee-1", OpIncome: 90.0, OpOutcome: 45.0, OpIncomeInstrument: nil, OpOutcomeInstrument: nil, Latitude: nil, Longitude: nil, Merchant: nil, IncomeBankID: nil, OutcomeBankID: nil, ReminderMarker: nil},
	}

	batch := mock.ExpectBatch()
	batch.ExpectExec("INSERT INTO transaction").WithArgs(transactions[0].ID, transactions[0].User, transactions[0].Date, transactions[0].Income, transactions[0].Outcome, transactions[0].Changed, transactions[0].IncomeInstrument, transactions[0].OutcomeInstrument, transactions[0].Created, transactions[0].OriginalPayee, transactions[0].Deleted, transactions[0].Viewed, transactions[0].Hold, transactions[0].QRCode, transactions[0].Source, transactions[0].IncomeAccount, transactions[0].OutcomeAccount, transactions[0].Tag, transactions[0].Comment, transactions[0].Payee, transactions[0].OpIncome, transactions[0].OpOutcome, transactions[0].OpIncomeInstrument, transactions[0].OpOutcomeInstrument, transactions[0].Latitude, transactions[0].Longitude, transactions[0].Merchant, transactions[0].IncomeBankID, transactions[0].OutcomeBankID, transactions[0].ReminderMarker).WillReturnResult(pgxmock.NewResult("INSERT", 1))

	err = db.SaveTransactions(context.Background(), transactions)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}
