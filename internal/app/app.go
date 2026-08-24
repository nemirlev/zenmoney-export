package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/nemirlev/zenmoney-export/v2/config"
	"github.com/nemirlev/zenmoney-export/v2/internal/db"
	"github.com/nemirlev/zenmoney-export/v2/internal/interfaces"
	"github.com/nemirlev/zenmoney-go-sdk/v3/api"
)

type storageFactory func(
	ctx context.Context,
	storageType interfaces.StorageType,
	connectionString string,
) (interfaces.Storage, error)

type zenClientFactory func(token string, opts ...api.Option) (*api.Client, error)

type Application struct {
	cfg       *config.Config
	db        interfaces.Storage
	zenClient *api.Client
	logger    *slog.Logger

	SyncService *SyncService
}

func NewApplication(ctx context.Context, cfg *config.Config) (*Application, error) {
	return newApplication(ctx, cfg, db.NewStorage, api.NewClient)
}

func newApplication(
	ctx context.Context,
	cfg *config.Config,
	newStorage storageFactory,
	newZenClient zenClientFactory,
) (*Application, error) {
	logger := installDefaultLogger(cfg)

	storage, err := newStorage(ctx, interfaces.StorageType(cfg.DBType), cfg.DBConfig)
	if err != nil {
		return nil, err
	}

	maxResponseSize, err := cfg.MaxResponseSizeBytes()
	if err != nil {
		if closeErr := storage.Close(context.WithoutCancel(ctx)); closeErr != nil {
			return nil, errors.Join(
				err,
				fmt.Errorf("close storage after API client configuration failure: %w", closeErr),
			)
		}
		return nil, err
	}

	zc, err := newZenClient(
		cfg.ZenMoneyToken,
		api.WithLogger(logger),
		api.WithMaxResponseSize(maxResponseSize),
	)
	if err != nil {
		if closeErr := storage.Close(context.WithoutCancel(ctx)); closeErr != nil {
			return nil, errors.Join(
				err,
				fmt.Errorf("close storage after API client initialization failure: %w", closeErr),
			)
		}
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

func (a *Application) Close(ctx context.Context) error {
	if a == nil || a.db == nil {
		return nil
	}
	return a.db.Close(ctx)
}

func installDefaultLogger(cfg *config.Config) *slog.Logger {
	logger := config.NewLogger(cfg)
	slog.SetDefault(logger)
	return logger
}
