package postgres

import (
	"context"
	"fmt"
	"strconv"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nemirlev/zenmoney-export/v2/internal/interfaces"
)

// decimalString converts an SDK float into the shortest non-exponential decimal
// representation that round-trips to the same float. PostgreSQL receives this as
// text and parses it as NUMERIC, avoiding an additional binary-float conversion
// in the database protocol.
func decimalString(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}

func optionalDecimalString(value *float64) any {
	if value == nil {
		// Preserve the typed nil for callers and pgxmock while pgx still encodes
		// it as SQL NULL before considering the target text format.
		return value
	}

	return decimalString(*value)
}

type DB struct {
	pool PgxIface
}

// batchSender is implemented by both pgxpool.Pool and pgx.Tx. Keeping the
// persistence helpers in terms of this small interface lets Save run every
// batch on its transaction while the public CRUD methods continue to use the
// pool directly.
type batchSender interface {
	SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults
}

type commandExecutor interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// PgxIface — interface for pgxpool.Pool
type PgxIface interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)

	Begin(ctx context.Context) (pgx.Tx, error)
	BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error)

	SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults

	Acquire(ctx context.Context) (*pgxpool.Conn, error)
	AcquireAllIdle(ctx context.Context) []*pgxpool.Conn
	AcquireFunc(ctx context.Context, f func(*pgxpool.Conn) error) error

	Stat() *pgxpool.Stat
	Config() *pgxpool.Config

	Reset()
	Close()
	Ping(ctx context.Context) error
}

// NewPostgresStorage creates a new PostgreSQL storage instance
func NewPostgresStorage(connectionString string) (interfaces.Storage, error) {
	config, err := pgxpool.ParseConfig(connectionString)
	if err != nil {
		return nil, fmt.Errorf("failed to parse postgres config: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return nil, fmt.Errorf("failed to create postgres connection pool: %w", err)
	}

	return &DB{
		pool: pool,
	}, nil
}

// Close closes the database connection pool
func (s *DB) Close(ctx context.Context) error {
	s.pool.Close()
	return nil
}

// Ping checks if the database is accessible
func (s *DB) Ping(ctx context.Context) error {
	return s.pool.Ping(ctx)
}
