package clickhouse

import (
	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

// Connector интерфейс для создания соединений с ClickHouse.
type Connector interface {
	Open(options *clickhouse.Options) (driver.Conn, error)
}

// DefaultConnector реализация интерфейса Connector, использующая clickhouse.Open.
type DefaultConnector struct{}

func (d *DefaultConnector) Open(options *clickhouse.Options) (driver.Conn, error) {
	return clickhouse.Open(options)
}
