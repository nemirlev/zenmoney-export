package postgres

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/nemirlev/zenmoney-export/v2/internal/interfaces"
	"github.com/nemirlev/zenmoney-go-sdk/v3/models"
	"github.com/stretchr/testify/require"
)

func TestSaveRollsBackEarlierEntitiesWhenLateEntityFails(t *testing.T) {
	tx := &recordingTx{failBatch: 2}
	pool := &recordingPool{tx: tx}
	db := &DB{pool: pool}
	response := &models.Response{
		ServerTimestamp: 42,
		Instrument:      []models.Instrument{{ID: 1, Title: "USD"}},
		Transaction:     []models.Transaction{{ID: "transaction-1"}},
	}

	err := db.Save(
		context.Background(),
		response,
		interfaces.SaveOptions{BatchSize: interfaces.DefaultBatchSize},
	)

	require.ErrorContains(t, err, "failed to save transactions")
	require.Equal(t, 2, tx.sendBatchCalls, "both entity batches must use the transaction")
	require.Zero(t, pool.sendBatchCalls, "Save must not write entity batches through the pool")
	require.Equal(
		t,
		1,
		tx.rollbackCalls,
		"the transaction containing earlier entities must be rolled back",
	)
	require.Zero(t, tx.commitCalls)
	require.Equal(t, 1, pool.statusWrites, "failed status is recorded only after rollback")
	require.True(t, pool.rollbackObservedAtStatus)
	require.Equal(t, "failed", pool.lastStatus)
}

func TestSaveDoesNotRollbackAfterSuccessfulCommit(t *testing.T) {
	tx := &recordingTx{}
	pool := &recordingPool{tx: tx}
	db := &DB{pool: pool}
	response := &models.Response{
		ServerTimestamp: 42,
		Instrument:      []models.Instrument{{ID: 1, Title: "USD"}},
		Deletion: []models.Deletion{{
			ID: "account-1", Object: string(models.EntityTypeAccount), User: 1, Stamp: 42,
		}},
	}

	err := db.Save(
		context.Background(),
		response,
		interfaces.SaveOptions{BatchSize: interfaces.DefaultBatchSize},
	)

	require.NoError(t, err)
	require.Equal(t, 1, tx.sendBatchCalls)
	require.Zero(t, pool.sendBatchCalls)
	require.Equal(
		t,
		1,
		pool.beginCalls,
		"deletions inside Save must not start a nested transaction",
	)
	require.Equal(
		t,
		2,
		tx.execCalls,
		"the delete and its history entry must use the Save transaction",
	)
	require.Equal(t, 1, tx.commitCalls)
	require.Zero(t, tx.rollbackCalls, "a committed transaction must not be rolled back")
	require.Equal(t, "completed", tx.lastStatus, "completed cursor must be part of the transaction")
	require.Zero(t, pool.statusWrites)
}

func TestSavePersistsRequestedSyncType(t *testing.T) {
	tx := &recordingTx{}
	db := &DB{pool: &recordingPool{tx: tx}}

	err := db.Save(
		context.Background(),
		&models.Response{ServerTimestamp: 42},
		interfaces.SaveOptions{SyncType: interfaces.SyncTypePartial},
	)

	require.NoError(t, err)
	require.Equal(t, interfaces.SyncTypePartial, tx.lastSyncType)
}

func TestSaveRejectsUnknownSyncTypeBeforeBeginningTransaction(t *testing.T) {
	tx := &recordingTx{}
	pool := &recordingPool{tx: tx}
	db := &DB{pool: pool}

	err := db.Save(
		context.Background(),
		&models.Response{},
		interfaces.SaveOptions{SyncType: "unknown"},
	)

	require.ErrorContains(t, err, `unsupported sync type "unknown"`)
	require.Zero(t, pool.beginCalls)
}

func TestSaveChunksEntityBatchesWithinOneTransaction(t *testing.T) {
	tx := &recordingTx{}
	pool := &recordingPool{tx: tx}
	db := &DB{pool: pool}
	response := &models.Response{
		ServerTimestamp: 42,
		Transaction: []models.Transaction{
			{ID: "transaction-0"},
			{ID: "transaction-1"},
			{ID: "transaction-2"},
			{ID: "transaction-3"},
			{ID: "transaction-4"},
		},
	}

	err := db.Save(context.Background(), response, interfaces.SaveOptions{BatchSize: 2})

	require.NoError(t, err)
	require.Equal(t, []int{2, 2, 1}, tx.batchSizes)
	require.Equal(t, 3, tx.closedBatches)
	require.Equal(t, 1, tx.commitCalls)
	require.Zero(t, tx.rollbackCalls)
	require.Equal(t, "completed", tx.lastStatus)
}

func TestSaveRollsBackWhenLaterChunkFails(t *testing.T) {
	tx := &recordingTx{failBatch: 2}
	pool := &recordingPool{tx: tx}
	db := &DB{pool: pool}
	response := &models.Response{
		ServerTimestamp: 42,
		Transaction: []models.Transaction{
			{ID: "transaction-0"},
			{ID: "transaction-1"},
			{ID: "transaction-2"},
			{ID: "transaction-3"},
		},
	}

	err := db.Save(context.Background(), response, interfaces.SaveOptions{BatchSize: 2})

	require.ErrorContains(t, err, "failed to save transaction 2")
	require.Equal(t, []int{2, 2}, tx.batchSizes)
	require.Equal(t, 2, tx.closedBatches)
	require.Equal(t, 1, tx.rollbackCalls)
	require.Zero(t, tx.commitCalls)
	require.Empty(
		t,
		tx.lastStatus,
		"completed cursor must not be written in the rolled-back transaction",
	)
	require.Equal(t, "failed", pool.lastStatus)
}

