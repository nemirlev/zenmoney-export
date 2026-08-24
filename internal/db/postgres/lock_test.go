package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nemirlev/zenmoney-export/v2/internal/interfaces"
	"github.com/stretchr/testify/require"
)

func TestDBSyncLockUsesRawConnectionOutsidePool(t *testing.T) {
	connConfig := &pgx.ConnConfig{}
	pool := &configOnlyPool{config: &pgxpool.Config{ConnConfig: connConfig}}
	conn := &recordingLockConnection{rows: []lockRow{{value: true}, {value: true}}}
	connector := &recordingLockConnector{connections: []syncLockConnection{conn}}
	db := &DB{pool: pool}

	lock, err := db.acquireSyncLock(context.Background(), connector)
	require.NoError(t, err)
	require.Zero(t, pool.acquireCalls, "the advisory lock must not consume a working pool slot")
	require.Len(t, connector.configs, 1)
	require.NotSame(
		t,
		connConfig,
		connector.configs[0],
		"the raw connection must receive an isolated config copy",
	)
	require.Zero(t, conn.closeCalls, "the raw connection must remain open while the lock is held")

	require.NoError(t, lock.Unlock(context.Background()))
	require.Equal(t, 1, conn.closeCalls)
	require.Equal(t, []string{
		"SELECT pg_try_advisory_lock($1, $2)",
		"SELECT pg_advisory_unlock($1, $2)",
	}, conn.queries)

	require.NoError(t, lock.Unlock(context.Background()))
	require.Equal(t, 1, conn.closeCalls)
}

func TestAcquireSyncLockReturnsBusySentinelAndClosesRawConnection(t *testing.T) {
	conn := &recordingLockConnection{rows: []lockRow{{value: false}}}
	connector := &recordingLockConnector{connections: []syncLockConnection{conn}}

	lock, err := acquireSyncLock(context.Background(), connector, nil)

	require.Nil(t, lock)
	require.ErrorIs(t, err, interfaces.ErrSyncAlreadyRunning)
	require.Equal(t, 1, conn.closeCalls)
}

func TestConcurrentSyncLockAttemptIsRejectedUntilOwnerUnlocks(t *testing.T) {
	ownerConn := &recordingLockConnection{rows: []lockRow{{value: true}, {value: true}}}
	contenderConn := &recordingLockConnection{rows: []lockRow{{value: false}}}
	connector := &recordingLockConnector{
		connections: []syncLockConnection{ownerConn, contenderConn},
	}

	owner, err := acquireSyncLock(context.Background(), connector, nil)
	require.NoError(t, err)

	contender, err := acquireSyncLock(context.Background(), connector, nil)
	require.Nil(t, contender)
	require.ErrorIs(t, err, interfaces.ErrSyncAlreadyRunning)
	require.Zero(
		t,
		ownerConn.closeCalls,
		"owner must retain its raw connection while contender checks",
	)
	require.Equal(t, 1, contenderConn.closeCalls)

	require.NoError(t, owner.Unlock(context.Background()))
	require.Equal(t, 1, ownerConn.closeCalls)
}

func TestAcquireSyncLockClosesConnectionAfterAmbiguousQueryError(t *testing.T) {
	queryErr := errors.New("query failed")
	conn := &recordingLockConnection{rows: []lockRow{{err: queryErr}}}
	connector := &recordingLockConnector{connections: []syncLockConnection{conn}}

	lock, err := acquireSyncLock(context.Background(), connector, nil)

	require.Nil(t, lock)
	require.ErrorIs(t, err, queryErr)
	require.Equal(t, 1, conn.closeCalls)
}

func TestUnlockClosesConnectionWhenAdvisoryUnlockFails(t *testing.T) {
	unlockErr := errors.New("connection interrupted")
	conn := &recordingLockConnection{rows: []lockRow{{value: true}, {err: unlockErr}}}
	connector := &recordingLockConnector{connections: []syncLockConnection{conn}}

	lock, err := acquireSyncLock(context.Background(), connector, nil)
	require.NoError(t, err)

	err = lock.Unlock(context.Background())

	require.ErrorIs(t, err, unlockErr)
	require.Equal(t, 1, conn.closeCalls, "closing the raw session guarantees lock release")
}

