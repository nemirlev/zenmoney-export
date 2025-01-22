package postgres

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/nemirlev/zenmoney-export/internal/interfaces"
	"time"
)

// SaveSyncStatus saves synchronization status to the database
// It creates a new record in the sync_status table with the provided status information
func (s *DB) SaveSyncStatus(ctx context.Context, status interfaces.SyncStatus) error {
	query := `
        INSERT INTO sync_status (
            started_at, finished_at, sync_type, server_timestamp,
            records_processed, status, error_message, created_at, updated_at
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
        RETURNING id`

	err := s.pool.QueryRow(ctx, query,
		status.StartedAt, status.FinishedAt, status.SyncType,
		status.ServerTimestamp, status.RecordsProcessed,
		status.Status, status.ErrorMessage,
		time.Now(), time.Now(),
	).Scan(&status.ID)

	if err != nil {
		return fmt.Errorf("failed to save sync status: %w", err)
	}

	return nil
}

// GetLastSyncStatus retrieves the latest synchronization status from the database
// Returns the most recent sync_status record ordered by ID
func (s *DB) GetLastSyncStatus(ctx context.Context) (interfaces.SyncStatus, error) {
	var status interfaces.SyncStatus
	query := `
        SELECT id, started_at, finished_at, sync_type, server_timestamp,
               records_processed, status, error_message, created_at, updated_at
        FROM sync_status
        ORDER BY id DESC
        LIMIT 1`

	err := s.pool.QueryRow(ctx, query).Scan(
		&status.ID, &status.StartedAt, &status.FinishedAt,
		&status.SyncType, &status.ServerTimestamp,
		&status.RecordsProcessed, &status.Status,
		&status.ErrorMessage, &status.CreatedAt, &status.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return interfaces.SyncStatus{}, fmt.Errorf("no sync status found")
		}
		return interfaces.SyncStatus{}, fmt.Errorf("failed to get last sync status: %w", err)
	}

	return status, nil
}