func TestSaveInChunksSkipsEmptySlice(t *testing.T) {
	tx := &recordingTx{}

	err := saveTransactions(context.Background(), tx, nil, 2)

	require.NoError(t, err)
	require.Zero(t, tx.sendBatchCalls)
}

type recordingPool struct {
	PgxIface
	tx                       *recordingTx
	beginCalls               int
	sendBatchCalls           int
	statusWrites             int
	lastStatus               string
	lastSyncType             interfaces.SyncType
	rollbackObservedAtStatus bool
}

func (p *recordingPool) Begin(context.Context) (pgx.Tx, error) {
	p.beginCalls++
	return p.tx, nil
}

func (p *recordingPool) SendBatch(context.Context, *pgx.Batch) pgx.BatchResults {
	p.sendBatchCalls++
	return &recordingBatchResults{}
}

func (p *recordingPool) QueryRow(_ context.Context, _ string, args ...any) pgx.Row {
	p.statusWrites++
	p.lastSyncType = interfaces.SyncType(args[2].(string))
	p.lastStatus = args[5].(string)
	p.rollbackObservedAtStatus = p.tx.rollbackCalls > 0
	return recordingRow{}
}

type recordingTx struct {
	pgx.Tx
	sendBatchCalls     int
	batchSizes         []int
	closedBatches      int
	failBatch          int
	onBatchError       func()
	rollbackCalls      int
	rollbackContextErr error
	commitCalls        int
	execCalls          int
	lastStatus         string
	lastSyncType       interfaces.SyncType
	copyFromCalls      int
	copiedRows         int
	copiedValues       [][]any
	failCopy           bool
	failMerge          bool
	stagingCreated     bool
	mergeExecuted      bool
}

func (tx *recordingTx) SendBatch(_ context.Context, batch *pgx.Batch) pgx.BatchResults {
	tx.sendBatchCalls++
	tx.batchSizes = append(tx.batchSizes, batch.Len())
	results := &recordingBatchResults{
		remaining: batch.Len(),
		onClose:   func() { tx.closedBatches++ },
	}
	if tx.sendBatchCalls == tx.failBatch {
		results.err = errors.New("late entity write failed")
		results.onError = tx.onBatchError
	}
	return results
}

func (tx *recordingTx) QueryRow(_ context.Context, _ string, args ...any) pgx.Row {
	tx.lastSyncType = interfaces.SyncType(args[2].(string))
	tx.lastStatus = args[5].(string)
	return recordingRow{}
}

func (tx *recordingTx) Rollback(ctx context.Context) error {
	tx.rollbackCalls++
	tx.rollbackContextErr = ctx.Err()
	return nil
}

func (tx *recordingTx) Commit(context.Context) error {
	tx.commitCalls++
	return nil
}

func (tx *recordingTx) Exec(_ context.Context, sql string, _ ...any) (pgconn.CommandTag, error) {
	tx.execCalls++
	if strings.Contains(sql, "CREATE TEMP TABLE staging_transaction") {
		tx.stagingCreated = true
	}
	if strings.Contains(sql, "FROM latest_staging_transaction") {
		tx.mergeExecuted = true
		if tx.failMerge {
			return pgconn.CommandTag{}, errors.New("merge failed")
		}
	}
	return pgconn.NewCommandTag("DELETE 1"), nil
}

func (tx *recordingTx) CopyFrom(
	_ context.Context,
	_ pgx.Identifier,
	_ []string,
	rowSrc pgx.CopyFromSource,
) (int64, error) {
	tx.copyFromCalls++
	if tx.failCopy {
		return 0, errors.New("copy failed")
	}

	for rowSrc.Next() {
		values, err := rowSrc.Values()
		if err != nil {
			return int64(tx.copiedRows), err
		}
		tx.copiedValues = append(tx.copiedValues, values)
		tx.copiedRows++
	}
	if err := rowSrc.Err(); err != nil {
		return int64(tx.copiedRows), err
	}
	return int64(tx.copiedRows), nil
}

type recordingBatchResults struct {
	pgx.BatchResults
	remaining int
	err       error
	onClose   func()
	onError   func()
}

func (r *recordingBatchResults) Exec() (pgconn.CommandTag, error) {
	if r.remaining > 0 {
		r.remaining--
	}
	if r.err != nil {
		if r.onError != nil {
			r.onError()
		}
		err := r.err
		r.err = nil
		return pgconn.CommandTag{}, err
	}
	return pgconn.NewCommandTag("INSERT 0 1"), nil
}

func (r *recordingBatchResults) Close() error {
	if r.onClose != nil {
		r.onClose()
	}
	return nil
}

type recordingRow struct {
	pgx.Row
}

func (recordingRow) Scan(dest ...any) error {
	if len(dest) > 0 {
		*dest[0].(*int64) = 1
	}
	return nil
}