func TestUnlockClosesConnectionWhenLockIsNotHeld(t *testing.T) {
	conn := &recordingLockConnection{rows: []lockRow{{value: true}, {value: false}}}
	connector := &recordingLockConnector{connections: []syncLockConnection{conn}}

	lock, err := acquireSyncLock(context.Background(), connector, nil)
	require.NoError(t, err)

	err = lock.Unlock(context.Background())

	require.ErrorContains(t, err, "was not held")
	require.Equal(t, 1, conn.closeCalls)
}

func TestUnlockUsesFreshBoundedContextAfterCallerCancellation(t *testing.T) {
	conn := &recordingLockConnection{rows: []lockRow{{value: true}, {value: true}}}
	connector := &recordingLockConnector{connections: []syncLockConnection{conn}}
	lock, err := acquireSyncLock(context.Background(), connector, nil)
	require.NoError(t, err)

	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()
	err = lock.Unlock(canceledCtx)

	require.NoError(t, err)
	require.NoError(t, conn.queryContextErrors[1], "unlock must not inherit caller cancellation")
	require.NoError(t, conn.closeContextErrors[0])
}

func TestUnlockDeadlineClosesBlackholedRawConnection(t *testing.T) {
	conn := &recordingLockConnection{rows: []lockRow{{value: true}, {block: true}}}
	connector := &recordingLockConnector{connections: []syncLockConnection{conn}}
	lock, err := acquireSyncLock(context.Background(), connector, nil)
	require.NoError(t, err)
	lock.(*postgresSyncLock).cleanupTimeout = 20 * time.Millisecond

	started := time.Now()
	err = lock.Unlock(context.Background())

	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.Less(t, time.Since(started), time.Second, "cleanup must be bounded")
	require.Equal(t, 1, conn.closeCalls, "deadline must still physically close the raw session")
	require.ErrorIs(t, conn.closeContextErrors[0], context.DeadlineExceeded)
}

type configOnlyPool struct {
	PgxIface
	config       *pgxpool.Config
	acquireCalls int
}

func (p *configOnlyPool) Config() *pgxpool.Config {
	return p.config
}

func (p *configOnlyPool) Acquire(context.Context) (*pgxpool.Conn, error) {
	p.acquireCalls++
	return nil, errors.New("working pool must not be used for advisory lock")
}

type recordingLockConnector struct {
	connections []syncLockConnection
	configs     []*pgx.ConnConfig
	err         error
}

func (f *recordingLockConnector) Connect(
	_ context.Context,
	config *pgx.ConnConfig,
) (syncLockConnection, error) {
	f.configs = append(f.configs, config)
	if f.err != nil {
		return nil, f.err
	}
	conn := f.connections[0]
	f.connections = f.connections[1:]
	return conn, nil
}

type recordingLockConnection struct {
	rows               []lockRow
	queries            []string
	queryContextErrors []error
	closeCalls         int
	closeContextErrors []error
	closeErr           error
}

func (c *recordingLockConnection) QueryRow(ctx context.Context, query string, _ ...any) pgx.Row {
	c.queries = append(c.queries, query)
	c.queryContextErrors = append(c.queryContextErrors, ctx.Err())
	row := c.rows[0]
	c.rows = c.rows[1:]
	row.ctx = ctx
	return &row
}

func (c *recordingLockConnection) Close(ctx context.Context) error {
	c.closeCalls++
	c.closeContextErrors = append(c.closeContextErrors, ctx.Err())
	return c.closeErr
}

type lockRow struct {
	ctx   context.Context
	value bool
	err   error
	block bool
}

func (r *lockRow) Scan(dest ...any) error {
	if r.block {
		<-r.ctx.Done()
		return r.ctx.Err()
	}
	if r.err != nil {
		return r.err
	}
	*dest[0].(*bool) = r.value
	return nil
}
