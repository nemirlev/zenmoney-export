package postgres

import (
	"context"
	"testing"

	"github.com/nemirlev/zenmoney-export/v2/internal/interfaces"
	"github.com/nemirlev/zenmoney-go-sdk/v3/models"
	"github.com/stretchr/testify/require"
)

func TestSaveCopyModeCopiesTransactionsAndCommitsCursor(t *testing.T) {
	tx := &recordingTx{}
	pool := &recordingPool{tx: tx}
	db := &DB{pool: pool}
	response := &models.Response{
		ServerTimestamp: 42,
		Instrument:      []models.Instrument{{ID: 1, Title: "USD"}},
		Transaction: []models.Transaction{
			{ID: "00000000-0000-0000-0000-000000000001", Date: "2026-08-22", Income: 0.1, Outcome: 12.34},
			{ID: "00000000-0000-0000-0000-000000000002", Date: "2026-08-22"},
		},
	}

	err := db.Save(context.Background(), response, interfaces.SaveOptions{
		BatchSize: 2,
		WriteMode: interfaces.WriteModeCopy,
	})

	require.NoError(t, err)
	require.Equal(t, 1, tx.sendBatchCalls, "non-transaction entities still use chunked batches")
	require.Equal(t, 1, tx.copyFromCalls)
	require.Equal(t, 2, tx.copiedRows)
	require.Equal(t, "2026-08-22", tx.copiedValues[0][2])
	require.Equal(t, "0.1", tx.copiedValues[0][3])
	require.Equal(t, "12.34", tx.copiedValues[0][4])
	require.True(t, tx.stagingCreated)
	require.True(t, tx.mergeExecuted)
	require.Equal(t, 1, tx.commitCalls)
	require.Zero(t, tx.rollbackCalls)
	require.Equal(t, "completed", tx.lastStatus)
}

func TestTransactionCopyMergeCastsAnalyticalTypes(t *testing.T) {
	require.Contains(t, mergeTransactionStagingSQL, "date::date")
	require.Contains(t, mergeTransactionStagingSQL, "income::numeric")
	require.Contains(t, mergeTransactionStagingSQL, "outcome::numeric")
	require.Contains(t, mergeTransactionStagingSQL, "op_income::numeric")
	require.Contains(t, mergeTransactionStagingSQL, "op_outcome::numeric")
	require.Contains(t, mergeTransactionStagingSQL, "income_account::uuid")
	require.Contains(t, mergeTransactionStagingSQL, "outcome_account), '')::uuid")
	require.Contains(t, mergeTransactionStagingSQL, "tag::uuid[]")
}

func TestSaveCopyModeRollsBackCopyFailureWithoutCompletedCursor(t *testing.T) {
	tx := &recordingTx{failCopy: true}
	pool := &recordingPool{tx: tx}
	db := &DB{pool: pool}
	response := &models.Response{
		ServerTimestamp: 42,
		Transaction:     []models.Transaction{{ID: "00000000-0000-0000-0000-000000000001"}},
	}

	err := db.Save(context.Background(), response, interfaces.SaveOptions{WriteMode: interfaces.WriteModeCopy})

	require.ErrorContains(t, err, "copy transactions to staging table")
	require.Equal(t, 1, tx.rollbackCalls)
	require.Zero(t, tx.commitCalls)
	require.Empty(t, tx.lastStatus)
	require.Equal(t, "failed", pool.lastStatus)
	require.False(t, tx.mergeExecuted)
}

func TestSaveCopyModeRollsBackMergeFailureWithoutCompletedCursor(t *testing.T) {
	tx := &recordingTx{failMerge: true}
	pool := &recordingPool{tx: tx}
	db := &DB{pool: pool}
	response := &models.Response{
		ServerTimestamp: 42,
		Transaction:     []models.Transaction{{ID: "00000000-0000-0000-0000-000000000001"}},
	}

	err := db.Save(context.Background(), response, interfaces.SaveOptions{WriteMode: interfaces.WriteModeCopy})

	require.ErrorContains(t, err, "merge transaction staging table")
	require.Equal(t, 1, tx.copyFromCalls)
	require.Equal(t, 1, tx.copiedRows)
	require.Equal(t, 1, tx.rollbackCalls)
	require.Zero(t, tx.commitCalls)
	require.Empty(t, tx.lastStatus)
	require.Equal(t, "failed", pool.lastStatus)
}

func TestSaveCopyModeSkipsStagingForEmptyTransactions(t *testing.T) {
	tx := &recordingTx{}
	pool := &recordingPool{tx: tx}
	db := &DB{pool: pool}

	err := db.Save(context.Background(), &models.Response{ServerTimestamp: 42}, interfaces.SaveOptions{
		WriteMode: interfaces.WriteModeCopy,
	})

	require.NoError(t, err)
	require.False(t, tx.stagingCreated)
	require.Zero(t, tx.copyFromCalls)
	require.False(t, tx.mergeExecuted)
	require.Equal(t, 1, tx.commitCalls)
	require.Equal(t, "completed", tx.lastStatus)
}

func TestSaveRejectsUnknownWriteModeBeforeBeginningTransaction(t *testing.T) {
	tx := &recordingTx{}
	pool := &recordingPool{tx: tx}
	db := &DB{pool: pool}

	err := db.Save(context.Background(), &models.Response{}, interfaces.SaveOptions{WriteMode: "fast"})

	require.ErrorContains(t, err, `unsupported write mode "fast"`)
	require.Zero(t, pool.beginCalls)
}
