package postgres

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/nemirlev/zenmoney-export/internal/interfaces"
	"github.com/nemirlev/zenmoney-go-sdk/v2/models"
)

// GetCountry retrieves a specific country by its ID
func (s *DB) GetCountry(ctx context.Context, id int) (*models.Country, error) {
	query := `
        SELECT id, title, currency, domain
        FROM country
        WHERE id = $1`

	country := &models.Country{}
	err := s.pool.QueryRow(ctx, query, id).Scan(
		&country.ID,
		&country.Title,
		&country.Currency,
		&country.Domain,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("country not found: %d", id)
		}
		return nil, fmt.Errorf("failed to get country: %w", err)
	}

	return country, nil
}

// ListCountries retrieves a list of countries based on the provided filter
func (s *DB) ListCountries(ctx context.Context, filter interfaces.Filter) ([]models.Country, error) {
	query := `
        SELECT id, title, currency, domain
        FROM country
        LIMIT $1 OFFSET $2`

	rows, err := s.pool.Query(ctx, query, filter.Limit, (filter.Page-1)*filter.Limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list countries: %w", err)
	}
	defer rows.Close()

	var countries []models.Country
	for rows.Next() {
		var country models.Country
		err := rows.Scan(
			&country.ID,
			&country.Title,
			&country.Currency,
			&country.Domain,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan country: %w", err)
		}
		countries = append(countries, country)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating countries: %w", err)
	}

	return countries, nil
}

// CreateCountry creates a new country record
func (s *DB) CreateCountry(ctx context.Context, country *models.Country) error {
	query := `
        INSERT INTO country (id, title, currency, domain)
        VALUES ($1, $2, $3, $4)`

	_, err := s.pool.Exec(ctx, query,
		country.ID,
		country.Title,
		country.Currency,
		country.Domain,
	)
	if err != nil {
		return fmt.Errorf("failed to create country: %w", err)
	}

	return nil
}

// UpdateCountry updates an existing country record
func (s *DB) UpdateCountry(ctx context.Context, country *models.Country) error {
	query := `
        UPDATE country
        SET title = $2, currency = $3, domain = $4
        WHERE id = $1`

	commandTag, err := s.pool.Exec(ctx, query,
		country.ID,
		country.Title,
		country.Currency,
		country.Domain,
	)
	if err != nil {
		return fmt.Errorf("failed to update country: %w", err)
	}

	if commandTag.RowsAffected() == 0 {
		return fmt.Errorf("country not found: %d", country.ID)
	}

	return nil
}

// DeleteCountry deletes a country by its ID
func (s *DB) DeleteCountry(ctx context.Context, id int) error {
	query := `DELETE FROM country WHERE id = $1`

	commandTag, err := s.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete country: %w", err)
	}

	if commandTag.RowsAffected() == 0 {
		return fmt.Errorf("country not found: %d", id)
	}

	return nil
}
