package postgres

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"

	"github.com/nemirlev/zenmoney-go-sdk/v3/models"
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
	txClosed := false
	defer func() {
		if txClosed {
			return
		}
		if rollbackErr := tx.Rollback(ctx); rollbackErr != nil {
			slog.Error("failed to rollback deletion transaction", "error", rollbackErr)
		}
	}()

	if err := deleteObjects(ctx, tx, deletions); err != nil {
		return err
	}

	err = tx.Commit(ctx)
	txClosed = true
	if err != nil {
		return fmt.Errorf("failed to commit deletion transaction: %w", err)
	}

	return nil
}

// deleteObjects executes deletions using the supplied executor. Save passes its
// transaction here so deletions and upserts share the same atomic boundary.
func deleteObjects(ctx context.Context, executor commandExecutor, deletions []models.Deletion) error {

	// Process each deletion
	for _, del := range deletions {
		query := ""
		args := []any{del.ID, del.User}
		switch del.Object {
		case string(models.EntityTypeCountry):
			countryID, parseErr := strconv.Atoi(del.ID)
			if parseErr != nil {
				return fmt.Errorf("invalid country ID %q: %w", del.ID, parseErr)
			}
			query = `DELETE FROM country WHERE id = $1`
			args = []any{countryID}
		case string(models.EntityTypeAccount):
			query = `DELETE FROM account WHERE id = $1 AND "user" = $2`
		case string(models.EntityTypeTag):
			query = `DELETE FROM tag WHERE id = $1 AND "user" = $2`
		case string(models.EntityTypeMerchant):
			query = `DELETE FROM merchant WHERE id = $1 AND "user" = $2`
		case string(models.EntityTypeBudget):
			// ZenMoney budgets have no ID and are removed by updating their
			// amount/lock fields. A Deletion cannot identify a budget row safely.
			return fmt.Errorf("budget deletion objects are unsupported")
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
		commandTag, err := executor.Exec(ctx, query, args...)
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
		_, err = executor.Exec(ctx, `
            INSERT INTO deletion_history (
                object_id, object_type, user_id, deleted_at
            ) VALUES ($1, $2, $3, to_timestamp($4))`,
			del.ID, del.Object, del.User, del.Stamp,
		)
		if err != nil {
			return fmt.Errorf("failed to record deletion history: %w", err)
		}
	}

	return nil
}
