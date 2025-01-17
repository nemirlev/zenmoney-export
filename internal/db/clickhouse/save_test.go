package clickhouse

import (
	"context"
	"errors"
	"testing"

	"github.com/nemirlev/zenexport/internal/logger"
	driverMocks "github.com/nemirlev/zenexport/tests/mocks/github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	clickhouseMocks "github.com/nemirlev/zenexport/tests/mocks/internal_/db/clickhouse"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func setupStore() (*Store, *driverMocks.Conn, *driverMocks.Batch) {
	mockConn := new(driverMocks.Conn)
	mockBatch := new(driverMocks.Batch)
	store := &Store{
		Conn: mockConn,
		Log:  logger.New(),
	}
	return store, mockConn, mockBatch
}

func TestStore_SaveBatch(t *testing.T) {
	store, mockConn, mockBatch := setupStore()

	tableName := "test_table"
	query := "INSERT INTO test_table VALUES (?, ?)"
	data := [][]interface{}{
		{1, "test1"},
		{2, "test2"},
	}

	mockConn.On("Exec", mock.Anything, "TRUNCATE TABLE IF EXISTS "+tableName).Return(nil)
	mockConn.On("PrepareBatch", mock.Anything, query).Return(mockBatch, nil)
	mockBatch.On("Append", 1, "test1").Return(nil)
	mockBatch.On("Append", 2, "test2").Return(nil)
	mockBatch.On("Send").Return(nil)

	err := store.saveBatch(context.Background(), tableName, query, data)

	assert.NoError(t, err)
	mockConn.AssertExpectations(t)
	mockBatch.AssertExpectations(t)
}

func TestStore_SaveBatch_TruncateError(t *testing.T) {
	store, mockConn, _ := setupStore()

	tableName := "test_table"
	query := "INSERT INTO test_table VALUES (?, ?)"
	data := [][]interface{}{
		{1, "test1"},
		{2, "test2"},
	}

	mockConn.On("Exec", mock.Anything, "TRUNCATE TABLE IF EXISTS "+tableName).Return(errors.New("truncate error"))

	err := store.saveBatch(context.Background(), tableName, query, data)

	assert.Error(t, err)
	assert.Equal(t, "truncate error", err.Error())
	mockConn.AssertExpectations(t)
}

func TestStore_SaveBatch_PrepareBatchError(t *testing.T) {
	store, mockConn, _ := setupStore()

	tableName := "test_table"
	query := "INSERT INTO test_table VALUES (?, ?)"
	data := [][]interface{}{
		{1, "test1"},
		{2, "test2"},
	}

	mockConn.On("Exec", mock.Anything, "TRUNCATE TABLE IF EXISTS "+tableName).Return(nil)
	mockConn.On("PrepareBatch", mock.Anything, query).Return(nil, errors.New("prepare batch error"))

	err := store.saveBatch(context.Background(), tableName, query, data)

	assert.Error(t, err)
	assert.Equal(t, "prepare batch error", err.Error())
	mockConn.AssertExpectations(t)
}

func TestStore_SaveBatch_AppendError(t *testing.T) {
	store, mockConn, mockBatch := setupStore()

	tableName := "test_table"
	query := "INSERT INTO test_table VALUES (?, ?)"
	data := [][]interface{}{
		{1, "test1"},
		{2, "test2"},
	}

	mockConn.On("Exec", mock.Anything, "TRUNCATE TABLE IF EXISTS "+tableName).Return(nil)
	mockConn.On("PrepareBatch", mock.Anything, query).Return(mockBatch, nil)
	mockBatch.On("Append", 1, "test1").Return(errors.New("append error"))

	err := store.saveBatch(context.Background(), tableName, query, data)

	assert.Error(t, err)
	assert.Equal(t, "append error", err.Error())
	mockConn.AssertExpectations(t)
	mockBatch.AssertExpectations(t)
}

func TestStore_SaveBatch_SendError(t *testing.T) {
	store, mockConn, mockBatch := setupStore()

	tableName := "test_table"
	query := "INSERT INTO test_table VALUES (?, ?)"
	data := [][]interface{}{
		{1, "test1"},
		{2, "test2"},
	}

	mockConn.On("Exec", mock.Anything, "TRUNCATE TABLE IF EXISTS "+tableName).Return(nil)
	mockConn.On("PrepareBatch", mock.Anything, query).Return(mockBatch, nil)
	mockBatch.On("Append", 1, "test1").Return(nil)
	mockBatch.On("Append", 2, "test2").Return(nil)
	mockBatch.On("Send").Return(errors.New("send error"))

	err := store.saveBatch(context.Background(), tableName, query, data)

	assert.Error(t, err)
	assert.Equal(t, "send error", err.Error())
	mockConn.AssertExpectations(t)
	mockBatch.AssertExpectations(t)
}

func TestStore_Save(t *testing.T) {
	mockConn := new(driverMocks.Conn)
	mockConn.On("Close").Return(nil)

	mockStore := &clickhouseMocks.Store{}
	mockStore.Conn = mockConn
	mockStore.Log = logger.New()

	ctx := context.Background()

	mockStore.On("connect").Return(nil)
	mockStore.On("saveInstruments", ctx, mock.Anything).Return(nil)
	mockStore.On("saveCountries", ctx, mock.Anything).Return(nil)
	mockStore.On("saveCompanies", ctx, mock.Anything).Return(nil)
	mockStore.On("saveUsers", ctx, mock.Anything).Return(nil)
	mockStore.On("saveAccounts", ctx, mock.Anything).Return(nil)
	mockStore.On("saveTags", ctx, mock.Anything).Return(nil)
	mockStore.On("saveMerchants", ctx, mock.Anything).Return(nil)
	mockStore.On("saveBudgets", ctx, mock.Anything).Return(nil)
	mockStore.On("saveReminders", ctx, mock.Anything).Return(nil)
	mockStore.On("saveReminderMarkers", ctx, mock.Anything).Return(nil)
	mockStore.On("saveTransactions", ctx, mock.Anything).Return(nil)

	data := test.GetTestResponse()

	err := mockStore.Save(data)
	assert.NoError(t, err)

	mockConn.AssertExpectations(t)
	mockStore.AssertExpectations(t)
}

func TestStore_Save_ErrorOnConnect(t *testing.T) {
	mockStore := &clickhouseMocks.Store{}
	mockStore.Log = logger.New()

	mockStore.On("connect").Return(errors.New("connection error"))

	data := test.GetTestResponse()

	err := mockStore.Save(data)
	assert.Error(t, err)
	assert.Equal(t, "connection error", err.Error())
}

func TestStore_Save_ErrorOnSave(t *testing.T) {
	mockConn := new(driverMocks.Conn)
	mockConn.On("Close").Return(nil)

	mockStore := &clickhouseMocks.Store{}
	mockStore.Conn = mockConn
	mockStore.Log = logger.New()

	ctx := context.Background()

	mockStore.On("connect").Return(nil)
	mockStore.On("saveInstruments", ctx, mock.Anything).Return(errors.New("save error"))

	data := test.GetTestResponse()

	err := mockStore.Save(data)
	assert.Error(t, err)
	assert.Equal(t, "save error", err.Error())

	mockConn.AssertExpectations(t)
	mockStore.AssertExpectations(t)
}
