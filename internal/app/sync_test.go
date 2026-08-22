package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/nemirlev/zenmoney-export/v2/config"
	"github.com/nemirlev/zenmoney-export/v2/internal/interfaces"
	"github.com/nemirlev/zenmoney-export/v2/mocks"
	"github.com/nemirlev/zenmoney-go-sdk/v3/models"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type recordingSyncClient struct {
	method   string
	since    time.Time
	entities []models.EntityType
	response models.Response
	events   *[]string
}

func (c *recordingSyncClient) FullSync(context.Context) (models.Response, error) {
	c.record("fetch")
	c.method = "full"
	return c.response, nil
}

func (c *recordingSyncClient) SyncSince(_ context.Context, since time.Time) (models.Response, error) {
	c.record("fetch")
	c.method = "since"
	c.since = since
	return c.response, nil
}

func (c *recordingSyncClient) ForceSyncEntitiesSince(
	_ context.Context,
	since time.Time,
	entities ...models.EntityType,
) (models.Response, error) {
	c.record("fetch")
	c.method = "force-entities"
	c.since = since
	c.entities = append([]models.EntityType(nil), entities...)
	return c.response, nil
}

func (c *recordingSyncClient) record(event string) {
	if c.events != nil {
		*c.events = append(*c.events, event)
	}
}

type testSyncLock struct {
	onUnlock func()
	err      error
}

func (l *testSyncLock) Unlock(context.Context) error {
	if l.onUnlock != nil {
		l.onUnlock()
	}
	return l.err
}

func TestSyncSelectsSDKMethod(t *testing.T) {
	tests := []struct {
		name         string
		lastSync     interfaces.SyncStatus
		params       SyncParams
		wantMethod   string
		wantEntities []models.EntityType
		wantSince    int64
	}{
		{
			name:       "first sync is full even with selected entities",
			params:     SyncParams{Entities: "accounts, transactions"},
			wantMethod: "full",
		},
		{
			name:       "regular incremental sync uses global cursor",
			lastSync:   interfaces.SyncStatus{ID: 1, ServerTimestamp: 123},
			params:     SyncParams{Entities: "all"},
			wantMethod: "since",
			wantSince:  123,
		},
		{
			name:         "selected entities are force fetched with regular diff",
			lastSync:     interfaces.SyncStatus{ID: 1, ServerTimestamp: 234},
			params:       SyncParams{Entities: " accounts,transaction,ACCOUNTS,reminder-markers "},
			wantMethod:   "force-entities",
			wantSince:    234,
			wantEntities: []models.EntityType{models.EntityTypeAccount, models.EntityTypeTransaction, models.EntityTypeReminderMarker},
		},
		{
			name:         "force with selected entities does not discard regular diff",
			lastSync:     interfaces.SyncStatus{ID: 1, ServerTimestamp: 345},
			params:       SyncParams{Entities: "budgets", Force: true},
			wantMethod:   "force-entities",
			wantSince:    345,
			wantEntities: []models.EntityType{models.EntityTypeBudget},
		},
		{
			name:       "force all performs full sync",
			lastSync:   interfaces.SyncStatus{ID: 1, ServerTimestamp: 456},
			params:     SyncParams{Entities: "all", Force: true},
			wantMethod: "full",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := mocks.NewStorage(t)
			storage.On("AcquireSyncLock", mock.Anything).Return(&testSyncLock{}, nil).Once()
			storage.On("GetLastSyncStatus", mock.Anything).Return(tt.lastSync, nil).Once()

			response := models.Response{ServerTimestamp: 999}
			storage.On("Save", mock.Anything, &response, interfaces.SaveOptions{BatchSize: interfaces.DefaultBatchSize}).Return(nil).Once()
			client := &recordingSyncClient{response: response}
			service := &SyncService{
				app: &Application{
					cfg: &config.Config{DBType: "postgres"},
					db:  storage,
				},
				client: client,
			}

			err := service.Sync(context.Background(), &tt.params)

			require.NoError(t, err)
			require.Equal(t, tt.wantMethod, client.method)
			require.Equal(t, tt.wantEntities, client.entities)
			if tt.wantSince != 0 {
				require.Equal(t, time.Unix(tt.wantSince, 0), client.since)
			}
		})
	}
}

