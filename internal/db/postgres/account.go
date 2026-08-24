package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/nemirlev/zenmoney-export/v2/internal/interfaces"
	"github.com/nemirlev/zenmoney-go-sdk/v3/models"
)

const accountSelectQuery = `
        SELECT id, "user", instrument, type, role, private, savings,
               title, in_balance, credit_limit, start_balance, balance,
               company, archive, enable_correction, balance_correction_type,
               to_char(start_date, 'YYYY-MM-DD') AS start_date,
               capitalization, percent, changed, sync_id,
               enable_sms, end_date_offset, end_date_offset_interval,
               payoff_step, payoff_interval
        FROM account`

const accountInsertQuery = `
        INSERT INTO account (
            id, "user", instrument, type, role, private, savings,
            title, in_balance, credit_limit, start_balance, balance,
            company, archive, enable_correction, balance_correction_type,
            start_date, capitalization, percent, changed, sync_id,
            enable_sms, end_date_offset, end_date_offset_interval,
            payoff_step, payoff_interval
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10::text::numeric,
                 $11::text::numeric, $12::text::numeric, $13,
                 $14, $15, $16, NULLIF(BTRIM($17::text), '')::date, $18,
                 $19::text::numeric, $20,
                 $21, $22, $23, $24, $25, $26)`

func scanAccount(row rowScanner) (models.Account, error) {
	var account models.Account
	err := row.Scan(
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
	return account, err
}

func accountValues(account *models.Account) []any {
	return []any{
		account.ID,
		account.User,
		account.Instrument,
		account.Type,
		account.Role,
		account.Private,
		account.Savings,
		account.Title,
		account.InBalance,
		optionalDecimalString(account.CreditLimit),
		optionalDecimalString(account.StartBalance),
		optionalDecimalString(account.Balance),
		account.Company,
		account.Archive,
		account.EnableCorrection,
		account.BalanceCorrectionType,
		account.StartDate,
		account.Capitalization,
		optionalDecimalString(account.Percent),
		account.Changed,
		account.SyncID,
		account.EnableSMS,
		account.EndDateOffset,
		account.EndDateOffsetInterval,
		account.PayoffStep,
		account.PayoffInterval,
	}
}

// GetAccount retrieves a specific account by its ID
func (s *DB) GetAccount(ctx context.Context, id string) (*models.Account, error) {
	account, err := scanAccount(s.pool.QueryRow(ctx, accountSelectQuery+` WHERE id = $1`, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("account not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get account: %w", err)
	}

	return &account, nil
}

// ListAccounts retrieves a list of accounts based on the provided filter
func (s *DB) ListAccounts(ctx context.Context, filter interfaces.Filter) ([]models.Account, error) {
	query, args := buildListQuery(accountSelectQuery, filter, false, "")

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list accounts: %w", err)
	}
	defer rows.Close()

	return collectRows(rows, scanAccount, "failed to scan account", "error iterating accounts")
}

// CreateAccount creates a new account record
func (s *DB) CreateAccount(ctx context.Context, account *models.Account) error {
	_, err := execCommand(
		ctx,
		s.pool,
		accountInsertQuery,
		"failed to create account",
		accountValues(account)...,
	)
	return err
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
            credit_limit = $10::text::numeric,
            start_balance = $11::text::numeric,
            balance = $12::text::numeric,
            company = $13,
            archive = $14,
            enable_correction = $15,
            balance_correction_type = $16,
            start_date = NULLIF(BTRIM($17::text), '')::date,
            capitalization = $18,
            percent = $19::text::numeric,
            changed = $20,
            sync_id = $21,
            enable_sms = $22,
            end_date_offset = $23,
            end_date_offset_interval = $24,
            payoff_step = $25,
            payoff_interval = $26
        WHERE id = $1`

	commandTag, err := execCommand(
		ctx,
		s.pool,
		query,
		"failed to update account",
		accountValues(account)...,
	)
	if err != nil {
		return err
	}

	return requireRowsAffected(commandTag, fmt.Sprintf("account not found: %s", account.ID))
}

// DeleteAccount deletes an account by its ID
func (s *DB) DeleteAccount(ctx context.Context, id string) error {
	query := `DELETE FROM account WHERE id = $1`

	commandTag, err := execCommand(ctx, s.pool, query, "failed to delete account", id)
	if err != nil {
		return err
	}

	return requireRowsAffected(commandTag, fmt.Sprintf("account not found: %s", id))
}
