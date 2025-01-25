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

// GetMerchant retrieves a specific merchant by its ID
func (s *DB) GetMerchant(ctx context.Context, id string) (*models.Merchant, error) {
	query := `
       SELECT id, "user", title, changed
       FROM merchant
       WHERE id = $1`

	merchant := &models.Merchant{}
	err := s.pool.QueryRow(ctx, query, id).Scan(
		&merchant.ID,
		&merchant.User,
		&merchant.Title,
		&merchant.Changed,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("merchant not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get merchant: %w", err)
	}

	return merchant, nil
}

// ListMerchants retrieves a list of merchants based on the provided filter
func (s *DB) ListMerchants(ctx context.Context, filter interfaces.Filter) ([]models.Merchant, error) {
	var conditions []string
	var args []interface{}
	argNum := 1

	if filter.UserID != nil {
		conditions = append(conditions, fmt.Sprintf(`"user" = $%d`, argNum))
		args = append(args, *filter.UserID)
		argNum++
	}

	query := `
       SELECT id, "user", title, changed
       FROM merchant`

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argNum, argNum+1)
	args = append(args, filter.Limit, (filter.Page-1)*filter.Limit)

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list merchants: %w", err)
	}
	defer rows.Close()

	var merchants []models.Merchant
	for rows.Next() {
		var merchant models.Merchant
		err := rows.Scan(
			&merchant.ID,
			&merchant.User,
			&merchant.Title,
			&merchant.Changed,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan merchant: %w", err)
		}
		merchants = append(merchants, merchant)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating merchants: %w", err)
	}

	return merchants, nil
}

// CreateMerchant creates a new merchant record
func (s *DB) CreateMerchant(ctx context.Context, merchant *models.Merchant) error {
	query := `
       INSERT INTO merchant (id, "user", title, changed)
       VALUES ($1, $2, $3, $4)`

	_, err := s.pool.Exec(ctx, query,
		merchant.ID,
		merchant.User,
		merchant.Title,
		merchant.Changed,
	)

	if err != nil {
		return fmt.Errorf("failed to create merchant: %w", err)
	}

	return nil
}

// UpdateMerchant updates an existing merchant record
func (s *DB) UpdateMerchant(ctx context.Context, merchant *models.Merchant) error {
	query := `
       UPDATE merchant 
       SET "user" = $2, title = $3, changed = $4
       WHERE id = $1`

	commandTag, err := s.pool.Exec(ctx, query,
		merchant.ID,
		merchant.User,
		merchant.Title,
		merchant.Changed,
	)

	if err != nil {
		return fmt.Errorf("failed to update merchant: %w", err)
	}

	if commandTag.RowsAffected() == 0 {
		return fmt.Errorf("merchant not found: %s", merchant.ID)
	}

	return nil
}

// DeleteMerchant deletes a merchant by its ID
func (s *DB) DeleteMerchant(ctx context.Context, id string) error {
	query := `DELETE FROM merchant WHERE id = $1`

	commandTag, err := s.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete merchant: %w", err)
	}

	if commandTag.RowsAffected() == 0 {
		return fmt.Errorf("merchant not found: %s", id)
	}

	return nil
}
