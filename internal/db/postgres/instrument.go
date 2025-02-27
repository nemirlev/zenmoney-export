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

// GetInstrument retrieves a specific instrument by its ID
func (s *DB) GetInstrument(ctx context.Context, id int) (*models.Instrument, error) {
	query := `
        SELECT id, title, short_title, symbol, rate, changed
        FROM instrument
        WHERE id = $1`

	instrument := &models.Instrument{}
	err := s.pool.QueryRow(ctx, query, id).Scan(
		&instrument.ID,
		&instrument.Title,
		&instrument.ShortTitle,
		&instrument.Symbol,
		&instrument.Rate,
		&instrument.Changed,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("instrument not found: %d", id)
		}
		return nil, fmt.Errorf("failed to get instrument: %w", err)
	}

	return instrument, nil
}

// ListInstruments retrieves a list of instruments based on the provided filter
func (s *DB) ListInstruments(ctx context.Context, filter interfaces.Filter) ([]models.Instrument, error) {
	var conditions []string
	var args []interface{}
	argNum := 1

	// Build the WHERE clause based on filter
	if filter.UserID != nil {
		conditions = append(conditions, fmt.Sprintf("user_id = $%d", argNum))
		args = append(args, *filter.UserID)
		argNum++
	}

	query := "SELECT id, title, short_title, symbol, rate, changed FROM instrument"
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	// Add pagination
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argNum, argNum+1)
	args = append(args, filter.Limit, (filter.Page-1)*filter.Limit)

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list instruments: %w", err)
	}
	defer rows.Close()

	var instruments []models.Instrument
	for rows.Next() {
		var instrument models.Instrument
		err := rows.Scan(
			&instrument.ID,
			&instrument.Title,
			&instrument.ShortTitle,
			&instrument.Symbol,
			&instrument.Rate,
			&instrument.Changed,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan instrument: %w", err)
		}
		instruments = append(instruments, instrument)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating instruments: %w", err)
	}

	return instruments, nil
}

// CreateInstrument creates a new instrument record
func (s *DB) CreateInstrument(ctx context.Context, instrument *models.Instrument) error {
	query := `
        INSERT INTO instrument (id, title, short_title, symbol, rate, changed)
        VALUES ($1, $2, $3, $4, $5, $6)`

	_, err := s.pool.Exec(ctx, query,
		instrument.ID,
		instrument.Title,
		instrument.ShortTitle,
		instrument.Symbol,
		instrument.Rate,
		instrument.Changed,
	)
	if err != nil {
		return fmt.Errorf("failed to create instrument: %w", err)
	}

	return nil
}

// UpdateInstrument updates an existing instrument record
func (s *DB) UpdateInstrument(ctx context.Context, instrument *models.Instrument) error {
	query := `
        UPDATE instrument
        SET title = $2, short_title = $3, symbol = $4, rate = $5, changed = $6
        WHERE id = $1`

	commandTag, err := s.pool.Exec(ctx, query,
		instrument.ID,
		instrument.Title,
		instrument.ShortTitle,
		instrument.Symbol,
		instrument.Rate,
		instrument.Changed,
	)
	if err != nil {
		return fmt.Errorf("failed to update instrument: %w", err)
	}

	if commandTag.RowsAffected() == 0 {
		return fmt.Errorf("instrument not found: %d", instrument.ID)
	}

	return nil
}

// DeleteInstrument deletes an instrument by its ID
func (s *DB) DeleteInstrument(ctx context.Context, id int) error {
	query := `DELETE FROM instrument WHERE id = $1`

	commandTag, err := s.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete instrument: %w", err)
	}

	if commandTag.RowsAffected() == 0 {
		return fmt.Errorf("instrument not found: %d", id)
	}

	return nil
}
