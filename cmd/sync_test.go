package cmd

import (
	"context"
	"errors"
	"testing"

	"github.com/nemirlev/zenmoney-export/v2/internal/app"
	"github.com/stretchr/testify/require"
)

type stubSyncRunner struct {
	syncErr     error
	daemonErr   error
	syncCalls   int
	daemonCalls int
}

func (r *stubSyncRunner) Sync(context.Context, *app.SyncParams) error {
	r.syncCalls++
	return r.syncErr
}

func (r *stubSyncRunner) DaemonSync(context.Context, *app.SyncParams, int) error {
	r.daemonCalls++
	return r.daemonErr
}

type recordingCloser struct {
	err      error
	calls    int
	contexts []context.Context
}

func (c *recordingCloser) Close(ctx context.Context) error {
	c.calls++
	c.contexts = append(c.contexts, ctx)
	return c.err
}

func TestSyncCommandValidatesEntities(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		wantError bool
	}{
		{name: "all", value: "all"},
		{name: "named aliases", value: " accounts,transactions "},
		{name: "unknown", value: "payments", wantError: true},
		{name: "empty list item", value: "accounts,,tags", wantError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewSyncCommand(&RootCommand{})
			require.NoError(t, cmd.Flags().Set("entities", tt.value))

			err := cmd.PreRunE(cmd, nil)
			if tt.wantError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestSyncCommandRejectsInvalidDaemonInterval(t *testing.T) {
	tests := []struct {
		name      string
		daemon    bool
		interval  string
		wantError bool
	}{
		{name: "zero daemon interval", daemon: true, interval: "0", wantError: true},
		{name: "negative daemon interval", daemon: true, interval: "-1", wantError: true},
		{name: "positive daemon interval", daemon: true, interval: "1"},
		{name: "interval is ignored for one-shot", daemon: false, interval: "0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewSyncCommand(&RootCommand{})
			require.NoError(t, cmd.Flags().Set("daemon", map[bool]string{true: "true", false: "false"}[tt.daemon]))
			require.NoError(t, cmd.Flags().Set("interval", tt.interval))

			err := cmd.Args(cmd, nil)
			if tt.wantError {
				require.ErrorContains(t, err, "greater than zero")
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestRunSyncAndCloseAlwaysClosesDatabase(t *testing.T) {
	operationError := errors.New("operation failed")
	tests := []struct {
		name      string
		daemon    bool
		cancel    bool
		runnerErr error
		wantErr   error
	}{
		{name: "one-shot success"},
		{name: "one-shot error", runnerErr: operationError, wantErr: operationError},
		{name: "daemon error", daemon: true, runnerErr: operationError, wantErr: operationError},
		{name: "cancellation is clean", cancel: true, runnerErr: context.Canceled},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			if tt.cancel {
				cancel()
			} else {
				defer cancel()
			}

			runner := &stubSyncRunner{}
			if tt.daemon {
				runner.daemonErr = tt.runnerErr
			} else {
				runner.syncErr = tt.runnerErr
			}
			closer := &recordingCloser{}

			err := runSyncAndClose(ctx, runner, closer, &app.SyncParams{}, tt.daemon, 30)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, 1, closer.calls)
			require.NoError(t, closer.contexts[0].Err(), "database close must not inherit cancellation")
			if tt.daemon {
				require.Equal(t, 1, runner.daemonCalls)
				require.Zero(t, runner.syncCalls)
			} else {
				require.Equal(t, 1, runner.syncCalls)
				require.Zero(t, runner.daemonCalls)
			}
		})
	}
}
