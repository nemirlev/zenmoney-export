package cmd

import (
	"context"
	"errors"
	"testing"

	"github.com/nemirlev/zenmoney-export/v2/config"
	"github.com/nemirlev/zenmoney-export/v2/internal/app"
	"github.com/nemirlev/zenmoney-export/v2/internal/interfaces"
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
			require.NoError(
				t,
				cmd.Flags().Set("daemon", map[bool]string{true: "true", false: "false"}[tt.daemon]),
			)
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

func TestSyncCommandBatchSizeValidationAndWiring(t *testing.T) {
	cmd := NewSyncCommand(&RootCommand{})
	require.Equal(t, "1000", cmd.Flags().Lookup("batch-size").DefValue)
	require.Equal(t, "batch", cmd.Flags().Lookup("write-mode").DefValue)

	for _, value := range []string{"0", "-1"} {
		require.NoError(t, cmd.Flags().Set("batch-size", value))
		require.ErrorContains(t, cmd.Args(cmd, nil), "--batch-size must be greater than zero")
	}

	opts := &config.SyncOptions{
		Entities:  "all",
		BatchSize: 37,
		WriteMode: "copy",
		Force:     true,
		DryRun:    true,
	}
	require.Equal(t, &app.SyncParams{
		Entities:  "all",
		BatchSize: 37,
		WriteMode: interfaces.WriteModeCopy,
		Force:     true,
		DryRun:    true,
	}, syncParamsFromOptions(opts))
	require.Equal(t, 1000, interfaces.DefaultBatchSize)
}

func TestSyncCommandRejectsUnknownWriteMode(t *testing.T) {
	cmd := NewSyncCommand(&RootCommand{})
	require.NoError(t, cmd.Flags().Set("write-mode", "COPY"))
	require.ErrorContains(t, cmd.Args(cmd, nil), "--write-mode must be one of: batch, copy")
}

type runSyncAndCloseTestCase struct {
	name            string
	daemon          bool
	cancel          bool
	runnerErr       error
	wantErr         error
	wantSyncCalls   int
	wantDaemonCalls int
}

func TestRunSyncAndCloseAlwaysClosesDatabase(t *testing.T) {
	operationError := errors.New("operation failed")
	tests := []runSyncAndCloseTestCase{
		{name: "one-shot success", wantSyncCalls: 1},
		{
			name:          "one-shot error",
			runnerErr:     operationError,
			wantErr:       operationError,
			wantSyncCalls: 1,
		},
		{
			name:            "daemon error",
			daemon:          true,
			runnerErr:       operationError,
			wantErr:         operationError,
			wantDaemonCalls: 1,
		},
		{
			name:          "cancellation is clean",
			cancel:        true,
			runnerErr:     context.Canceled,
			wantSyncCalls: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testRunSyncAndClose(t, tt)
		})
	}
}

func testRunSyncAndClose(t *testing.T, tt runSyncAndCloseTestCase) {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	if tt.cancel {
		cancel()
	}

	runner := &stubSyncRunner{syncErr: tt.runnerErr, daemonErr: tt.runnerErr}
	closer := &recordingCloser{}

	err := runSyncAndClose(ctx, runner, closer, &app.SyncParams{}, tt.daemon, 30)

	if tt.wantErr != nil {
		require.ErrorIs(t, err, tt.wantErr)
	} else {
		require.NoError(t, err)
	}
	require.Equal(t, 1, closer.calls)
	require.NoError(
		t,
		closer.contexts[0].Err(),
		"database close must not inherit cancellation",
	)
	require.Equal(t, tt.wantSyncCalls, runner.syncCalls)
	require.Equal(t, tt.wantDaemonCalls, runner.daemonCalls)
}
