// internal/db/postgres.go

package postgres

import (
	"context"
	"errors"
	"fmt"
	"github.com/nemirlev/zenmoney-export/internal/db"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nemirlev/zenmoney-go-sdk/v2/models"
)

type DB struct {
	pool *pgxpool.Pool
}

// NewPostgresStorage creates a new PostgreSQL storage instance
func NewPostgresStorage(connectionString string) (db.Storage, error) {
	config, err := pgxpool.ParseConfig(connectionString)
	if err != nil {
		return nil, fmt.Errorf("failed to parse postgres config: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return nil, fmt.Errorf("failed to create postgres connection pool: %w", err)
	}

	return &DB{
		pool: pool,
	}, nil
}

// Close closes the database connection pool
func (s *DB) Close(ctx context.Context) error {
	s.pool.Close()
	return nil
}

// Ping checks if the database is accessible
func (s *DB) Ping(ctx context.Context) error {
	return s.pool.Ping(ctx)
}

// Save saves the entire API response to database
func (s *DB) Save(ctx context.Context, response *models.Response) error {
	// Begin transaction
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Save all entities within transaction
	if len(response.Instrument) > 0 {
		if err := s.SaveInstruments(ctx, response.Instrument); err != nil {
			return fmt.Errorf("failed to save instruments: %w", err)
		}
	}

	if len(response.Country) > 0 {
		if err := s.SaveCountries(ctx, response.Country); err != nil {
			return fmt.Errorf("failed to save countries: %w", err)
		}
	}

	if len(response.Company) > 0 {
		if err := s.SaveCompanies(ctx, response.Company); err != nil {
			return fmt.Errorf("failed to save companies: %w", err)
		}
	}

	if len(response.User) > 0 {
		if err := s.SaveUsers(ctx, response.User); err != nil {
			return fmt.Errorf("failed to save users: %w", err)
		}
	}

	if len(response.Account) > 0 {
		if err := s.SaveAccounts(ctx, response.Account); err != nil {
			return fmt.Errorf("failed to save accounts: %w", err)
		}
	}

	if len(response.Tag) > 0 {
		if err := s.SaveTags(ctx, response.Tag); err != nil {
			return fmt.Errorf("failed to save tags: %w", err)
		}
	}

	if len(response.Merchant) > 0 {
		if err := s.SaveMerchants(ctx, response.Merchant); err != nil {
			return fmt.Errorf("failed to save merchants: %w", err)
		}
	}

	if len(response.Budget) > 0 {
		if err := s.SaveBudgets(ctx, response.Budget); err != nil {
			return fmt.Errorf("failed to save budgets: %w", err)
		}
	}

	if len(response.Reminder) > 0 {
		if err := s.SaveReminders(ctx, response.Reminder); err != nil {
			return fmt.Errorf("failed to save reminders: %w", err)
		}
	}

	if len(response.ReminderMarker) > 0 {
		if err := s.SaveReminderMarkers(ctx, response.ReminderMarker); err != nil {
			return fmt.Errorf("failed to save reminder markers: %w", err)
		}
	}

	if len(response.Transaction) > 0 {
		if err := s.SaveTransactions(ctx, response.Transaction); err != nil {
			return fmt.Errorf("failed to save transactions: %w", err)
		}
	}

	// Process deletions if any
	if len(response.Deletion) > 0 {
		if err := s.DeleteObjects(ctx, response.Deletion); err != nil {
			return fmt.Errorf("failed to process deletions: %w", err)
		}
	}

	// Save sync status
	status := db.SyncStatus{
		StartedAt:        time.Now(),
		FinishedAt:       nil, // Will be set after commit
		SyncType:         "full",
		ServerTimestamp:  int64(response.ServerTimestamp),
		RecordsProcessed: s.countRecords(response),
		Status:           "completed",
		ErrorMessage:     nil,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	if err := s.SaveSyncStatus(ctx, status); err != nil {
		return fmt.Errorf("failed to save sync status: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// countRecords counts total number of records in response
func (s *DB) countRecords(response *models.Response) int {
	return len(response.Instrument) +
		len(response.Country) +
		len(response.Company) +
		len(response.User) +
		len(response.Account) +
		len(response.Tag) +
		len(response.Merchant) +
		len(response.Budget) +
		len(response.Reminder) +
		len(response.ReminderMarker) +
		len(response.Transaction) +
		len(response.Deletion)
}

// SaveSyncStatus saves synchronization status to the database
// It creates a new record in the sync_status table with the provided status information
func (s *DB) SaveSyncStatus(ctx context.Context, status db.SyncStatus) error {
	query := `
        INSERT INTO sync_status (
            started_at, finished_at, sync_type, server_timestamp,
            records_processed, status, error_message, created_at, updated_at
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
        RETURNING id`

	err := s.pool.QueryRow(ctx, query,
		status.StartedAt, status.FinishedAt, status.SyncType,
		status.ServerTimestamp, status.RecordsProcessed,
		status.Status, status.ErrorMessage,
		time.Now(), time.Now(),
	).Scan(&status.ID)

	if err != nil {
		return fmt.Errorf("failed to save sync status: %w", err)
	}

	return nil
}

// GetLastSyncStatus retrieves the latest synchronization status from the database
// Returns the most recent sync_status record ordered by ID
func (s *DB) GetLastSyncStatus(ctx context.Context) (db.SyncStatus, error) {
	var status db.SyncStatus
	query := `
        SELECT id, started_at, finished_at, sync_type, server_timestamp,
               records_processed, status, error_message, created_at, updated_at
        FROM sync_status
        ORDER BY id DESC
        LIMIT 1`

	err := s.pool.QueryRow(ctx, query).Scan(
		&status.ID, &status.StartedAt, &status.FinishedAt,
		&status.SyncType, &status.ServerTimestamp,
		&status.RecordsProcessed, &status.Status,
		&status.ErrorMessage, &status.CreatedAt, &status.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return db.SyncStatus{}, fmt.Errorf("no sync status found")
		}
		return db.SyncStatus{}, fmt.Errorf("failed to get last sync status: %w", err)
	}

	return status, nil
}

// SaveInstruments saves a batch of instruments to the database
// It performs an upsert operation: inserts new records and updates existing ones based on their ID
func (s *DB) SaveInstruments(ctx context.Context, instruments []models.Instrument) error {
	if len(instruments) == 0 {
		return nil
	}

	query := `
        INSERT INTO instrument (id, title, short_title, symbol, rate, changed)
        VALUES ($1, $2, $3, $4, $5, $6)
        ON CONFLICT (id) DO UPDATE SET
            title = EXCLUDED.title,
            short_title = EXCLUDED.short_title,
            symbol = EXCLUDED.symbol,
            rate = EXCLUDED.rate,
            changed = EXCLUDED.changed`

	batch := &pgx.Batch{}
	for _, inst := range instruments {
		batch.Queue(query, inst.ID, inst.Title, inst.ShortTitle, inst.Symbol, inst.Rate, inst.Changed)
	}

	br := s.pool.SendBatch(ctx, batch)
	defer br.Close()

	for i := 0; i < batch.Len(); i++ {
		if _, err := br.Exec(); err != nil {
			return fmt.Errorf("failed to save instrument %d: %w", i, err)
		}
	}

	return nil
}

// SaveCountries saves a batch of countries to the database
// It performs an upsert operation: inserts new records and updates existing ones based on their ID
func (s *DB) SaveCountries(ctx context.Context, countries []models.Country) error {
	if len(countries) == 0 {
		return nil
	}

	query := `
        INSERT INTO country (id, title, currency, domain)
        VALUES ($1, $2, $3, $4)
        ON CONFLICT (id) DO UPDATE SET
            title = EXCLUDED.title,
            currency = EXCLUDED.currency,
            domain = EXCLUDED.domain`

	batch := &pgx.Batch{}
	for _, country := range countries {
		batch.Queue(query, country.ID, country.Title, country.Currency, country.Domain)
	}

	br := s.pool.SendBatch(ctx, batch)
	defer br.Close()

	for i := 0; i < batch.Len(); i++ {
		if _, err := br.Exec(); err != nil {
			return fmt.Errorf("failed to save country %d: %w", i, err)
		}
	}

	return nil
}

// SaveCompanies saves a batch of companies to the database
// It performs an upsert operation: inserts new records and updates existing ones based on their ID
func (s *DB) SaveCompanies(ctx context.Context, companies []models.Company) error {
	if len(companies) == 0 {
		return nil
	}

	query := `
        INSERT INTO company (
            id, title, full_title, www, country, deleted,
            country_code, changed
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
        ON CONFLICT (id) DO UPDATE SET
            title = EXCLUDED.title,
            full_title = EXCLUDED.full_title,
            www = EXCLUDED.www,
            country = EXCLUDED.country,
            deleted = EXCLUDED.deleted,
            country_code = EXCLUDED.country_code,
            changed = EXCLUDED.changed`

	batch := &pgx.Batch{}
	for _, company := range companies {
		batch.Queue(query,
			company.ID, company.Title, company.FullTitle,
			company.Www, company.Country, company.Deleted,
			company.CountryCode, company.Changed,
		)
	}

	br := s.pool.SendBatch(ctx, batch)
	defer br.Close()

	for i := 0; i < batch.Len(); i++ {
		if _, err := br.Exec(); err != nil {
			return fmt.Errorf("failed to save company %d: %w", i, err)
		}
	}

	return nil
}

// SaveUsers saves a batch of users to the database
// It performs an upsert operation: inserts new records and updates existing ones based on their ID
func (s *DB) SaveUsers(ctx context.Context, users []models.User) error {
	if len(users) == 0 {
		return nil
	}

	query := `
        INSERT INTO "user" (
            id, country, login, parent, country_code, email,
            changed, currency, paid_till, month_start_day,
            is_forecast_enabled, plan_balance_mode, plan_settings,
            subscription, subscription_renewal_date
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
        ON CONFLICT (id) DO UPDATE SET
            country = EXCLUDED.country,
            login = EXCLUDED.login,
            parent = EXCLUDED.parent,
            country_code = EXCLUDED.country_code,
            email = EXCLUDED.email,
            changed = EXCLUDED.changed,
            currency = EXCLUDED.currency,
            paid_till = EXCLUDED.paid_till,
            month_start_day = EXCLUDED.month_start_day,
            is_forecast_enabled = EXCLUDED.is_forecast_enabled,
            plan_balance_mode = EXCLUDED.plan_balance_mode,
            plan_settings = EXCLUDED.plan_settings,
            subscription = EXCLUDED.subscription,
            subscription_renewal_date = EXCLUDED.subscription_renewal_date`

	batch := &pgx.Batch{}
	for _, user := range users {
		batch.Queue(query,
			user.ID, user.Country, user.Login, user.Parent,
			user.CountryCode, user.Email, user.Changed,
			user.Currency, user.PaidTill, user.MonthStartDay,
			user.IsForecastEnabled, user.PlanBalanceMode,
			user.PlanSettings, user.Subscription,
			user.SubscriptionRenewalDate,
		)
	}

	br := s.pool.SendBatch(ctx, batch)
	defer br.Close()

	for i := 0; i < batch.Len(); i++ {
		if _, err := br.Exec(); err != nil {
			return fmt.Errorf("failed to save user %d: %w", i, err)
		}
	}

	return nil
}

// SaveAccounts saves a batch of accounts to the database
func (s *DB) SaveAccounts(ctx context.Context, accounts []models.Account) error {
	if len(accounts) == 0 {
		return nil
	}

	query := `
        INSERT INTO account (
            id, "user", instrument, type, role, private, savings,
            title, in_balance, credit_limit, start_balance, balance,
            company, archive, enable_correction, balance_correction_type,
            start_date, capitalization, percent, changed, sync_id,
            enable_sms, end_date_offset, end_date_offset_interval,
            payoff_step, payoff_interval
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13,
                  $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26)
        ON CONFLICT (id) DO UPDATE SET
            "user" = EXCLUDED.user,
            instrument = EXCLUDED.instrument,
            type = EXCLUDED.type,
            role = EXCLUDED.role,
            private = EXCLUDED.private,
            savings = EXCLUDED.savings,
            title = EXCLUDED.title,
            in_balance = EXCLUDED.in_balance,
            credit_limit = EXCLUDED.credit_limit,
            start_balance = EXCLUDED.start_balance,
            balance = EXCLUDED.balance,
            company = EXCLUDED.company,
            archive = EXCLUDED.archive,
            enable_correction = EXCLUDED.enable_correction,
            balance_correction_type = EXCLUDED.balance_correction_type,
            start_date = EXCLUDED.start_date,
            capitalization = EXCLUDED.capitalization,
            percent = EXCLUDED.percent,
            changed = EXCLUDED.changed,
            sync_id = EXCLUDED.sync_id,
            enable_sms = EXCLUDED.enable_sms,
            end_date_offset = EXCLUDED.end_date_offset,
            end_date_offset_interval = EXCLUDED.end_date_offset_interval,
            payoff_step = EXCLUDED.payoff_step,
            payoff_interval = EXCLUDED.payoff_interval`

	batch := &pgx.Batch{}
	for _, account := range accounts {
		batch.Queue(query,
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
	}

	br := s.pool.SendBatch(ctx, batch)
	defer br.Close()

	for i := 0; i < batch.Len(); i++ {
		if _, err := br.Exec(); err != nil {
			return fmt.Errorf("failed to save account %d: %w", i, err)
		}
	}

	return nil
}

// SaveTags saves a batch of tags to the database
func (s *DB) SaveTags(ctx context.Context, tags []models.Tag) error {
	if len(tags) == 0 {
		return nil
	}

	query := `
        INSERT INTO tag (
            id, "user", changed, icon, budget_income, budget_outcome,
            required, color, picture, title, show_income, show_outcome,
            parent, static_id
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
        ON CONFLICT (id) DO UPDATE SET
            "user" = EXCLUDED.user,
            changed = EXCLUDED.changed,
            icon = EXCLUDED.icon,
            budget_income = EXCLUDED.budget_income,
            budget_outcome = EXCLUDED.budget_outcome,
            required = EXCLUDED.required,
            color = EXCLUDED.color,
            picture = EXCLUDED.picture,
            title = EXCLUDED.title,
            show_income = EXCLUDED.show_income,
            show_outcome = EXCLUDED.show_outcome,
            parent = EXCLUDED.parent,
            static_id = EXCLUDED.static_id`

	batch := &pgx.Batch{}
	for _, tag := range tags {
		batch.Queue(query,
			tag.ID,
			tag.User,
			tag.Changed,
			tag.Icon,
			tag.BudgetIncome,
			tag.BudgetOutcome,
			tag.Required,
			tag.Color,
			tag.Picture,
			tag.Title,
			tag.ShowIncome,
			tag.ShowOutcome,
			tag.Parent,
			tag.StaticID,
		)
	}

	br := s.pool.SendBatch(ctx, batch)
	defer br.Close()

	for i := 0; i < batch.Len(); i++ {
		if _, err := br.Exec(); err != nil {
			return fmt.Errorf("failed to save tag %d: %w", i, err)
		}
	}

	return nil
}

// SaveMerchants saves a batch of merchants to the database
func (s *DB) SaveMerchants(ctx context.Context, merchants []models.Merchant) error {
	if len(merchants) == 0 {
		return nil
	}

	query := `
        INSERT INTO merchant (id, "user", title, changed)
        VALUES ($1, $2, $3, $4)
        ON CONFLICT (id) DO UPDATE SET
            "user" = EXCLUDED.user,
            title = EXCLUDED.title,
            changed = EXCLUDED.changed`

	batch := &pgx.Batch{}
	for _, merchant := range merchants {
		batch.Queue(query,
			merchant.ID,
			merchant.User,
			merchant.Title,
			merchant.Changed,
		)
	}

	br := s.pool.SendBatch(ctx, batch)
	defer br.Close()

	for i := 0; i < batch.Len(); i++ {
		if _, err := br.Exec(); err != nil {
			return fmt.Errorf("failed to save merchant %d: %w", i, err)
		}
	}

	return nil
}

// SaveBudgets saves a batch of budgets to the database
func (s *DB) SaveBudgets(ctx context.Context, budgets []models.Budget) error {
	if len(budgets) == 0 {
		return nil
	}

	query := `
        INSERT INTO budget (
            "user", changed, date, tag, income, outcome,
            income_lock, outcome_lock, is_income_forecast, is_outcome_forecast
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
        ON CONFLICT ("user", date, tag) DO UPDATE SET
            changed = EXCLUDED.changed,
            income = EXCLUDED.income,
            outcome = EXCLUDED.outcome,
            income_lock = EXCLUDED.income_lock,
            outcome_lock = EXCLUDED.outcome_lock,
            is_income_forecast = EXCLUDED.is_income_forecast,
            is_outcome_forecast = EXCLUDED.is_outcome_forecast`

	batch := &pgx.Batch{}
	for _, budget := range budgets {
		batch.Queue(query,
			budget.User,
			budget.Changed,
			budget.Date,
			budget.Tag,
			budget.Income,
			budget.Outcome,
			budget.IncomeLock,
			budget.OutcomeLock,
			budget.IsIncomeForecast,
			budget.IsOutcomeForecast,
		)
	}

	br := s.pool.SendBatch(ctx, batch)
	defer br.Close()

	for i := 0; i < batch.Len(); i++ {
		if _, err := br.Exec(); err != nil {
			return fmt.Errorf("failed to save budget %d: %w", i, err)
		}
	}

	return nil
}

// SaveReminders saves a batch of reminders to the database
func (s *DB) SaveReminders(ctx context.Context, reminders []models.Reminder) error {
	if len(reminders) == 0 {
		return nil
	}

	query := `
        INSERT INTO reminder (
            id, "user", income, outcome, changed, income_instrument,
            outcome_instrument, step, points, tag, start_date, end_date,
            notify, interval, income_account, outcome_account, comment,
            payee, merchant
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13,
                  $14, $15, $16, $17, $18, $19)
        ON CONFLICT (id) DO UPDATE SET
            "user" = EXCLUDED.user,
            income = EXCLUDED.income,
            outcome = EXCLUDED.outcome,
            changed = EXCLUDED.changed,
            income_instrument = EXCLUDED.income_instrument,
            outcome_instrument = EXCLUDED.outcome_instrument,
            step = EXCLUDED.step,
            points = EXCLUDED.points,
            tag = EXCLUDED.tag,
            start_date = EXCLUDED.start_date,
            end_date = EXCLUDED.end_date,
            notify = EXCLUDED.notify,
            interval = EXCLUDED.interval,
            income_account = EXCLUDED.income_account,
            outcome_account = EXCLUDED.outcome_account,
            comment = EXCLUDED.comment,
            payee = EXCLUDED.payee,
            merchant = EXCLUDED.merchant`

	batch := &pgx.Batch{}
	for _, reminder := range reminders {
		batch.Queue(query,
			reminder.ID,
			reminder.User,
			reminder.Income,
			reminder.Outcome,
			reminder.Changed,
			reminder.IncomeInstrument,
			reminder.OutcomeInstrument,
			reminder.Step,
			reminder.Points,
			reminder.Tag,
			reminder.StartDate,
			reminder.EndDate,
			reminder.Notify,
			reminder.Interval,
			reminder.IncomeAccount,
			reminder.OutcomeAccount,
			reminder.Comment,
			reminder.Payee,
			reminder.Merchant,
		)
	}

	br := s.pool.SendBatch(ctx, batch)
	defer br.Close()

	for i := 0; i < batch.Len(); i++ {
		if _, err := br.Exec(); err != nil {
			return fmt.Errorf("failed to save reminder %d: %w", i, err)
		}
	}

	return nil
}

// SaveReminderMarkers saves a batch of reminder markers to the database
func (s *DB) SaveReminderMarkers(ctx context.Context, markers []models.ReminderMarker) error {
	if len(markers) == 0 {
		return nil
	}

	query := `
        INSERT INTO reminder_marker (
            id, "user", date, income, outcome, changed,
            income_instrument, outcome_instrument, state, is_forecast,
            reminder, income_account, outcome_account, comment,
            payee, merchant, notify, tag
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13,
                  $14, $15, $16, $17, $18)
        ON CONFLICT (id) DO UPDATE SET
            "user" = EXCLUDED.user,
            date = EXCLUDED.date,
            income = EXCLUDED.income,
            outcome = EXCLUDED.outcome,
            changed = EXCLUDED.changed,
            income_instrument = EXCLUDED.income_instrument,
            outcome_instrument = EXCLUDED.outcome_instrument,
            state = EXCLUDED.state,
            is_forecast = EXCLUDED.is_forecast,
            reminder = EXCLUDED.reminder,
            income_account = EXCLUDED.income_account,
            outcome_account = EXCLUDED.outcome_account,
            comment = EXCLUDED.comment,
            payee = EXCLUDED.payee,
            merchant = EXCLUDED.merchant,
            notify = EXCLUDED.notify,
            tag = EXCLUDED.tag`

	batch := &pgx.Batch{}
	for _, marker := range markers {
		batch.Queue(query,
			marker.ID,
			marker.User,
			marker.Date,
			marker.Income,
			marker.Outcome,
			marker.Changed,
			marker.IncomeInstrument,
			marker.OutcomeInstrument,
			marker.State,
			marker.IsForecast,
			marker.Reminder,
			marker.IncomeAccount,
			marker.OutcomeAccount,
			marker.Comment,
			marker.Payee,
			marker.Merchant,
			marker.Notify,
			marker.Tag,
		)
	}

	br := s.pool.SendBatch(ctx, batch)
	defer br.Close()

	for i := 0; i < batch.Len(); i++ {
		if _, err := br.Exec(); err != nil {
			return fmt.Errorf("failed to save reminder marker %d: %w", i, err)
		}
	}

	return nil
}

// SaveTransactions saves a batch of transactions to the database
func (s *DB) SaveTransactions(ctx context.Context, transactions []models.Transaction) error {
	if len(transactions) == 0 {
		return nil
	}

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
                 $25, $26, $27, $28, $29, $30)
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

	batch := &pgx.Batch{}
	for _, tx := range transactions {
		batch.Queue(query,
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
	}

	br := s.pool.SendBatch(ctx, batch)
	defer br.Close()

	for i := 0; i < batch.Len(); i++ {
		if _, err := br.Exec(); err != nil {
			return fmt.Errorf("failed to save transaction %d: %w", i, err)
		}
	}

	return nil
}

// DeleteObjects handles deletion of multiple objects from different tables
// based on the Deletion objects received from ZenMoney API.
// It processes deletions in a single transaction to ensure data consistency.
// Each Deletion object contains:
// - ID: the object's ID
// - Object: the type of object (e.g., "transaction", "account", etc.)
// - User: the user ID
// - Stamp: timestamp of deletion
func (s *DB) DeleteObjects(ctx context.Context, deletions []models.Deletion) error {
	if len(deletions) == 0 {
		return nil
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Process each deletion
	for _, del := range deletions {
		query := ""
		switch del.Object {
		case string(models.EntityTypeAccount):
			query = `DELETE FROM account WHERE id = $1 AND "user" = $2`
		case string(models.EntityTypeTag):
			query = `DELETE FROM tag WHERE id = $1 AND "user" = $2`
		case string(models.EntityTypeMerchant):
			query = `DELETE FROM merchant WHERE id = $1 AND "user" = $2`
		case string(models.EntityTypeBudget):
			query = `DELETE FROM budget WHERE "user" = $1 AND date = $2`
		case string(models.EntityTypeReminder):
			query = `DELETE FROM reminder WHERE id = $1 AND "user" = $2`
		case string(models.EntityTypeReminderMarker):
			query = `DELETE FROM reminder_marker WHERE id = $1 AND "user" = $2`
		case string(models.EntityTypeTransaction):
			query = `DELETE FROM transaction WHERE id = $1 AND "user" = $2`
		default:
			return fmt.Errorf("unsupported object type for deletion: %s", del.Object)
		}

		// Execute the delete query
		commandTag, err := tx.Exec(ctx, query, del.ID, del.User)
		if err != nil {
			return fmt.Errorf("failed to delete %s with ID %s: %w", del.Object, del.ID, err)
		}

		// Check if any row was actually deleted
		if commandTag.RowsAffected() == 0 {
			// Log warning but don't return error as the object might have been already deleted
			fmt.Printf("warning: no %s found for deletion with ID %s and user %d\n",
				del.Object, del.ID, del.User)
		}

		// Record the deletion in deletion_history table for audit
		_, err = tx.Exec(ctx, `
            INSERT INTO deletion_history (
                object_id, object_type, user_id, deleted_at
            ) VALUES ($1, $2, $3, to_timestamp($4))`,
			del.ID, del.Object, del.User, del.Stamp,
		)
		if err != nil {
			return fmt.Errorf("failed to record deletion history: %w", err)
		}
	}

	// Commit the transaction
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit deletion transaction: %w", err)
	}

	return nil
}

// GetInstrument retrieves a specific instrument by its ID
func (s *DB) GetInstrument(ctx context.Context, id int) (*models.Instrument, error) {
	query := `
        SELECT id, title, short_title, symbol, rate, changed
        FROM instrument
        WHERE id = $1`

	instrument := &models.Instrument{}
	err := s.pool.QueryRow(ctx, query, id).Scan(
		&instrument.ID,
		&instrument.Title,
		&instrument.ShortTitle,
		&instrument.Symbol,
		&instrument.Rate,
		&instrument.Changed,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("instrument not found: %d", id)
		}
		return nil, fmt.Errorf("failed to get instrument: %w", err)
	}

	return instrument, nil
}

// ListInstruments retrieves a list of instruments based on the provided filter
func (s *DB) ListInstruments(ctx context.Context, filter db.Filter) ([]models.Instrument, error) {
	var conditions []string
	var args []interface{}
	argNum := 1

	// Build the WHERE clause based on filter
	if filter.UserID != nil {
		conditions = append(conditions, fmt.Sprintf("user_id = $%d", argNum))
		args = append(args, *filter.UserID)
		argNum++
	}

	query := "SELECT id, title, short_title, symbol, rate, changed FROM instrument"
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	// Add pagination
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argNum, argNum+1)
	args = append(args, filter.Limit, (filter.Page-1)*filter.Limit)

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list instruments: %w", err)
	}
	defer rows.Close()

	var instruments []models.Instrument
	for rows.Next() {
		var instrument models.Instrument
		err := rows.Scan(
			&instrument.ID,
			&instrument.Title,
			&instrument.ShortTitle,
			&instrument.Symbol,
			&instrument.Rate,
			&instrument.Changed,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan instrument: %w", err)
		}
		instruments = append(instruments, instrument)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating instruments: %w", err)
	}

	return instruments, nil
}

// CreateInstrument creates a new instrument record
func (s *DB) CreateInstrument(ctx context.Context, instrument *models.Instrument) error {
	query := `
        INSERT INTO instrument (id, title, short_title, symbol, rate, changed)
        VALUES ($1, $2, $3, $4, $5, $6)`

	_, err := s.pool.Exec(ctx, query,
		instrument.ID,
		instrument.Title,
		instrument.ShortTitle,
		instrument.Symbol,
		instrument.Rate,
		instrument.Changed,
	)
	if err != nil {
		return fmt.Errorf("failed to create instrument: %w", err)
	}

	return nil
}

// UpdateInstrument updates an existing instrument record
func (s *DB) UpdateInstrument(ctx context.Context, instrument *models.Instrument) error {
	query := `
        UPDATE instrument
        SET title = $2, short_title = $3, symbol = $4, rate = $5, changed = $6
        WHERE id = $1`

	commandTag, err := s.pool.Exec(ctx, query,
		instrument.ID,
		instrument.Title,
		instrument.ShortTitle,
		instrument.Symbol,
		instrument.Rate,
		instrument.Changed,
	)
	if err != nil {
		return fmt.Errorf("failed to update instrument: %w", err)
	}

	if commandTag.RowsAffected() == 0 {
		return fmt.Errorf("instrument not found: %d", instrument.ID)
	}

	return nil
}

// DeleteInstrument deletes an instrument by its ID
func (s *DB) DeleteInstrument(ctx context.Context, id int) error {
	query := `DELETE FROM instrument WHERE id = $1`

	commandTag, err := s.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete instrument: %w", err)
	}

	if commandTag.RowsAffected() == 0 {
		return fmt.Errorf("instrument not found: %d", id)
	}

	return nil
}

// GetCompany retrieves a specific company by its ID
func (s *DB) GetCompany(ctx context.Context, id int) (*models.Company, error) {
	query := `
        SELECT id, title, full_title, www, country, deleted, country_code, changed
        FROM company
        WHERE id = $1`

	company := &models.Company{}
	err := s.pool.QueryRow(ctx, query, id).Scan(
		&company.ID,
		&company.Title,
		&company.FullTitle,
		&company.Www,
		&company.Country,
		&company.Deleted,
		&company.CountryCode,
		&company.Changed,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("company not found: %d", id)
		}
		return nil, fmt.Errorf("failed to get company: %w", err)
	}

	return company, nil
}

