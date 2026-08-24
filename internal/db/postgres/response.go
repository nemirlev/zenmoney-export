package postgres

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/nemirlev/zenmoney-export/v2/internal/interfaces"
	"github.com/nemirlev/zenmoney-go-sdk/v3/models"
)

// Save saves the entire API response to database
func (s *DB) Save(
	ctx context.Context,
	response *models.Response,
	options interfaces.SaveOptions,
) error {
	batchSize := normalizeBatchSize(options.BatchSize)
	writeMode := options.WriteMode
	if writeMode == "" {
		writeMode = interfaces.WriteModeBatch
	}
	if writeMode != interfaces.WriteModeBatch && writeMode != interfaces.WriteModeCopy {
		return fmt.Errorf("unsupported write mode %q", writeMode)
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
			slog.Error("failed to rollback transaction", "error", rollbackErr)
		}
	}()

	status := interfaces.SyncStatus{
		StartedAt:        time.Now(),
		FinishedAt:       nil,
		SyncType:         "full", // TODO: implement incremental sync type
		ServerTimestamp:  response.ServerTimestamp,
		RecordsProcessed: s.countRecords(response),
		Status:           "in_progress",
		ErrorMessage:     nil,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	fail := func(saveErr error) error {
		rollbackErr := tx.Rollback(ctx)
		txClosed = true
		if rollbackErr != nil && rollbackErr != pgx.ErrTxClosed {
			slog.Error("failed to rollback transaction", "error", rollbackErr)
		}

		finishedAt := time.Now()
		status.FinishedAt = &finishedAt
		status.Status = "failed"
		errorMessage := saveErr.Error()
		status.ErrorMessage = &errorMessage
		if statusErr := s.SaveSyncStatus(ctx, status); statusErr != nil {
			slog.Error("failed to save failed sync status", "error", statusErr)
		}

		return saveErr
	}

	if len(response.Instrument) > 0 {
		if err = saveInstruments(ctx, tx, response.Instrument, batchSize); err != nil {
			return fail(fmt.Errorf("failed to save instruments: %w", err))
		}
	}

	if len(response.Country) > 0 {
		if err = saveCountries(ctx, tx, response.Country, batchSize); err != nil {
			return fail(fmt.Errorf("failed to save countries: %w", err))
		}
	}

	if len(response.Company) > 0 {
		if err = saveCompanies(ctx, tx, response.Company, batchSize); err != nil {
			return fail(fmt.Errorf("failed to save companies: %w", err))
		}
	}

	if len(response.User) > 0 {
		if err = saveUsers(ctx, tx, response.User, batchSize); err != nil {
			return fail(fmt.Errorf("failed to save users: %w", err))
		}
	}

	if len(response.Account) > 0 {
		if err = saveAccounts(ctx, tx, response.Account, batchSize); err != nil {
			return fail(fmt.Errorf("failed to save accounts: %w", err))
		}
	}

	if len(response.Tag) > 0 {
		if err = saveTags(ctx, tx, response.Tag, batchSize); err != nil {
			return fail(fmt.Errorf("failed to save tags: %w", err))
		}
	}

	if len(response.Merchant) > 0 {
		if err = saveMerchants(ctx, tx, response.Merchant, batchSize); err != nil {
			return fail(fmt.Errorf("failed to save merchants: %w", err))
		}
	}

	if len(response.Budget) > 0 {
		if err = saveBudgets(ctx, tx, response.Budget, batchSize); err != nil {
			return fail(fmt.Errorf("failed to save budgets: %w", err))
		}
	}

	if len(response.Reminder) > 0 {
		if err = saveReminders(ctx, tx, response.Reminder, batchSize); err != nil {
			return fail(fmt.Errorf("failed to save reminders: %w", err))
		}
	}

	if len(response.ReminderMarker) > 0 {
		if err = saveReminderMarkers(ctx, tx, response.ReminderMarker, batchSize); err != nil {
			return fail(fmt.Errorf("failed to save reminder markers: %w", err))
		}
	}

	if len(response.Transaction) > 0 {
		if writeMode == interfaces.WriteModeCopy {
			err = copyTransactions(ctx, tx, response.Transaction)
		} else {
			err = saveTransactions(ctx, tx, response.Transaction, batchSize)
		}
		if err != nil {
			return fail(fmt.Errorf("failed to save transactions: %w", err))
		}
	}

	if len(response.Deletion) > 0 {
		if err = deleteObjects(ctx, tx, response.Deletion); err != nil {
			return fail(fmt.Errorf("failed to process deletions: %w", err))
		}
	}

	finishedAt := time.Now()
	status.FinishedAt = &finishedAt
	status.Status = "completed"
	if err = s.saveSyncStatus(ctx, tx, status); err != nil {
		return fail(fmt.Errorf("failed to save completed sync status: %w", err))
	}

	err = tx.Commit(ctx)
	txClosed = true
	if err != nil {
		return fail(fmt.Errorf("failed to commit transaction: %w", err))
	}

	return nil
}

// countRecords counts total number of records in response
func (s *DB) countRecords(response *models.Response) int {
	return len(response.Instrument) +
		len(response.Country) +
		len(response.Company) +
		len(response.User) +
		len(response.Account) +
		len(response.Tag) +
		len(response.Merchant) +
		len(response.Budget) +
		len(response.Reminder) +
		len(response.ReminderMarker) +
		len(response.Transaction) +
		len(response.Deletion)
}
