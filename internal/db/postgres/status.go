package postgres

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/nemirlev/zenmoney-export/v2/internal/interfaces"
)

type syncStatusQueryRower interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// SaveSyncStatus saves synchronization status to the database
// It creates a new record in the sync_status table with the provided status information
func (s *DB) SaveSyncStatus(ctx context.Context, status interfaces.SyncStatus) error {
	return s.saveSyncStatus(ctx, s.pool, status)
}

// saveSyncStatus saves a synchronization status using the provided executor.
// Passing a transaction keeps a completed status atomic with the synchronized data.
func (s *DB) saveSyncStatus(
	ctx context.Context,
	executor syncStatusQueryRower,
	status interfaces.SyncStatus,
) error {
	query := `
        INSERT INTO sync_status (
            started_at, finished_at, sync_type, server_timestamp,
            records_processed, status, error_message, created_at, updated_at
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
        RETURNING id`

	err := executor.QueryRow(ctx, query,
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

// GetLastSyncStatus retrieves the latest completed synchronization status from the database.
// Failed or unfinished runs must not advance the incremental synchronization cursor.
func (s *DB) GetLastSyncStatus(ctx context.Context) (interfaces.SyncStatus, error) {
	var status interfaces.SyncStatus
	query := `
        SELECT id, started_at, finished_at, sync_type, server_timestamp,
               records_processed, status, error_message, created_at, updated_at
        FROM sync_status
		WHERE status = 'completed'
        ORDER BY id DESC
        LIMIT 1`

	err := s.pool.QueryRow(ctx, query).Scan(
		&status.ID, &status.StartedAt, &status.FinishedAt,
		&status.SyncType, &status.ServerTimestamp,
		&status.RecordsProcessed, &status.Status,
		&status.ErrorMessage, &status.CreatedAt, &status.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			slog.Info("no previous sync element found in database")
			return interfaces.SyncStatus{}, nil
		}
		return interfaces.SyncStatus{}, fmt.Errorf("failed to get last sync status: %w", err)
	}

	return status, nil
}
