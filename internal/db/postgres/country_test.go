package postgres

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/nemirlev/zenmoney-export/v2/internal/interfaces"
	"github.com/nemirlev/zenmoney-go-sdk/v2/models"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"testing"
)

// TestGetCountry_Success tests the successful retrieval of a country
func TestGetCountry_Success(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	countryID := 1
	expectedCountry := &models.Country{
		ID:       countryID,
		Title:    "Test Country",
		Currency: 1,
		Domain:   "test",
	}

	rows := mock.NewRows([]string{"id", "title", "currency", "domain"}).
		AddRow(expectedCountry.ID, expectedCountry.Title, expectedCountry.Currency, expectedCountry.Domain)

	mock.ExpectQuery(`SELECT id, title, currency, domain FROM country WHERE id = \$1`).
		WithArgs(countryID).
		WillReturnRows(rows)

	result, err := db.GetCountry(context.Background(), countryID)
	assert.NoError(t, err)
	assert.Equal(t, expectedCountry, result)

	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestGetCountry_NotFound tests the case when the country is not found
func TestGetCountry_NotFound(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	countryID := 999

	mock.ExpectQuery(`SELECT id, title, currency, domain FROM country WHERE id = \$1`).
		WithArgs(countryID).
		WillReturnError(pgx.ErrNoRows)

	result, err := db.GetCountry(context.Background(), countryID)
	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "country not found")

	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestGetCountry_QueryError tests the case when there is a query error
func TestGetCountry_QueryError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	countryID := 1

	mock.ExpectQuery(`SELECT id, title, currency, domain FROM country WHERE id = \$1`).
		WithArgs(countryID).
		WillReturnError(errors.New("query error"))

	result, err := db.GetCountry(context.Background(), countryID)
	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get country")

	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestListCountries_Success tests the successful retrieval of a list of countries
func TestListCountries_Success(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	filter := interfaces.Filter{
		Limit: 10,
		Page:  1,
	}

	rows := mock.NewRows([]string{"id", "title", "currency", "domain"}).
		AddRow(1, "Country 1", 1, "domain1").
		AddRow(2, "Country 2", 2, "domain2")

	mock.ExpectQuery(`SELECT id, title, currency, domain FROM country LIMIT \$1 OFFSET \$2`).
		WithArgs(10, 0).
		WillReturnRows(rows)

	countries, err := db.ListCountries(context.Background(), filter)
	assert.NoError(t, err)
	assert.Len(t, countries, 2)
	assert.Equal(t, "Country 1", countries[0].Title)
	assert.Equal(t, "Country 2", countries[1].Title)

	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestListCountries_QueryError tests the case when there is a query error
func TestListCountries_QueryError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	filter := interfaces.Filter{
		Limit: 10,
		Page:  1,
	}

	mock.ExpectQuery(`SELECT id, title, currency, domain FROM country LIMIT \$1 OFFSET \$2`).
		WithArgs(10, 0).
		WillReturnError(errors.New("query error"))

	countries, err := db.ListCountries(context.Background(), filter)
	assert.Error(t, err)
	assert.Nil(t, countries)
	assert.Contains(t, err.Error(), "failed to list countries")

	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestUpdateCountry_Success tests the successful update of a country
func TestUpdateCountry_Success(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	country := &models.Country{
		ID:       1,
		Title:    "Updated Country",
		Currency: 2,
		Domain:   "updated",
	}

	mock.ExpectExec(`UPDATE country SET title = \$2, currency = \$3, domain = \$4 WHERE id = \$1`).
		WithArgs(
			country.ID, country.Title, country.Currency, country.Domain,
		).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	err = db.UpdateCountry(context.Background(), country)
	assert.NoError(t, err)

	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestUpdateCountry_NotFound tests the case when the country is not found
func TestUpdateCountry_NotFound(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	country := &models.Country{
		ID:       999,
		Title:    "Non-existing Country",
		Currency: 2,
		Domain:   "nonexisting",
	}

	mock.ExpectExec(`UPDATE country SET title = \$2, currency = \$3, domain = \$4 WHERE id = \$1`).
		WithArgs(
			country.ID, country.Title, country.Currency, country.Domain,
		).
		WillReturnResult(pgxmock.NewResult("UPDATE", 0))

	err = db.UpdateCountry(context.Background(), country)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "country not found")

	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestUpdateCountry_QueryError tests the case when there is a query error
func TestUpdateCountry_QueryError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	country := &models.Country{
		ID:       1,
		Title:    "Updated Country",
		Currency: 2,
		Domain:   "updated",
	}

	mock.ExpectExec(`UPDATE country SET title = \$2, currency = \$3, domain = \$4 WHERE id = \$1`).
		WithArgs(
			country.ID, country.Title, country.Currency, country.Domain,
		).
		WillReturnError(errors.New("update error"))

	err = db.UpdateCountry(context.Background(), country)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to update country")

	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestCreateCountry_Success tests the successful creation of a country
func TestCreateCountry_Success(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	country := &models.Country{
		ID:       1,
		Title:    "New Country",
		Currency: 1,
		Domain:   "new",
	}

	mock.ExpectExec(`INSERT INTO country \(id, title, currency, domain\) VALUES \(\$1, \$2, \$3, \$4\)`).
		WithArgs(country.ID, country.Title, country.Currency, country.Domain).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	err = db.CreateCountry(context.Background(), country)
	assert.NoError(t, err)

	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestCreateCountry_QueryError tests the case when there is a query error
func TestCreateCountry_QueryError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	country := &models.Country{
		ID:       1,
		Title:    "New Country",
		Currency: 1,
		Domain:   "new",
	}

	mock.ExpectExec(`INSERT INTO country \(id, title, currency, domain\) VALUES \(\$1, \$2, \$3, \$4\)`).
		WithArgs(country.ID, country.Title, country.Currency, country.Domain).
		WillReturnError(errors.New("insert error"))

	err = db.CreateCountry(context.Background(), country)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create country")

	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestDeleteCountry_Success tests the successful deletion of a country
func TestDeleteCountry_Success(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	countryID := 1

	mock.ExpectExec(`DELETE FROM country WHERE id = \$1`).
		WithArgs(countryID).
		WillReturnResult(pgxmock.NewResult("DELETE", 1))

	err = db.DeleteCountry(context.Background(), countryID)
	assert.NoError(t, err)

	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestDeleteCountry_NotFound tests the case when the country is not found
func TestDeleteCountry_NotFound(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	countryID := 999

	mock.ExpectExec(`DELETE FROM country WHERE id = \$1`).
		WithArgs(countryID).
		WillReturnResult(pgxmock.NewResult("DELETE", 0))

	err = db.DeleteCountry(context.Background(), countryID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "country not found")

	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestDeleteCountry_QueryError tests the case when there is a query error
func TestDeleteCountry_QueryError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	countryID := 1

	mock.ExpectExec(`DELETE FROM country WHERE id = \$1`).
		WithArgs(countryID).
		WillReturnError(errors.New("delete error"))

	err = db.DeleteCountry(context.Background(), countryID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to delete country")

	assert.NoError(t, mock.ExpectationsWereMet())
}
