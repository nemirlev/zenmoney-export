package postgres

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/nemirlev/zenmoney-export/v2/internal/analytics"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/require"
)

func expectAnalyticsSnapshotBegin(mock pgxmock.PgxPoolIface) {
	mock.ExpectBeginTx(analyticsSnapshotTxOptions)
}

func analyticsTestRange() analytics.DateRange {
	return analytics.DateRange{
		From: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC),
	}
}

func TestAnalyticsSQLScopesAndClassifiesConservatively(t *testing.T) {
	t.Parallel()

	require.Contains(t, analyticsTransactionScopeSQL, `t."user"::bigint = ANY($1::bigint[])`)
	require.Contains(t, analyticsTransactionScopeSQL, `income_account."user" = t."user"`)
	require.Contains(t, analyticsTransactionScopeSQL, `t.deleted IS FALSE`)
	require.Contains(
		t,
		analyticsTransactionScopeSQL,
		`t.hold IS FALSE OR ($4::boolean AND t.hold IS TRUE)`,
	)
	require.Contains(t, analyticsTransactionScopeSQL, `income_account.in_balance IS TRUE`)
	require.Contains(t, analyticsTransactionScopeSQL, `t.date >= $2::date`)
	require.Contains(t, analyticsTransactionScopeSQL, `t.date < $3::date`)
	require.Contains(
		t,
		analyticsTransactionScopeSQL,
		`income_account IS NOT DISTINCT FROM outcome_account`,
	)
	require.Contains(t, analyticsTransactionScopeSQL, `account_type IS DISTINCT FROM 'debt'`)
	require.Contains(t, analyticsTransactionScopeSQL, `leg.amount * source.rate / target.rate`)
	require.Contains(t, analyticsTransactionScopeSQL, `report_user.currency`)
	require.Contains(t, analyticsTransactionScopeSQL, `selected_outcome_account."user" = t."user"`)
	require.Contains(t, analyticsTransactionScopeSQL, `selected_merchant."user" = t."user"`)
	require.Contains(t, searchTransactionsSQL, `outcome_account."user" = t."user"`)
	require.Contains(t, searchTransactionsSQL, `merchant."user" = t."user"`)
	require.Contains(t, searchTransactionsSQL, `merchant.id AS merchant`)
}

func TestSpendingCategorySQLDeduplicatesRollsUpAndEqualSplits(t *testing.T) {
	t.Parallel()

	require.Contains(t, spendingCategoriesSQL, `SELECT DISTINCT outcome.transaction_id`)
	require.Contains(t, spendingCategoriesSQL, `COALESCE(parent.id, category.id) AS category_id`)
	require.Contains(t, spendingCategoriesSQL, `outcome.report_amount / counts.tag_count`)
	require.Contains(t, spendingCategoriesSQL, `NOT EXISTS`)
	require.Contains(t, spendingCategoriesSQL, `NULL::uuid AS category_id`)
}

func TestBudgetSQLUsesStrictNormalizedMonthRange(t *testing.T) {
	t.Parallel()

	require.Contains(t, budgetProgressCTESQL, `budget.date >= $2::date`)
	require.Contains(t, budgetProgressCTESQL, `budget.date < $3::date`)
	require.NotContains(t, budgetProgressCTESQL, `date_trunc('month', $2::date)`)
}

func TestAnalyticsUsersDiscoversAllDatabaseUsers(t *testing.T) {
	db, mock := newTestDB(t)
	principal := analytics.Principal{Subject: "local", AllUsers: true}

	expectAnalyticsSnapshotBegin(mock)
	mock.ExpectQuery(regexp.QuoteMeta(analyticsUsersSQL)).
		WithArgs(true, []int64(nil)).
		WillReturnRows(mock.NewRows([]string{"id", "label", "currency"}).
			AddRow(int64(7), "Family", "RUB").
			AddRow(int64(42), "Personal", "RUB"))
	mock.ExpectCommit()

	users, err := db.AnalyticsUsers(context.Background(), principal)
	require.NoError(t, err)
	require.Equal(t, []analytics.AnalyticsUser{
		{UserID: 7, ID: "user:7", Label: "Family", Currency: "RUB"},
		{UserID: 42, ID: "user:42", Label: "Personal", Currency: "RUB"},
	}, users)
}

func TestAnalyticsUsersRejectsMissingAllowlistUser(t *testing.T) {
	db, mock := newTestDB(t)
	principal := analytics.Principal{Subject: "restricted", UserIDs: []int64{7, 42}}

	expectAnalyticsSnapshotBegin(mock)
	mock.ExpectQuery(regexp.QuoteMeta(analyticsUsersSQL)).
		WithArgs(false, principal.UserIDs).
		WillReturnRows(mock.NewRows([]string{"id", "label", "currency"}).
			AddRow(int64(7), "Family", "RUB"))
	mock.ExpectRollback()

	_, err := db.AnalyticsUsers(context.Background(), principal)
	require.ErrorIs(t, err, ErrAnalyticsUserCatalog)
}

