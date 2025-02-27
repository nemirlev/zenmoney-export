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

// GetTag retrieves a specific tag by its ID
func (s *DB) GetTag(ctx context.Context, id string) (*models.Tag, error) {
	query := `
        SELECT id, "user", changed, icon, budget_income, budget_outcome,
               required, color, picture, title, show_income, show_outcome,
               parent, static_id
        FROM tag
        WHERE id = $1`

	tag := &models.Tag{}
	err := s.pool.QueryRow(ctx, query, id).Scan(
		&tag.ID,
		&tag.User,
		&tag.Changed,
		&tag.Icon,
		&tag.BudgetIncome,
		&tag.BudgetOutcome,
		&tag.Required,
		&tag.Color,
		&tag.Picture,
		&tag.Title,
		&tag.ShowIncome,
		&tag.ShowOutcome,
		&tag.Parent,
		&tag.StaticID,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("tag not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get tag: %w", err)
	}

	return tag, nil
}

// ListTags retrieves a list of tags based on the provided filter
func (s *DB) ListTags(ctx context.Context, filter interfaces.Filter) ([]models.Tag, error) {
	var conditions []string
	var args []interface{}
	argNum := 1

	// Build WHERE clause based on filter
	if filter.UserID != nil {
		conditions = append(conditions, fmt.Sprintf(`"user" = $%d`, argNum))
		args = append(args, *filter.UserID)
		argNum++
	}

	query := `
        SELECT id, "user", changed, icon, budget_income, budget_outcome,
               required, color, picture, title, show_income, show_outcome,
               parent, static_id
        FROM tag`

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	// Add pagination
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argNum, argNum+1)
	args = append(args, filter.Limit, (filter.Page-1)*filter.Limit)

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list tags: %w", err)
	}
	defer rows.Close()

	var tags []models.Tag
	for rows.Next() {
		var tag models.Tag
		err := rows.Scan(
			&tag.ID,
			&tag.User,
			&tag.Changed,
			&tag.Icon,
			&tag.BudgetIncome,
			&tag.BudgetOutcome,
			&tag.Required,
			&tag.Color,
			&tag.Picture,
			&tag.Title,
			&tag.ShowIncome,
			&tag.ShowOutcome,
			&tag.Parent,
			&tag.StaticID,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan tag: %w", err)
		}
		tags = append(tags, tag)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating tags: %w", err)
	}

	return tags, nil
}

// CreateTag creates a new tag record
func (s *DB) CreateTag(ctx context.Context, tag *models.Tag) error {
	query := `
        INSERT INTO tag (
            id, "user", changed, icon, budget_income, budget_outcome,
            required, color, picture, title, show_income, show_outcome,
            parent, static_id
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`

	_, err := s.pool.Exec(ctx, query,
		tag.ID,
		tag.User,
		tag.Changed,
		tag.Icon,
		tag.BudgetIncome,
		tag.BudgetOutcome,
		tag.Required,
		tag.Color,
		tag.Picture,
		tag.Title,
		tag.ShowIncome,
		tag.ShowOutcome,
		tag.Parent,
		tag.StaticID,
	)

	if err != nil {
		return fmt.Errorf("failed to create tag: %w", err)
	}

	return nil
}

// UpdateTag updates an existing tag record
func (s *DB) UpdateTag(ctx context.Context, tag *models.Tag) error {
	query := `
        UPDATE tag SET
            "user" = $2,
            changed = $3,
            icon = $4,
            budget_income = $5,
            budget_outcome = $6,
            required = $7,
            color = $8,
            picture = $9,
            title = $10,
            show_income = $11,
            show_outcome = $12,
            parent = $13,
            static_id = $14
        WHERE id = $1`

	commandTag, err := s.pool.Exec(ctx, query,
		tag.ID,
		tag.User,
		tag.Changed,
		tag.Icon,
		tag.BudgetIncome,
		tag.BudgetOutcome,
		tag.Required,
		tag.Color,
		tag.Picture,
		tag.Title,
		tag.ShowIncome,
		tag.ShowOutcome,
		tag.Parent,
		tag.StaticID,
	)

	if err != nil {
		return fmt.Errorf("failed to update tag: %w", err)
	}

	if commandTag.RowsAffected() == 0 {
		return fmt.Errorf("tag not found: %s", tag.ID)
	}

	return nil
}

// DeleteTag deletes a tag by its ID
func (s *DB) DeleteTag(ctx context.Context, id string) error {
	query := `DELETE FROM tag WHERE id = $1`

	commandTag, err := s.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete tag: %w", err)
	}

	if commandTag.RowsAffected() == 0 {
		return fmt.Errorf("tag not found: %s", id)
	}

	return nil
}
