package postgres

import (
	"context"
	"fmt"
	"github.com/nemirlev/zenmoney-export/internal/interfaces"
	"github.com/nemirlev/zenmoney-go-sdk/v2/models"
	"strings"
)

// GetAccount retrieves a specific account by its ID
func (s *DB) GetAccount(ctx context.Context, id string) (*models.Account, error) {
	query := `
        SELECT id, "user", instrument, type, role, private, savings,
               title, in_balance, credit_limit, start_balance, balance,
               company, archive, enable_correction, balance_correction_type,
               start_date, capitalization, percent, changed, sync_id,
               enable_sms, end_date_offset, end_date_offset_interval,
               payoff_step, payoff_interval
        FROM account
        WHERE id = $1`

	account := &models.Account{}
	err := s.pool.QueryRow(ctx, query, id).Scan(
		&account.ID,
		&account.User,
		&account.Instrument,
		&account.Type,
		&account.Role,
		&account.Private,
		&account.Savings,
		&account.Title,
		&account.InBalance,
		&account.CreditLimit,
		&account.StartBalance,
		&account.Balance,
		&account.Company,
		&account.Archive,
		&account.EnableCorrection,
		&account.BalanceCorrectionType,
		&account.StartDate,
		&account.Capitalization,
		&account.Percent,
		&account.Changed,
		&account.SyncID,
		&account.EnableSMS,
		&account.EndDateOffset,
		&account.EndDateOffsetInterval,
		&account.PayoffStep,
		&account.PayoffInterval,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("account not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get account: %w", err)
	}

	return account, nil
}

// ListAccounts retrieves a list of accounts based on the provided filter
func (s *DB) ListAccounts(ctx context.Context, filter interfaces.Filter) ([]models.Account, error) {
	var conditions []string
	var args []interface{}
	argNum := 1

	// Build WHERE clause based on filter
	if filter.UserID != nil {
		conditions = append(conditions, fmt.Sprintf(`"user" = $%d`, argNum))
		args = append(args, *filter.UserID)
		argNum++
	}

	query := `
        SELECT id, "user", instrument, type, role, private, savings,
               title, in_balance, credit_limit, start_balance, balance,
               company, archive, enable_correction, balance_correction_type,
               start_date, capitalization, percent, changed, sync_id,
               enable_sms, end_date_offset, end_date_offset_interval,
               payoff_step, payoff_interval
        FROM account`

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	// Add pagination
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argNum, argNum+1)
	args = append(args, filter.Limit, (filter.Page-1)*filter.Limit)

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list accounts: %w", err)
	}
	defer rows.Close()

	var accounts []models.Account
	for rows.Next() {
		var account models.Account
		err := rows.Scan(
			&account.ID,
			&account.User,
			&account.Instrument,
			&account.Type,
			&account.Role,
			&account.Private,
			&account.Savings,
			&account.Title,
			&account.InBalance,
			&account.CreditLimit,
			&account.StartBalance,
			&account.Balance,
			&account.Company,
			&account.Archive,
			&account.EnableCorrection,
			&account.BalanceCorrectionType,
			&account.StartDate,
			&account.Capitalization,
			&account.Percent,
			&account.Changed,
			&account.SyncID,
			&account.EnableSMS,
			&account.EndDateOffset,
			&account.EndDateOffsetInterval,
			&account.PayoffStep,
			&account.PayoffInterval,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan account: %w", err)
		}
		accounts = append(accounts, account)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating accounts: %w", err)
	}

	return accounts, nil
}

// CreateAccount creates a new account record
func (s *DB) CreateAccount(ctx context.Context, account *models.Account) error {
	query := `
        INSERT INTO account (
            id, "user", instrument, type, role, private, savings,
            title, in_balance, credit_limit, start_balance, balance,
            company, archive, enable_correction, balance_correction_type,
            start_date, capitalization, percent, changed, sync_id,
            enable_sms, end_date_offset, end_date_offset_interval,
            payoff_step, payoff_interval
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13,
                 $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26)`

	_, err := s.pool.Exec(ctx, query,
		account.ID,
		account.User,
		account.Instrument,
		account.Type,
		account.Role,
		account.Private,
		account.Savings,
		account.Title,
		account.InBalance,
		account.CreditLimit,
		account.StartBalance,
		account.Balance,
		account.Company,
		account.Archive,
		account.EnableCorrection,
		account.BalanceCorrectionType,
		account.StartDate,
		account.Capitalization,
		account.Percent,
		account.Changed,
		account.SyncID,
		account.EnableSMS,
		account.EndDateOffset,
		account.EndDateOffsetInterval,
		account.PayoffStep,
		account.PayoffInterval,
	)

	if err != nil {
		return fmt.Errorf("failed to create account: %w", err)
	}

	return nil
}

// UpdateAccount updates an existing account record
func (s *DB) UpdateAccount(ctx context.Context, account *models.Account) error {
	query := `
        UPDATE account SET
            "user" = $2,
            instrument = $3,
            type = $4,
            role = $5,
            private = $6,
            savings = $7,
            title = $8,
            in_balance = $9,
            credit_limit = $10,
            start_balance = $11,
            balance = $12,
            company = $13,
            archive = $14,
            enable_correction = $15,
            balance_correction_type = $16,
            start_date = $17,
            capitalization = $18,
            percent = $19,
            changed = $20,
            sync_id = $21,
            enable_sms = $22,
            end_date_offset = $23,
            end_date_offset_interval = $24,
            payoff_step = $25,
            payoff_interval = $26
        WHERE id = $1`

	commandTag, err := s.pool.Exec(ctx, query,
		account.ID,
		account.User,
		account.Instrument,
		account.Type,
		account.Role,
		account.Private,
		account.Savings,
		account.Title,
		account.InBalance,
		account.CreditLimit,
		account.StartBalance,
		account.Balance,
		account.Company,
		account.Archive,
		account.EnableCorrection,
		account.BalanceCorrectionType,
		account.StartDate,
		account.Capitalization,
		account.Percent,
		account.Changed,
		account.SyncID,
		account.EnableSMS,
		account.EndDateOffset,
		account.EndDateOffsetInterval,
		account.PayoffStep,
		account.PayoffInterval,
	)

	if err != nil {
		return fmt.Errorf("failed to update account: %w", err)
	}

	if commandTag.RowsAffected() == 0 {
		return fmt.Errorf("account not found: %s", account.ID)
	}

	return nil
}

// DeleteAccount deletes an account by its ID
func (s *DB) DeleteAccount(ctx context.Context, id string) error {
	query := `DELETE FROM account WHERE id = $1`

	commandTag, err := s.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete account: %w", err)
	}

	if commandTag.RowsAffected() == 0 {
		return fmt.Errorf("account not found: %s", id)
	}

	return nil
}
