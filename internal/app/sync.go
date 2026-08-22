package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/nemirlev/zenmoney-export/v2/internal/interfaces"
	"github.com/nemirlev/zenmoney-go-sdk/v3/models"
)

type zenMoneySyncClient interface {
	FullSync(ctx context.Context) (models.Response, error)
	SyncSince(ctx context.Context, lastSync time.Time) (models.Response, error)
	ForceSyncEntitiesSince(ctx context.Context, lastSync time.Time, entityTypes ...models.EntityType) (models.Response, error)
}

type SyncService struct {
	app    *Application
	client zenMoneySyncClient
}

const (
	initialDaemonRetryDelay = time.Minute
	maximumDaemonRetryDelay = 15 * time.Minute
)

type exponentialBackoff struct {
	initial time.Duration
	maximum time.Duration
	current time.Duration
}

type contextWaitFunc func(ctx context.Context, duration time.Duration) error

func NewSyncService(app *Application) *SyncService {
	return &SyncService{app: app, client: app.zenClient}
}

type SyncParams struct {
	FromDate  string
	ToDate    string
	Entities  string
	BatchSize int
	WriteMode interfaces.WriteMode
	Force     bool
	DryRun    bool
}

type syncSummary struct {
	ServerTimestamp int64
	Instruments     int
	Countries       int
	Companies       int
	Users           int
	Accounts        int
	Tags            int
	Merchants       int
	Budgets         int
	Reminders       int
	ReminderMarkers int
	Transactions    int
	Deletions       int
	Total           int
}

func summarizeResponse(data *models.Response) syncSummary {
	summary := syncSummary{
		ServerTimestamp: data.ServerTimestamp,
		Instruments:     len(data.Instrument),
		Countries:       len(data.Country),
		Companies:       len(data.Company),
		Users:           len(data.User),
		Accounts:        len(data.Account),
		Tags:            len(data.Tag),
		Merchants:       len(data.Merchant),
		Budgets:         len(data.Budget),
		Reminders:       len(data.Reminder),
		ReminderMarkers: len(data.ReminderMarker),
		Transactions:    len(data.Transaction),
		Deletions:       len(data.Deletion),
	}
	summary.Total = summary.Instruments + summary.Countries + summary.Companies +
		summary.Users + summary.Accounts + summary.Tags + summary.Merchants +
		summary.Budgets + summary.Reminders + summary.ReminderMarkers +
		summary.Transactions + summary.Deletions

	return summary
}

func logSyncSummary(message string, summary syncSummary) {
	slog.Info(message,
		"server_timestamp", summary.ServerTimestamp,
		"counts", slog.GroupValue(
			slog.Int("instruments", summary.Instruments),
			slog.Int("countries", summary.Countries),
			slog.Int("companies", summary.Companies),
			slog.Int("users", summary.Users),
			slog.Int("accounts", summary.Accounts),
			slog.Int("tags", summary.Tags),
			slog.Int("merchants", summary.Merchants),
			slog.Int("budgets", summary.Budgets),
			slog.Int("reminders", summary.Reminders),
			slog.Int("reminder_markers", summary.ReminderMarkers),
			slog.Int("transactions", summary.Transactions),
			slog.Int("deletions", summary.Deletions),
		),
		"total", summary.Total,
	)
}

// ParseSyncEntities parses the CLI entity selection. "all" means a regular
// diff (or a full sync when combined with --force); named entities are sent as
// forceFetch while the same global diff cursor is preserved.
func ParseSyncEntities(value string) (bool, []models.EntityType, error) {
	aliases := map[string]models.EntityType{
		"instrument":      models.EntityTypeInstrument,
		"instruments":     models.EntityTypeInstrument,
		"country":         models.EntityTypeCountry,
		"countries":       models.EntityTypeCountry,
		"company":         models.EntityTypeCompany,
		"companies":       models.EntityTypeCompany,
		"user":            models.EntityTypeUser,
		"users":           models.EntityTypeUser,
		"account":         models.EntityTypeAccount,
		"accounts":        models.EntityTypeAccount,
		"tag":             models.EntityTypeTag,
		"tags":            models.EntityTypeTag,
		"merchant":        models.EntityTypeMerchant,
		"merchants":       models.EntityTypeMerchant,
		"budget":          models.EntityTypeBudget,
		"budgets":         models.EntityTypeBudget,
		"reminder":        models.EntityTypeReminder,
		"reminders":       models.EntityTypeReminder,
		"remindermarker":  models.EntityTypeReminderMarker,
		"remindermarkers": models.EntityTypeReminderMarker,
		"transaction":     models.EntityTypeTransaction,
		"transactions":    models.EntityTypeTransaction,
	}

	parts := strings.Split(value, ",")
	entities := make([]models.EntityType, 0, len(parts))
	seen := make(map[models.EntityType]struct{}, len(parts))
	all := false

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			return false, nil, fmt.Errorf("entities must not contain empty values")
		}

		normalized := strings.NewReplacer("-", "", "_", "").Replace(strings.ToLower(part))
		if normalized == "all" {
			all = true
			continue
		}

		entityType, ok := aliases[normalized]
		if !ok {
			return false, nil, fmt.Errorf("unknown entity %q", part)
		}
		if _, ok := seen[entityType]; ok {
			continue
		}
		seen[entityType] = struct{}{}
		entities = append(entities, entityType)
	}

	if all && len(entities) > 0 {
		return false, nil, fmt.Errorf("entity %q cannot be combined with named entities", "all")
	}
	return all, entities, nil
}

