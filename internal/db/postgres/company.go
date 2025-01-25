package postgres

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/nemirlev/zenmoney-export/v2/internal/interfaces"
	"github.com/nemirlev/zenmoney-go-sdk/v2/models"
	"strings"
)

// GetCompany retrieves a specific company by its ID
func (s *DB) GetCompany(ctx context.Context, id int) (*models.Company, error) {
	query := `
        SELECT id, title, full_title, www, country, deleted, country_code, changed
        FROM company
        WHERE id = $1`

	company := &models.Company{}
	err := s.pool.QueryRow(ctx, query, id).Scan(
		&company.ID,
		&company.Title,
		&company.FullTitle,
		&company.Www,
		&company.Country,
		&company.Deleted,
		&company.CountryCode,
		&company.Changed,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("company not found: %d", id)
		}
		return nil, fmt.Errorf("failed to get company: %w", err)
	}

	return company, nil
}

// ListCompanies retrieves a list of companies based on the provided filter
func (s *DB) ListCompanies(ctx context.Context, filter interfaces.Filter) ([]models.Company, error) {
	var conditions []string
	var args []interface{}
	argNum := 1

	// Build the WHERE clause based on filter
	if filter.UserID != nil {
		conditions = append(conditions, fmt.Sprintf("user_id = $%d", argNum))
		args = append(args, *filter.UserID)
		argNum++
	}

	query := `
        SELECT id, title, full_title, www, country, deleted, country_code, changed
        FROM company`

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	// Add pagination
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argNum, argNum+1)
	args = append(args, filter.Limit, (filter.Page-1)*filter.Limit)

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list companies: %w", err)
	}
	defer rows.Close()

	var companies []models.Company
	for rows.Next() {
		var company models.Company
		err := rows.Scan(
			&company.ID,
			&company.Title,
			&company.FullTitle,
			&company.Www,
			&company.Country,
			&company.Deleted,
			&company.CountryCode,
			&company.Changed,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan company: %w", err)
		}
		companies = append(companies, company)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating companies: %w", err)
	}

	return companies, nil
}

// CreateCompany creates a new company record
func (s *DB) CreateCompany(ctx context.Context, company *models.Company) error {
	query := `
        INSERT INTO company (
            id, title, full_title, www, country, deleted,
            country_code, changed
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	_, err := s.pool.Exec(ctx, query,
		company.ID,
		company.Title,
		company.FullTitle,
		company.Www,
		company.Country,
		company.Deleted,
		company.CountryCode,
		company.Changed,
	)
	if err != nil {
		return fmt.Errorf("failed to create company: %w", err)
	}

	return nil
}

// UpdateCompany updates an existing company record
func (s *DB) UpdateCompany(ctx context.Context, company *models.Company) error {
	query := `
        UPDATE company
        SET title = $2, full_title = $3, www = $4, country = $5,
            deleted = $6, country_code = $7, changed = $8
        WHERE id = $1`

	commandTag, err := s.pool.Exec(ctx, query,
		company.ID,
		company.Title,
		company.FullTitle,
		company.Www,
		company.Country,
		company.Deleted,
		company.CountryCode,
		company.Changed,
	)
	if err != nil {
		return fmt.Errorf("failed to update company: %w", err)
	}

	if commandTag.RowsAffected() == 0 {
		return fmt.Errorf("company not found: %d", company.ID)
	}

	return nil
}

// DeleteCompany deletes a company by its ID
func (s *DB) DeleteCompany(ctx context.Context, id int) error {
	query := `DELETE FROM company WHERE id = $1`

	commandTag, err := s.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete company: %w", err)
	}

	if commandTag.RowsAffected() == 0 {
		return fmt.Errorf("company not found: %d", id)
	}

	return nil
}
