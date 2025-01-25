package postgres

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nemirlev/zenmoney-export/v2/internal/interfaces"
)

type DB struct {
	pool *pgxpool.Pool
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
