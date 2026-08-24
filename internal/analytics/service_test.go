package analytics

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const (
	testAccountID  = "11111111-1111-4111-8111-111111111111"
	testCategoryID = "22222222-2222-4222-8222-222222222222"
)

type fakeStore struct {
	principal       Principal
	spendingQuery   SpendingSummaryQuery
	searchQuery     TransactionSearchQuery
	spendingCalls   int
	spendingData    SpendingSummaryData
	cashflowData    CashflowData
	budgetData      BudgetProgressData
	transactionPage TransactionPage
	freshnessData   FreshnessData
	err             error
}

func (s *fakeStore) SpendingSummary(
	_ context.Context,
	principal Principal,
	query SpendingSummaryQuery,
) (SpendingSummaryData, error) {
	s.principal = principal
	s.spendingQuery = query
	s.spendingCalls++
	return s.spendingData, s.err
}

func (s *fakeStore) Cashflow(
	context.Context,
	Principal,
	CashflowQuery,
) (CashflowData, error) {
	return s.cashflowData, s.err
}

func (s *fakeStore) BudgetProgress(
	context.Context,
	Principal,
	BudgetProgressQuery,
) (BudgetProgressData, error) {
	return s.budgetData, s.err
}

func (s *fakeStore) SearchTransactions(
	_ context.Context,
	_ Principal,
	query TransactionSearchQuery,
) (TransactionPage, error) {
	s.searchQuery = query
	return s.transactionPage, s.err
}

func (s *fakeStore) DataFreshness(context.Context, Principal) (FreshnessData, error) {
	return s.freshnessData, s.err
}

func TestSpendingSummaryNormalizesAuthorizationPeriodAndFilters(t *testing.T) {
	lastSync := time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC)
	store := &fakeStore{spendingData: SpendingSummaryData{
		Currency: "RUB", Total: "12.30", TransactionCount: 2, LastSyncAt: &lastSync,
		Categories: []SpendingCategory{{
			ID: "category:" + testCategoryID, CategoryID: testCategoryID, Title: "Food",
			Amount: "12.30", SharePercent: "100", TransactionCount: 2,
		}},
	}}
	service := newTestService(t, store)

	result, err := service.GetSpendingSummary(
		context.Background(),
		Principal{Subject: " auth-user ", UserIDs: []int64{9, 4, 9}},
		SpendingSummaryRequest{
			Period:  PeriodRequest{From: "2026-08-01", To: "2026-08-24", Timezone: "Europe/Moscow"},
			Filters: Filters{AccountIDs: []string{testAccountID, testAccountID}},
		},
	)

	require.NoError(t, err)
	require.Equal(t, Principal{Subject: "auth-user", UserIDs: []int64{4, 9}}, store.principal)
	require.Equal(t, "2026-08-01", store.spendingQuery.Range.From.Format(dateLayout))
	require.Equal(t, "2026-08-25", store.spendingQuery.Range.To.Format(dateLayout))
	require.Equal(t, []string{testAccountID}, store.spendingQuery.Filters.AccountIDs)
	require.Equal(t, "RUB", result.Metadata.Currency)
	require.True(t, result.Metadata.Period.FromInclusive)
	require.True(t, result.Metadata.Period.ToInclusive)
	require.Equal(t, "2026-08-24", result.Metadata.Period.To)
	require.Equal(t, Decimal("12.30"), result.Total)
	require.Equal(t, "12.30", result.Table.Rows[0].Cells[1].Value)
	require.NotNil(t, result.Metadata.NormalizedRequest)
	require.Empty(t, result.Metadata.NormalizedRequest.SpendingSummary.Filters.AccountIDs[1:])
	require.Contains(t, result.Metadata.Rules.MultiTag, "split equally")
}

func TestPublicRequestsDoNotContainModelSelectedCurrency(t *testing.T) {
	for name, request := range map[string]any{
		"spending": SpendingSummaryRequest{},
		"cashflow": CashflowRequest{},
		"budget":   BudgetProgressRequest{},
		"search":   TransactionSearchRequest{},
	} {
		t.Run(name, func(t *testing.T) {
			payload, err := json.Marshal(request)
			require.NoError(t, err)
			require.NotContains(t, string(payload), "currency")
		})
	}
}

func TestServiceRejectsInvalidRequestBeforeStore(t *testing.T) {
	store := &fakeStore{}
	service := newTestService(t, store)
	principal := Principal{Subject: "user", UserIDs: []int64{1}}

	tests := []struct {
		name    string
		request SpendingSummaryRequest
		want    string
	}{
		{
			name: "reversed dates",
			request: SpendingSummaryRequest{Period: PeriodRequest{
				From: "2026-08-25", To: "2026-08-24",
			}},
			want: "must not precede",
		},
		{
			name: "invalid timezone",
			request: SpendingSummaryRequest{Period: PeriodRequest{
				From: "2026-08-01", To: "2026-08-24", Timezone: "Mars/Base",
			}},
			want: "invalid timezone",
		},
		{
			name: "invalid filter",
			request: SpendingSummaryRequest{
				Period:  PeriodRequest{From: "2026-08-01", To: "2026-08-24"},
				Filters: Filters{AccountIDs: []string{"not-a-uuid"}},
			},
			want: "invalid account ID",
		},
		{
			name: "excessive limit",
			request: SpendingSummaryRequest{
				Period: PeriodRequest{From: "2026-08-01", To: "2026-08-24"}, Limit: 101,
			},
			want: "between 1 and 100",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := service.GetSpendingSummary(context.Background(), principal, test.request)
			require.ErrorContains(t, err, test.want)
		})
	}
	require.Zero(t, store.spendingCalls)
}

