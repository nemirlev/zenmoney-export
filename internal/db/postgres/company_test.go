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

// TestGetCompany_Success tests the successful retrieval of a company
func TestGetCompany_Success(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create mock pool: %v", err)
	}
	defer mock.Close()

	db := &DB{pool: mock}

	companyID := 1
	expectedCompany := &models.Company{
		ID:          companyID,
		Title:       "Test Company",
		FullTitle:   "Test Company Full Title",
		Www:         "https://testcompany.com",
		Country:     1,
		Deleted:     false,
		CountryCode: "TC",
		Changed:     1234567890,
	}

	rows := mock.NewRows([]string{
		"id", "title", "full_title", "www", "country", "deleted", "country_code", "changed",
	}).AddRow(
		expectedCompany.ID, expectedCompany.Title, expectedCompany.FullTitle, expectedCompany.Www,
		expectedCompany.Country, expectedCompany.Deleted, expectedCompany.CountryCode, expectedCompany.Changed,
	)

	mock.ExpectQuery(`SELECT id, title, full_title, www, country, deleted, country_code, changed FROM company WHERE id = \$1`).
		WithArgs(companyID).
		WillReturnRows(rows)

	result, err := db.GetCompany(context.Background(), companyID)
	assert.NoError(t, err)
	assert.Equal(t, expectedCompany, result)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations were not met: %v", err)
	}
}

// TestGetCompany_NotFound tests the case when the company is not found
func TestGetCompany_NotFound(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create mock pool: %v", err)
	}
	defer mock.Close()

	db := &DB{pool: mock}

	companyID := 999

	mock.ExpectQuery(`SELECT id, title, full_title, www, country, deleted, country_code, changed FROM company WHERE id = \$1`).
		WithArgs(companyID).
		WillReturnError(pgx.ErrNoRows)

	result, err := db.GetCompany(context.Background(), companyID)
	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "company not found")

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations were not met: %v", err)
	}
}

// TestListCompanies_Success tests the successful retrieval of a list of companies
func TestListCompanies_Success(t *testing.T) {
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
		"id", "title", "full_title", "www", "country", "deleted", "country_code", "changed",
	}).AddRow(
		1, "Test Company", "Test Company Full Title", "https://testcompany.com", 1, false, "TC", 1234567890,
	)

	mock.ExpectQuery(`SELECT id, title, full_title, www, country, deleted, country_code, changed FROM company WHERE user_id = \$1 LIMIT \$2 OFFSET \$3`).
		WithArgs(1, 10, 0).
		WillReturnRows(rows)

	companies, err := db.ListCompanies(context.Background(), filter)
	assert.NoError(t, err)
	assert.Len(t, companies, 1)
	assert.Equal(t, "Test Company", companies[0].Title)

	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestListCompanies_QueryError tests the case when there is a query error
func TestListCompanies_QueryError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	filter := interfaces.Filter{
		UserID: ptr(1),
		Limit:  10,
		Page:   1,
	}

	mock.ExpectQuery(`SELECT id, title, full_title, www, country, deleted, country_code, changed FROM company WHERE user_id = \$1 LIMIT \$2 OFFSET \$3`).
		WithArgs(1, 10, 0).
		WillReturnError(errors.New("query error"))

	companies, err := db.ListCompanies(context.Background(), filter)
	assert.Error(t, err)
	assert.Nil(t, companies)
	assert.Contains(t, err.Error(), "failed to list companies")

	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestCreateCompany_Success tests the successful creation of a company
func TestCreateCompany_Success(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	company := &models.Company{
		ID:          1,
		Title:       "Test Company",
		FullTitle:   "Test Company Full Title",
		Www:         "https://testcompany.com",
		Country:     1,
		Deleted:     false,
		CountryCode: "TC",
		Changed:     1234567890,
	}

	mock.ExpectExec(`INSERT INTO company`).
		WithArgs(
			company.ID, company.Title, company.FullTitle, company.Www,
			company.Country, company.Deleted, company.CountryCode, company.Changed,
		).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	err = db.CreateCompany(context.Background(), company)
	assert.NoError(t, err)

	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestCreateCompany_QueryError tests the case when there is a query error
func TestCreateCompany_QueryError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	company := &models.Company{
		ID:          1,
		Title:       "Test Company",
		FullTitle:   "Test Company Full Title",
		Www:         "https://testcompany.com",
		Country:     1,
		Deleted:     false,
		CountryCode: "TC",
		Changed:     1234567890,
	}

	mock.ExpectExec(`INSERT INTO company`).
		WithArgs(
			company.ID, company.Title, company.FullTitle, company.Www,
			company.Country, company.Deleted, company.CountryCode, company.Changed,
		).
		WillReturnError(errors.New("insert error"))

	err = db.CreateCompany(context.Background(), company)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create company")

	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestUpdateCompany_Success tests the successful update of a company
func TestUpdateCompany_Success(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	company := &models.Company{
		ID:          1,
		Title:       "Updated Company",
		FullTitle:   "Updated Company Full Title",
		Www:         "https://updatedcompany.com",
		Country:     1,
		Deleted:     false,
		CountryCode: "UC",
		Changed:     1234567891,
	}

	mock.ExpectExec(`UPDATE company SET`).
		WithArgs(
			company.ID, company.Title, company.FullTitle, company.Www,
			company.Country, company.Deleted, company.CountryCode, company.Changed,
		).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	err = db.UpdateCompany(context.Background(), company)
	assert.NoError(t, err)

	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestUpdateCompany_NotFound tests the case when the company is not found
func TestUpdateCompany_NotFound(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	company := &models.Company{
		ID:          1,
		Title:       "Updated Company",
		FullTitle:   "Updated Company Full Title",
		Www:         "https://updatedcompany.com",
		Country:     1,
		Deleted:     false,
		CountryCode: "UC",
		Changed:     1234567891,
	}

	mock.ExpectExec(`UPDATE company SET`).
		WithArgs(
			company.ID, company.Title, company.FullTitle, company.Www,
			company.Country, company.Deleted, company.CountryCode, company.Changed,
		).
		WillReturnResult(pgxmock.NewResult("UPDATE", 0))

	err = db.UpdateCompany(context.Background(), company)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "company not found")

	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestDeleteCompany_Success tests the successful deletion of a company
func TestDeleteCompany_Success(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	companyID := 1

	mock.ExpectExec(`DELETE FROM company WHERE id = \$1`).
		WithArgs(companyID).
		WillReturnResult(pgxmock.NewResult("DELETE", 1))

	err = db.DeleteCompany(context.Background(), companyID)
	assert.NoError(t, err)

	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestDeleteCompany_NotFound tests the case when the company is not found
func TestDeleteCompany_NotFound(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	companyID := 999

	mock.ExpectExec(`DELETE FROM company WHERE id = \$1`).
		WithArgs(companyID).
		WillReturnResult(pgxmock.NewResult("DELETE", 0))

	err = db.DeleteCompany(context.Background(), companyID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "company not found")

	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestDeleteCompany_QueryError tests the case when there is a query error
func TestDeleteCompany_QueryError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	companyID := 1

	mock.ExpectExec(`DELETE FROM company WHERE id = \$1`).
		WithArgs(companyID).
		WillReturnError(errors.New("delete error"))

	err = db.DeleteCompany(context.Background(), companyID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to delete company")

	assert.NoError(t, mock.ExpectationsWereMet())
}
