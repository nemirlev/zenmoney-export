package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nemirlev/zenmoney-export/v2/internal/interfaces"
)

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