func TestServiceRejectsMissingAuthorization(t *testing.T) {
	service := newTestService(t, &fakeStore{})
	request := SpendingSummaryRequest{Period: PeriodRequest{From: "2026-08-01", To: "2026-08-24"}}

	_, err := service.GetSpendingSummary(context.Background(), Principal{}, request)

	require.ErrorContains(t, err, "principal subject")
}

func TestSearchCursorIsOpaqueAndStoreReceivesTypedCursor(t *testing.T) {
	next := &TransactionCursor{
		Date: time.Date(2026, 8, 20, 0, 0, 0, 0, time.UTC), Created: 42,
		ID: "33333333-3333-4333-8333-333333333333",
	}
	store := &fakeStore{transactionPage: TransactionPage{
		Currency: "RUB", NextCursor: next, Items: []TransactionItem{{
			ID: next.ID, Date: "2026-08-20", Direction: DirectionTransfer,
			Amount: "0", Income: "100", Outcome: "100", Currency: "RUB",
		}},
	}}
	service := newTestService(t, store)
	principal := Principal{Subject: "user", UserIDs: []int64{1}}
	request := TransactionSearchRequest{
		Period: PeriodRequest{From: "2026-08-01", To: "2026-08-24"}, PageSize: 10,
	}

	first, err := service.SearchTransactions(context.Background(), principal, request)
	require.NoError(t, err)
	require.NotEmpty(t, first.NextCursor)
	require.NotContains(t, first.NextCursor, next.ID)

	request.Cursor = first.NextCursor
	_, err = service.SearchTransactions(context.Background(), principal, request)
	require.NoError(t, err)
	require.Equal(t, next.ID, store.searchQuery.Cursor.ID)
	require.Equal(t, next.Created, store.searchQuery.Cursor.Created)

	request.Cursor = "not-base64"
	_, err = service.SearchTransactions(context.Background(), principal, request)
	require.ErrorContains(t, err, "invalid transaction cursor")
}

func TestAllDataReportsExposeStableStructuredMetadataAndTables(t *testing.T) {
	finished := time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC)
	zero := Decimal("0")
	store := &fakeStore{
		spendingData: SpendingSummaryData{Currency: "RUB", Total: zero},
		cashflowData: CashflowData{Currency: "RUB", Totals: CashflowTotals{
			Income: zero, Outcome: zero, Net: zero,
		}},
		budgetData: BudgetProgressData{Currency: "RUB", Totals: BudgetTotals{
			Budget: zero, Spent: zero, Remaining: zero,
		}},
		transactionPage: TransactionPage{Currency: "RUB"},
		freshnessData: FreshnessData{LastCompleted: &SyncSnapshot{
			StartedAt: finished.Add(-time.Minute), FinishedAt: &finished, Status: "completed",
		}},
	}
	service := newTestService(t, store)
	service.now = func() time.Time { return finished.Add(time.Hour) }
	principal := Principal{Subject: "user", UserIDs: []int64{1}}
	period := PeriodRequest{From: "2026-08-01", To: "2026-08-24"}

	spending, err := service.GetSpendingSummary(
		context.Background(), principal, SpendingSummaryRequest{Period: period},
	)
	require.NoError(t, err)
	require.Equal(t, SchemaVersion, spending.Metadata.SchemaVersion)
	require.NotNil(t, spending.Categories)
	require.NotNil(t, spending.Table.Rows)

	cashflow, err := service.GetCashflow(
		context.Background(), principal, CashflowRequest{Period: period},
	)
	require.NoError(t, err)
	require.Equal(t, ReportCashflow, cashflow.Metadata.ReportKind)
	require.NotNil(t, cashflow.Points)

	budget, err := service.GetBudgetProgress(
		context.Background(), principal, BudgetProgressRequest{Period: period},
	)
	require.NoError(t, err)
	require.Equal(t, ReportBudgetProgress, budget.Metadata.ReportKind)
	require.NotNil(t, budget.Rows)

	transactions, err := service.SearchTransactions(
		context.Background(), principal, TransactionSearchRequest{Period: period},
	)
	require.NoError(t, err)
	require.Equal(t, ReportTransactions, transactions.Metadata.ReportKind)
	require.NotNil(t, transactions.Items)

	freshness, err := service.GetDataFreshness(
		context.Background(), principal, DataFreshnessRequest{},
	)
	require.NoError(t, err)
	require.Equal(t, ReportDataFreshness, freshness.Metadata.ReportKind)
	require.Equal(t, int64(3600), *freshness.AgeSeconds)
	require.False(t, freshness.Stale)
	require.Empty(t, freshness.Metadata.Currency)
}

func TestServiceWrapsStoreErrors(t *testing.T) {
	store := &fakeStore{err: errors.New("database unavailable")}
	service := newTestService(t, store)

	_, err := service.GetSpendingSummary(
		context.Background(),
		Principal{Subject: "user", UserIDs: []int64{1}},
		SpendingSummaryRequest{Period: PeriodRequest{From: "2026-08-01", To: "2026-08-24"}},
	)

	require.ErrorContains(t, err, "get spending summary")
	require.ErrorContains(t, err, "database unavailable")
}

func newTestService(t *testing.T, store AnalyticsStore) *Service {
	t.Helper()
	service, err := NewService(store, Limits{MaxPeriodDays: 365, MaxPageSize: 100, MaxChartPoints: 100})
	require.NoError(t, err)
	return service
}
