package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/nemirlev/zenmoney-export/v2/internal/interfaces"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
)

func TestSaveSyncStatus_Success(t *testing.T) {
	db, mock := newTestDB(t)

	status := interfaces.SyncStatus{
		StartedAt:        time.Now(),
		FinishedAt:       nil,
		SyncType:         "full",
		ServerTimestamp:  123456789,
		RecordsProcessed: 10,
		Status:           "in_progress",
		ErrorMessage:     nil,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	mock.ExpectQuery(`INSERT INTO sync_status`).
		WithArgs(
			status.StartedAt, status.FinishedAt, status.SyncType,
			status.ServerTimestamp, status.RecordsProcessed,
			status.Status, status.ErrorMessage,
			pgxmock.AnyArg(), pgxmock.AnyArg(),
		).
		WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow(int64(1)))

	err := db.SaveSyncStatus(context.Background(), status)
	assert.NoError(t, err)
}

func TestSaveSyncStatus_Error(t *testing.T) {
	db, mock := newTestDB(t)

	status := interfaces.SyncStatus{
		StartedAt:        time.Now(),
		FinishedAt:       nil,
		SyncType:         "full",
		ServerTimestamp:  123456789,
		RecordsProcessed: 10,
		Status:           "in_progress",
		ErrorMessage:     nil,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	mock.ExpectQuery(`INSERT INTO sync_status`).
		WithArgs(
			status.StartedAt, status.FinishedAt, status.SyncType,
			status.ServerTimestamp, status.RecordsProcessed,
			status.Status, status.ErrorMessage,
			pgxmock.AnyArg(), pgxmock.AnyArg(),
		).
		WillReturnError(errors.New("insert error"))

	err := db.SaveSyncStatus(context.Background(), status)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to save sync status")
}

const lastCompletedSyncStatusQuery = `SELECT id, started_at, finished_at, sync_type, server_timestamp, records_processed, status, error_message, created_at, updated_at FROM sync_status WHERE status = 'completed' ORDER BY id DESC LIMIT 1`

func TestGetLastSyncStatus_ReturnsLatestCompleted(t *testing.T) {
	db, mock := newTestDB(t)

	expectedStatus := interfaces.SyncStatus{
		ID:               1,
		StartedAt:        time.Now(),
		FinishedAt:       nil,
		SyncType:         "full",
		ServerTimestamp:  123456789,
		RecordsProcessed: 10,
		Status:           "completed",
		ErrorMessage:     nil,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	rows := mock.NewRows([]string{
		"id", "started_at", "finished_at", "sync_type", "server_timestamp",
		"records_processed", "status", "error_message", "created_at", "updated_at",
	}).AddRow(
		expectedStatus.ID, expectedStatus.StartedAt, expectedStatus.FinishedAt,
		expectedStatus.SyncType, expectedStatus.ServerTimestamp,
		expectedStatus.RecordsProcessed, expectedStatus.Status,
		expectedStatus.ErrorMessage, expectedStatus.CreatedAt, expectedStatus.UpdatedAt,
	)

	mock.ExpectQuery(lastCompletedSyncStatusQuery).
		WillReturnRows(rows)

	status, err := db.GetLastSyncStatus(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, expectedStatus, status)

}

func TestGetLastSyncStatus_NoCompletedSync(t *testing.T) {
	db, mock := newTestDB(t)

	mock.ExpectQuery(lastCompletedSyncStatusQuery).
		WillReturnError(pgx.ErrNoRows)

	status, err := db.GetLastSyncStatus(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, interfaces.SyncStatus{}, status)

}

func TestGetLastSyncStatus_Error(t *testing.T) {
	db, mock := newTestDB(t)

	mock.ExpectQuery(lastCompletedSyncStatusQuery).
		WillReturnError(errors.New("query error"))

	status, err := db.GetLastSyncStatus(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get last sync status")
	assert.Equal(t, interfaces.SyncStatus{}, status)

}
