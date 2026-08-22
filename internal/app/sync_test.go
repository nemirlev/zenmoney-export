package app

import (
	"context"
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
}

func (c *recordingSyncClient) FullSync(context.Context) (models.Response, error) {
	c.method = "full"
	return c.response, nil
}

func (c *recordingSyncClient) SyncSince(_ context.Context, since time.Time) (models.Response, error) {
	c.method = "since"
	c.since = since
	return c.response, nil
}

func (c *recordingSyncClient) ForceSyncEntitiesSince(
	_ context.Context,
	since time.Time,
	entities ...models.EntityType,
) (models.Response, error) {
	c.method = "force-entities"
	c.since = since
	c.entities = append([]models.EntityType(nil), entities...)
	return c.response, nil
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
			storage.On("GetLastSyncStatus", mock.Anything).Return(tt.lastSync, nil).Once()

			response := models.Response{ServerTimestamp: 999}
			storage.On("Save", mock.Anything, &response).Return(nil).Once()
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
		})
	}
}
