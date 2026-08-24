package postgres

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/nemirlev/zenmoney-export/v2/internal/interfaces"
)

// Use a two-key advisory lock so its namespace is easy to identify and remains
// stable across exporter releases. The lock is database-wide by design.
const (
	syncAdvisoryLockNamespace int32 = 0x5A454E // "ZEN"
	syncAdvisoryLockKey       int32 = 1
	syncLockCleanupTimeout          = 5 * time.Second
)

type syncLockConnection interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Close(ctx context.Context) error
}

type syncLockConnectionFactory interface {
	Connect(ctx context.Context, config *pgx.ConnConfig) (syncLockConnection, error)
}

type pgxSyncLockConnectionFactory struct{}

func (pgxSyncLockConnectionFactory) Connect(
	ctx context.Context,
	config *pgx.ConnConfig,
) (syncLockConnection, error) {
	return pgx.ConnectConfig(ctx, config)
}

// AcquireSyncLock attempts to acquire the exporter lock without waiting.
func (s *DB) AcquireSyncLock(ctx context.Context) (interfaces.SyncLock, error) {
	return s.acquireSyncLock(ctx, pgxSyncLockConnectionFactory{})
}

func (s *DB) acquireSyncLock(
	ctx context.Context,
	factory syncLockConnectionFactory,
) (interfaces.SyncLock, error) {
	poolConfig := s.pool.Config()
	if poolConfig == nil || poolConfig.ConnConfig == nil {
		return nil, errors.New("postgres connection config is not available for sync lock")
	}

	// The advisory lock deliberately uses a raw connection outside pgxpool.
	// Keeping a pooled connection checked out for the complete API fetch would
	// deadlock Save and cursor queries when pool_max_conns=1.
	return acquireSyncLock(ctx, factory, poolConfig.ConnConfig.Copy())
}

func acquireSyncLock(
	ctx context.Context,
	factory syncLockConnectionFactory,
	config *pgx.ConnConfig,
) (interfaces.SyncLock, error) {
	conn, err := factory.Connect(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("open dedicated postgres connection: %w", err)
	}

	var acquired bool
	err = conn.QueryRow(
		ctx,
		"SELECT pg_try_advisory_lock($1, $2)",
		syncAdvisoryLockNamespace,
		syncAdvisoryLockKey,
	).Scan(&acquired)
	if err != nil {
		return nil, errors.Join(
			fmt.Errorf("try postgres advisory lock: %w", err),
			closeSyncLockConnection(conn, ctx, syncLockCleanupTimeout),
		)
	}
	if !acquired {
		return nil, errors.Join(
			interfaces.ErrSyncAlreadyRunning,
			closeSyncLockConnection(conn, ctx, syncLockCleanupTimeout),
		)
	}

	return &postgresSyncLock{conn: conn, cleanupTimeout: syncLockCleanupTimeout}, nil
}

type postgresSyncLock struct {
	mu             sync.Mutex
	conn           syncLockConnection
	cleanupTimeout time.Duration
}

func (l *postgresSyncLock) Unlock(ctx context.Context) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.conn == nil {
		return nil
	}
	conn := l.conn
	l.conn = nil
	cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), l.cleanupTimeout)
	defer cancel()

	var unlocked bool
	err := conn.QueryRow(
		cleanupCtx,
		"SELECT pg_advisory_unlock($1, $2)",
		syncAdvisoryLockNamespace,
		syncAdvisoryLockKey,
	).Scan(&unlocked)
	if err != nil {
		return errors.Join(
			fmt.Errorf("release postgres advisory lock: %w", err),
			closeSyncLockConnectionWithContext(conn, cleanupCtx),
		)
	}
	if !unlocked {
		return errors.Join(
			errors.New("postgres advisory lock was not held by its dedicated connection"),
			closeSyncLockConnectionWithContext(conn, cleanupCtx),
		)
	}

	return closeSyncLockConnectionWithContext(conn, cleanupCtx)
}

func closeSyncLockConnection(
	conn syncLockConnection,
	parent context.Context,
	timeout time.Duration,
) error {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(parent), timeout)
	defer cancel()
	return closeSyncLockConnectionWithContext(conn, ctx)
}

func closeSyncLockConnectionWithContext(conn syncLockConnection, ctx context.Context) error {
	// pgx always closes the underlying net.Conn even if this bounded context has
	// expired, which guarantees that PostgreSQL releases the session lock.
	if err := conn.Close(ctx); err != nil {
		return fmt.Errorf("close dedicated postgres connection: %w", err)
	}
	return nil
}
