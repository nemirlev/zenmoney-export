package clickhouse

import (
	"context"
	"errors"
	"testing"

	"github.com/nemirlev/zenexport/internal/config"
	"github.com/nemirlev/zenexport/internal/logger"
	driverMocks "github.com/nemirlev/zenexport/tests/mocks/github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	clickhouseMocks "github.com/nemirlev/zenexport/tests/mocks/internal_/db/clickhouse"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestStore_Connect(t *testing.T) {
	mockConn := new(driverMocks.Conn)
	mockConn.On("Ping", mock.Anything).Return(nil)

	mockConnector := new(clickhouseMocks.Connector)
	mockConnector.On("Open", mock.Anything).Return(mockConn, nil)

	cfg := &config.Config{
		ClickhouseServer:   "localhost",
		ClickhouseDB:       "test_db",
		ClickhouseUser:     "user",
		ClickhousePassword: "password",
	}

	log := logger.New()
	store := &Store{
		Log:       log,
		Config:    cfg,
		Connector: mockConnector,
	}

	err := store.connect()

	assert.NoError(t, err)
	mockConn.AssertExpectations(t)
	mockConnector.AssertExpectations(t)
}

func TestStore_Connect_Error(t *testing.T) {
	mockConn := new(driverMocks.Conn)
	mockConn.On("Ping", mock.Anything).Return(errors.New("ping error"))

	mockConnector := new(clickhouseMocks.Connector)
	mockConnector.On("Open", mock.Anything).Return(mockConn, nil)

	cfg := &config.Config{
		ClickhouseServer:   "localhost",
		ClickhouseDB:       "test_db",
		ClickhouseUser:     "user",
		ClickhousePassword: "password",
	}

	log := logger.New()
	store := &Store{
		Log:       log,
		Config:    cfg,
		Connector: mockConnector,
	}

	err := store.connect()

	assert.Error(t, err)
	assert.Equal(t, "ping error", err.Error())
	mockConn.AssertExpectations(t)
	mockConnector.AssertExpectations(t)
}

func TestStore_ExecuteBatch(t *testing.T) {
	mockConn := new(driverMocks.Conn)
	mockBatch := new(driverMocks.Batch)
	ctx := context.Background()
	query := "INSERT INTO test_table VALUES (?, ?)"
	data := [][]interface{}{
		{1, "test1"},
		{2, "test2"},
	}

	mockConn.On("PrepareBatch", ctx, query).Return(mockBatch, nil)
	mockBatch.On("Append", 1, "test1").Return(nil).Once()
	mockBatch.On("Append", 2, "test2").Return(nil).Once()
	mockBatch.On("Send").Return(nil)

	log := logger.New()
	store := &Store{
		Conn: mockConn,
		Log:  log,
	}

	err := store.executeBatch(ctx, query, data)

	assert.NoError(t, err)
	mockConn.AssertExpectations(t)
	mockBatch.AssertExpectations(t)
}

func TestStore_ExecuteBatch_PrepareError(t *testing.T) {
	mockConn := new(driverMocks.Conn)
	ctx := context.Background()
	query := "INSERT INTO test_table VALUES (?, ?)"
	data := [][]interface{}{
		{1, "test1"},
		{2, "test2"},
	}

	mockConn.On("PrepareBatch", ctx, query).Return(nil, errors.New("prepare error"))

	log := logger.New()
	store := &Store{
		Conn: mockConn,
		Log:  log,
	}

	err := store.executeBatch(ctx, query, data)

	assert.Error(t, err)
	assert.Equal(t, "prepare error", err.Error())
	mockConn.AssertExpectations(t)
}

func TestStore_ExecuteBatch_AppendError(t *testing.T) {
	mockConn := new(driverMocks.Conn)
	mockBatch := new(driverMocks.Batch)
	ctx := context.Background()
	query := "INSERT INTO test_table VALUES (?, ?)"
	data := [][]interface{}{
		{1, "test1"},
	}

	mockConn.On("PrepareBatch", ctx, query).Return(mockBatch, nil)
	mockBatch.On("Append", 1, "test1").Return(errors.New("append error"))

	log := logger.New()
	store := &Store{
		Conn: mockConn,
		Log:  log,
	}

	err := store.executeBatch(ctx, query, data)

	assert.Error(t, err)
	assert.Equal(t, "append error", err.Error())
	mockConn.AssertExpectations(t)
	mockBatch.AssertExpectations(t)
}

func TestStore_TruncateTable(t *testing.T) {
	mockConn := new(driverMocks.Conn)
	ctx := context.Background()
	tableName := "test_table"

	query := "TRUNCATE TABLE IF EXISTS " + tableName
	mockConn.On("Exec", ctx, query).Return(nil)

	log := logger.New()
	store := &Store{
		Conn: mockConn,
		Log:  log,
	}

	err := store.truncateTable(ctx, tableName)

	assert.NoError(t, err)
	mockConn.AssertExpectations(t)
}
