package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/nemirlev/zenmoney-go-sdk/v3/models"
)

const createTransactionStagingSQL = `
CREATE TEMP TABLE staging_transaction (
    staging_order BIGINT GENERATED ALWAYS AS IDENTITY,
    id TEXT,
    "user" INT,
    date TEXT,
    income TEXT,
    outcome TEXT,
    changed BIGINT,
    income_instrument INT,
    outcome_instrument INT,
    created BIGINT,
    original_payee TEXT,
    deleted BOOLEAN,
    viewed BOOLEAN,
    hold BOOLEAN,
    qr_code TEXT,
    source TEXT,
    income_account TEXT,
    outcome_account TEXT,
    tag TEXT[],
    comment TEXT,
    payee TEXT,
    op_income TEXT,
    op_outcome TEXT,
    op_income_instrument INT,
    op_outcome_instrument INT,
    latitude DOUBLE PRECISION,
    longitude DOUBLE PRECISION,
    merchant TEXT,
    income_bank_id TEXT,
    outcome_bank_id TEXT,
    reminder_marker TEXT
) ON COMMIT DROP`

var transactionCopyColumns = []string{
	"id", "user", "date", "income", "outcome", "changed",
	"income_instrument", "outcome_instrument", "created", "original_payee",
	"deleted", "viewed", "hold", "qr_code", "source", "income_account",
	"outcome_account", "tag", "comment", "payee", "op_income", "op_outcome",
	"op_income_instrument", "op_outcome_instrument", "latitude", "longitude",
	"merchant", "income_bank_id", "outcome_bank_id", "reminder_marker",
}

const mergeTransactionStagingSQL = `
WITH latest_staging_transaction AS (
    SELECT DISTINCT ON (id) *
    FROM staging_transaction
    ORDER BY id, staging_order DESC
)
INSERT INTO transaction (
    id, "user", date, income, outcome, changed, income_instrument,
    outcome_instrument, created, original_payee, deleted, viewed,
    hold, qr_code, source, income_account, outcome_account, tag,
    comment, payee, op_income, op_outcome, op_income_instrument,
    op_outcome_instrument, latitude, longitude, merchant,
    income_bank_id, outcome_bank_id, reminder_marker
)
SELECT
    id::uuid,
    "user",
    date::date,
    income::numeric,
    outcome::numeric,
    changed,
    income_instrument,
    outcome_instrument,
    created,
    original_payee,
    deleted,
    viewed,
    hold,
    qr_code,
    source,
    income_account::uuid,
    NULLIF(BTRIM(outcome_account), '')::uuid,
    tag::uuid[],
    comment,
    payee,
    op_income::numeric,
    op_outcome::numeric,
    op_income_instrument,
    op_outcome_instrument,
    latitude,
    longitude,
    NULLIF(BTRIM(merchant), '')::uuid,
    income_bank_id,
    outcome_bank_id,
    NULLIF(BTRIM(reminder_marker), '')::uuid
FROM latest_staging_transaction
ON CONFLICT (id) DO UPDATE SET
    "user" = EXCLUDED.user,
    date = EXCLUDED.date,
    income = EXCLUDED.income,
    outcome = EXCLUDED.outcome,
    changed = EXCLUDED.changed,
    income_instrument = EXCLUDED.income_instrument,
    outcome_instrument = EXCLUDED.outcome_instrument,
    created = EXCLUDED.created,
    original_payee = EXCLUDED.original_payee,
    deleted = EXCLUDED.deleted,
    viewed = EXCLUDED.viewed,
    hold = EXCLUDED.hold,
    qr_code = EXCLUDED.qr_code,
    source = EXCLUDED.source,
    income_account = EXCLUDED.income_account,
    outcome_account = EXCLUDED.outcome_account,
    tag = EXCLUDED.tag,
    comment = EXCLUDED.comment,
    payee = EXCLUDED.payee,
    op_income = EXCLUDED.op_income,
    op_outcome = EXCLUDED.op_outcome,
    op_income_instrument = EXCLUDED.op_income_instrument,
    op_outcome_instrument = EXCLUDED.op_outcome_instrument,
    latitude = EXCLUDED.latitude,
    longitude = EXCLUDED.longitude,
    merchant = EXCLUDED.merchant,
    income_bank_id = EXCLUDED.income_bank_id,
    outcome_bank_id = EXCLUDED.outcome_bank_id,
    reminder_marker = EXCLUDED.reminder_marker`

func copyTransactions(ctx context.Context, tx pgx.Tx, transactions []models.Transaction) error {
	if len(transactions) == 0 {
		return nil
	}

	if _, err := tx.Exec(ctx, createTransactionStagingSQL); err != nil {
		return fmt.Errorf("create transaction staging table: %w", err)
	}

	rows := pgx.CopyFromSlice(len(transactions), func(i int) ([]any, error) {
		transaction := transactions[i]
		return []any{
			transaction.ID,
			transaction.User,
			transaction.Date,
			decimalString(transaction.Income),
			decimalString(transaction.Outcome),
			transaction.Changed,
			transaction.IncomeInstrument,
			transaction.OutcomeInstrument,
			transaction.Created,
			transaction.OriginalPayee,
			transaction.Deleted,
			transaction.Viewed,
			transaction.Hold,
			optionalCopyValue(transaction.QRCode),
			transaction.Source,
			transaction.IncomeAccount,
			optionalCopyValue(transaction.OutcomeAccount),
			transaction.Tag,
			optionalCopyValue(transaction.Comment),
			transaction.Payee,
			decimalString(transaction.OpIncome),
			decimalString(transaction.OpOutcome),
			optionalCopyValue(transaction.OpIncomeInstrument),
			optionalCopyValue(transaction.OpOutcomeInstrument),
			optionalCopyValue(transaction.Latitude),
			optionalCopyValue(transaction.Longitude),
			optionalCopyValue(transaction.Merchant),
			optionalCopyValue(transaction.IncomeBankID),
			optionalCopyValue(transaction.OutcomeBankID),
			optionalCopyValue(transaction.ReminderMarker),
		}, nil
	})

	if _, err := tx.CopyFrom(ctx, pgx.Identifier{"staging_transaction"}, transactionCopyColumns, rows); err != nil {
		return fmt.Errorf("copy transactions to staging table: %w", err)
	}

	if _, err := tx.Exec(ctx, mergeTransactionStagingSQL); err != nil {
		return fmt.Errorf("merge transaction staging table: %w", err)
	}

	return nil
}

func optionalCopyValue[T any](value *T) any {
	if value == nil {
		return nil
	}
	return *value
}
