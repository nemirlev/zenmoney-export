package postgres

import (
	"context"
	"fmt"
	"github.com/nemirlev/zenmoney-go-sdk/v2/models"
)

// DeleteObjects handles deletion of multiple objects from different tables
// based on the Deletion objects received from ZenMoney API.
// It processes deletions in a single transaction to ensure data consistency.
// Each Deletion object contains:
// - ID: the object's ID
// - Object: the type of object (e.g., "transaction", "account", etc.)
// - User: the user ID
// - Stamp: timestamp of deletion
func (s *DB) DeleteObjects(ctx context.Context, deletions []models.Deletion) error {
	if len(deletions) == 0 {
		return nil
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Process each deletion
	for _, del := range deletions {
		query := ""
		switch del.Object {
		case string(models.EntityTypeAccount):
			query = `DELETE FROM account WHERE id = $1 AND "user" = $2`
		case string(models.EntityTypeTag):
			query = `DELETE FROM tag WHERE id = $1 AND "user" = $2`
		case string(models.EntityTypeMerchant):
			query = `DELETE FROM merchant WHERE id = $1 AND "user" = $2`
		case string(models.EntityTypeBudget):
			query = `DELETE FROM budget WHERE "user" = $1 AND date = $2`
		case string(models.EntityTypeReminder):
			query = `DELETE FROM reminder WHERE id = $1 AND "user" = $2`
		case string(models.EntityTypeReminderMarker):
			query = `DELETE FROM reminder_marker WHERE id = $1 AND "user" = $2`
		case string(models.EntityTypeTransaction):
			query = `DELETE FROM transaction WHERE id = $1 AND "user" = $2`
		default:
			return fmt.Errorf("unsupported object type for deletion: %s", del.Object)
		}

		// Execute the delete query
		commandTag, err := tx.Exec(ctx, query, del.ID, del.User)
		if err != nil {
			return fmt.Errorf("failed to delete %s with ID %s: %w", del.Object, del.ID, err)
		}

		// Check if any row was actually deleted
		if commandTag.RowsAffected() == 0 {
			// Log warning but don't return error as the object might have been already deleted
			fmt.Printf("warning: no %s found for deletion with ID %s and user %d\n",
				del.Object, del.ID, del.User)
		}

		// Record the deletion in deletion_history table for audit
		_, err = tx.Exec(ctx, `
            INSERT INTO deletion_history (
                object_id, object_type, user_id, deleted_at
            ) VALUES ($1, $2, $3, to_timestamp($4))`,
			del.ID, del.Object, del.User, del.Stamp,
		)
		if err != nil {
			return fmt.Errorf("failed to record deletion history: %w", err)
		}
	}

	// Commit the transaction
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit deletion transaction: %w", err)
	}

	return nil
}
