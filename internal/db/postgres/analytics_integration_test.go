package postgres

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/nemirlev/zenmoney-export/v2/internal/analytics"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
)

func TestAnalyticsQueriesAgainstPostgres(t *testing.T) {
	postgresVersion := os.Getenv("POSTGRES_VERSION")
	if postgresVersion == "" {
		postgresVersion = "18"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	container, err := tcpostgres.Run(
		ctx,
		"postgres:"+postgresVersion+"-alpine",
		tcpostgres.WithDatabase("zenanalytics"),
		tcpostgres.WithUsername("zenanalytics"),
		tcpostgres.WithPassword("zenanalytics"),
		tcpostgres.BasicWaitStrategies(),
	)
	if err != nil {
		t.Skipf("Docker is unavailable for analytics integration test: %v", err)
	}
	testcontainers.CleanupContainer(t, container)
	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)
	migrations, err := filepath.Glob(
		filepath.Join("..", "..", "..", "migrations", "postgres", "*.up.sql"),
	)
	require.NoError(t, err)
	applyMigrations(t, ctx, dsn, migrations)

	connection, err := pgx.Connect(ctx, dsn)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, connection.Close(context.Background())) })
	seedAnalyticsIntegrationData(t, ctx, connection)

	store, err := NewPostgresAnalyticsStore(ctx, dsn)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, store.Close(context.Background())) })
	principal := analytics.Principal{Subject: "integration", UserIDs: []int64{42}}
	dateRange := analytics.DateRange{
		From: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC),
	}

	spending, err := store.SpendingSummary(ctx, principal, analytics.SpendingSummaryQuery{
		Range: dateRange,
		Limit: 10,
	})
	require.NoError(t, err)
	require.Equal(t, "RUB", spending.Currency)
	require.Equal(t, analytics.Decimal("1800.0000000000000000"), spending.Total)
	require.Equal(t, int64(4), spending.TransactionCount)
	require.Len(t, spending.Categories, 3)
	require.Equal(t, analytics.Decimal("1200.0000000000000000"), spending.Categories[0].Amount)
	require.Equal(t, "Food", spending.Categories[0].Title)
	require.Equal(t, "category:uncategorized", spending.Categories[1].CategoryID)
	require.Equal(t, analytics.Decimal("200.0000000000000000"), spending.Categories[2].Amount)
	require.Equal(t, "Transport", spending.Categories[2].Title)
	limitedSpending, err := store.SpendingSummary(ctx, principal, analytics.SpendingSummaryQuery{
		Range: dateRange,
		Limit: 1,
	})
	require.NoError(t, err)
	require.Len(t, limitedSpending.Categories, 1)
	require.True(t, limitedSpending.HasMore)

	spendingWithHold, err := store.SpendingSummary(ctx, principal, analytics.SpendingSummaryQuery{
		Range: dateRange,
		Filters: analytics.AppliedFilters{
			IncludeHold: true,
		},
		Limit: 10,
	})
	require.NoError(t, err)
	require.Equal(t, analytics.Decimal("2000.0000000000000000"), spendingWithHold.Total)

	cashflow, err := store.Cashflow(ctx, principal, analytics.CashflowQuery{
		Range:       dateRange,
		Granularity: analytics.GranularityMonth,
		MaxPoints:   2,
	})
	require.NoError(t, err)
	require.Equal(t, analytics.Decimal("2000.0000000000000000"), cashflow.Totals.Income)
	require.Equal(t, analytics.Decimal("1800.0000000000000000"), cashflow.Totals.Outcome)
	require.Equal(t, analytics.Decimal("200.0000000000000000"), cashflow.Totals.Net)

	budget, err := store.BudgetProgress(ctx, principal, analytics.BudgetProgressQuery{
		Range: dateRange,
		Limit: 10,
	})
	require.NoError(t, err)
	require.Len(t, budget.Rows, 2)
	require.Equal(t, analytics.Decimal("3000"), budget.Totals.Budget)
	require.Equal(t, analytics.Decimal("1800.0000000000000000"), budget.Totals.Spent)
	require.Equal(t, "category:all", budget.Rows[0].CategoryID)
	require.Equal(t, analytics.Decimal("1800.0000000000000000"), budget.Rows[0].Spent)
	require.Equal(t, analytics.Decimal("1200.0000000000000000"), budget.Rows[1].Spent)
	limitedBudget, err := store.BudgetProgress(ctx, principal, analytics.BudgetProgressQuery{
		Range: dateRange,
		Limit: 1,
	})
	require.NoError(t, err)
	require.Len(t, limitedBudget.Rows, 1)
	require.True(t, limitedBudget.HasMore)

	page, err := store.SearchTransactions(ctx, principal, analytics.TransactionSearchQuery{
		Range: dateRange,
		Limit: 10,
	})
	require.NoError(t, err)
	require.Len(t, page.Items, 6)
	require.Contains(t, page.Items, analytics.TransactionItem{
		ID:               "00000000-0000-0000-0000-000000000103",
		Date:             "2026-08-12",
		Direction:        analytics.DirectionTransfer,
		Amount:           "0.0000000000000000",
		Income:           "500.0000000000000000",
		Outcome:          "500.0000000000000000",
		Currency:         "RUB",
		AccountID:        "00000000-0000-0000-0000-000000000001",
		AccountTitle:     "Primary",
		IncomeAccountID:  new("00000000-0000-0000-0000-000000000001"),
		OutcomeAccountID: new("00000000-0000-0000-0000-000000000002"),
		CategoryIDs:      []string{"category:uncategorized"},
		CategoryTitles:   []string{"Uncategorized"},
		MerchantID:       new("merchant:none"),
		MerchantTitle:    "No merchant",
	})
	firstPage, err := store.SearchTransactions(ctx, principal, analytics.TransactionSearchQuery{
		Range: dateRange,
		Limit: 1,
	})
	require.NoError(t, err)
	require.Len(t, firstPage.Items, 1)
	require.NotNil(t, firstPage.NextCursor)
	secondPage, err := store.SearchTransactions(ctx, principal, analytics.TransactionSearchQuery{
		Range:  dateRange,
		Cursor: firstPage.NextCursor,
		Limit:  1,
	})
	require.NoError(t, err)
	require.Len(t, secondPage.Items, 1)
	require.NotEqual(t, firstPage.Items[0].ID, secondPage.Items[0].ID)

	freshness, err := store.DataFreshness(ctx, principal)
	require.NoError(t, err)
	require.NotNil(t, freshness.LastCompleted)
	require.Equal(t, int64(123), freshness.LastCompleted.ServerTimestamp)

	verifyAnalyticsSnapshotIsolation(t, ctx, store, connection)
}