func TestParseSyncEntitiesAliases(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected models.EntityType
	}{
		{name: "instruments", value: "instrument,instruments", expected: models.EntityTypeInstrument},
		{name: "countries", value: "country,countries", expected: models.EntityTypeCountry},
		{name: "companies", value: "company,companies", expected: models.EntityTypeCompany},
		{name: "users", value: "user,users", expected: models.EntityTypeUser},
		{name: "accounts", value: "account,accounts", expected: models.EntityTypeAccount},
		{name: "tags", value: "tag,tags", expected: models.EntityTypeTag},
		{name: "merchants", value: "merchant,merchants", expected: models.EntityTypeMerchant},
		{name: "budgets", value: "budget,budgets", expected: models.EntityTypeBudget},
		{name: "reminders", value: "reminder,reminders", expected: models.EntityTypeReminder},
		{name: "reminder markers", value: "reminderMarker,reminder_markers", expected: models.EntityTypeReminderMarker},
		{name: "transactions", value: "transaction,transactions", expected: models.EntityTypeTransaction},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			all, entities, err := ParseSyncEntities(tt.value)
			require.NoError(t, err)
			require.False(t, all)
			require.Equal(t, []models.EntityType{tt.expected}, entities)
		})
	}
}

func TestSyncRejectsInvalidEntitiesBeforeStorageOrAPI(t *testing.T) {
	tests := []string{"", "   ", "accounts,,tags", "unknown", "all,accounts"}

	for _, value := range tests {
		t.Run(value, func(t *testing.T) {
			storage := mocks.NewStorage(t)
			client := &recordingSyncClient{}
			service := &SyncService{
				app:    &Application{cfg: &config.Config{DBType: "postgres"}, db: storage},
				client: client,
			}

			err := service.Sync(context.Background(), &SyncParams{Entities: value})

			require.ErrorContains(t, err, "invalid entities")
			require.Empty(t, client.method)
			storage.AssertNotCalled(t, "GetLastSyncStatus", mock.Anything)
			storage.AssertNotCalled(t, "AcquireSyncLock", mock.Anything)
		})
	}
}

func TestSummarizeResponseCountsEveryEntityType(t *testing.T) {
	response := models.Response{
		ServerTimestamp: 987654321,
		Instrument:      make([]models.Instrument, 1),
		Country:         make([]models.Country, 2),
		Company:         make([]models.Company, 3),
		User:            make([]models.User, 4),
		Account:         make([]models.Account, 5),
		Tag:             make([]models.Tag, 6),
		Merchant:        make([]models.Merchant, 7),
		Budget:          make([]models.Budget, 8),
		Reminder:        make([]models.Reminder, 9),
		ReminderMarker:  make([]models.ReminderMarker, 10),
		Transaction:     make([]models.Transaction, 11),
		Deletion:        make([]models.Deletion, 12),
	}

	summary := summarizeResponse(&response)

	require.Equal(t, syncSummary{
		ServerTimestamp: 987654321,
		Instruments:     1,
		Countries:       2,
		Companies:       3,
		Users:           4,
		Accounts:        5,
		Tags:            6,
		Merchants:       7,
		Budgets:         8,
		Reminders:       9,
		ReminderMarkers: 10,
		Transactions:    11,
		Deletions:       12,
		Total:           78,
	}, summary)
}

func TestSyncDryRunLogsSummaryWithoutSaving(t *testing.T) {
	storage := mocks.NewStorage(t)
	storage.On("GetLastSyncStatus", mock.Anything).Return(interfaces.SyncStatus{}, nil).Once()

	response := models.Response{
		ServerTimestamp: 777,
		Account:         []models.Account{{Title: "sensitive account name"}},
		Tag:             []models.Tag{{Title: "private category"}},
		Deletion:        []models.Deletion{{ID: "sensitive-deletion-id", Object: "account"}},
	}
	client := &recordingSyncClient{response: response}
	service := &SyncService{
		app: &Application{
			cfg: &config.Config{DBType: "postgres"},
			db:  storage,
		},
		client: client,
	}

	var logs bytes.Buffer
	previousLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&logs, nil)))
	t.Cleanup(func() { slog.SetDefault(previousLogger) })

	err := service.Sync(context.Background(), &SyncParams{Entities: "all", DryRun: true})

	require.NoError(t, err)
	storage.AssertNotCalled(t, "Save", mock.Anything, mock.Anything, mock.Anything)
	storage.AssertNotCalled(t, "SaveSyncStatus", mock.Anything, mock.Anything)
	storage.AssertNotCalled(t, "AcquireSyncLock", mock.Anything)
	require.NotContains(t, logs.String(), "sensitive account name")
	require.NotContains(t, logs.String(), "private category")
	require.NotContains(t, logs.String(), "sensitive-deletion-id")

	var summaryLog struct {
		Message         string         `json:"msg"`
		ServerTimestamp int64          `json:"server_timestamp"`
		Counts          map[string]int `json:"counts"`
		Total           int            `json:"total"`
	}
	for _, line := range strings.Split(strings.TrimSpace(logs.String()), "\n") {
		var entry struct {
			Message string `json:"msg"`
		}
		require.NoError(t, json.Unmarshal([]byte(line), &entry))
		if entry.Message == "Dry run sync summary" {
			require.NoError(t, json.Unmarshal([]byte(line), &summaryLog))
			break
		}
	}

	require.Equal(t, "Dry run sync summary", summaryLog.Message)
	require.Equal(t, int64(777), summaryLog.ServerTimestamp)
	require.Equal(t, 3, summaryLog.Total)
	require.Equal(t, map[string]int{
		"instruments":      0,
		"countries":        0,
		"companies":        0,
		"users":            0,
		"accounts":         1,
		"tags":             1,
		"merchants":        0,
		"budgets":          0,
		"reminders":        0,
		"reminder_markers": 0,
		"transactions":     0,
		"deletions":        1,
	}, summaryLog.Counts)
}