// ListCompanies retrieves a list of companies based on the provided filter
func (s *DB) ListCompanies(ctx context.Context, filter db.Filter) ([]models.Company, error) {
	var conditions []string
	var args []interface{}
	argNum := 1

	// Build the WHERE clause based on filter
	if filter.UserID != nil {
		conditions = append(conditions, fmt.Sprintf("user_id = $%d", argNum))
		args = append(args, *filter.UserID)
		argNum++
	}

	query := `
        SELECT id, title, full_title, www, country, deleted, country_code, changed
        FROM company`

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	// Add pagination
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argNum, argNum+1)
	args = append(args, filter.Limit, (filter.Page-1)*filter.Limit)

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list companies: %w", err)
	}
	defer rows.Close()

	var companies []models.Company
	for rows.Next() {
		var company models.Company
		err := rows.Scan(
			&company.ID,
			&company.Title,
			&company.FullTitle,
			&company.Www,
			&company.Country,
			&company.Deleted,
			&company.CountryCode,
			&company.Changed,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan company: %w", err)
		}
		companies = append(companies, company)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating companies: %w", err)
	}

	return companies, nil
}

// CreateCompany creates a new company record
func (s *DB) CreateCompany(ctx context.Context, company *models.Company) error {
	query := `
        INSERT INTO company (
            id, title, full_title, www, country, deleted,
            country_code, changed
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	_, err := s.pool.Exec(ctx, query,
		company.ID,
		company.Title,
		company.FullTitle,
		company.Www,
		company.Country,
		company.Deleted,
		company.CountryCode,
		company.Changed,
	)
	if err != nil {
		return fmt.Errorf("failed to create company: %w", err)
	}

	return nil
}

// UpdateCompany updates an existing company record
func (s *DB) UpdateCompany(ctx context.Context, company *models.Company) error {
	query := `
        UPDATE company
        SET title = $2, full_title = $3, www = $4, country = $5,
            deleted = $6, country_code = $7, changed = $8
        WHERE id = $1`

	commandTag, err := s.pool.Exec(ctx, query,
		company.ID,
		company.Title,
		company.FullTitle,
		company.Www,
		company.Country,
		company.Deleted,
		company.CountryCode,
		company.Changed,
	)
	if err != nil {
		return fmt.Errorf("failed to update company: %w", err)
	}

	if commandTag.RowsAffected() == 0 {
		return fmt.Errorf("company not found: %d", company.ID)
	}

	return nil
}

// DeleteCompany deletes a company by its ID
func (s *DB) DeleteCompany(ctx context.Context, id int) error {
	query := `DELETE FROM company WHERE id = $1`

	commandTag, err := s.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete company: %w", err)
	}

	if commandTag.RowsAffected() == 0 {
		return fmt.Errorf("company not found: %d", id)
	}

	return nil
}

// GetUser retrieves a specific user by their ID
func (s *DB) GetUser(ctx context.Context, id int) (*models.User, error) {
	query := `
        SELECT id, country, login, parent, country_code, email,
               changed, currency, paid_till, month_start_day,
               is_forecast_enabled, plan_balance_mode, plan_settings,
               subscription, subscription_renewal_date
        FROM "user"
        WHERE id = $1`

	user := &models.User{}
	err := s.pool.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.Country,
		&user.Login,
		&user.Parent,
		&user.CountryCode,
		&user.Email,
		&user.Changed,
		&user.Currency,
		&user.PaidTill,
		&user.MonthStartDay,
		&user.IsForecastEnabled,
		&user.PlanBalanceMode,
		&user.PlanSettings,
		&user.Subscription,
		&user.SubscriptionRenewalDate,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("user not found: %d", id)
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

// ListUsers retrieves a list of users based on the provided filter
func (s *DB) ListUsers(ctx context.Context, filter db.Filter) ([]models.User, error) {
	var conditions []string
	var args []interface{}
	argNum := 1

	// Build the WHERE clause based on filter
	if filter.UserID != nil {
		conditions = append(conditions, fmt.Sprintf("id = $%d", argNum))
		args = append(args, *filter.UserID)
		argNum++
	}

	query := `
        SELECT id, country, login, parent, country_code, email,
               changed, currency, paid_till, month_start_day,
               is_forecast_enabled, plan_balance_mode, plan_settings,
               subscription, subscription_renewal_date
        FROM "user"`

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	// Add pagination
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argNum, argNum+1)
	args = append(args, filter.Limit, (filter.Page-1)*filter.Limit)

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var user models.User
		err := rows.Scan(
			&user.ID,
			&user.Country,
			&user.Login,
			&user.Parent,
			&user.CountryCode,
			&user.Email,
			&user.Changed,
			&user.Currency,
			&user.PaidTill,
			&user.MonthStartDay,
			&user.IsForecastEnabled,
			&user.PlanBalanceMode,
			&user.PlanSettings,
			&user.Subscription,
			&user.SubscriptionRenewalDate,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, user)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating users: %w", err)
	}

	return users, nil
}

// CreateUser creates a new user record
func (s *DB) CreateUser(ctx context.Context, user *models.User) error {
	query := `
        INSERT INTO "user" (
            id, country, login, parent, country_code, email,
            changed, currency, paid_till, month_start_day,
            is_forecast_enabled, plan_balance_mode, plan_settings,
            subscription, subscription_renewal_date
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)`

	_, err := s.pool.Exec(ctx, query,
		user.ID,
		user.Country,
		user.Login,
		user.Parent,
		user.CountryCode,
		user.Email,
		user.Changed,
		user.Currency,
		user.PaidTill,
		user.MonthStartDay,
		user.IsForecastEnabled,
		user.PlanBalanceMode,
		user.PlanSettings,
		user.Subscription,
		user.SubscriptionRenewalDate,
	)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

// UpdateUser updates an existing user record
func (s *DB) UpdateUser(ctx context.Context, user *models.User) error {
	query := `
        UPDATE "user"
        SET country = $2, login = $3, parent = $4, country_code = $5,
            email = $6, changed = $7, currency = $8, paid_till = $9,
            month_start_day = $10, is_forecast_enabled = $11,
            plan_balance_mode = $12, plan_settings = $13,
            subscription = $14, subscription_renewal_date = $15
        WHERE id = $1`

	commandTag, err := s.pool.Exec(ctx, query,
		user.ID,
		user.Country,
		user.Login,
		user.Parent,
		user.CountryCode,
		user.Email,
		user.Changed,
		user.Currency,
		user.PaidTill,
		user.MonthStartDay,
		user.IsForecastEnabled,
		user.PlanBalanceMode,
		user.PlanSettings,
		user.Subscription,
		user.SubscriptionRenewalDate,
	)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	if commandTag.RowsAffected() == 0 {
		return fmt.Errorf("user not found: %d", user.ID)
	}

	return nil
}

// DeleteUser deletes a user by their ID
func (s *DB) DeleteUser(ctx context.Context, id int) error {
	query := `DELETE FROM "user" WHERE id = $1`

	commandTag, err := s.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	if commandTag.RowsAffected() == 0 {
		return fmt.Errorf("user not found: %d", id)
	}

	return nil
}

// GetCountry retrieves a specific country by its ID
func (s *DB) GetCountry(ctx context.Context, id int) (*models.Country, error) {
	query := `
        SELECT id, title, currency, domain
        FROM country
        WHERE id = $1`

	country := &models.Country{}
	err := s.pool.QueryRow(ctx, query, id).Scan(
		&country.ID,
		&country.Title,
		&country.Currency,
		&country.Domain,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("country not found: %d", id)
		}
		return nil, fmt.Errorf("failed to get country: %w", err)
	}

	return country, nil
}

// ListCountries retrieves a list of countries based on the provided filter
func (s *DB) ListCountries(ctx context.Context, filter db.Filter) ([]models.Country, error) {
	query := `
        SELECT id, title, currency, domain
        FROM country
        LIMIT $1 OFFSET $2`

	rows, err := s.pool.Query(ctx, query, filter.Limit, (filter.Page-1)*filter.Limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list countries: %w", err)
	}
	defer rows.Close()

	var countries []models.Country
	for rows.Next() {
		var country models.Country
		err := rows.Scan(
			&country.ID,
			&country.Title,
			&country.Currency,
			&country.Domain,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan country: %w", err)
		}
		countries = append(countries, country)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating countries: %w", err)
	}

	return countries, nil
}

// CreateCountry creates a new country record
func (s *DB) CreateCountry(ctx context.Context, country *models.Country) error {
	query := `
        INSERT INTO country (id, title, currency, domain)
        VALUES ($1, $2, $3, $4)`

	_, err := s.pool.Exec(ctx, query,
		country.ID,
		country.Title,
		country.Currency,
		country.Domain,
	)
	if err != nil {
		return fmt.Errorf("failed to create country: %w", err)
	}

	return nil
}

// UpdateCountry updates an existing country record
func (s *DB) UpdateCountry(ctx context.Context, country *models.Country) error {
	query := `
        UPDATE country
        SET title = $2, currency = $3, domain = $4
        WHERE id = $1`

	commandTag, err := s.pool.Exec(ctx, query,
		country.ID,
		country.Title,
		country.Currency,
		country.Domain,
	)
	if err != nil {
		return fmt.Errorf("failed to update country: %w", err)
	}

	if commandTag.RowsAffected() == 0 {
		return fmt.Errorf("country not found: %d", country.ID)
	}

	return nil
}

// DeleteCountry deletes a country by its ID
func (s *DB) DeleteCountry(ctx context.Context, id int) error {
	query := `DELETE FROM country WHERE id = $1`

	commandTag, err := s.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete country: %w", err)
	}

	if commandTag.RowsAffected() == 0 {
		return fmt.Errorf("country not found: %d", id)
	}

	return nil
}

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
func (s *DB) ListAccounts(ctx context.Context, filter db.Filter) ([]models.Account, error) {
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

// GetTag retrieves a specific tag by its ID
func (s *DB) GetTag(ctx context.Context, id string) (*models.Tag, error) {
	query := `
        SELECT id, "user", changed, icon, budget_income, budget_outcome,
               required, color, picture, title, show_income, show_outcome,
               parent, static_id
        FROM tag
        WHERE id = $1`

	tag := &models.Tag{}
	err := s.pool.QueryRow(ctx, query, id).Scan(
		&tag.ID,
		&tag.User,
		&tag.Changed,
		&tag.Icon,
		&tag.BudgetIncome,
		&tag.BudgetOutcome,
		&tag.Required,
		&tag.Color,
		&tag.Picture,
		&tag.Title,
		&tag.ShowIncome,
		&tag.ShowOutcome,
		&tag.Parent,
		&tag.StaticID,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("tag not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get tag: %w", err)
	}

	return tag, nil
}

// ListTags retrieves a list of tags based on the provided filter
func (s *DB) ListTags(ctx context.Context, filter db.Filter) ([]models.Tag, error) {
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
        SELECT id, "user", changed, icon, budget_income, budget_outcome,
               required, color, picture, title, show_income, show_outcome,
               parent, static_id
        FROM tag`

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	// Add pagination
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argNum, argNum+1)
	args = append(args, filter.Limit, (filter.Page-1)*filter.Limit)

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list tags: %w", err)
	}
	defer rows.Close()

	var tags []models.Tag
	for rows.Next() {
		var tag models.Tag
		err := rows.Scan(
			&tag.ID,
			&tag.User,
			&tag.Changed,
			&tag.Icon,
			&tag.BudgetIncome,
			&tag.BudgetOutcome,
			&tag.Required,
			&tag.Color,
			&tag.Picture,
			&tag.Title,
			&tag.ShowIncome,
			&tag.ShowOutcome,
			&tag.Parent,
			&tag.StaticID,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan tag: %w", err)
		}
		tags = append(tags, tag)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating tags: %w", err)
	}

	return tags, nil
}

