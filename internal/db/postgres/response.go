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
	options, err := normalizeSaveOptions(options)
	if err != nil {
		return err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	session := responseSaveSession{
		database: s,
		ctx:      ctx,
		tx:       tx,
	}
	defer session.rollbackIfOpen()
	session.status = interfaces.SyncStatus{
		StartedAt:        time.Now(),
		FinishedAt:       nil,
		SyncType:         string(options.SyncType),
		ServerTimestamp:  response.ServerTimestamp,
		RecordsProcessed: s.countRecords(response),
		Status:           "in_progress",
		ErrorMessage:     nil,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	if err := saveResponse(ctx, tx, response, options); err != nil {
		return session.fail(err)
	}
	return session.complete()
}

func normalizeSaveOptions(options interfaces.SaveOptions) (interfaces.SaveOptions, error) {
	options.BatchSize = normalizeBatchSize(options.BatchSize)
	if options.WriteMode == "" {
		options.WriteMode = interfaces.WriteModeBatch
	}
	if options.WriteMode != interfaces.WriteModeBatch &&
		options.WriteMode != interfaces.WriteModeCopy {
		return interfaces.SaveOptions{}, fmt.Errorf(
			"unsupported write mode %q",
			options.WriteMode,
		)
	}
	if options.SyncType == "" {
		options.SyncType = interfaces.SyncTypeFull
	}
	if options.SyncType != interfaces.SyncTypeFull &&
		options.SyncType != interfaces.SyncTypePartial &&
		options.SyncType != interfaces.SyncTypeForce {
		return interfaces.SaveOptions{}, fmt.Errorf(
			"unsupported sync type %q",
			options.SyncType,
		)
	}
	return options, nil
}

type responseSaveSession struct {
	database *DB
	ctx      context.Context
	tx       pgx.Tx
	status   interfaces.SyncStatus
	closed   bool
}

func (s *responseSaveSession) rollbackIfOpen() {
	if s.closed {
		return
	}
	if err := s.tx.Rollback(s.ctx); err != nil {
		slog.Error("failed to rollback transaction", "error", err)
	}
}

func (s *responseSaveSession) fail(saveErr error) error {
	rollbackErr := s.tx.Rollback(s.ctx)
	s.closed = true
	if rollbackErr != nil && rollbackErr != pgx.ErrTxClosed {
		slog.Error("failed to rollback transaction", "error", rollbackErr)
	}

	finishedAt := time.Now()
	s.status.FinishedAt = &finishedAt
	s.status.Status = "failed"
	errorMessage := saveErr.Error()
	s.status.ErrorMessage = &errorMessage
	if statusErr := s.database.SaveSyncStatus(s.ctx, s.status); statusErr != nil {
		slog.Error("failed to save failed sync status", "error", statusErr)
	}

	return saveErr
}

func (s *responseSaveSession) complete() error {
	finishedAt := time.Now()
	s.status.FinishedAt = &finishedAt
	s.status.Status = "completed"
	if err := s.database.saveSyncStatus(s.ctx, s.tx, s.status); err != nil {
		return s.fail(fmt.Errorf("failed to save completed sync status: %w", err))
	}

	err := s.tx.Commit(s.ctx)
	s.closed = true
	if err != nil {
		return s.fail(fmt.Errorf("failed to commit transaction: %w", err))
	}
	return nil
}

func saveResponse(
	ctx context.Context,
	tx pgx.Tx,
	response *models.Response,
	options interfaces.SaveOptions,
) error {
	if err := saveInstruments(ctx, tx, response.Instrument, options.BatchSize); err != nil {
		return fmt.Errorf("failed to save instruments: %w", err)
	}
	if err := saveCountries(ctx, tx, response.Country, options.BatchSize); err != nil {
		return fmt.Errorf("failed to save countries: %w", err)
	}
	if err := saveCompanies(ctx, tx, response.Company, options.BatchSize); err != nil {
		return fmt.Errorf("failed to save companies: %w", err)
	}
	if err := saveUsers(ctx, tx, response.User, options.BatchSize); err != nil {
		return fmt.Errorf("failed to save users: %w", err)
	}
	if err := saveAccounts(ctx, tx, response.Account, options.BatchSize); err != nil {
		return fmt.Errorf("failed to save accounts: %w", err)
	}
	if err := saveTags(ctx, tx, response.Tag, options.BatchSize); err != nil {
		return fmt.Errorf("failed to save tags: %w", err)
	}
	if err := saveMerchants(ctx, tx, response.Merchant, options.BatchSize); err != nil {
		return fmt.Errorf("failed to save merchants: %w", err)
	}
	if err := saveBudgets(ctx, tx, response.Budget, options.BatchSize); err != nil {
		return fmt.Errorf("failed to save budgets: %w", err)
	}
	if err := saveReminders(ctx, tx, response.Reminder, options.BatchSize); err != nil {
		return fmt.Errorf("failed to save reminders: %w", err)
	}
	if err := saveReminderMarkers(
		ctx,
		tx,
		response.ReminderMarker,
		options.BatchSize,
	); err != nil {
		return fmt.Errorf("failed to save reminder markers: %w", err)
	}

	if err := saveResponseTransactions(ctx, tx, response.Transaction, options); err != nil {
		return fmt.Errorf("failed to save transactions: %w", err)
	}
	if err := deleteObjects(ctx, tx, response.Deletion); err != nil {
		return fmt.Errorf("failed to process deletions: %w", err)
	}
	return nil
}

func saveResponseTransactions(
	ctx context.Context,
	tx pgx.Tx,
	transactions []models.Transaction,
	options interfaces.SaveOptions,
) error {
	if len(transactions) == 0 {
		return nil
	}
	if options.WriteMode == interfaces.WriteModeCopy {
		return copyTransactions(ctx, tx, transactions)
	}
	return saveTransactions(ctx, tx, transactions, options.BatchSize)
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