func TestSyncHoldsLockAcrossCursorFetchAndSave(t *testing.T) {
	storage := mocks.NewStorage(t)
	events := make([]string, 0, 5)
	lock := &testSyncLock{onUnlock: func() { events = append(events, "unlock") }}
	storage.On("AcquireSyncLock", mock.Anything).
		Run(func(mock.Arguments) { events = append(events, "lock") }).
		Return(lock, nil).
		Once()
	storage.On("GetLastSyncStatus", mock.Anything).
		Run(func(mock.Arguments) { events = append(events, "cursor") }).
		Return(interfaces.SyncStatus{}, nil).
		Once()
	response := models.Response{ServerTimestamp: 42}
	storage.On("Save", mock.Anything, &response, interfaces.SaveOptions{BatchSize: interfaces.DefaultBatchSize}).
		Run(func(mock.Arguments) { events = append(events, "save") }).
		Return(nil).
		Once()
	service := &SyncService{
		app: &Application{cfg: &config.Config{DBType: "postgres"}, db: storage},
		client: &recordingSyncClient{
			response: response,
			events:   &events,
		},
	}

	err := service.Sync(context.Background(), &SyncParams{Entities: "all"})

	require.NoError(t, err)
	require.Equal(t, []string{"lock", "cursor", "fetch", "save", "unlock"}, events)
}

func TestSyncPassesConfiguredBatchSizeToStorage(t *testing.T) {
	storage := mocks.NewStorage(t)
	storage.On("AcquireSyncLock", mock.Anything).Return(&testSyncLock{}, nil).Once()
	storage.On("GetLastSyncStatus", mock.Anything).Return(interfaces.SyncStatus{}, nil).Once()
	response := models.Response{ServerTimestamp: 42}
	storage.On("Save", mock.Anything, &response, interfaces.SaveOptions{BatchSize: 37}).Return(nil).Once()
	service := &SyncService{
		app:    &Application{cfg: &config.Config{DBType: "postgres"}, db: storage},
		client: &recordingSyncClient{response: response},
	}

	err := service.Sync(context.Background(), &SyncParams{Entities: "all", BatchSize: 37})

	require.NoError(t, err)
}

func TestSyncPassesCopyWriteModeToStorage(t *testing.T) {
	storage := mocks.NewStorage(t)
	storage.On("AcquireSyncLock", mock.Anything).Return(&testSyncLock{}, nil).Once()
	storage.On("GetLastSyncStatus", mock.Anything).Return(interfaces.SyncStatus{}, nil).Once()
	response := models.Response{ServerTimestamp: 42}
	storage.On("Save", mock.Anything, &response, interfaces.SaveOptions{
		BatchSize: interfaces.DefaultBatchSize,
		WriteMode: interfaces.WriteModeCopy,
	}).Return(nil).Once()
	service := &SyncService{
		app:    &Application{cfg: &config.Config{DBType: "postgres"}, db: storage},
		client: &recordingSyncClient{response: response},
	}

	err := service.Sync(context.Background(), &SyncParams{Entities: "all", WriteMode: interfaces.WriteModeCopy})

	require.NoError(t, err)
}