// CreateTag creates a new tag record
func (s *DB) CreateTag(ctx context.Context, tag *models.Tag) error {
	query := `
        INSERT INTO tag (
            id, "user", changed, icon, budget_income, budget_outcome,
            required, color, picture, title, show_income, show_outcome,
            parent, static_id
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`

	_, err := s.pool.Exec(ctx, query,
		tag.ID,
		tag.User,
		tag.Changed,
		tag.Icon,
		tag.BudgetIncome,
		tag.BudgetOutcome,
		tag.Required,
		tag.Color,
		tag.Picture,
		tag.Title,
		tag.ShowIncome,
		tag.ShowOutcome,
		tag.Parent,
		tag.StaticID,
	)

	if err != nil {
		return fmt.Errorf("failed to create tag: %w", err)
	}

	return nil
}

// UpdateTag updates an existing tag record
func (s *DB) UpdateTag(ctx context.Context, tag *models.Tag) error {
	query := `
        UPDATE tag SET
            "user" = $2,
            changed = $3,
            icon = $4,
            budget_income = $5,
            budget_outcome = $6,
            required = $7,
            color = $8,
            picture = $9,
            title = $10,
            show_income = $11,
            show_outcome = $12,
            parent = $13,
            static_id = $14
        WHERE id = $1`

	commandTag, err := s.pool.Exec(ctx, query,
		tag.ID,
		tag.User,
		tag.Changed,
		tag.Icon,
		tag.BudgetIncome,
		tag.BudgetOutcome,
		tag.Required,
		tag.Color,
		tag.Picture,
		tag.Title,
		tag.ShowIncome,
		tag.ShowOutcome,
		tag.Parent,
		tag.StaticID,
	)

	if err != nil {
		return fmt.Errorf("failed to update tag: %w", err)
	}

	if commandTag.RowsAffected() == 0 {
		return fmt.Errorf("tag not found: %s", tag.ID)
	}

	return nil
}