func TestSpendingSummaryUsesAuthenticatedScopeAndStableUncategorizedID(t *testing.T) {
	db, mock := newTestDB(t)
	ctx := context.Background()
	principal := analytics.Principal{Subject: "subject", UserIDs: []int64{42}}
	query := analytics.SpendingSummaryQuery{Range: analyticsTestRange(), Limit: 1}

	expectAnalyticsSnapshotBegin(mock)
	mock.ExpectQuery(regexp.QuoteMeta(resolveAnalyticsCurrencySQL)).
		WithArgs(principal.UserIDs).
		WillReturnRows(mock.NewRows([]string{"currency", "currency_count", "user_count"}).
			AddRow("RUB", int64(1), int64(1)))
	baseArgs := []any{
		principal.UserIDs, query.Range.From, query.Range.To, false,
		[]string{},
		[]string{},
		[]string{},
	}
	mock.ExpectQuery(regexp.QuoteMeta(spendingTotalsSQL)).
		WithArgs(baseArgs...).
		WillReturnRows(mock.NewRows([]string{"total", "transaction_count", "invalid_rate_count"}).
			AddRow("1250.50", int64(2), int64(0)))
	mock.ExpectQuery(regexp.QuoteMeta(spendingCategoriesSQL)).
		WithArgs(append(baseArgs, 2)...).
		WillReturnRows(mock.NewRows([]string{
			"category_id", "category_title", "amount", "share", "transaction_count",
		}).AddRow(nil, nil, "1250.50", "100", int64(2)).
			AddRow("00000000-0000-0000-0000-000000000020", "Food", "10", "1", int64(1)))
	finished := time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC)
	mock.ExpectQuery(regexp.QuoteMeta(latestCompletedSyncSQL)).
		WillReturnRows(mock.NewRows([]string{
			"started_at",
			"finished_at",
			"status",
			"sync_type",
			"server_timestamp",
			"records_processed",
		}).AddRow(finished.Add(-time.Minute), finished, "completed", "partial", int64(99), int64(3)))
	mock.ExpectCommit()

	data, err := db.SpendingSummary(ctx, principal, query)
	require.NoError(t, err)
	require.Equal(t, "RUB", data.Currency)
	require.Equal(t, analytics.Decimal("1250.50"), data.Total)
	require.Equal(t, int64(2), data.TransactionCount)
	require.Len(t, data.Categories, 1)
	require.True(t, data.HasMore)
	require.Equal(t, "category:uncategorized", data.Categories[0].CategoryID)
	require.Equal(t, "spending:category:uncategorized", data.Categories[0].ID)
	require.Equal(t, "Uncategorized", data.Categories[0].Title)
}

func TestSpendingSummaryRejectsMissingRate(t *testing.T) {
	db, mock := newTestDB(t)
	principal := analytics.Principal{UserIDs: []int64{42}}
	query := analytics.SpendingSummaryQuery{Range: analyticsTestRange(), Limit: 10}

	expectAnalyticsSnapshotBegin(mock)
	mock.ExpectQuery(regexp.QuoteMeta(resolveAnalyticsCurrencySQL)).
		WithArgs(principal.UserIDs).
		WillReturnRows(mock.NewRows([]string{"currency", "currency_count", "user_count"}).
			AddRow("RUB", int64(1), int64(1)))
	mock.ExpectQuery(regexp.QuoteMeta(spendingTotalsSQL)).
		WithArgs(
			principal.UserIDs, query.Range.From, query.Range.To, false,
			[]string{}, []string{}, []string{},
		).
		WillReturnRows(mock.NewRows([]string{"total", "transaction_count", "invalid_rate_count"}).
			AddRow("0", int64(0), int64(1)))
	mock.ExpectRollback()

	_, err := db.SpendingSummary(context.Background(), principal, query)
	require.ErrorIs(t, err, ErrAnalyticsRate)
}

func TestCashflowBucketSQLOnlyAcceptsKnownGranularities(t *testing.T) {
	t.Parallel()

	day, err := cashflowPointsSQL(analytics.GranularityDay)
	require.NoError(t, err)
	require.Contains(t, day, `date + 1 AS bucket_to`)
	week, err := cashflowPointsSQL(analytics.GranularityWeek)
	require.NoError(t, err)
	require.Contains(t, week, `date_trunc('week', date)::date + 7`)
	month, err := cashflowPointsSQL(analytics.GranularityMonth)
	require.NoError(t, err)
	require.Contains(t, month, `interval '1 month'`)
	_, err = cashflowPointsSQL(analytics.Granularity("quarter; DROP TABLE transaction"))
	require.Error(t, err)
}

