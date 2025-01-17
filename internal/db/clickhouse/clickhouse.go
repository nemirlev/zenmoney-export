package clickhouse

import (
	"context"
	"errors"
	"fmt"
	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/nemirlev/zenexport/internal/config"
	"github.com/nemirlev/zenexport/internal/logger"
)

type Store struct {
	Conn      driver.Conn
	Log       logger.Log
	Config    *config.Config
	Connector Connector
}

// connect устанавливает соединение с базой данных ClickHouse, используя параметры, указанные в конфигурации.
// Параметры:
// - s: указатель на структуру Store, содержащую конфигурацию и логгер.
func (s *Store) connect() error {
	var (
		ctx = context.Background()
	)

	conn, err := s.Connector.Open(&clickhouse.Options{
		Addr: []string{fmt.Sprintf("%s:9000", s.Config.ClickhouseServer)},
		Auth: clickhouse.Auth{
			Database: s.Config.ClickhouseDB,
			Username: s.Config.ClickhouseUser,
			Password: s.Config.ClickhousePassword,
		},
		Debugf: func(format string, v ...interface{}) {
			s.Log.Debug(format, v)
		},
	})

	if err != nil {
		return err
	}

	if err := conn.Ping(ctx); err != nil {
		var exception *clickhouse.Exception
		if errors.As(err, &exception) {
			s.Log.WithError(err, "Exception [%d] %s \n%s\n", exception.Code, exception.Message, exception.StackTrace)
		}
		return err
	}

	s.Conn = conn
	return nil
}

// executeBatch выполняет пакетный запрос в ClickHouse.
// Параметры:
// - ctx: контекст для управления временем выполнения и отменой запроса.
// - query: строка с SQL-запросом для выполнения пакетного вставки данных.
// - data: срез с данными, которые будут вставлены в таблицу.
func (s *Store) executeBatch(ctx context.Context, query string, data [][]interface{}) error {
	batch, err := s.Conn.PrepareBatch(ctx, query)
	if err != nil {
		s.Log.WithError(err, "error on prepare batch Clickhouse")
		return err
	}

	for _, item := range data {
		if err := batch.Append(item...); err != nil {
			s.Log.WithError(err, "error append batch in clickhouse")
			return err
		}
	}

	if err := batch.Send(); err != nil {
		s.Log.WithError(err, "error send batch in clickhouse")
		return err
	}
	return nil
}

// truncateTable очищает указанную таблицу в базе данных ClickHouse.
// Параметры:
// - ctx: контекст для управления временем выполнения и отменой запроса.
// - tableName: имя таблицы, которую необходимо очистить.
func (s *Store) truncateTable(ctx context.Context, tableName string) error {
	query := fmt.Sprintf("TRUNCATE TABLE IF EXISTS %s", tableName)
	return s.Conn.Exec(ctx, query)
}
