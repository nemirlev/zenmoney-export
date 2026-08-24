package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nemirlev/zenmoney-export/v2/internal/analytics"
)

const (
	maxAnalyticsUsers     = 100
	maxAnalyticsRows      = 500
	aggregateCategoryUUID = "00000000-0000-0000-0000-000000000000"
)

var (
	ErrAnalyticsAccessScope = errors.New("analytics principal has no authorized users")
	ErrAnalyticsUserCatalog = errors.New("analytics user catalog is invalid")
	ErrAnalyticsLimit       = errors.New("analytics result limit is invalid")
	ErrAnalyticsCurrency    = errors.New("analytics users do not have one report currency")
	ErrAnalyticsRate        = errors.New("analytics conversion rate is unavailable")
	ErrAnalyticsRange       = errors.New("analytics date range is invalid")
)

const analyticsUsersSQL = `
SELECT report_user.id::bigint,
       COALESCE(NULLIF(BTRIM(report_user.login), ''), 'User ' || report_user.id::text),
       COALESCE(report_instrument.short_title, '')
FROM "user" report_user
LEFT JOIN instrument report_instrument ON report_instrument.id = report_user.currency
WHERE $1::boolean
   OR report_user.id::bigint = ANY($2::bigint[])
ORDER BY report_user.id
LIMIT 101`

var analyticsSnapshotTxOptions = pgx.TxOptions{
	IsoLevel:   pgx.RepeatableRead,
	AccessMode: pgx.ReadOnly,
}

type analyticsQueryExecutor interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

func withAnalyticsSnapshot[T any](
	ctx context.Context,
	pool PgxIface,
	read func(analyticsQueryExecutor) (T, error),
) (T, error) {
	var zero T
	tx, err := pool.BeginTx(ctx, analyticsSnapshotTxOptions)
	if err != nil {
		return zero, fmt.Errorf("begin analytics snapshot: %w", err)
	}

	result, err := read(tx)
	if err != nil {
		if rollbackErr := tx.Rollback(
			ctx,
		); rollbackErr != nil &&
			!errors.Is(rollbackErr, pgx.ErrTxClosed) {
			return zero, errors.Join(
				err,
				fmt.Errorf("rollback analytics snapshot: %w", rollbackErr),
			)
		}
		return zero, err
	}
	if err := tx.Commit(ctx); err != nil {
		return zero, fmt.Errorf("commit analytics snapshot: %w", err)
	}
	return result, nil
}

const resolveAnalyticsCurrencySQL = `
SELECT COALESCE(MIN(report_instrument.short_title), ''),
       COUNT(DISTINCT report_user.currency),
       COUNT(DISTINCT report_user.id)
FROM "user" report_user
JOIN instrument report_instrument ON report_instrument.id = report_user.currency
WHERE report_user.id::bigint = ANY($1::bigint[])`