func TestCashflowReturnsBoundedServerCurrencyPoints(t *testing.T) {
	db, mock := newTestDB(t)
	principal := analytics.Principal{UserIDs: []int64{42}}
	query := analytics.CashflowQuery{
		Range:       analyticsTestRange(),
		Granularity: analytics.GranularityMonth,
		MaxPoints:   12,
	}
	pointsSQL, err := cashflowPointsSQL(query.Granularity)
	require.NoError(t, err)

	expectAnalyticsSnapshotBegin(mock)
	mock.ExpectQuery(regexp.QuoteMeta(resolveAnalyticsCurrencySQL)).
		WithArgs(principal.UserIDs).
		WillReturnRows(mock.NewRows([]string{"currency", "currency_count", "user_count"}).
			AddRow("RUB", int64(1), int64(1)))
	baseArgs := []any{
		principal.UserIDs, query.Range.From, query.Range.To, false,
		[]string{},
		[]string{},
		[]string{},
	}
	mock.ExpectQuery(regexp.QuoteMeta(cashflowTotalsSQL)).
		WithArgs(baseArgs...).
		WillReturnRows(mock.NewRows([]string{"income", "outcome", "net", "invalid"}).
			AddRow("2000", "750", "1250", int64(0)))
	mock.ExpectQuery(regexp.QuoteMeta(pointsSQL)).
		WithArgs(append(baseArgs, 13)...).
		WillReturnRows(mock.NewRows([]string{
			"from", "to", "income", "outcome", "net", "invalid",
		}).AddRow("2026-08-01", "2026-09-01", "2000", "750", "1250", int64(0)))
	mock.ExpectQuery(regexp.QuoteMeta(latestCompletedSyncSQL)).WillReturnError(pgx.ErrNoRows)
	mock.ExpectCommit()

	data, err := db.Cashflow(context.Background(), principal, query)
	require.NoError(t, err)
	require.Equal(t, "RUB", data.Currency)
	require.Equal(t, analytics.Decimal("1250"), data.Totals.Net)
	require.Len(t, data.Points, 1)
	require.Equal(t, "cashflow:month:2026-08-01", data.Points[0].ID)
}

func TestSearchTransactionsUsesKeysetAndStableFallbacks(t *testing.T) {
	db, mock := newTestDB(t)
	principal := analytics.Principal{UserIDs: []int64{42}}
	query := analytics.TransactionSearchQuery{
		Range: analyticsTestRange(),
		Text:  "market",
		Limit: 2,
	}

	expectAnalyticsSnapshotBegin(mock)
	mock.ExpectQuery(regexp.QuoteMeta(resolveAnalyticsCurrencySQL)).
		WithArgs(principal.UserIDs).
		WillReturnRows(mock.NewRows([]string{"currency", "currency_count", "user_count"}).
			AddRow("RUB", int64(1), int64(1)))
	baseArgs := []any{
		principal.UserIDs, query.Range.From, query.Range.To, false,
		[]string{},
		[]string{},
		[]string{},
		"market", nil, int64(0), nil, 3,
	}
	mock.ExpectQuery(regexp.QuoteMeta(searchTransactionsSQL)).
		WithArgs(baseArgs...).
		WillReturnRows(mock.NewRows([]string{
			"id", "date", "created", "direction", "amount", "income", "outcome",
			"income_account", "outcome_account", "account_title", "category_ids",
			"category_titles", "merchant_id", "merchant_title", "hold", "invalid_rate",
		}).AddRow(
			"00000000-0000-0000-0000-000000000001", "2026-08-20", int64(100),
			"outcome", "-25", "0", "25",
			"00000000-0000-0000-0000-000000000010", nil, "Main",
			[]string{"category:uncategorized"}, []string{"Uncategorized"}, nil, nil, false, false,
		))
	mock.ExpectQuery(regexp.QuoteMeta(latestCompletedSyncSQL)).WillReturnError(pgx.ErrNoRows)
	mock.ExpectCommit()

	page, err := db.SearchTransactions(context.Background(), principal, query)
	require.NoError(t, err)
	require.Equal(t, "RUB", page.Currency)
	require.Len(t, page.Items, 1)
	require.Equal(t, analytics.DirectionOutcome, page.Items[0].Direction)
	require.Equal(t, "merchant:none", *page.Items[0].MerchantID)
	require.Equal(t, "No merchant", page.Items[0].MerchantTitle)
	require.Nil(t, page.NextCursor)
	require.Contains(t, searchTransactionsSQL, `(t.date, t.created, t.id) <`)
	require.Contains(
		t,
		searchTransactionsSQL,
		`ORDER BY searched.date DESC, searched.created DESC, searched.id DESC`,
	)
}

