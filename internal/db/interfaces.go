// internal/db/interfaces.go

package db

import (
	"context"
	"time"

	"github.com/nemirlev/zenmoney-go-sdk/v2/models"
)

// StorageType represents the type of storage
type StorageType string

const (
	PostgresStorage   StorageType = "postgres"
	MySQLStorage      StorageType = "mysql"
	MongoStorage      StorageType = "mongodb"
	ClickhouseStorage StorageType = "clickhouse"
	RedisStorage      StorageType = "redis"
	InMemoryStorage   StorageType = "memory"
)

// Storage is an interface for working with the database
type Storage interface {
	Close(ctx context.Context) error
	Ping(ctx context.Context) error

	SaveSyncStatus(ctx context.Context, status SyncStatus) error
	GetLastSyncStatus(ctx context.Context) (SyncStatus, error)

	Save(ctx context.Context, response *models.Response) error

	SaveInstruments(ctx context.Context, instruments []models.Instrument) error
	SaveCountries(ctx context.Context, countries []models.Country) error
	SaveCompanies(ctx context.Context, companies []models.Company) error
	SaveUsers(ctx context.Context, users []models.User) error
	SaveAccounts(ctx context.Context, accounts []models.Account) error
	SaveTags(ctx context.Context, tags []models.Tag) error
	SaveMerchants(ctx context.Context, merchants []models.Merchant) error
	SaveBudgets(ctx context.Context, budgets []models.Budget) error
	SaveReminders(ctx context.Context, reminders []models.Reminder) error
	SaveReminderMarkers(ctx context.Context, markers []models.ReminderMarker) error
	SaveTransactions(ctx context.Context, transactions []models.Transaction) error

	DeleteObjects(ctx context.Context, deletions []models.Deletion) error

	GetInstrument(ctx context.Context, id int) (*models.Instrument, error)
	ListInstruments(ctx context.Context, filter Filter) ([]models.Instrument, error)
	CreateInstrument(ctx context.Context, instrument *models.Instrument) error
	UpdateInstrument(ctx context.Context, instrument *models.Instrument) error
	DeleteInstrument(ctx context.Context, id int) error

	GetCompany(ctx context.Context, id int) (*models.Company, error)
	ListCompanies(ctx context.Context, filter Filter) ([]models.Company, error)
	CreateCompany(ctx context.Context, company *models.Company) error
	UpdateCompany(ctx context.Context, company *models.Company) error
	DeleteCompany(ctx context.Context, id int) error

	GetUser(ctx context.Context, id int) (*models.User, error)
	ListUsers(ctx context.Context, filter Filter) ([]models.User, error)
	CreateUser(ctx context.Context, user *models.User) error
	UpdateUser(ctx context.Context, user *models.User) error
	DeleteUser(ctx context.Context, id int) error

	GetCountry(ctx context.Context, id int) (*models.Country, error)
	ListCountries(ctx context.Context, filter Filter) ([]models.Country, error)
	CreateCountry(ctx context.Context, country *models.Country) error
	UpdateCountry(ctx context.Context, country *models.Country) error
	DeleteCountry(ctx context.Context, id int) error

	GetAccount(ctx context.Context, id string) (*models.Account, error)
	ListAccounts(ctx context.Context, filter Filter) ([]models.Account, error)
	CreateAccount(ctx context.Context, account *models.Account) error
	UpdateAccount(ctx context.Context, account *models.Account) error
	DeleteAccount(ctx context.Context, id string) error

	GetTag(ctx context.Context, id string) (*models.Tag, error)
	ListTags(ctx context.Context, filter Filter) ([]models.Tag, error)
	CreateTag(ctx context.Context, tag *models.Tag) error
	UpdateTag(ctx context.Context, tag *models.Tag) error
	DeleteTag(ctx context.Context, id string) error

	GetMerchant(ctx context.Context, id string) (*models.Merchant, error)
	ListMerchants(ctx context.Context, filter Filter) ([]models.Merchant, error)
	CreateMerchant(ctx context.Context, merchant *models.Merchant) error
	UpdateMerchant(ctx context.Context, merchant *models.Merchant) error
	DeleteMerchant(ctx context.Context, id string) error

	GetBudget(ctx context.Context, userID int, tagID string, date time.Time) (*models.Budget, error)
	ListBudgets(ctx context.Context, filter Filter) ([]models.Budget, error)
	CreateBudget(ctx context.Context, budget *models.Budget) error
	UpdateBudget(ctx context.Context, budget *models.Budget) error
	DeleteBudget(ctx context.Context, userID int, tagID string, date time.Time) error

	GetReminder(ctx context.Context, id string) (*models.Reminder, error)
	ListReminders(ctx context.Context, filter Filter) ([]models.Reminder, error)
	CreateReminder(ctx context.Context, reminder *models.Reminder) error
	UpdateReminder(ctx context.Context, reminder *models.Reminder) error
	DeleteReminder(ctx context.Context, id string) error

	GetReminderMarker(ctx context.Context, id string) (*models.ReminderMarker, error)
	ListReminderMarkers(ctx context.Context, filter Filter) ([]models.ReminderMarker, error)
	CreateReminderMarker(ctx context.Context, marker *models.ReminderMarker) error
	UpdateReminderMarker(ctx context.Context, marker *models.ReminderMarker) error
	DeleteReminderMarker(ctx context.Context, id string) error

	GetTransaction(ctx context.Context, id string) (*models.Transaction, error)
	ListTransactions(ctx context.Context, filter Filter) ([]models.Transaction, error)
	CreateTransaction(ctx context.Context, tx *models.Transaction) error
	UpdateTransaction(ctx context.Context, tx *models.Transaction) error
	DeleteTransaction(ctx context.Context, id string) error
}

// Filter is a filter for listing objects
type Filter struct {
	UserID    *int       `json:"userId,omitempty"`
	StartDate *time.Time `json:"startDate,omitempty"`
	EndDate   *time.Time `json:"endDate,omitempty"`
	Page      int        `json:"page"`
	Limit     int        `json:"limit"`
}

// SyncStatus is a status of the synchronization process
type SyncStatus struct {
	ID               int64
	StartedAt        time.Time
	FinishedAt       *time.Time
	SyncType         string // full, partial, force
	ServerTimestamp  int64
	RecordsProcessed int
	Status           string // completed, failed
	ErrorMessage     *string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}