func verifyAnalyticsSnapshotIsolation(
	t *testing.T,
	ctx context.Context,
	store *DB,
	outside *pgx.Conn,
) {
	t.Helper()

	observed, err := withAnalyticsSnapshot(ctx, store.pool, func(executor analyticsQueryExecutor) (
		[]string,
		error,
	) {
		var readOnly string
		var isolation string
		if err := executor.QueryRow(
			ctx,
			`SELECT current_setting('transaction_read_only'), current_setting('transaction_isolation')`,
		).Scan(&readOnly, &isolation); err != nil {
			return nil, err
		}
		var before string
		if err := executor.QueryRow(ctx, `SELECT rate::text FROM instrument WHERE id = 1`).
			Scan(&before); err != nil {
			return nil, err
		}
		if _, err := outside.Exec(
			ctx,
			`UPDATE instrument SET rate = 101 WHERE id = 1`,
		); err != nil {
			return nil, err
		}
		var after string
		if err := executor.QueryRow(ctx, `SELECT rate::text FROM instrument WHERE id = 1`).
			Scan(&after); err != nil {
			return nil, err
		}
		return []string{readOnly, isolation, before, after}, nil
	})
	require.NoError(t, err)
	require.Equal(t, []string{"on", "repeatable read", "100", "100"}, observed)

	var committed string
	require.NoError(
		t,
		outside.QueryRow(ctx, `SELECT rate::text FROM instrument WHERE id = 1`).Scan(&committed),
	)
	require.Equal(t, "101", committed)
}

