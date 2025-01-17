package clickhouse

import (
	"errors"
	"testing"

	"github.com/ClickHouse/clickhouse-go/v2"
	driverMocks "github.com/nemirlev/zenexport/tests/mocks/github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	clickhouseMocks "github.com/nemirlev/zenexport/tests/mocks/internal_/db/clickhouse"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type MockConnector struct {
	clickhouseMocks.Connector
}

func TestDefaultConnector_Open(t *testing.T) {
	mockConn := new(driverMocks.Conn)
	mockConnector := new(clickhouseMocks.Connector)

	options := &clickhouse.Options{
		Addr: []string{"localhost:9000"},
		Auth: clickhouse.Auth{
			Database: "test_db",
			Username: "user",
			Password: "password",
		},
	}

	mockConnector.On("Open", options).Return(mockConn, nil)

	conn, err := mockConnector.Open(options)

	require.NoError(t, err)
	assert.Equal(t, mockConn, conn)
	mockConnector.AssertExpectations(t)
}

func TestDefaultConnector_Open_Error(t *testing.T) {
	mockConnector := new(clickhouseMocks.Connector)

	options := &clickhouse.Options{
		Addr: []string{"localhost:9000"},
		Auth: clickhouse.Auth{
			Database: "test_db",
			Username: "user",
			Password: "password",
		},
	}

	mockConnector.On("Open", options).Return(nil, errors.New("connection error"))

	conn, err := mockConnector.Open(options)

	assert.Error(t, err)
	assert.Nil(t, conn)
	assert.Equal(t, "connection error", err.Error())
	mockConnector.AssertExpectations(t)
}
