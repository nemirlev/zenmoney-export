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

func TestGetUser_Success(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	expectedUser := &models.User{
		ID:                      1,
		Country:                 1,
		Login:                   "testuser",
		Parent:                  nil,
		CountryCode:             "US",
		Email:                   "testuser@example.com",
		Changed:                 1234567890,
		Currency:                1,
		PaidTill:                1234567890,
		MonthStartDay:           1,
		IsForecastEnabled:       true,
		PlanBalanceMode:         "balance",
		PlanSettings:            "settings",
		Subscription:            "subscription",
		SubscriptionRenewalDate: nil,
	}

	rows := mock.NewRows([]string{
		"id", "country", "login", "parent", "country_code", "email",
		"changed", "currency", "paid_till", "month_start_day",
		"is_forecast_enabled", "plan_balance_mode", "plan_settings",
		"subscription", "subscription_renewal_date",
	}).AddRow(
		expectedUser.ID, expectedUser.Country, expectedUser.Login, expectedUser.Parent, expectedUser.CountryCode,
		expectedUser.Email, expectedUser.Changed, expectedUser.Currency, expectedUser.PaidTill, expectedUser.MonthStartDay,
		expectedUser.IsForecastEnabled, expectedUser.PlanBalanceMode, expectedUser.PlanSettings, expectedUser.Subscription,
		expectedUser.SubscriptionRenewalDate,
	)

	mock.ExpectQuery(`SELECT id, country, login, parent, country_code, email,`).
		WithArgs(expectedUser.ID).
		WillReturnRows(rows)

	result, err := db.GetUser(context.Background(), expectedUser.ID)
	assert.NoError(t, err)
	assert.Equal(t, expectedUser, result)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetUser_NotFound(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	userID := 1

	mock.ExpectQuery(`SELECT id, country, login, parent, country_code, email,`).
		WithArgs(userID).
		WillReturnError(pgx.ErrNoRows)

	result, err := db.GetUser(context.Background(), userID)
	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), fmt.Sprintf("user not found: %d", userID))

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetUser_QueryError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	userID := 1

	mock.ExpectQuery(`SELECT id, country, login, parent, country_code, email,`).
		WithArgs(userID).
		WillReturnError(errors.New("query error"))

	result, err := db.GetUser(context.Background(), userID)
	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get user")

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestListUsers_Success(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	filter := interfaces.Filter{
		UserID: ptr(1),
		Limit:  10,
		Page:   1,
	}

	rows := mock.NewRows([]string{
		"id", "country", "login", "parent", "country_code", "email",
		"changed", "currency", "paid_till", "month_start_day",
		"is_forecast_enabled", "plan_balance_mode", "plan_settings",
		"subscription", "subscription_renewal_date",
	}).AddRow(
		1, 1, "testuser", nil, "US", "testuser@example.com",
		1234567890, 1, 1234567890, 1,
		true, "balance", "settings",
		"subscription", nil,
	)

	mock.ExpectQuery(`SELECT id, country, login, parent, country_code, email,`).
		WithArgs(1, 10, 0).
		WillReturnRows(rows)

	users, err := db.ListUsers(context.Background(), filter)
	assert.NoError(t, err)
	assert.Len(t, users, 1)
	assert.Equal(t, 1, users[0].ID)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestListUsers_QueryError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	filter := interfaces.Filter{
		UserID: ptr(1),
		Limit:  10,
		Page:   1,
	}

	mock.ExpectQuery(`SELECT id, country, login, parent, country_code, email,`).
		WithArgs(1, 10, 0).
		WillReturnError(errors.New("query error"))

	users, err := db.ListUsers(context.Background(), filter)
	assert.Error(t, err)
	assert.Nil(t, users)
	assert.Contains(t, err.Error(), "failed to list users")

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateUser_Success(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	user := &models.User{
		ID:                      1,
		Country:                 1,
		Login:                   "testuser",
		Parent:                  nil,
		CountryCode:             "US",
		Email:                   "testuser@example.com",
		Changed:                 1234567890,
		Currency:                1,
		PaidTill:                1234567890,
		MonthStartDay:           1,
		IsForecastEnabled:       true,
		PlanBalanceMode:         "balance",
		PlanSettings:            "settings",
		Subscription:            "subscription",
		SubscriptionRenewalDate: nil,
	}

	mock.ExpectExec(`INSERT INTO "user"`).
		WithArgs(
			user.ID, user.Country, user.Login, user.Parent, user.CountryCode, user.Email,
			user.Changed, user.Currency, user.PaidTill, user.MonthStartDay,
			user.IsForecastEnabled, user.PlanBalanceMode, user.PlanSettings,
			user.Subscription, user.SubscriptionRenewalDate,
		).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	err = db.CreateUser(context.Background(), user)
	assert.NoError(t, err)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateUser_QueryError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	user := &models.User{
		ID:                      1,
		Country:                 1,
		Login:                   "testuser",
		Parent:                  nil,
		CountryCode:             "US",
		Email:                   "testuser@example.com",
		Changed:                 1234567890,
		Currency:                1,
		PaidTill:                1234567890,
		MonthStartDay:           1,
		IsForecastEnabled:       true,
		PlanBalanceMode:         "balance",
		PlanSettings:            "settings",
		Subscription:            "subscription",
		SubscriptionRenewalDate: nil,
	}

	mock.ExpectExec(`INSERT INTO "user"`).
		WithArgs(
			user.ID, user.Country, user.Login, user.Parent, user.CountryCode, user.Email,
			user.Changed, user.Currency, user.PaidTill, user.MonthStartDay,
			user.IsForecastEnabled, user.PlanBalanceMode, user.PlanSettings,
			user.Subscription, user.SubscriptionRenewalDate,
		).
		WillReturnError(errors.New("insert error"))

	err = db.CreateUser(context.Background(), user)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create user")

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateUser_Success(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	user := &models.User{
		ID:                      1,
		Country:                 1,
		Login:                   "updateduser",
		Parent:                  nil,
		CountryCode:             "US",
		Email:                   "updateduser@example.com",
		Changed:                 1234567890,
		Currency:                1,
		PaidTill:                1234567890,
		MonthStartDay:           1,
		IsForecastEnabled:       true,
		PlanBalanceMode:         "balance",
		PlanSettings:            "settings",
		Subscription:            "subscription",
		SubscriptionRenewalDate: nil,
	}

	mock.ExpectExec(`UPDATE "user" SET`).
		WithArgs(
			user.ID, user.Country, user.Login, user.Parent, user.CountryCode, user.Email,
			user.Changed, user.Currency, user.PaidTill, user.MonthStartDay,
			user.IsForecastEnabled, user.PlanBalanceMode, user.PlanSettings,
			user.Subscription, user.SubscriptionRenewalDate,
		).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	err = db.UpdateUser(context.Background(), user)
	assert.NoError(t, err)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateUser_NotFound(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	user := &models.User{
		ID:                      1,
		Country:                 1,
		Login:                   "updateduser",
		Parent:                  nil,
		CountryCode:             "US",
		Email:                   "updateduser@example.com",
		Changed:                 1234567890,
		Currency:                1,
		PaidTill:                1234567890,
		MonthStartDay:           1,
		IsForecastEnabled:       true,
		PlanBalanceMode:         "balance",
		PlanSettings:            "settings",
		Subscription:            "subscription",
		SubscriptionRenewalDate: nil,
	}

	mock.ExpectExec(`UPDATE "user" SET`).
		WithArgs(
			user.ID, user.Country, user.Login, user.Parent, user.CountryCode, user.Email,
			user.Changed, user.Currency, user.PaidTill, user.MonthStartDay,
			user.IsForecastEnabled, user.PlanBalanceMode, user.PlanSettings,
			user.Subscription, user.SubscriptionRenewalDate,
		).
		WillReturnResult(pgxmock.NewResult("UPDATE", 0))

	err = db.UpdateUser(context.Background(), user)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "user not found")

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateUser_QueryError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	user := &models.User{
		ID:                      1,
		Country:                 1,
		Login:                   "updateduser",
		Parent:                  nil,
		CountryCode:             "US",
		Email:                   "updateduser@example.com",
		Changed:                 1234567890,
		Currency:                1,
		PaidTill:                1234567890,
		MonthStartDay:           1,
		IsForecastEnabled:       true,
		PlanBalanceMode:         "balance",
		PlanSettings:            "settings",
		Subscription:            "subscription",
		SubscriptionRenewalDate: nil,
	}

	mock.ExpectExec(`UPDATE "user" SET`).
		WithArgs(
			user.ID, user.Country, user.Login, user.Parent, user.CountryCode, user.Email,
			user.Changed, user.Currency, user.PaidTill, user.MonthStartDay,
			user.IsForecastEnabled, user.PlanBalanceMode, user.PlanSettings,
			user.Subscription, user.SubscriptionRenewalDate,
		).
		WillReturnError(errors.New("update error"))

	err = db.UpdateUser(context.Background(), user)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to update user")

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDeleteUser_Success(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	userID := 1

	mock.ExpectExec(`DELETE FROM "user" WHERE id = \$1`).
		WithArgs(userID).
		WillReturnResult(pgxmock.NewResult("DELETE", 1))

	err = db.DeleteUser(context.Background(), userID)
	assert.NoError(t, err)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDeleteUser_NotFound(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	userID := 1

	mock.ExpectExec(`DELETE FROM "user" WHERE id = \$1`).
		WithArgs(userID).
		WillReturnResult(pgxmock.NewResult("DELETE", 0))

	err = db.DeleteUser(context.Background(), userID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "user not found")

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDeleteUser_QueryError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	userID := 1

	mock.ExpectExec(`DELETE FROM "user" WHERE id = \$1`).
		WithArgs(userID).
		WillReturnError(errors.New("delete error"))

	err = db.DeleteUser(context.Background(), userID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to delete user")

	assert.NoError(t, mock.ExpectationsWereMet())
}
