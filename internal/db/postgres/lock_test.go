package postgres

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/nemirlev/zenmoney-export/v2/internal/interfaces"
	"github.com/stretchr/testify/require"
)

func TestAcquireSyncLockAndUnlockUseSameDedicatedConnection(t *testing.T) {
	conn := &recordingLockConnection{rows: []lockRow{{value: true}, {value: true}}}

	lock, err := acquireSyncLock(context.Background(), recordingLockAcquirer{conn: conn})
	require.NoError(t, err)
	require.Zero(t, conn.releaseCalls, "the acquired connection must stay checked out")

	err = lock.Unlock(context.Background())
	require.NoError(t, err)
	require.Equal(t, 1, conn.releaseCalls)
	require.Zero(t, conn.destroyCalls)
	require.Equal(t, []string{
		"SELECT pg_try_advisory_lock($1, $2)",
		"SELECT pg_advisory_unlock($1, $2)",
	}, conn.queries)

	// Unlock is intentionally idempotent so deferred cleanup cannot reuse a
	// connection that has already returned to the pool.
	require.NoError(t, lock.Unlock(context.Background()))
	require.Equal(t, 1, conn.releaseCalls)
}

func TestAcquireSyncLockReturnsBusySentinel(t *testing.T) {
	conn := &recordingLockConnection{rows: []lockRow{{value: false}}}

	lock, err := acquireSyncLock(context.Background(), recordingLockAcquirer{conn: conn})

	require.Nil(t, lock)
	require.ErrorIs(t, err, interfaces.ErrSyncAlreadyRunning)
	require.Equal(t, 1, conn.releaseCalls)
	require.Zero(t, conn.destroyCalls)
}

func TestConcurrentSyncLockAttemptIsRejectedUntilOwnerUnlocks(t *testing.T) {
	ownerConn := &recordingLockConnection{rows: []lockRow{{value: true}, {value: true}}}
	contenderConn := &recordingLockConnection{rows: []lockRow{{value: false}}}
	acquirer := &sequenceLockAcquirer{connections: []syncLockConnection{ownerConn, contenderConn}}

	owner, err := acquireSyncLock(context.Background(), acquirer)
	require.NoError(t, err)

	contender, err := acquireSyncLock(context.Background(), acquirer)
	require.Nil(t, contender)
	require.ErrorIs(t, err, interfaces.ErrSyncAlreadyRunning)
	require.Zero(t, ownerConn.releaseCalls, "owner must retain its dedicated connection while contender checks")
	require.Equal(t, 1, contenderConn.releaseCalls)

	require.NoError(t, owner.Unlock(context.Background()))
	require.Equal(t, 1, ownerConn.releaseCalls)
}

func TestAcquireSyncLockDestroysConnectionAfterAmbiguousQueryError(t *testing.T) {
	queryErr := errors.New("query failed")
	conn := &recordingLockConnection{rows: []lockRow{{err: queryErr}}}

	lock, err := acquireSyncLock(context.Background(), recordingLockAcquirer{conn: conn})

	require.Nil(t, lock)
	require.ErrorIs(t, err, queryErr)
	require.Zero(t, conn.releaseCalls)
	require.Equal(t, 1, conn.destroyCalls)
}

func TestUnlockDestroysConnectionWhenAdvisoryUnlockFails(t *testing.T) {
	unlockErr := errors.New("connection interrupted")
	conn := &recordingLockConnection{rows: []lockRow{{value: true}, {err: unlockErr}}}

	lock, err := acquireSyncLock(context.Background(), recordingLockAcquirer{conn: conn})
	require.NoError(t, err)

	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()
	err = lock.Unlock(canceledCtx)

	require.ErrorIs(t, err, unlockErr)
	require.Zero(t, conn.releaseCalls, "a possibly locked session must not return to the pool")
	require.Equal(t, 1, conn.destroyCalls)
	require.NoError(t, conn.destroyContextErr, "connection destruction must survive caller cancellation")
}

func TestUnlockDestroysConnectionWhenLockIsNotHeld(t *testing.T) {
	conn := &recordingLockConnection{rows: []lockRow{{value: true}, {value: false}}}

	lock, err := acquireSyncLock(context.Background(), recordingLockAcquirer{conn: conn})
	require.NoError(t, err)

	err = lock.Unlock(context.Background())

	require.ErrorContains(t, err, "was not held")
	require.Zero(t, conn.releaseCalls)
	require.Equal(t, 1, conn.destroyCalls)
}

type recordingLockAcquirer struct {
	conn syncLockConnection
	err  error
}

type sequenceLockAcquirer struct {
	connections []syncLockConnection
}

func (a *sequenceLockAcquirer) Acquire(context.Context) (syncLockConnection, error) {
	conn := a.connections[0]
	a.connections = a.connections[1:]
	return conn, nil
}

func (a recordingLockAcquirer) Acquire(context.Context) (syncLockConnection, error) {
	return a.conn, a.err
}

type recordingLockConnection struct {
	rows              []lockRow
	queries           []string
	releaseCalls      int
	destroyCalls      int
	destroyContextErr error
}

func (c *recordingLockConnection) QueryRow(_ context.Context, query string, _ ...any) pgx.Row {
	c.queries = append(c.queries, query)
	row := c.rows[0]
	c.rows = c.rows[1:]
	return row
}

func (c *recordingLockConnection) Release() {
	c.releaseCalls++
}

func (c *recordingLockConnection) Destroy(ctx context.Context) error {
	c.destroyCalls++
	c.destroyContextErr = ctx.Err()
	return nil
}

type lockRow struct {
	value bool
	err   error
}

func (r lockRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	*dest[0].(*bool) = r.value
	return nil
}