// DeleteTag deletes a tag by its ID
func (s *DB) DeleteTag(ctx context.Context, id string) error {
	query := `DELETE FROM tag WHERE id = $1`

	commandTag, err := s.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete tag: %w", err)
	}

	if commandTag.RowsAffected() == 0 {
		return fmt.Errorf("tag not found: %s", id)
	}

	return nil
}

// GetMerchant retrieves a specific merchant by its ID
func (s *DB) GetMerchant(ctx context.Context, id string) (*models.Merchant, error) {
	query := `
       SELECT id, "user", title, changed
       FROM merchant
       WHERE id = $1`

	merchant := &models.Merchant{}
	err := s.pool.QueryRow(ctx, query, id).Scan(
		&merchant.ID,
		&merchant.User,
		&merchant.Title,
		&merchant.Changed,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("merchant not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get merchant: %w", err)
	}

	return merchant, nil
}

// ListMerchants retrieves a list of merchants based on the provided filter
func (s *DB) ListMerchants(ctx context.Context, filter db.Filter) ([]models.Merchant, error) {
	var conditions []string
	var args []interface{}
	argNum := 1

	if filter.UserID != nil {
		conditions = append(conditions, fmt.Sprintf(`"user" = $%d`, argNum))
		args = append(args, *filter.UserID)
		argNum++
	}

	query := `
       SELECT id, "user", title, changed
       FROM merchant`

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argNum, argNum+1)
	args = append(args, filter.Limit, (filter.Page-1)*filter.Limit)

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list merchants: %w", err)
	}
	defer rows.Close()

	var merchants []models.Merchant
	for rows.Next() {
		var merchant models.Merchant
		err := rows.Scan(
			&merchant.ID,
			&merchant.User,
			&merchant.Title,
			&merchant.Changed,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan merchant: %w", err)
		}
		merchants = append(merchants, merchant)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating merchants: %w", err)
	}

	return merchants, nil
}

