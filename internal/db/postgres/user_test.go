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

func TestGetUser_Success(t *testing.T) {
	db, mock := newTestDB(t)

	expectedUser := testUser("testuser", "testuser@example.com")

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
}

func TestGetUser_NotFound(t *testing.T) {
	db, mock := newTestDB(t)

	userID := 1

	mock.ExpectQuery(`SELECT id, country, login, parent, country_code, email,`).
		WithArgs(userID).
		WillReturnError(pgx.ErrNoRows)

	result, err := db.GetUser(context.Background(), userID)
	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), fmt.Sprintf("user not found: %d", userID))
}

func TestGetUser_QueryError(t *testing.T) {
	db, mock := newTestDB(t)

	userID := 1

	mock.ExpectQuery(`SELECT id, country, login, parent, country_code, email,`).
		WithArgs(userID).
		WillReturnError(errors.New("query error"))

	result, err := db.GetUser(context.Background(), userID)
	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get user")
}

func TestListUsers_Success(t *testing.T) {
	db, mock := newTestDB(t)

	filter := testPageFilter()

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
}

func TestListUsers_QueryError(t *testing.T) {
	db, mock := newTestDB(t)

	filter := testPageFilter()

	mock.ExpectQuery(`SELECT id, country, login, parent, country_code, email,`).
		WithArgs(1, 10, 0).
		WillReturnError(errors.New("query error"))

	users, err := db.ListUsers(context.Background(), filter)
	assert.Error(t, err)
	assert.Nil(t, users)
	assert.Contains(t, err.Error(), "failed to list users")
}

func TestCreateUser_Success(t *testing.T) {
	db, mock := newTestDB(t)

	user := testUser("testuser", "testuser@example.com")

	mock.ExpectExec(`INSERT INTO "user"`).
		WithArgs(userArgs(user)...).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	err := db.CreateUser(context.Background(), user)
	assert.NoError(t, err)
}

func TestCreateUser_QueryError(t *testing.T) {
	db, mock := newTestDB(t)

	user := testUser("testuser", "testuser@example.com")

	mock.ExpectExec(`INSERT INTO "user"`).
		WithArgs(userArgs(user)...).
		WillReturnError(errors.New("insert error"))

	err := db.CreateUser(context.Background(), user)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create user")
}

func TestUpdateUser_Success(t *testing.T) {
	db, mock := newTestDB(t)

	user := testUser("updateduser", "updateduser@example.com")

	mock.ExpectExec(`UPDATE "user" SET`).
		WithArgs(userArgs(user)...).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	err := db.UpdateUser(context.Background(), user)
	assert.NoError(t, err)
}

func TestUpdateUser_NotFound(t *testing.T) {
	db, mock := newTestDB(t)

	user := testUser("updateduser", "updateduser@example.com")

	mock.ExpectExec(`UPDATE "user" SET`).
		WithArgs(userArgs(user)...).
		WillReturnResult(pgxmock.NewResult("UPDATE", 0))

	err := db.UpdateUser(context.Background(), user)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "user not found")
}

func TestUpdateUser_QueryError(t *testing.T) {
	db, mock := newTestDB(t)

	user := testUser("updateduser", "updateduser@example.com")

	mock.ExpectExec(`UPDATE "user" SET`).
		WithArgs(userArgs(user)...).
		WillReturnError(errors.New("update error"))

	err := db.UpdateUser(context.Background(), user)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to update user")
}

func TestDeleteUser_Success(t *testing.T) {
	db, mock := newTestDB(t)

	userID := 1

	mock.ExpectExec(`DELETE FROM "user" WHERE id = \$1`).
		WithArgs(userID).
		WillReturnResult(pgxmock.NewResult("DELETE", 1))

	err := db.DeleteUser(context.Background(), userID)
	assert.NoError(t, err)
}

func TestDeleteUser_NotFound(t *testing.T) {
	db, mock := newTestDB(t)

	userID := 1

	mock.ExpectExec(`DELETE FROM "user" WHERE id = \$1`).
		WithArgs(userID).
		WillReturnResult(pgxmock.NewResult("DELETE", 0))

	err := db.DeleteUser(context.Background(), userID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "user not found")
}

func TestDeleteUser_QueryError(t *testing.T) {
	db, mock := newTestDB(t)

	userID := 1

	mock.ExpectExec(`DELETE FROM "user" WHERE id = \$1`).
		WithArgs(userID).
		WillReturnError(errors.New("delete error"))

	err := db.DeleteUser(context.Background(), userID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to delete user")
}
