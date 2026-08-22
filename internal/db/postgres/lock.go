package postgres

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nemirlev/zenmoney-export/v2/internal/interfaces"
)

// Use a two-key advisory lock so its namespace is easy to identify and remains
// stable across exporter releases. The lock is database-wide by design.
const (
	syncAdvisoryLockNamespace int32 = 0x5A454E // "ZEN"
	syncAdvisoryLockKey       int32 = 1
)

type syncLockConnection interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Release()
	Destroy(ctx context.Context) error
}

type syncLockConnectionAcquirer interface {
	Acquire(ctx context.Context) (syncLockConnection, error)
}

type pgxPoolLockConnectionAcquirer struct {
	pool PgxIface
}

func (a pgxPoolLockConnectionAcquirer) Acquire(ctx context.Context) (syncLockConnection, error) {
	conn, err := a.pool.Acquire(ctx)
	if err != nil {
		return nil, err
	}
	return &pgxPoolLockConnection{Conn: conn}, nil
}

type pgxPoolLockConnection struct {
	*pgxpool.Conn
}

func (c *pgxPoolLockConnection) Destroy(ctx context.Context) error {
	return c.Hijack().Close(ctx)
}

// AcquireSyncLock attempts to acquire the exporter lock without waiting.
func (s *DB) AcquireSyncLock(ctx context.Context) (interfaces.SyncLock, error) {
	return acquireSyncLock(ctx, pgxPoolLockConnectionAcquirer{pool: s.pool})
}

func acquireSyncLock(ctx context.Context, acquirer syncLockConnectionAcquirer) (interfaces.SyncLock, error) {
	conn, err := acquirer.Acquire(ctx)
	if err != nil {
		return nil, fmt.Errorf("acquire dedicated postgres connection: %w", err)
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
			destroySyncLockConnection(conn, ctx),
		)
	}
	if !acquired {
		conn.Release()
		return nil, interfaces.ErrSyncAlreadyRunning
	}

	return &postgresSyncLock{conn: conn}, nil
}

type postgresSyncLock struct {
	mu   sync.Mutex
	conn syncLockConnection
}

func (l *postgresSyncLock) Unlock(ctx context.Context) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.conn == nil {
		return nil
	}
	conn := l.conn
	l.conn = nil

	var unlocked bool
	err := conn.QueryRow(
		ctx,
		"SELECT pg_advisory_unlock($1, $2)",
		syncAdvisoryLockNamespace,
		syncAdvisoryLockKey,
	).Scan(&unlocked)
	if err != nil {
		return errors.Join(
			fmt.Errorf("release postgres advisory lock: %w", err),
			destroySyncLockConnection(conn, ctx),
		)
	}
	if !unlocked {
		return errors.Join(
			errors.New("postgres advisory lock was not held by its dedicated connection"),
			destroySyncLockConnection(conn, ctx),
		)
	}

	conn.Release()
	return nil
}

func destroySyncLockConnection(conn syncLockConnection, ctx context.Context) error {
	if err := conn.Destroy(context.WithoutCancel(ctx)); err != nil {
		return fmt.Errorf("destroy dedicated postgres connection: %w", err)
	}
	return nil
}