func TestSyncReturnsBusyLockWithoutReadingCursorOrCallingAPI(t *testing.T) {
	storage := mocks.NewStorage(t)
	storage.On("AcquireSyncLock", mock.Anything).
		Return(nil, interfaces.ErrSyncAlreadyRunning).
		Once()
	client := &recordingSyncClient{}
	service := &SyncService{
		app:    &Application{cfg: &config.Config{DBType: "postgres"}, db: storage},
		client: client,
	}

	err := service.Sync(context.Background(), &SyncParams{Entities: "all"})

	require.ErrorIs(t, err, interfaces.ErrSyncAlreadyRunning)
	require.ErrorContains(t, err, "another exporter instance")
	require.Empty(t, client.method)
	storage.AssertNotCalled(t, "GetLastSyncStatus", mock.Anything)
	storage.AssertNotCalled(t, "Save", mock.Anything, mock.Anything, mock.Anything)
}

func TestSyncReturnsUnlockFailureAfterSaving(t *testing.T) {
	unlockErr := errors.New("unlock failed")
	storage := mocks.NewStorage(t)
	storage.On("AcquireSyncLock", mock.Anything).
		Return(&testSyncLock{err: unlockErr}, nil).
		Once()
	storage.On("GetLastSyncStatus", mock.Anything).Return(interfaces.SyncStatus{}, nil).Once()
	response := models.Response{ServerTimestamp: 42}
	storage.On("Save", mock.Anything, &response, interfaces.SaveOptions{BatchSize: interfaces.DefaultBatchSize}).Return(nil).Once()
	service := &SyncService{
		app:    &Application{cfg: &config.Config{DBType: "postgres"}, db: storage},
		client: &recordingSyncClient{response: response},
	}

	err := service.Sync(context.Background(), &SyncParams{Entities: "all"})

	require.ErrorIs(t, err, unlockErr)
	require.ErrorContains(t, err, "release sync lock")
}

func TestDaemonSyncRejectsNonPositiveIntervals(t *testing.T) {
	service := &SyncService{}

	for _, interval := range []int{0, -1} {
		err := service.DaemonSync(context.Background(), &SyncParams{}, interval)
		require.ErrorContains(t, err, "greater than zero")
	}
}

func TestRunDaemonSyncUsesExponentialRetryAndResetsAfterSuccess(t *testing.T) {
	syncError := errors.New("sync failed")
	stopError := errors.New("stop test loop")
	results := []error{syncError, syncError, nil, syncError}
	syncCalls := 0
	waits := make([]time.Duration, 0, len(results))

	err := runDaemonSync(
		context.Background(),
		func(context.Context, *SyncParams) error {
			result := results[syncCalls]
			syncCalls++
			return result
		},
		&SyncParams{},
		10*time.Second,
		&exponentialBackoff{initial: time.Second, maximum: 4 * time.Second},
		func(_ context.Context, duration time.Duration) error {
			waits = append(waits, duration)
			if len(waits) == len(results) {
				return stopError
			}
			return nil
		},
	)

	require.ErrorIs(t, err, stopError)
	require.Equal(t, len(results), syncCalls, "the first sync must run before the first wait")
	require.Equal(t, []time.Duration{
		time.Second,
		2 * time.Second,
		10 * time.Second,
		time.Second,
	}, waits)
}

func TestRunDaemonSyncTreatsBusyLockAsNormalInterval(t *testing.T) {
	stopError := errors.New("stop test loop")
	waits := make([]time.Duration, 0, 2)
	results := []error{errors.Join(errors.New("acquire sync lock"), interfaces.ErrSyncAlreadyRunning), errors.New("real failure")}
	syncCalls := 0

	err := runDaemonSync(
		context.Background(),
		func(context.Context, *SyncParams) error {
			result := results[syncCalls]
			syncCalls++
			return result
		},
		&SyncParams{},
		10*time.Second,
		&exponentialBackoff{initial: time.Second, maximum: 4 * time.Second},
		func(_ context.Context, duration time.Duration) error {
			waits = append(waits, duration)
			if len(waits) == len(results) {
				return stopError
			}
			return nil
		},
	)

	require.ErrorIs(t, err, stopError)
	require.Equal(t, []time.Duration{10 * time.Second, time.Second}, waits)
}

func TestRunDaemonSyncCancellationInterruptsWaitCleanly(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	syncStarted := make(chan struct{})
	result := make(chan error, 1)

	go func() {
		result <- runDaemonSync(
			ctx,
			func(context.Context, *SyncParams) error {
				close(syncStarted)
				return nil
			},
			&SyncParams{},
			time.Hour,
			&exponentialBackoff{initial: time.Second, maximum: 4 * time.Second},
			waitForContext,
		)
	}()

	<-syncStarted
	cancel()

	select {
	case err := <-result:
		require.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("daemon did not stop after context cancellation")
	}
}
