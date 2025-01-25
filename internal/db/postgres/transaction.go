package postgres

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/nemirlev/zenmoney-export/v2/internal/interfaces"
	"github.com/nemirlev/zenmoney-go-sdk/v2/models"
	"strings"
)

// GetTransaction retrieves a specific transaction by its ID
func (s *DB) GetTransaction(ctx context.Context, id string) (*models.Transaction, error) {
	query := `
        SELECT id, "user", date, income, outcome, changed, income_instrument,
               outcome_instrument, created, original_payee, deleted, viewed,
               hold, qr_code, source, income_account, outcome_account, tag,
               comment, payee, op_income, op_outcome, op_income_instrument,
               op_outcome_instrument, latitude, longitude, merchant,
               income_bank_id, outcome_bank_id, reminder_marker
        FROM transaction
        WHERE id = $1`

	tx := &models.Transaction{}
	err := s.pool.QueryRow(ctx, query, id).Scan(
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

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("transaction not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get transaction: %w", err)
	}

	return tx, nil
}

// ListTransactions retrieves a list of transactions based on the provided filter
func (s *DB) ListTransactions(ctx context.Context, filter interfaces.Filter) ([]models.Transaction, error) {
	var conditions []string
	var args []interface{}
	argNum := 1

	if filter.UserID != nil {
		conditions = append(conditions, fmt.Sprintf(`"user" = $%d`, argNum))
		args = append(args, *filter.UserID)
		argNum++
	}

	if filter.StartDate != nil {
		conditions = append(conditions, fmt.Sprintf(`date >= $%d`, argNum))
		args = append(args, filter.StartDate.Format("2006-01-02"))
		argNum++
	}

	if filter.EndDate != nil {
		conditions = append(conditions, fmt.Sprintf(`date <= $%d`, argNum))
		args = append(args, filter.EndDate.Format("2006-01-02"))
		argNum++
	}

	query := `
        SELECT id, "user", date, income, outcome, changed, income_instrument,
               outcome_instrument, created, original_payee, deleted, viewed,
               hold, qr_code, source, income_account, outcome_account, tag,
               comment, payee, op_income, op_outcome, op_income_instrument,
               op_outcome_instrument, latitude, longitude, merchant,
               income_bank_id, outcome_bank_id, reminder_marker
        FROM transaction`

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	query += ` ORDER BY date DESC, created DESC`
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argNum, argNum+1)
	args = append(args, filter.Limit, (filter.Page-1)*filter.Limit)

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list transactions: %w", err)
	}
	defer rows.Close()

	var transactions []models.Transaction
	for rows.Next() {
		var tx models.Transaction
		err := rows.Scan(
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
		if err != nil {
			return nil, fmt.Errorf("failed to scan transaction: %w", err)
		}
		transactions = append(transactions, tx)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating transactions: %w", err)
	}

	return transactions, nil
}

// CreateTransaction creates a new transaction record
func (s *DB) CreateTransaction(ctx context.Context, tx *models.Transaction) error {
	query := `
        INSERT INTO transaction (
            id, "user", date, income, outcome, changed, income_instrument,
            outcome_instrument, created, original_payee, deleted, viewed,
            hold, qr_code, source, income_account, outcome_account, tag,
            comment, payee, op_income, op_outcome, op_income_instrument,
            op_outcome_instrument, latitude, longitude, merchant,
            income_bank_id, outcome_bank_id, reminder_marker
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13,
                  $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24,
                  $25, $26, $27, $28, $29, $30)`

	_, err := s.pool.Exec(ctx, query,
		tx.ID,
		tx.User,
		tx.Date,
		tx.Income,
		tx.Outcome,
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
		tx.OpIncome,
		tx.OpOutcome,
		tx.OpIncomeInstrument,
		tx.OpOutcomeInstrument,
		tx.Latitude,
		tx.Longitude,
		tx.Merchant,
		tx.IncomeBankID,
		tx.OutcomeBankID,
		tx.ReminderMarker,
	)

	if err != nil {
		return fmt.Errorf("failed to create transaction: %w", err)
	}

	return nil
}

// UpdateTransaction updates an existing transaction record
func (s *DB) UpdateTransaction(ctx context.Context, tx *models.Transaction) error {
	query := `
        UPDATE transaction SET
            "user" = $2,
            date = $3,
            income = $4,
            outcome = $5,
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
            income_account = $16,
            outcome_account = $17,
            tag = $18,
            comment = $19,
            payee = $20,
            op_income = $21,
            op_outcome = $22,
            op_income_instrument = $23,
            op_outcome_instrument = $24,
            latitude = $25,
            longitude = $26,
            merchant = $27,
            income_bank_id = $28,
            outcome_bank_id = $29,
            reminder_marker = $30
        WHERE id = $1`

	commandTag, err := s.pool.Exec(ctx, query,
		tx.ID,
		tx.User,
		tx.Date,
		tx.Income,
		tx.Outcome,
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
		tx.OpIncome,
		tx.OpOutcome,
		tx.OpIncomeInstrument,
		tx.OpOutcomeInstrument,
		tx.Latitude,
		tx.Longitude,
		tx.Merchant,
		tx.IncomeBankID,
		tx.OutcomeBankID,
		tx.ReminderMarker,
	)

	if err != nil {
		return fmt.Errorf("failed to update transaction: %w", err)
	}

	if commandTag.RowsAffected() == 0 {
		return fmt.Errorf("transaction not found: %s", tx.ID)
	}

	return nil
}

// DeleteTransaction deletes a transaction by its ID
func (s *DB) DeleteTransaction(ctx context.Context, id string) error {
	query := `DELETE FROM transaction WHERE id = $1`

	commandTag, err := s.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete transaction: %w", err)
	}

	if commandTag.RowsAffected() == 0 {
		return fmt.Errorf("transaction not found: %s", id)
	}

	return nil
}
