package app

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/nemirlev/zenmoney-export/v2/config"
	"github.com/nemirlev/zenmoney-export/v2/internal/interfaces"
	"github.com/nemirlev/zenmoney-export/v2/mocks"
	"github.com/nemirlev/zenmoney-go-sdk/v3/api"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestInstallDefaultLoggerHonorsConfiguredLevel(t *testing.T) {
	originalLogger := slog.Default()
	t.Cleanup(func() {
		slog.SetDefault(originalLogger)
	})

	tests := []struct {
		name         string
		level        string
		debugEnabled bool
	}{
		{name: "info suppresses debug", level: "info", debugEnabled: false},
		{name: "debug enables debug", level: "debug", debugEnabled: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := installDefaultLogger(&config.Config{LogLevel: tt.level})

			if slog.Default() != logger {
				t.Fatal("configured application logger was not installed as slog default")
			}
			if got := logger.Handler().Enabled(context.Background(), slog.LevelDebug); got != tt.debugEnabled {
				t.Errorf("debug enabled = %t, want %t", got, tt.debugEnabled)
			}
			if !logger.Handler().Enabled(context.Background(), slog.LevelInfo) {
				t.Error("info logging must remain enabled")
			}
		})
	}
}

func TestNewApplicationClosesStorageWhenAPIClientInitializationFails(t *testing.T) {
	originalLogger := slog.Default()
	t.Cleanup(func() { slog.SetDefault(originalLogger) })

	storage := mocks.NewStorage(t)
	storage.On("Close", mock.MatchedBy(func(ctx context.Context) bool {
		return ctx.Err() == nil
	})).Return(nil).Once()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	clientError := errors.New("invalid API client configuration")

	application, err := newApplication(
		ctx,
		&config.Config{DBType: "postgres", DBConfig: "postgres://example", ZenMoneyToken: "token"},
		func(context.Context, interfaces.StorageType, string) (interfaces.Storage, error) {
			return storage, nil
		},
		func(string, ...api.Option) (*api.Client, error) {
			return nil, clientError
		},
	)

	require.Nil(t, application)
	require.ErrorIs(t, err, clientError)
}

func TestApplicationCloseClosesStorage(t *testing.T) {
	storage := mocks.NewStorage(t)
	storage.On("Close", mock.Anything).Return(nil).Once()
	application := &Application{db: storage}

	require.NoError(t, application.Close(context.Background()))
}