// CreateMerchant creates a new merchant record
func (s *DB) CreateMerchant(ctx context.Context, merchant *models.Merchant) error {
	query := `
       INSERT INTO merchant (id, "user", title, changed)
       VALUES ($1, $2, $3, $4)`

	_, err := s.pool.Exec(ctx, query,
		merchant.ID,
		merchant.User,
		merchant.Title,
		merchant.Changed,
	)

	if err != nil {
		return fmt.Errorf("failed to create merchant: %w", err)
	}

	return nil
}

// UpdateMerchant updates an existing merchant record
func (s *DB) UpdateMerchant(ctx context.Context, merchant *models.Merchant) error {
	query := `
       UPDATE merchant 
       SET "user" = $2, title = $3, changed = $4
       WHERE id = $1`

	commandTag, err := s.pool.Exec(ctx, query,
		merchant.ID,
		merchant.User,
		merchant.Title,
		merchant.Changed,
	)

	if err != nil {
		return fmt.Errorf("failed to update merchant: %w", err)
	}

	if commandTag.RowsAffected() == 0 {
		return fmt.Errorf("merchant not found: %s", merchant.ID)
	}

	return nil
}

// DeleteMerchant deletes a merchant by its ID
func (s *DB) DeleteMerchant(ctx context.Context, id string) error {
	query := `DELETE FROM merchant WHERE id = $1`

	commandTag, err := s.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete merchant: %w", err)
	}

	if commandTag.RowsAffected() == 0 {
		return fmt.Errorf("merchant not found: %s", id)
	}

	return nil
}

