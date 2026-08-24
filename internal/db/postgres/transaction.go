package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/nemirlev/zenmoney-export/v2/internal/interfaces"
	"github.com/nemirlev/zenmoney-go-sdk/v3/models"
)

const transactionSelectQuery = `
        SELECT id, "user", to_char(date, 'YYYY-MM-DD') AS date,
               income, outcome, changed, income_instrument,
               outcome_instrument, created, original_payee, deleted, viewed,
               hold, qr_code, source, income_account, outcome_account, tag,
               comment, payee, op_income, op_outcome, op_income_instrument,
               op_outcome_instrument, latitude, longitude, merchant,
               income_bank_id, outcome_bank_id, reminder_marker
        FROM transaction`

const transactionInsertQuery = `
        INSERT INTO transaction (
            id, "user", date, income, outcome, changed, income_instrument,
            outcome_instrument, created, original_payee, deleted, viewed,
            hold, qr_code, source, income_account, outcome_account, tag,
            comment, payee, op_income, op_outcome, op_income_instrument,
            op_outcome_instrument, latitude, longitude, merchant,
            income_bank_id, outcome_bank_id, reminder_marker
        ) VALUES ($1, $2, $3::text::date, $4::text::numeric,
                  $5::text::numeric, $6, $7, $8, $9, $10, $11, $12, $13,
                  $14, $15, $16::text::uuid,
                  NULLIF(BTRIM($17::text), '')::uuid, $18, $19, $20,
                  $21::text::numeric, $22::text::numeric, $23, $24,
                  $25, $26, $27, $28, $29, $30)`

func scanTransaction(row rowScanner) (models.Transaction, error) {
	var tx models.Transaction
	err := row.Scan(
		&tx.ID,
		&tx.User,
		&tx.Date,
		&tx.Income,
		&tx.Outcome,
		&tx.Changed,
		&tx.IncomeInstrument,
		&tx.OutcomeInstrument,
		&tx.Created,
		&tx.OriginalPayee,
		&tx.Deleted,
		&tx.Viewed,
		&tx.Hold,
		&tx.QRCode,
		&tx.Source,
		&tx.IncomeAccount,
		&tx.OutcomeAccount,
		&tx.Tag,
		&tx.Comment,
		&tx.Payee,
		&tx.OpIncome,
		&tx.OpOutcome,
		&tx.OpIncomeInstrument,
		&tx.OpOutcomeInstrument,
		&tx.Latitude,
		&tx.Longitude,
		&tx.Merchant,
		&tx.IncomeBankID,
		&tx.OutcomeBankID,
		&tx.ReminderMarker,
	)
	return tx, err
}

func transactionValues(tx *models.Transaction) []any {
	return []any{
		tx.ID,
		tx.User,
		tx.Date,
		decimalString(tx.Income),
		decimalString(tx.Outcome),
		tx.Changed,
		tx.IncomeInstrument,
		tx.OutcomeInstrument,
		tx.Created,
		tx.OriginalPayee,
		tx.Deleted,
		tx.Viewed,
		tx.Hold,
		tx.QRCode,
		tx.Source,
		tx.IncomeAccount,
		tx.OutcomeAccount,
		tx.Tag,
		tx.Comment,
		tx.Payee,
		decimalString(tx.OpIncome),
		decimalString(tx.OpOutcome),
		tx.OpIncomeInstrument,
		tx.OpOutcomeInstrument,
		tx.Latitude,
		tx.Longitude,
		tx.Merchant,
		tx.IncomeBankID,
		tx.OutcomeBankID,
		tx.ReminderMarker,
	}
}

// GetTransaction retrieves a specific transaction by its ID
func (s *DB) GetTransaction(ctx context.Context, id string) (*models.Transaction, error) {
	tx, err := scanTransaction(s.pool.QueryRow(ctx, transactionSelectQuery+` WHERE id = $1`, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("transaction not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get transaction: %w", err)
	}

	return &tx, nil
}

// ListTransactions retrieves a list of transactions based on the provided filter
func (s *DB) ListTransactions(
	ctx context.Context,
	filter interfaces.Filter,
) ([]models.Transaction, error) {
	query, args := buildListQuery(
		transactionSelectQuery,
		filter,
		true,
		` ORDER BY date DESC, created DESC`,
	)

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list transactions: %w", err)
	}
	defer rows.Close()

	return collectRows(
		rows,
		scanTransaction,
		"failed to scan transaction",
		"error iterating transactions",
	)
}

// CreateTransaction creates a new transaction record
func (s *DB) CreateTransaction(ctx context.Context, tx *models.Transaction) error {
	_, err := execCommand(
		ctx,
		s.pool,
		transactionInsertQuery,
		"failed to create transaction",
		transactionValues(tx)...,
	)
	return err
}

// UpdateTransaction updates an existing transaction record
func (s *DB) UpdateTransaction(ctx context.Context, tx *models.Transaction) error {
	query := `
        UPDATE transaction SET
            "user" = $2,
            date = $3::text::date,
            income = $4::text::numeric,
            outcome = $5::text::numeric,
            changed = $6,
            income_instrument = $7,
            outcome_instrument = $8,
            created = $9,
            original_payee = $10,
            deleted = $11,
            viewed = $12,
            hold = $13,
            qr_code = $14,
            source = $15,
            income_account = $16::text::uuid,
            outcome_account = NULLIF(BTRIM($17::text), '')::uuid,
            tag = $18,
            comment = $19,
            payee = $20,
            op_income = $21::text::numeric,
            op_outcome = $22::text::numeric,
            op_income_instrument = $23,
            op_outcome_instrument = $24,
            latitude = $25,
            longitude = $26,
            merchant = $27,
            income_bank_id = $28,
            outcome_bank_id = $29,
            reminder_marker = $30
        WHERE id = $1`

	commandTag, err := execCommand(
		ctx,
		s.pool,
		query,
		"failed to update transaction",
		transactionValues(tx)...,
	)
	if err != nil {
		return err
	}

	return requireRowsAffected(commandTag, fmt.Sprintf("transaction not found: %s", tx.ID))
}

// DeleteTransaction deletes a transaction by its ID
func (s *DB) DeleteTransaction(ctx context.Context, id string) error {
	query := `DELETE FROM transaction WHERE id = $1`

	commandTag, err := execCommand(ctx, s.pool, query, "failed to delete transaction", id)
	if err != nil {
		return err
	}

	return requireRowsAffected(commandTag, fmt.Sprintf("transaction not found: %s", id))
}