// NewPostgresAnalyticsStore opens the dedicated read-only analytics runtime's
// connection pool. Authentication scope remains a per-call concern.
func NewPostgresAnalyticsStore(ctx context.Context, connectionString string) (*DB, error) {
	config, err := pgxpool.ParseConfig(connectionString)
	if err != nil {
		return nil, fmt.Errorf("parse analytics postgres config: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("create analytics postgres pool: %w", err)
	}

	return &DB{pool: pool}, nil
}

func validateAnalyticsQuery(principal analytics.Principal, dateRange analytics.DateRange) error {
	if err := validateAnalyticsPrincipal(principal); err != nil {
		return err
	}
	if dateRange.From.IsZero() || dateRange.To.IsZero() || !dateRange.From.Before(dateRange.To) {
		return ErrAnalyticsRange
	}
	return nil
}

func validateAnalyticsPrincipal(principal analytics.Principal) error {
	if principal.AllUsers || len(principal.UserIDs) == 0 {
		return ErrAnalyticsAccessScope
	}
	if len(principal.UserIDs) > maxAnalyticsUsers {
		return fmt.Errorf(
			"%w: at most %d authorized users",
			ErrAnalyticsAccessScope,
			maxAnalyticsUsers,
		)
	}
	return nil
}

func validateAnalyticsCatalogPrincipal(principal analytics.Principal) error {
	if principal.AllUsers == (len(principal.UserIDs) > 0) {
		return ErrAnalyticsAccessScope
	}
	if len(principal.UserIDs) > maxAnalyticsUsers {
		return fmt.Errorf(
			"%w: at most %d authorized users",
			ErrAnalyticsAccessScope,
			maxAnalyticsUsers,
		)
	}
	for _, userID := range principal.UserIDs {
		if userID <= 0 {
			return ErrAnalyticsAccessScope
		}
	}
	return nil
}

func (s *DB) AnalyticsUsers(
	ctx context.Context,
	principal analytics.Principal,
) ([]analytics.AnalyticsUser, error) {
	if err := validateAnalyticsCatalogPrincipal(principal); err != nil {
		return nil, err
	}
	return withAnalyticsSnapshot(ctx, s.pool, func(executor analyticsQueryExecutor) (
		[]analytics.AnalyticsUser,
		error,
	) {
		rows, err := executor.Query(ctx, analyticsUsersSQL, principal.AllUsers, principal.UserIDs)
		if err != nil {
			return nil, fmt.Errorf("query analytics users: %w", err)
		}
		defer rows.Close()

		users := make([]analytics.AnalyticsUser, 0)
		for rows.Next() {
			var user analytics.AnalyticsUser
			if err := rows.Scan(&user.UserID, &user.Label, &user.Currency); err != nil {
				return nil, fmt.Errorf("scan analytics user: %w", err)
			}
			if user.UserID <= 0 || strings.TrimSpace(user.Currency) == "" {
				return nil, ErrAnalyticsUserCatalog
			}
			user.ID = fmt.Sprintf("user:%d", user.UserID)
			users = append(users, user)
		}
		if err := rows.Err(); err != nil {
			return nil, fmt.Errorf("iterate analytics users: %w", err)
		}
		if len(users) == 0 || len(users) > maxAnalyticsUsers {
			return nil, fmt.Errorf(
				"%w: expected between 1 and %d users",
				ErrAnalyticsUserCatalog,
				maxAnalyticsUsers,
			)
		}
		if !principal.AllUsers && len(users) != uniqueUserCount(principal.UserIDs) {
			return nil, ErrAnalyticsUserCatalog
		}
		return users, nil
	})
}

func validateAnalyticsLimit(limit int) error {
	if limit < 1 || limit > maxAnalyticsRows {
		return fmt.Errorf("%w: must be between 1 and %d", ErrAnalyticsLimit, maxAnalyticsRows)
	}
	return nil
}

func nonNilStrings(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}

func (s *DB) analyticsReportCurrency(
	ctx context.Context,
	executor analyticsQueryExecutor,
	principal analytics.Principal,
) (string, error) {
	var currency string
	var currencyCount int64
	var userCount int64
	err := executor.QueryRow(ctx, resolveAnalyticsCurrencySQL, principal.UserIDs).Scan(
		&currency,
		&currencyCount,
		&userCount,
	)
	if err != nil {
		return "", fmt.Errorf("resolve analytics currency: %w", err)
	}
	if currency == "" || currencyCount != 1 ||
		userCount != int64(uniqueUserCount(principal.UserIDs)) {
		return "", ErrAnalyticsCurrency
	}
	return currency, nil
}

func uniqueUserCount(userIDs []int64) int {
	unique := make(map[int64]struct{}, len(userIDs))
	for _, userID := range userIDs {
		unique[userID] = struct{}{}
	}
	return len(unique)
}

func analyticsQueryArgs(
	principal analytics.Principal,
	dateRange analytics.DateRange,
	filters analytics.AppliedFilters,
) []any {
	return []any{
		principal.UserIDs,
		dateRange.From,
		dateRange.To,
		filters.IncludeHold,
		nonNilStrings(filters.AccountIDs),
		nonNilStrings(filters.CategoryIDs),
		nonNilStrings(filters.MerchantIDs),
	}
}

func (s *DB) lastCompletedSyncAt(
	ctx context.Context,
	executor analyticsQueryExecutor,
) (*time.Time, error) {
	var snapshot analytics.SyncSnapshot
	var finishedAt time.Time
	err := executor.QueryRow(ctx, latestCompletedSyncSQL).Scan(
		&snapshot.StartedAt,
		&finishedAt,
		&snapshot.Status,
		&snapshot.SyncType,
		&snapshot.ServerTimestamp,
		&snapshot.RecordsProcessed,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read latest completed sync: %w", err)
	}
	return &finishedAt, nil
}

func (s *DB) SpendingSummary(
	ctx context.Context,
	principal analytics.Principal,
	query analytics.SpendingSummaryQuery,
) (analytics.SpendingSummaryData, error) {
	if err := validateAnalyticsQuery(principal, query.Range); err != nil {
		return analytics.SpendingSummaryData{}, err
	}
	if err := validateAnalyticsLimit(query.Limit); err != nil {
		return analytics.SpendingSummaryData{}, err
	}
	return withAnalyticsSnapshot(ctx, s.pool, func(executor analyticsQueryExecutor) (
		analytics.SpendingSummaryData,
		error,
	) {
		return s.spendingSummarySnapshot(ctx, executor, principal, query)
	})
}

func (s *DB) spendingSummarySnapshot(
	ctx context.Context,
	executor analyticsQueryExecutor,
	principal analytics.Principal,
	query analytics.SpendingSummaryQuery,
) (analytics.SpendingSummaryData, error) {
	currency, err := s.analyticsReportCurrency(ctx, executor, principal)
	if err != nil {
		return analytics.SpendingSummaryData{}, err
	}
	args := analyticsQueryArgs(principal, query.Range, query.Filters)

	var data analytics.SpendingSummaryData
	data.Currency = currency
	var invalidRates int64
	if err := executor.QueryRow(ctx, spendingTotalsSQL, args...).Scan(
		&data.Total,
		&data.TransactionCount,
		&invalidRates,
	); err != nil {
		return analytics.SpendingSummaryData{}, fmt.Errorf("query spending totals: %w", err)
	}
	if invalidRates != 0 {
		return analytics.SpendingSummaryData{}, fmt.Errorf(
			"%w: %d spending legs",
			ErrAnalyticsRate,
			invalidRates,
		)
	}

	categoryArgs := append(append([]any{}, args...), query.Limit+1)
	rows, err := executor.Query(ctx, spendingCategoriesSQL, categoryArgs...)
	if err != nil {
		return analytics.SpendingSummaryData{}, fmt.Errorf("query spending categories: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var categoryID sql.NullString
		var title sql.NullString
		var category analytics.SpendingCategory
		if err := rows.Scan(
			&categoryID,
			&title,
			&category.Amount,
			&category.SharePercent,
			&category.TransactionCount,
		); err != nil {
			return analytics.SpendingSummaryData{}, fmt.Errorf("scan spending category: %w", err)
		}
		if !categoryID.Valid {
			category.CategoryID = "category:uncategorized"
			category.ID = "spending:category:uncategorized"
			category.Title = "Uncategorized"
		} else {
			category.CategoryID = categoryID.String
			category.ID = "spending:category:" + categoryID.String
			if !title.Valid || strings.TrimSpace(title.String) == "" {
				category.Title = "Unnamed category"
			} else {
				category.Title = title.String
			}
		}
		data.Categories = append(data.Categories, category)
	}
	if err := rows.Err(); err != nil {
		return analytics.SpendingSummaryData{}, fmt.Errorf("iterate spending categories: %w", err)
	}
	if len(data.Categories) > query.Limit {
		data.HasMore = true
		data.Categories = data.Categories[:query.Limit]
	}

	data.LastSyncAt, err = s.lastCompletedSyncAt(ctx, executor)
	if err != nil {
		return analytics.SpendingSummaryData{}, err
	}
	return data, nil
}

func cashflowBucketSQL(granularity analytics.Granularity) (string, string, error) {
	switch granularity {
	case analytics.GranularityDay:
		return "date", "date + 1", nil
	case analytics.GranularityWeek:
		return "date_trunc('week', date)::date", "date_trunc('week', date)::date + 7", nil
	case analytics.GranularityMonth:
		return "date_trunc('month', date)::date", "(date_trunc('month', date) + interval '1 month')::date", nil
	default:
		return "", "", fmt.Errorf("unsupported cashflow granularity %q", granularity)
	}
}

func cashflowPointsSQL(granularity analytics.Granularity) (string, error) {
	bucket, bucketEnd, err := cashflowBucketSQL(granularity)
	if err != nil {
		return "", err
	}
	return `WITH ` + analyticsTransactionScopeSQL + `,
bucketed AS (
    SELECT ` + bucket + ` AS bucket_from,
           ` + bucketEnd + ` AS bucket_to,
           COALESCE(SUM(report_amount) FILTER (
               WHERE direction = 'income' AND NOT invalid_rate
           ), 0) AS income,
           COALESCE(SUM(report_amount) FILTER (
               WHERE direction = 'outcome' AND NOT invalid_rate
           ), 0) AS outcome,
           COUNT(*) FILTER (WHERE invalid_rate) AS invalid_rate_count
    FROM converted_legs
    GROUP BY bucket_from, bucket_to
)
SELECT to_char(bucket_from, 'YYYY-MM-DD'),
       to_char(bucket_to, 'YYYY-MM-DD'),
       income::text,
       outcome::text,
       (income - outcome)::text,
       invalid_rate_count
FROM bucketed
ORDER BY bucket_from
LIMIT $8::integer`, nil
}

func (s *DB) Cashflow(
	ctx context.Context,
	principal analytics.Principal,
	query analytics.CashflowQuery,
) (analytics.CashflowData, error) {
	if err := validateAnalyticsQuery(principal, query.Range); err != nil {
		return analytics.CashflowData{}, err
	}
	if err := validateAnalyticsLimit(query.MaxPoints); err != nil {
		return analytics.CashflowData{}, err
	}
	return withAnalyticsSnapshot(ctx, s.pool, func(executor analyticsQueryExecutor) (
		analytics.CashflowData,
		error,
	) {
		return s.cashflowSnapshot(ctx, executor, principal, query)
	})
}

func (s *DB) cashflowSnapshot(
	ctx context.Context,
	executor analyticsQueryExecutor,
	principal analytics.Principal,
	query analytics.CashflowQuery,
) (analytics.CashflowData, error) {
	pointsSQL, err := cashflowPointsSQL(query.Granularity)
	if err != nil {
		return analytics.CashflowData{}, err
	}
	currency, err := s.analyticsReportCurrency(ctx, executor, principal)
	if err != nil {
		return analytics.CashflowData{}, err
	}
	args := analyticsQueryArgs(principal, query.Range, query.Filters)

	var data analytics.CashflowData
	data.Currency = currency
	var invalidRates int64
	if err := executor.QueryRow(ctx, cashflowTotalsSQL, args...).Scan(
		&data.Totals.Income,
		&data.Totals.Outcome,
		&data.Totals.Net,
		&invalidRates,
	); err != nil {
		return analytics.CashflowData{}, fmt.Errorf("query cashflow totals: %w", err)
	}
	if invalidRates != 0 {
		return analytics.CashflowData{}, fmt.Errorf(
			"%w: %d cashflow legs",
			ErrAnalyticsRate,
			invalidRates,
		)
	}

	pointArgs := append(append([]any{}, args...), query.MaxPoints+1)
	rows, err := executor.Query(ctx, pointsSQL, pointArgs...)
	if err != nil {
		return analytics.CashflowData{}, fmt.Errorf("query cashflow points: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var point analytics.CashflowPoint
		var pointInvalidRates int64
		if err := rows.Scan(
			&point.From,
			&point.To,
			&point.Income,
			&point.Outcome,
			&point.Net,
			&pointInvalidRates,
		); err != nil {
			return analytics.CashflowData{}, fmt.Errorf("scan cashflow point: %w", err)
		}
		if pointInvalidRates != 0 {
			return analytics.CashflowData{}, ErrAnalyticsRate
		}
		point.ID = "cashflow:" + string(query.Granularity) + ":" + point.From
		point.Label = point.From
		data.Points = append(data.Points, point)
	}
	if err := rows.Err(); err != nil {
		return analytics.CashflowData{}, fmt.Errorf("iterate cashflow points: %w", err)
	}
	if len(data.Points) > query.MaxPoints {
		return analytics.CashflowData{}, fmt.Errorf(
			"%w: cashflow has more than %d points",
			ErrAnalyticsLimit,
			query.MaxPoints,
		)
	}

	data.LastSyncAt, err = s.lastCompletedSyncAt(ctx, executor)
	if err != nil {
		return analytics.CashflowData{}, err
	}
	return data, nil
}

func (s *DB) SearchTransactions(
	ctx context.Context,
	principal analytics.Principal,
	query analytics.TransactionSearchQuery,
) (analytics.TransactionPage, error) {
	if err := validateAnalyticsQuery(principal, query.Range); err != nil {
		return analytics.TransactionPage{}, err
	}
	if err := validateAnalyticsLimit(query.Limit); err != nil {
		return analytics.TransactionPage{}, err
	}
	return withAnalyticsSnapshot(ctx, s.pool, func(executor analyticsQueryExecutor) (
		analytics.TransactionPage,
		error,
	) {
		return s.searchTransactionsSnapshot(ctx, executor, principal, query)
	})
}

func (s *DB) searchTransactionsSnapshot(
	ctx context.Context,
	executor analyticsQueryExecutor,
	principal analytics.Principal,
	query analytics.TransactionSearchQuery,
) (analytics.TransactionPage, error) {
	currency, err := s.analyticsReportCurrency(ctx, executor, principal)
	if err != nil {
		return analytics.TransactionPage{}, err
	}
	args := analyticsQueryArgs(principal, query.Range, query.Filters)
	args = append(args, strings.TrimSpace(query.Text))
	if query.Cursor == nil {
		args = append(args, nil, int64(0), nil)
	} else {
		args = append(args, query.Cursor.Date, query.Cursor.Created, query.Cursor.ID)
	}
	args = append(args, query.Limit+1)

	rows, err := executor.Query(ctx, searchTransactionsSQL, args...)
	if err != nil {
		return analytics.TransactionPage{}, fmt.Errorf("search transactions: %w", err)
	}
	defer rows.Close()

	page := analytics.TransactionPage{Currency: currency}
	var lastCursor *analytics.TransactionCursor
	hasMore := false
	for rows.Next() {
		var sourceID string
		var created int64
		var direction string
		var accountTitle sql.NullString
		var incomeAccountID string
		var outcomeAccountID sql.NullString
		var merchantID sql.NullString
		var merchantTitle sql.NullString
		var invalidRate bool
		var item analytics.TransactionItem
		if err := rows.Scan(
			&sourceID,
			&item.Date,
			&created,
			&direction,
			&item.Amount,
			&item.Income,
			&item.Outcome,
			&incomeAccountID,
			&outcomeAccountID,
			&accountTitle,
			&item.CategoryIDs,
			&item.CategoryTitles,
			&merchantID,
			&merchantTitle,
			&item.Hold,
			&invalidRate,
		); err != nil {
			return analytics.TransactionPage{}, fmt.Errorf("scan transaction search row: %w", err)
		}
		if invalidRate {
			return analytics.TransactionPage{}, fmt.Errorf(
				"%w: transaction %s",
				ErrAnalyticsRate,
				sourceID,
			)
		}
		date, err := time.Parse("2006-01-02", item.Date)
		if err != nil {
			return analytics.TransactionPage{}, fmt.Errorf("parse transaction date: %w", err)
		}
		if len(page.Items) == query.Limit {
			hasMore = true
			continue
		}

		item.ID = sourceID
		item.Direction = analytics.TransactionDirection(direction)
		item.Currency = currency
		item.AccountID = incomeAccountID
		item.IncomeAccountID = new(incomeAccountID)
		if outcomeAccountID.Valid {
			item.OutcomeAccountID = new(outcomeAccountID.String)
		}
		if !accountTitle.Valid || strings.TrimSpace(accountTitle.String) == "" {
			item.AccountTitle = "Unnamed account"
		} else {
			item.AccountTitle = accountTitle.String
		}
		if !merchantID.Valid {
			noneID := "merchant:none"
			item.MerchantID = &noneID
			item.MerchantTitle = "No merchant"
		} else {
			item.MerchantID = new(merchantID.String)
			if !merchantTitle.Valid || strings.TrimSpace(merchantTitle.String) == "" {
				item.MerchantTitle = "Unnamed merchant"
			} else {
				item.MerchantTitle = merchantTitle.String
			}
		}
		page.Items = append(page.Items, item)
		lastCursor = &analytics.TransactionCursor{Date: date, Created: created, ID: sourceID}
	}
	if err := rows.Err(); err != nil {
		return analytics.TransactionPage{}, fmt.Errorf("iterate transaction search rows: %w", err)
	}
	if hasMore {
		page.NextCursor = lastCursor
	}
	page.LastSyncAt, err = s.lastCompletedSyncAt(ctx, executor)
	if err != nil {
		return analytics.TransactionPage{}, err
	}
	return page, nil
}

func decimalPointer(value sql.NullString) *analytics.Decimal {
	if !value.Valid {
		return nil
	}
	decimal := analytics.Decimal(value.String)
	return &decimal
}

func (s *DB) BudgetProgress(
	ctx context.Context,
	principal analytics.Principal,
	query analytics.BudgetProgressQuery,
) (analytics.BudgetProgressData, error) {
	if err := validateAnalyticsQuery(principal, query.Range); err != nil {
		return analytics.BudgetProgressData{}, err
	}
	if err := validateAnalyticsLimit(query.Limit); err != nil {
		return analytics.BudgetProgressData{}, err
	}
	return withAnalyticsSnapshot(ctx, s.pool, func(executor analyticsQueryExecutor) (
		analytics.BudgetProgressData,
		error,
	) {
		return s.budgetProgressSnapshot(ctx, executor, principal, query)
	})
}

func (s *DB) budgetProgressSnapshot(
	ctx context.Context,
	executor analyticsQueryExecutor,
	principal analytics.Principal,
	query analytics.BudgetProgressQuery,
) (analytics.BudgetProgressData, error) {
	currency, err := s.analyticsReportCurrency(ctx, executor, principal)
	if err != nil {
		return analytics.BudgetProgressData{}, err
	}
	args := analyticsQueryArgs(principal, query.Range, query.Filters)
	data := analytics.BudgetProgressData{Currency: currency}
	var totalPercent sql.NullString
	var invalidRates int64
	if err := executor.QueryRow(ctx, budgetProgressTotalsSQL, args...).Scan(
		&data.Totals.Budget,
		&data.Totals.Spent,
		&data.Totals.Remaining,
		&totalPercent,
		&invalidRates,
	); err != nil {
		return analytics.BudgetProgressData{}, fmt.Errorf("query budget progress totals: %w", err)
	}
	if invalidRates != 0 {
		return analytics.BudgetProgressData{}, fmt.Errorf(
			"%w: %d budget progress values",
			ErrAnalyticsRate,
			invalidRates,
		)
	}
	data.Totals.Percent = decimalPointer(totalPercent)

	rowArgs := append(append([]any{}, args...), query.Limit+1)
	rows, err := executor.Query(ctx, budgetProgressRowsSQL, rowArgs...)
	if err != nil {
		return analytics.BudgetProgressData{}, fmt.Errorf("query budget progress rows: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var categoryID sql.NullString
		var title sql.NullString
		var percent sql.NullString
		var row analytics.BudgetProgressRow
		if err := rows.Scan(
			&categoryID,
			&title,
			&row.Budget,
			&row.Spent,
			&row.Remaining,
			&percent,
			&row.TransactionCount,
		); err != nil {
			return analytics.BudgetProgressData{}, fmt.Errorf("scan budget progress row: %w", err)
		}
		if !categoryID.Valid {
			row.CategoryID = "category:uncategorized"
			row.ID = "budget:category:uncategorized"
			row.Title = "Uncategorized"
		} else if categoryID.String == aggregateCategoryUUID {
			row.CategoryID = "category:all"
			row.ID = "budget:category:all"
			row.Title = "All categories"
		} else {
			row.CategoryID = categoryID.String
			row.ID = "budget:category:" + categoryID.String
			if !title.Valid || strings.TrimSpace(title.String) == "" {
				row.Title = "Unnamed category"
			} else {
				row.Title = title.String
			}
		}
		row.Percent = decimalPointer(percent)
		data.Rows = append(data.Rows, row)
	}
	if err := rows.Err(); err != nil {
		return analytics.BudgetProgressData{}, fmt.Errorf("iterate budget progress rows: %w", err)
	}
	if len(data.Rows) > query.Limit {
		data.HasMore = true
		data.Rows = data.Rows[:query.Limit]
	}
	data.LastSyncAt, err = s.lastCompletedSyncAt(ctx, executor)
	if err != nil {
		return analytics.BudgetProgressData{}, err
	}
	return data, nil
}

func (s *DB) readSyncSnapshot(
	ctx context.Context,
	executor analyticsQueryExecutor,
	querySQL string,
) (*analytics.SyncSnapshot, error) {
	var snapshot analytics.SyncSnapshot
	var finishedAt sql.NullTime
	err := executor.QueryRow(ctx, querySQL).Scan(
		&snapshot.StartedAt,
		&finishedAt,
		&snapshot.Status,
		&snapshot.SyncType,
		&snapshot.ServerTimestamp,
		&snapshot.RecordsProcessed,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if finishedAt.Valid {
		snapshot.FinishedAt = &finishedAt.Time
	}
	return &snapshot, nil
}

func (s *DB) DataFreshness(
	ctx context.Context,
	principal analytics.Principal,
) (analytics.FreshnessData, error) {
	if err := validateAnalyticsCatalogPrincipal(principal); err != nil {
		return analytics.FreshnessData{}, err
	}
	return withAnalyticsSnapshot(ctx, s.pool, func(executor analyticsQueryExecutor) (
		analytics.FreshnessData,
		error,
	) {
		return s.dataFreshnessSnapshot(ctx, executor)
	})
}

func (s *DB) dataFreshnessSnapshot(
	ctx context.Context,
	executor analyticsQueryExecutor,
) (analytics.FreshnessData, error) {
	completed, err := s.readSyncSnapshot(ctx, executor, latestCompletedSyncSQL)
	if err != nil {
		return analytics.FreshnessData{}, fmt.Errorf("read completed sync freshness: %w", err)
	}
	attempt, err := s.readSyncSnapshot(ctx, executor, latestSyncAttemptSQL)
	if err != nil {
		return analytics.FreshnessData{}, fmt.Errorf("read latest sync freshness: %w", err)
	}
	return analytics.FreshnessData{LastCompleted: completed, LastAttempt: attempt}, nil
}

var _ analytics.AnalyticsStore = (*DB)(nil)