// GetBudget retrieves a specific budget by user ID, tag ID and date
func (s *DB) GetBudget(ctx context.Context, userID int, tagID string, date time.Time) (*models.Budget, error) {
	query := `
        SELECT "user", changed, date, tag, income, outcome, 
               income_lock, outcome_lock, is_income_forecast, is_outcome_forecast
        FROM budget
        WHERE "user" = $1 AND tag = $2 AND date = $3`

	budget := &models.Budget{}
	err := s.pool.QueryRow(ctx, query, userID, tagID, date.Format("2006-01-02")).Scan(
		&budget.User,
		&budget.Changed,
		&budget.Date,
		&budget.Tag,
		&budget.Income,
		&budget.Outcome,
		&budget.IncomeLock,
		&budget.OutcomeLock,
		&budget.IsIncomeForecast,
		&budget.IsOutcomeForecast,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("budget not found for user %d, tag %s, date %s",
				userID, tagID, date.Format("2006-01-02"))
		}
		return nil, fmt.Errorf("failed to get budget: %w", err)
	}

	return budget, nil
}

// ListBudgets retrieves a list of budgets based on the provided filter
func (s *DB) ListBudgets(ctx context.Context, filter db.Filter) ([]models.Budget, error) {
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
        SELECT "user", changed, date, tag, income, outcome,
               income_lock, outcome_lock, is_income_forecast, is_outcome_forecast
        FROM budget`

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argNum, argNum+1)
	args = append(args, filter.Limit, (filter.Page-1)*filter.Limit)

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list budgets: %w", err)
	}
	defer rows.Close()

	var budgets []models.Budget
	for rows.Next() {
		var budget models.Budget
		err := rows.Scan(
			&budget.User,
			&budget.Changed,
			&budget.Date,
			&budget.Tag,
			&budget.Income,
			&budget.Outcome,
			&budget.IncomeLock,
			&budget.OutcomeLock,
			&budget.IsIncomeForecast,
			&budget.IsOutcomeForecast,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan budget: %w", err)
		}
		budgets = append(budgets, budget)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating budgets: %w", err)
	}

	return budgets, nil
}

// CreateBudget creates a new budget record
func (s *DB) CreateBudget(ctx context.Context, budget *models.Budget) error {
	query := `
        INSERT INTO budget (
            "user", changed, date, tag, income, outcome,
            income_lock, outcome_lock, is_income_forecast, is_outcome_forecast
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`

	_, err := s.pool.Exec(ctx, query,
		budget.User,
		budget.Changed,
		budget.Date,
		budget.Tag,
		budget.Income,
		budget.Outcome,
		budget.IncomeLock,
		budget.OutcomeLock,
		budget.IsIncomeForecast,
		budget.IsOutcomeForecast,
	)

	if err != nil {
		return fmt.Errorf("failed to create budget: %w", err)
	}

	return nil
}

// UpdateBudget updates an existing budget record
func (s *DB) UpdateBudget(ctx context.Context, budget *models.Budget) error {
	query := `
        UPDATE budget 
        SET changed = $4,
            income = $5,
            outcome = $6,
            income_lock = $7,
            outcome_lock = $8,
            is_income_forecast = $9,
            is_outcome_forecast = $10
        WHERE "user" = $1 AND tag = $2 AND date = $3`

	commandTag, err := s.pool.Exec(ctx, query,
		budget.User,
		budget.Tag,
		budget.Date,
		budget.Changed,
		budget.Income,
		budget.Outcome,
		budget.IncomeLock,
		budget.OutcomeLock,
		budget.IsIncomeForecast,
		budget.IsOutcomeForecast,
	)

	if err != nil {
		return fmt.Errorf("failed to update budget: %w", err)
	}

	if commandTag.RowsAffected() == 0 {
		return fmt.Errorf("budget not found for user %d, tag %s, date %s",
			budget.User, *budget.Tag, budget.Date)
	}

	return nil
}

// DeleteBudget deletes a budget by user ID, tag ID and date
func (s *DB) DeleteBudget(ctx context.Context, userID int, tagID string, date time.Time) error {
	query := `DELETE FROM budget WHERE "user" = $1 AND tag = $2 AND date = $3`

	commandTag, err := s.pool.Exec(ctx, query, userID, tagID, date.Format("2006-01-02"))
	if err != nil {
		return fmt.Errorf("failed to delete budget: %w", err)
	}

	if commandTag.RowsAffected() == 0 {
		return fmt.Errorf("budget not found for user %d, tag %s, date %s",
			userID, tagID, date.Format("2006-01-02"))
	}

	return nil
}

// GetReminder retrieves a specific reminder by its ID
func (s *DB) GetReminder(ctx context.Context, id string) (*models.Reminder, error) {
	query := `
        SELECT id, "user", income, outcome, changed, income_instrument,
               outcome_instrument, step, points, tag, start_date, end_date,
               notify, interval, income_account, outcome_account, comment,
               payee, merchant
        FROM reminder
        WHERE id = $1`

	reminder := &models.Reminder{}
	err := s.pool.QueryRow(ctx, query, id).Scan(
		&reminder.ID,
		&reminder.User,
		&reminder.Income,
		&reminder.Outcome,
		&reminder.Changed,
		&reminder.IncomeInstrument,
		&reminder.OutcomeInstrument,
		&reminder.Step,
		&reminder.Points,
		&reminder.Tag,
		&reminder.StartDate,
		&reminder.EndDate,
		&reminder.Notify,
		&reminder.Interval,
		&reminder.IncomeAccount,
		&reminder.OutcomeAccount,
		&reminder.Comment,
		&reminder.Payee,
		&reminder.Merchant,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("reminder not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get reminder: %w", err)
	}

	return reminder, nil
}

// ListReminders retrieves a list of reminders based on the provided filter
func (s *DB) ListReminders(ctx context.Context, filter db.Filter) ([]models.Reminder, error) {
	var conditions []string
	var args []interface{}
	argNum := 1

	if filter.UserID != nil {
		conditions = append(conditions, fmt.Sprintf(`"user" = $%d`, argNum))
		args = append(args, *filter.UserID)
		argNum++
	}

	query := `
        SELECT id, "user", income, outcome, changed, income_instrument,
               outcome_instrument, step, points, tag, start_date, end_date,
               notify, interval, income_account, outcome_account, comment,
               payee, merchant
        FROM reminder`

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argNum, argNum+1)
	args = append(args, filter.Limit, (filter.Page-1)*filter.Limit)

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list reminders: %w", err)
	}
	defer rows.Close()

	var reminders []models.Reminder
	for rows.Next() {
		var reminder models.Reminder
		err := rows.Scan(
			&reminder.ID,
			&reminder.User,
			&reminder.Income,
			&reminder.Outcome,
			&reminder.Changed,
			&reminder.IncomeInstrument,
			&reminder.OutcomeInstrument,
			&reminder.Step,
			&reminder.Points,
			&reminder.Tag,
			&reminder.StartDate,
			&reminder.EndDate,
			&reminder.Notify,
			&reminder.Interval,
			&reminder.IncomeAccount,
			&reminder.OutcomeAccount,
			&reminder.Comment,
			&reminder.Payee,
			&reminder.Merchant,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan reminder: %w", err)
		}
		reminders = append(reminders, reminder)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating reminders: %w", err)
	}

	return reminders, nil
}

// CreateReminder creates a new reminder record
func (s *DB) CreateReminder(ctx context.Context, reminder *models.Reminder) error {
	query := `
        INSERT INTO reminder (
            id, "user", income, outcome, changed, income_instrument,
            outcome_instrument, step, points, tag, start_date, end_date,
            notify, interval, income_account, outcome_account, comment,
            payee, merchant
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, 
                  $14, $15, $16, $17, $18, $19)`

	_, err := s.pool.Exec(ctx, query,
		reminder.ID,
		reminder.User,
		reminder.Income,
		reminder.Outcome,
		reminder.Changed,
		reminder.IncomeInstrument,
		reminder.OutcomeInstrument,
		reminder.Step,
		reminder.Points,
		reminder.Tag,
		reminder.StartDate,
		reminder.EndDate,
		reminder.Notify,
		reminder.Interval,
		reminder.IncomeAccount,
		reminder.OutcomeAccount,
		reminder.Comment,
		reminder.Payee,
		reminder.Merchant,
	)

	if err != nil {
		return fmt.Errorf("failed to create reminder: %w", err)
	}

	return nil
}

// UpdateReminder updates an existing reminder record
func (s *DB) UpdateReminder(ctx context.Context, reminder *models.Reminder) error {
	query := `
        UPDATE reminder SET
            "user" = $2,
            income = $3,
            outcome = $4,
            changed = $5,
            income_instrument = $6,
            outcome_instrument = $7,
            step = $8,
            points = $9,
            tag = $10,
            start_date = $11,
            end_date = $12,
            notify = $13,
            interval = $14,
            income_account = $15,
            outcome_account = $16,
            comment = $17,
            payee = $18,
            merchant = $19
        WHERE id = $1`

	commandTag, err := s.pool.Exec(ctx, query,
		reminder.ID,
		reminder.User,
		reminder.Income,
		reminder.Outcome,
		reminder.Changed,
		reminder.IncomeInstrument,
		reminder.OutcomeInstrument,
		reminder.Step,
		reminder.Points,
		reminder.Tag,
		reminder.StartDate,
		reminder.EndDate,
		reminder.Notify,
		reminder.Interval,
		reminder.IncomeAccount,
		reminder.OutcomeAccount,
		reminder.Comment,
		reminder.Payee,
		reminder.Merchant,
	)

	if err != nil {
		return fmt.Errorf("failed to update reminder: %w", err)
	}

	if commandTag.RowsAffected() == 0 {
		return fmt.Errorf("reminder not found: %s", reminder.ID)
	}

	return nil
}

// DeleteReminder deletes a reminder by its ID
func (s *DB) DeleteReminder(ctx context.Context, id string) error {
	query := `DELETE FROM reminder WHERE id = $1`

	commandTag, err := s.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete reminder: %w", err)
	}

	if commandTag.RowsAffected() == 0 {
		return fmt.Errorf("reminder not found: %s", id)
	}

	return nil
}

// GetReminderMarker retrieves a specific reminder marker by its ID
func (s *DB) GetReminderMarker(ctx context.Context, id string) (*models.ReminderMarker, error) {
	query := `
        SELECT id, "user", date, income, outcome, changed,
               income_instrument, outcome_instrument, state, is_forecast,
               reminder, income_account, outcome_account, comment,
               payee, merchant, notify, tag
        FROM reminder_marker
        WHERE id = $1`

	marker := &models.ReminderMarker{}
	err := s.pool.QueryRow(ctx, query, id).Scan(
		&marker.ID,
		&marker.User,
		&marker.Date,
		&marker.Income,
		&marker.Outcome,
		&marker.Changed,
		&marker.IncomeInstrument,
		&marker.OutcomeInstrument,
		&marker.State,
		&marker.IsForecast,
		&marker.Reminder,
		&marker.IncomeAccount,
		&marker.OutcomeAccount,
		&marker.Comment,
		&marker.Payee,
		&marker.Merchant,
		&marker.Notify,
		&marker.Tag,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("reminder marker not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get reminder marker: %w", err)
	}

	return marker, nil
}

// ListReminderMarkers retrieves a list of reminder markers based on the provided filter
func (s *DB) ListReminderMarkers(ctx context.Context, filter db.Filter) ([]models.ReminderMarker, error) {
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
        SELECT id, "user", date, income, outcome, changed,
               income_instrument, outcome_instrument, state, is_forecast,
               reminder, income_account, outcome_account, comment,
               payee, merchant, notify, tag
        FROM reminder_marker`

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argNum, argNum+1)
	args = append(args, filter.Limit, (filter.Page-1)*filter.Limit)

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list reminder markers: %w", err)
	}
	defer rows.Close()

	var markers []models.ReminderMarker
	for rows.Next() {
		var marker models.ReminderMarker
		err := rows.Scan(
			&marker.ID,
			&marker.User,
			&marker.Date,
			&marker.Income,
			&marker.Outcome,
			&marker.Changed,
			&marker.IncomeInstrument,
			&marker.OutcomeInstrument,
			&marker.State,
			&marker.IsForecast,
			&marker.Reminder,
			&marker.IncomeAccount,
			&marker.OutcomeAccount,
			&marker.Comment,
			&marker.Payee,
			&marker.Merchant,
			&marker.Notify,
			&marker.Tag,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan reminder marker: %w", err)
		}
		markers = append(markers, marker)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating reminder markers: %w", err)
	}

	return markers, nil
}

