package postgres

import (
	"context"
	"fmt"
	"github.com/nemirlev/zenmoney-export/internal/interfaces"
	"github.com/nemirlev/zenmoney-go-sdk/v2/models"
	"log/slog"
	"time"
)

// Save saves the entire API response to database
func (s *DB) Save(ctx context.Context, response *models.Response) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	status := interfaces.SyncStatus{
		StartedAt:        time.Now(),
		FinishedAt:       nil,
		SyncType:         "full", // TODO: implement incremental sync type
		ServerTimestamp:  int64(response.ServerTimestamp),
		RecordsProcessed: s.countRecords(response),
		Status:           "in_progress",
		ErrorMessage:     nil,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	defer func() {
		now := time.Now()
		status.FinishedAt = &now

		if err != nil {
			status.Status = "failed"
			errorMessage := err.Error()
			status.ErrorMessage = &errorMessage
		} else {
			status.Status = "completed"
		}

		if saveErr := s.SaveSyncStatus(ctx, status); saveErr != nil {
			slog.Error("Failed to save sync status on Save method", "error", saveErr)
		}
	}()

	if len(response.Instrument) > 0 {
		if err = s.SaveInstruments(ctx, response.Instrument); err != nil {
			return fmt.Errorf("failed to save instruments: %w", err)
		}
	}

	if len(response.Country) > 0 {
		if err = s.SaveCountries(ctx, response.Country); err != nil {
			return fmt.Errorf("failed to save countries: %w", err)
		}
	}

	if len(response.Company) > 0 {
		if err = s.SaveCompanies(ctx, response.Company); err != nil {
			return fmt.Errorf("failed to save companies: %w", err)
		}
	}

	if len(response.User) > 0 {
		if err = s.SaveUsers(ctx, response.User); err != nil {
			return fmt.Errorf("failed to save users: %w", err)
		}
	}

	if len(response.Account) > 0 {
		if err = s.SaveAccounts(ctx, response.Account); err != nil {
			return fmt.Errorf("failed to save accounts: %w", err)
		}
	}

	if len(response.Tag) > 0 {
		if err = s.SaveTags(ctx, response.Tag); err != nil {
			return fmt.Errorf("failed to save tags: %w", err)
		}
	}

	if len(response.Merchant) > 0 {
		if err = s.SaveMerchants(ctx, response.Merchant); err != nil {
			return fmt.Errorf("failed to save merchants: %w", err)
		}
	}

	if len(response.Budget) > 0 {
		if err = s.SaveBudgets(ctx, response.Budget); err != nil {
			return fmt.Errorf("failed to save budgets: %w", err)
		}
	}

	if len(response.Reminder) > 0 {
		if err = s.SaveReminders(ctx, response.Reminder); err != nil {
			return fmt.Errorf("failed to save reminders: %w", err)
		}
	}

	if len(response.ReminderMarker) > 0 {
		if err = s.SaveReminderMarkers(ctx, response.ReminderMarker); err != nil {
			return fmt.Errorf("failed to save reminder markers: %w", err)
		}
	}

	if len(response.Transaction) > 0 {
		if err = s.SaveTransactions(ctx, response.Transaction); err != nil {
			return fmt.Errorf("failed to save transactions: %w", err)
		}
	}

	if len(response.Deletion) > 0 {
		if err = s.DeleteObjects(ctx, response.Deletion); err != nil {
			return fmt.Errorf("failed to process deletions: %w", err)
		}
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
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
