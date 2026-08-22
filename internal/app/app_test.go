package app

import (
	"context"
	"log/slog"
	"testing"

	"github.com/nemirlev/zenmoney-export/v2/config"
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
