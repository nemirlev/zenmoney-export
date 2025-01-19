package db

import (
	"context"
	"fmt"
)

// NewStorage создает новое хранилище указанного типа
// Usage example:
/*
func main() {
    storage, err := NewStorage(ctx, PostgresStorage, "postgres://...")
    if err != nil {
        log.Fatal(err)
    }

    // Get Transactions on period
    transactions, err := storage.ListTransactions(ctx, Filter{
        UserID:    &userID,
        StartDate: &startDate,
        EndDate:   &endDate,
        Page:      1,
        Limit:     100,
    })

	// Save all Zen Money Response (github.com/nemirlev/zenmoney-go-sdk/v2/models.Response)
	err = storage.Save(ctx, response)

    // Bulk operations in sync process
    err = storage.SaveTransactions(ctx, newTransactions)

	// Delete from delete array (github.com/nemirlev/zenmoney-go-sdk/v2/models.Deletion)
    err = storage.DeleteObjects(ctx, []Deletions{
				{Object: "transaction", ID: 123},
				{Object: "transaction", ID: 124},)
}
*/
func NewStorage(ctx context.Context, storageType StorageType, connectionString string) (Storage, error) {
	switch storageType {
	case PostgresStorage:
		return NewPostgresStorage(connectionString)
	//case MySQLStorage:
	//	return NewMySQLStorage(connectionString)
	//case MongoStorage:
	//	return NewMongoStorage(connectionString)
	//case ClickhouseStorage:
	//	return NewClickhouseStorage(connectionString)
	//case RedisStorage:
	//	return NewRedisStorage(connectionString)
	//case InMemoryStorage:
	//	return NewInMemoryStorage()
	default:
		return nil, fmt.Errorf("unsupported storage type: %s", storageType)
	}
}
