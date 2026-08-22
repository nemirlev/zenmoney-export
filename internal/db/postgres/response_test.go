package postgres

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
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

	err := db.Save(context.Background(), response)

	require.ErrorContains(t, err, "failed to save transactions")
	require.Equal(t, 2, tx.sendBatchCalls, "both entity batches must use the transaction")
	require.Zero(t, pool.sendBatchCalls, "Save must not write entity batches through the pool")
	require.Equal(t, 1, tx.rollbackCalls, "the transaction containing earlier entities must be rolled back")
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

	err := db.Save(context.Background(), response)

	require.NoError(t, err)
	require.Equal(t, 1, tx.sendBatchCalls)
	require.Zero(t, pool.sendBatchCalls)
	require.Equal(t, 1, pool.beginCalls, "deletions inside Save must not start a nested transaction")
	require.Equal(t, 2, tx.execCalls, "the delete and its history entry must use the Save transaction")
	require.Equal(t, 1, tx.commitCalls)
	require.Zero(t, tx.rollbackCalls, "a committed transaction must not be rolled back")
	require.Equal(t, "completed", tx.lastStatus, "completed cursor must be part of the transaction")
	require.Zero(t, pool.statusWrites)
}

type recordingPool struct {
	PgxIface
	tx                       *recordingTx
	beginCalls               int
	sendBatchCalls           int
	statusWrites             int
	lastStatus               string
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
	p.lastStatus = args[5].(string)
	p.rollbackObservedAtStatus = p.tx.rollbackCalls > 0
	return recordingRow{}
}

type recordingTx struct {
	pgx.Tx
	sendBatchCalls int
	failBatch      int
	rollbackCalls  int
	commitCalls    int
	execCalls      int
	lastStatus     string
}

func (tx *recordingTx) SendBatch(_ context.Context, batch *pgx.Batch) pgx.BatchResults {
	tx.sendBatchCalls++
	results := &recordingBatchResults{remaining: batch.Len()}
	if tx.sendBatchCalls == tx.failBatch {
		results.err = errors.New("late entity write failed")
	}
	return results
}

func (tx *recordingTx) QueryRow(_ context.Context, _ string, args ...any) pgx.Row {
	tx.lastStatus = args[5].(string)
	return recordingRow{}
}

func (tx *recordingTx) Rollback(context.Context) error {
	tx.rollbackCalls++
	return nil
}

func (tx *recordingTx) Commit(context.Context) error {
	tx.commitCalls++
	return nil
}

func (tx *recordingTx) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	tx.execCalls++
	return pgconn.NewCommandTag("DELETE 1"), nil
}

type recordingBatchResults struct {
	pgx.BatchResults
	remaining int
	err       error
}

func (r *recordingBatchResults) Exec() (pgconn.CommandTag, error) {
	if r.remaining > 0 {
		r.remaining--
	}
	if r.err != nil {
		err := r.err
		r.err = nil
		return pgconn.CommandTag{}, err
	}
	return pgconn.NewCommandTag("INSERT 0 1"), nil
}

func (r *recordingBatchResults) Close() error {
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
