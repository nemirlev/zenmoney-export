package app

import (
	"context"
	"github.com/nemirlev/zenmoney-export/v2/config"
	"github.com/nemirlev/zenmoney-export/v2/internal/db"
	"github.com/nemirlev/zenmoney-export/v2/internal/interfaces"
	"github.com/nemirlev/zenmoney-go-sdk/v2/api"
	"log/slog"
)

type Application struct {
	cfg       *config.Config
	db        interfaces.Storage
	zenClient *api.Client
	logger    *slog.Logger

	SyncService *SyncService
}

func NewApplication(ctx context.Context, cfg *config.Config) (*Application, error) {
	logger := config.NewLogger(cfg)

	storage, err := db.NewStorage(ctx, interfaces.StorageType(cfg.DBType), cfg.DBConfig)
	if err != nil {
		return nil, err
	}

	zc, err := api.NewClient(cfg.ZenMoneyToken)
	if err != nil {
		return nil, err
	}

	app := &Application{
		cfg:       cfg,
		db:        storage,
		zenClient: zc,
		logger:    logger,
	}

	app.SyncService = NewSyncService(app)

	return app, nil
}