func TestBudgetProgressReturnsExactDecimalRows(t *testing.T) {
	db, mock := newTestDB(t)
	principal := analytics.Principal{UserIDs: []int64{42}}
	query := analytics.BudgetProgressQuery{Range: analyticsTestRange(), Limit: 1}
	categoryID := "00000000-0000-0000-0000-000000000020"

	expectAnalyticsSnapshotBegin(mock)
	mock.ExpectQuery(regexp.QuoteMeta(resolveAnalyticsCurrencySQL)).
		WithArgs(principal.UserIDs).
		WillReturnRows(mock.NewRows([]string{"currency", "currency_count", "user_count"}).
			AddRow("RUB", int64(1), int64(1)))
	baseArgs := []any{
		principal.UserIDs, query.Range.From, query.Range.To, false,
		[]string{},
		[]string{},
		[]string{},
	}
	mock.ExpectQuery(regexp.QuoteMeta(budgetProgressTotalsSQL)).
		WithArgs(baseArgs...).
		WillReturnRows(mock.NewRows([]string{"budget", "spent", "remaining", "percent", "invalid"}).
			AddRow("1000.00", "250.00", "750.00", "25", int64(0)))
	mock.ExpectQuery(regexp.QuoteMeta(budgetProgressRowsSQL)).
		WithArgs(append(baseArgs, 2)...).
		WillReturnRows(mock.NewRows([]string{
			"category_id", "title", "budget", "spent", "remaining", "percent", "transaction_count",
		}).AddRow(categoryID, "Food", "1000.00", "250.00", "750.00", "25", int64(2)).
			AddRow("00000000-0000-0000-0000-000000000021", "Transport", "500", "50", "450", "10", int64(1)),
		)
	mock.ExpectQuery(regexp.QuoteMeta(latestCompletedSyncSQL)).WillReturnError(pgx.ErrNoRows)
	mock.ExpectCommit()

	data, err := db.BudgetProgress(context.Background(), principal, query)
	require.NoError(t, err)
	require.Equal(t, "RUB", data.Currency)
	require.Equal(t, analytics.Decimal("1000.00"), data.Totals.Budget)
	require.Equal(t, analytics.Decimal("25"), *data.Totals.Percent)
	require.Len(t, data.Rows, 1)
	require.True(t, data.HasMore)
	require.Equal(t, "budget:category:"+categoryID, data.Rows[0].ID)
	require.Equal(t, analytics.Decimal("250.00"), data.Rows[0].Spent)
}

func TestDataFreshnessReturnsDatabaseWideCompletedAndLatestSnapshots(t *testing.T) {
	db, mock := newTestDB(t)
	principal := analytics.Principal{UserIDs: []int64{42}}
	started := time.Date(2026, 8, 24, 10, 0, 0, 0, time.UTC)
	finished := started.Add(time.Minute)

	expectAnalyticsSnapshotBegin(mock)
	mock.ExpectQuery(regexp.QuoteMeta(latestCompletedSyncSQL)).
		WillReturnRows(mock.NewRows([]string{
			"started_at",
			"finished_at",
			"status",
			"sync_type",
			"server_timestamp",
			"records_processed",
		}).AddRow(started, finished, "completed", "partial", int64(10), int64(20)))
	mock.ExpectQuery(regexp.QuoteMeta(latestSyncAttemptSQL)).
		WillReturnRows(mock.NewRows([]string{
			"started_at",
			"finished_at",
			"status",
			"sync_type",
			"server_timestamp",
			"records_processed",
		}).AddRow(started.Add(time.Hour), nil, "failed", "partial", int64(11), int64(0)))
	mock.ExpectCommit()

	data, err := db.DataFreshness(context.Background(), principal)
	require.NoError(t, err)
	require.NotNil(t, data.LastCompleted)
	require.Equal(t, int64(10), data.LastCompleted.ServerTimestamp)
	require.NotNil(t, data.LastAttempt)
	require.Equal(t, "failed", data.LastAttempt.Status)
}

func TestAnalyticsRejectsUnscopedAndOversizedQueriesWithoutDatabaseAccess(t *testing.T) {
	db, _ := newTestDB(t)

	_, err := db.SpendingSummary(
		context.Background(),
		analytics.Principal{},
		analytics.SpendingSummaryQuery{
			Range: analyticsTestRange(),
			Limit: 10,
		},
	)
	require.ErrorIs(t, err, ErrAnalyticsAccessScope)

	_, err = db.SearchTransactions(
		context.Background(),
		analytics.Principal{UserIDs: []int64{1}},
		analytics.TransactionSearchQuery{
			Range: analyticsTestRange(),
			Limit: maxAnalyticsRows + 1,
		},
	)
	require.ErrorIs(t, err, ErrAnalyticsLimit)
}
