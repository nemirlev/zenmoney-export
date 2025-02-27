package app

import (
	"context"
	"github.com/nemirlev/zenmoney-go-sdk/v2/models"
	"log/slog"
	"time"
)

type SyncService struct {
	app *Application
}

func NewSyncService(app *Application) *SyncService {
	return &SyncService{app: app}
}

type SyncParams struct {
	FromDate string
	ToDate   string
	Entities string
	Force    bool
	DryRun   bool
}

func (s *SyncService) Sync(ctx context.Context, p *SyncParams) error {
	slog.Info("Starting sync",
		"db_type", s.app.cfg.DBType,
		"entities", p.Entities,
		"force", p.Force,
	)

	lastSync, err := s.app.db.GetLastSyncStatus(ctx)
	if err != nil {
		return err
	}

	var data models.Response
	if lastSync.ID == 0 || p.Force {
		data, err = s.app.zenClient.FullSync(ctx)
		if err != nil {
			return err
		}
	} else {
		data, err = s.app.zenClient.SyncSince(ctx, time.Unix(lastSync.ServerTimestamp, 0))
		if err != nil {
			return err
		}
	}

	if !p.DryRun {
		err = s.app.db.Save(ctx, &data)
		if err != nil {
			return err
		}
	} else {
		slog.Info("Dry run mode: skipping save")
	}

	slog.Info("Sync completed successfully")
	return nil
}

func (s *SyncService) DaemonSync(ctx context.Context, p *SyncParams, interval int) error {
	slog.Info("Running daemon sync", "interval_minutes", interval)
	for {
		if err := s.Sync(ctx, p); err != nil {
			slog.Error("sync failed", "error", err)
		}
		time.Sleep(time.Duration(interval) * time.Minute)
	}
}