func seedAnalyticsIntegrationData(t *testing.T, ctx context.Context, connection *pgx.Conn) {
	t.Helper()
	statements := []string{
		`INSERT INTO instrument (id, title, short_title, symbol, rate, changed) VALUES
            (1, 'US Dollar', 'USD', '$', 100, 1),
            (2, 'Ruble', 'RUB', 'RUB', 1, 1)`,
		`INSERT INTO "user" (id, changed, login, currency, paid_till, month_start_day,
            is_forecast_enabled, plan_balance_mode, plan_settings, subscription)
         VALUES (42, 1, 'analytics', 2, 0, 1, false, 'balance', '[]', ''),
                (43, 1, 'other', 2, 0, 1, false, 'balance', '[]', '')`,
		`INSERT INTO account (id, changed, "user", instrument, type, title, in_balance, archive) VALUES
            ('00000000-0000-0000-0000-000000000001', 1, 42, 1, 'checking', 'Primary', true, false),
            ('00000000-0000-0000-0000-000000000002', 1, 42, 1, 'checking', 'Secondary', true, false),
			('00000000-0000-0000-0000-000000000003', 1, 42, 1, 'checking', 'Excluded', false, false),
            ('00000000-0000-0000-0000-000000000004', 1, 43, 1, 'checking', 'Other user', true, false)`,
		`INSERT INTO tag (id, changed, "user", title, parent, archive) VALUES
            ('00000000-0000-0000-0000-000000000010', 1, 42, 'Food', NULL, false),
            ('00000000-0000-0000-0000-000000000011', 1, 42, 'Groceries', '00000000-0000-0000-0000-000000000010', false),
			('00000000-0000-0000-0000-000000000012', 1, 42, 'Cafe', '00000000-0000-0000-0000-000000000010', false),
            ('00000000-0000-0000-0000-000000000013', 1, 42, 'Transport', NULL, false),
            ('00000000-0000-0000-0000-000000000014', 1, 42, 'Bus', '00000000-0000-0000-0000-000000000013', false)`,
		`INSERT INTO transaction (
            id, "user", date, income, outcome, income_instrument, outcome_instrument,
            created, deleted, hold, income_account, outcome_account, tag
         ) VALUES
            ('00000000-0000-0000-0000-000000000101', 42, '2026-08-10', 0, 10, 1, 1, 101, false, false,
             '00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000001',
             ARRAY['00000000-0000-0000-0000-000000000011'::uuid, '00000000-0000-0000-0000-000000000012'::uuid]),
            ('00000000-0000-0000-0000-000000000102', 42, '2026-08-11', 20, 0, 1, 1, 102, false, false,
             '00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000001', '{}'::uuid[]),
            ('00000000-0000-0000-0000-000000000103', 42, '2026-08-12', 5, 5, 1, 1, 103, false, false,
             '00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000002', '{}'::uuid[]),
            ('00000000-0000-0000-0000-000000000104', 42, '2026-08-13', 0, 100, 1, 1, 104, true, false,
             '00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000001', '{}'::uuid[]),
            ('00000000-0000-0000-0000-000000000105', 42, '2026-08-14', 0, 2, 1, 1, 105, false, true,
             '00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000001', '{}'::uuid[]),
            ('00000000-0000-0000-0000-000000000106', 42, '2026-08-15', 0, 50, 1, 1, 106, false, false,
             '00000000-0000-0000-0000-000000000003', '00000000-0000-0000-0000-000000000003', '{}'::uuid[]),
            ('00000000-0000-0000-0000-000000000107', 42, '2026-08-16', 0, 3, 1, 1, 107, false, false,
			 '00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000001', '{}'::uuid[]),
            ('00000000-0000-0000-0000-000000000108', 42, '2026-08-17', 0, 4, 1, 1, 108, false, false,
             '00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000001',
             ARRAY['00000000-0000-0000-0000-000000000011'::uuid, '00000000-0000-0000-0000-000000000014'::uuid]),
            ('00000000-0000-0000-0000-000000000109', 42, '2026-08-01', 0, 1, 1, 1, 109, false, false,
             '00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000001', '{}'::uuid[]),
            ('00000000-0000-0000-0000-000000000110', 42, '2026-09-01', 0, 1, 1, 1, 110, false, false,
             '00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000001', '{}'::uuid[]),
            ('00000000-0000-0000-0000-000000000111', 43, '2026-08-18', 0, 999, 1, 1, 111, false, false,
             '00000000-0000-0000-0000-000000000004', '00000000-0000-0000-0000-000000000004', '{}'::uuid[])`,
		`INSERT INTO budget (changed, "user", tag, date, income, outcome) VALUES
			(1, 42, '00000000-0000-0000-0000-000000000010', '2026-08-01', 0, 2000),
            (1, 42, '00000000-0000-0000-0000-000000000000', '2026-08-01', 0, 3000)`,
		`INSERT INTO sync_status (
            started_at, finished_at, sync_type, server_timestamp,
            records_processed, status, created_at, updated_at
         ) VALUES (CURRENT_TIMESTAMP - interval '1 minute', CURRENT_TIMESTAMP,
                   'partial', 123, 7, 'completed', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`,
	}
	for _, statement := range statements {
		_, err := connection.Exec(ctx, statement)
		require.NoError(t, err)
	}
}
