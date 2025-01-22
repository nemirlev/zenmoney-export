package postgres

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/nemirlev/zenmoney-go-sdk/v2/models"
)

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