func (s *SyncService) Sync(ctx context.Context, p *SyncParams) (syncErr error) {
	allEntities, entities, err := ParseSyncEntities(p.Entities)
	if err != nil {
		return fmt.Errorf("invalid entities: %w", err)
	}

	slog.Info("Starting sync",
		"db_type", s.app.cfg.DBType,
		"entities", p.Entities,
		"force", p.Force,
	)

	if !p.DryRun {
		lock, err := s.app.db.AcquireSyncLock(ctx)
		if err != nil {
			return fmt.Errorf("acquire sync lock: %w", err)
		}
		defer func() {
			if err := lock.Unlock(context.WithoutCancel(ctx)); err != nil {
				syncErr = errors.Join(syncErr, fmt.Errorf("release sync lock: %w", err))
			}
		}()
	}

	lastSync, err := s.app.db.GetLastSyncStatus(ctx)
	if err != nil {
		return err
	}

	var data models.Response
	if lastSync.ID == 0 || (allEntities && p.Force) {
		data, err = s.client.FullSync(ctx)
	} else if allEntities {
		data, err = s.client.SyncSince(ctx, time.Unix(lastSync.ServerTimestamp, 0))
	} else {
		data, err = s.client.ForceSyncEntitiesSince(
			ctx,
			time.Unix(lastSync.ServerTimestamp, 0),
			entities...,
		)
	}
	if err != nil {
		return err
	}

	summary := summarizeResponse(&data)
	if !p.DryRun {
		batchSize := p.BatchSize
		if batchSize <= 0 {
			batchSize = interfaces.DefaultBatchSize
		}
		err = s.app.db.Save(ctx, &data, interfaces.SaveOptions{BatchSize: batchSize, WriteMode: p.WriteMode})
		if err != nil {
			return err
		}
	} else {
		logSyncSummary("Dry run sync summary", summary)
		slog.Info("Dry run mode: skipping save and sync cursor update")
	}

	slog.Info("Sync completed successfully")
	return nil
}

func (s *SyncService) DaemonSync(ctx context.Context, p *SyncParams, interval int) error {
	if interval <= 0 {
		return fmt.Errorf("daemon sync interval must be greater than zero minutes")
	}

	slog.Info("Running daemon sync", "interval_minutes", interval)
	return runDaemonSync(
		ctx,
		s.Sync,
		p,
		time.Duration(interval)*time.Minute,
		&exponentialBackoff{
			initial: initialDaemonRetryDelay,
			maximum: maximumDaemonRetryDelay,
		},
		waitForContext,
	)
}

func runDaemonSync(
	ctx context.Context,
	sync func(context.Context, *SyncParams) error,
	params *SyncParams,
	interval time.Duration,
	backoff *exponentialBackoff,
	wait contextWaitFunc,
) error {
	for {
		if ctx.Err() != nil {
			return nil
		}

		err := sync(ctx, params)
		if ctx.Err() != nil {
			return nil
		}

		delay := interval
		if errors.Is(err, interfaces.ErrSyncAlreadyRunning) {
			slog.Info("Skipping sync because another exporter instance is already running")
			backoff.Reset()
		} else if err != nil {
			slog.Error("sync failed", "error", err)
			delay = backoff.Next()
		} else {
			backoff.Reset()
		}

		if err := wait(ctx, delay); err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err
		}
	}
}

func (b *exponentialBackoff) Next() time.Duration {
	if b.current == 0 {
		b.current = b.initial
		return b.current
	}
	if b.current >= b.maximum/2 {
		b.current = b.maximum
		return b.current
	}
	b.current *= 2
	return b.current
}

func (b *exponentialBackoff) Reset() {
	b.current = 0
}

func waitForContext(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