// CreateReminderMarker creates a new reminder marker record
func (s *DB) CreateReminderMarker(ctx context.Context, marker *models.ReminderMarker) error {
	query := `
        INSERT INTO reminder_marker (
            id, "user", date, income, outcome, changed,
            income_instrument, outcome_instrument, state, is_forecast,
            reminder, income_account, outcome_account, comment,
            payee, merchant, notify, tag
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, 
                  $13, $14, $15, $16, $17, $18)`

	_, err := s.pool.Exec(ctx, query,
		marker.ID,
		marker.User,
		marker.Date,
		marker.Income,
		marker.Outcome,
		marker.Changed,
		marker.IncomeInstrument,
		marker.OutcomeInstrument,
		marker.State,
		marker.IsForecast,
		marker.Reminder,
		marker.IncomeAccount,
		marker.OutcomeAccount,
		marker.Comment,
		marker.Payee,
		marker.Merchant,
		marker.Notify,
		marker.Tag,
	)

	if err != nil {
		return fmt.Errorf("failed to create reminder marker: %w", err)
	}

	return nil
}

// UpdateReminderMarker updates an existing reminder marker record
func (s *DB) UpdateReminderMarker(ctx context.Context, marker *models.ReminderMarker) error {
	query := `
        UPDATE reminder_marker SET
            "user" = $2,
            date = $3,
            income = $4,
            outcome = $5,
            changed = $6,
            income_instrument = $7,
            outcome_instrument = $8,
            state = $9,
            is_forecast = $10,
            reminder = $11,
            income_account = $12,
            outcome_account = $13,
            comment = $14,
            payee = $15,
            merchant = $16,
            notify = $17,
            tag = $18
        WHERE id = $1`

	commandTag, err := s.pool.Exec(ctx, query,
		marker.ID,
		marker.User,
		marker.Date,
		marker.Income,
		marker.Outcome,
		marker.Changed,
		marker.IncomeInstrument,
		marker.OutcomeInstrument,
		marker.State,
		marker.IsForecast,
		marker.Reminder,
		marker.IncomeAccount,
		marker.OutcomeAccount,
		marker.Comment,
		marker.Payee,
		marker.Merchant,
		marker.Notify,
		marker.Tag,
	)

	if err != nil {
		return fmt.Errorf("failed to update reminder marker: %w", err)
	}

	if commandTag.RowsAffected() == 0 {
		return fmt.Errorf("reminder marker not found: %s", marker.ID)
	}

	return nil
}

// DeleteReminderMarker deletes a reminder marker by its ID
func (s *DB) DeleteReminderMarker(ctx context.Context, id string) error {
	query := `DELETE FROM reminder_marker WHERE id = $1`

	commandTag, err := s.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete reminder marker: %w", err)
	}

	if commandTag.RowsAffected() == 0 {
		return fmt.Errorf("reminder marker not found: %s", id)
	}

	return nil
}

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
func (s *DB) ListTransactions(ctx context.Context, filter db.Filter) ([]models.Transaction, error) {
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
