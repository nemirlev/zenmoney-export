package app

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
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
			if got := logger.Handler().
				Enabled(context.Background(), slog.LevelDebug); got != tt.debugEnabled {
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
		&config.Config{
			DBType:            "postgres",
			DBConfig:          "postgres://example",
			ZenMoneyToken:     "token",
			MaxResponseSizeMB: config.DefaultMaxResponseSizeMB,
		},
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

func TestNewApplicationConfiguresZenMoneyResponseLimit(t *testing.T) {
	originalLogger := slog.Default()
	t.Cleanup(func() { slog.SetDefault(originalLogger) })

	storage := mocks.NewStorage(t)
	storage.On("Close", mock.Anything).Return(nil).Once()

	oversizedResponse := `{"padding":"` + strings.Repeat("a", 1<<20) + `"}`
	httpClient := &http.Client{
		Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(oversizedResponse)),
				Header:     make(http.Header),
			}, nil
		}),
	}

	application, err := newApplication(
		context.Background(),
		&config.Config{
			DBType:            "postgres",
			DBConfig:          "postgres://example",
			ZenMoneyToken:     "token",
			MaxResponseSizeMB: 1,
		},
		func(context.Context, interfaces.StorageType, string) (interfaces.Storage, error) {
			return storage, nil
		},
		func(token string, opts ...api.Option) (*api.Client, error) {
			return api.NewClient(token, append(opts,
				api.WithHTTPClient(httpClient),
				api.WithRetryPolicy(0, 0),
			)...)
		},
	)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, application.Close(context.Background())) })

	_, err = application.zenClient.FullSync(context.Background())

	require.ErrorContains(t, err, "response body exceeds configured limit")
}

func TestApplicationCloseClosesStorage(t *testing.T) {
	storage := mocks.NewStorage(t)
	storage.On("Close", mock.Anything).Return(nil).Once()
	application := &Application{db: storage}

	require.NoError(t, application.Close(context.Background()))
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
