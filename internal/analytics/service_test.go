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
	userCatalog        []AnalyticsUser
	usersErr           error
	usersCalls         int
	catalogPrincipal   Principal
	principal          Principal
	spendingQuery      SpendingSummaryQuery
	budgetQuery        BudgetProgressQuery
	searchQuery        TransactionSearchQuery
	spendingCalls      int
	spendingData       SpendingSummaryData
	cashflowData       CashflowData
	cashflowCalls      int
	cashflowPrincipals []Principal
	cashflowQueries    []CashflowQuery
	cashflowDataSet    []CashflowData
	budgetData         BudgetProgressData
	transactionPage    TransactionPage
	freshnessData      FreshnessData
	freshnessPrincipal Principal
	err                error
}

func (s *fakeStore) AnalyticsUsers(
	_ context.Context,
	principal Principal,
) ([]AnalyticsUser, error) {
	s.usersCalls++
	s.catalogPrincipal = principal
	if s.usersErr != nil {
		return nil, s.usersErr
	}
	if s.userCatalog != nil {
		return cloneAnalyticsUsers(s.userCatalog), nil
	}
	users := make([]AnalyticsUser, 0, len(principal.UserIDs))
	for _, userID := range principal.UserIDs {
		users = append(users, AnalyticsUser{
			UserID: userID, ID: analyticsUserKey(userID), Currency: "RUB",
		})
	}
	return users, nil
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
	_ context.Context,
	principal Principal,
	query CashflowQuery,
) (CashflowData, error) {
	s.cashflowPrincipals = append(s.cashflowPrincipals, principal)
	s.cashflowQueries = append(s.cashflowQueries, query)
	s.cashflowCalls++
	if s.cashflowCalls <= len(s.cashflowDataSet) {
		return s.cashflowDataSet[s.cashflowCalls-1], s.err
	}
	return s.cashflowData, s.err
}

func (s *fakeStore) BudgetProgress(
	_ context.Context,
	_ Principal,
	query BudgetProgressQuery,
) (BudgetProgressData, error) {
	s.budgetQuery = query
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

func (s *fakeStore) DataFreshness(_ context.Context, principal Principal) (FreshnessData, error) {
	s.freshnessPrincipal = principal
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

func TestAllUsersPrincipalResolvesCatalogAndCanonicalizesEmptySelection(t *testing.T) {
	store := &fakeStore{
		userCatalog: []AnalyticsUser{
			{UserID: 9, ID: "user:9", Label: " Second ", Currency: "RUB"},
			{UserID: 4, ID: "user:4", Label: "First", Currency: "RUB"},
		},
		spendingData: SpendingSummaryData{Currency: "RUB", Total: "0"},
	}
	service := newTestService(t, store)

	result, err := service.GetSpendingSummary(
		context.Background(),
		Principal{Subject: " database-owner ", AllUsers: true},
		SpendingSummaryRequest{Period: PeriodRequest{From: "2026-08-01", To: "2026-08-01"}},
	)

	require.NoError(t, err)
	require.Equal(t, Principal{Subject: "database-owner", AllUsers: true}, store.catalogPrincipal)
	require.Equal(t, Principal{Subject: "database-owner", UserIDs: []int64{4, 9}}, store.principal)
	require.Equal(t, []string{"user:4", "user:9"}, result.Metadata.NormalizedRequest.SpendingSummary.Users.UserIDs)
	require.Equal(t, []AnalyticsUser{
		{UserID: 4, ID: "user:4", Label: "First", Currency: "RUB"},
		{UserID: 9, ID: "user:9", Label: "Second", Currency: "RUB"},
	}, result.Metadata.Users)
}

func TestUserSelectionNarrowsAllowlistAndRejectsUnknownUsers(t *testing.T) {
	store := &fakeStore{
		userCatalog: []AnalyticsUser{
			{UserID: 4, ID: "user:4", Currency: "RUB"},
			{UserID: 9, ID: "user:9", Currency: "RUB"},
		},
		spendingData: SpendingSummaryData{Currency: "RUB", Total: "0"},
	}
	service := newTestService(t, store)
	principal := Principal{Subject: "owner", UserIDs: []int64{9, 4}}
	period := PeriodRequest{From: "2026-08-01", To: "2026-08-01"}

	result, err := service.GetSpendingSummary(
		context.Background(), principal,
		SpendingSummaryRequest{
			Period: period,
			Users:  UserSelection{UserIDs: []string{" user:9 ", "user:9"}},
		},
	)
	require.NoError(t, err)
	require.Equal(t, Principal{Subject: "owner", UserIDs: []int64{9}}, store.principal)
	require.Equal(t, []string{"user:9"}, result.Metadata.NormalizedRequest.SpendingSummary.Users.UserIDs)
	require.Equal(t, []AnalyticsUser{{UserID: 9, ID: "user:9", Currency: "RUB"}}, result.Metadata.Users)

	before := store.spendingCalls
	_, err = service.GetSpendingSummary(
		context.Background(), principal,
		SpendingSummaryRequest{
			Period: period, Users: UserSelection{UserIDs: []string{"user:10"}},
		},
	)
	require.ErrorContains(t, err, "unknown or not authorized")
	require.Equal(t, before, store.spendingCalls)
}

func TestUserCatalogCannotExpandExplicitPrincipal(t *testing.T) {
	store := &fakeStore{
		userCatalog:  []AnalyticsUser{{UserID: 2, ID: "user:2", Currency: "RUB"}},
		spendingData: SpendingSummaryData{Currency: "RUB", Total: "0"},
	}
	service := newTestService(t, store)

	_, err := service.GetSpendingSummary(
		context.Background(), Principal{Subject: "owner", UserIDs: []int64{1}},
		SpendingSummaryRequest{Period: PeriodRequest{From: "2026-08-01", To: "2026-08-01"}},
	)

	require.ErrorContains(t, err, "outside principal scope")
	require.Zero(t, store.spendingCalls)
}

func TestAggregateResultCurrencyMustMatchEverySelectedUser(t *testing.T) {
	store := &fakeStore{
		userCatalog: []AnalyticsUser{
			{UserID: 1, ID: "user:1", Currency: "RUB"},
			{UserID: 2, ID: "user:2", Currency: "USD"},
		},
		spendingData: SpendingSummaryData{Currency: "RUB", Total: "0"},
	}
	service := newTestService(t, store)

	_, err := service.GetSpendingSummary(
		context.Background(), Principal{Subject: "owner", AllUsers: true},
		SpendingSummaryRequest{Period: PeriodRequest{From: "2026-08-01", To: "2026-08-01"}},
	)

	require.ErrorContains(t, err, "does not match selected analytics user user:2")
}

func TestPrincipalScopeMustBeExplicitAndUnambiguous(t *testing.T) {
	service := newTestService(t, &fakeStore{
		spendingData: SpendingSummaryData{Currency: "RUB", Total: "0"},
	})
	request := SpendingSummaryRequest{
		Period: PeriodRequest{From: "2026-08-01", To: "2026-08-01"},
	}

	_, err := service.GetSpendingSummary(
		context.Background(), Principal{Subject: "owner", AllUsers: true, UserIDs: []int64{1}}, request,
	)
	require.ErrorContains(t, err, "cannot combine")

	_, err = service.GetSpendingSummary(
		context.Background(), Principal{Subject: "owner"}, request,
	)
	require.ErrorContains(t, err, "no authorized users")
}

func TestServiceUsesConfiguredTimezoneWhenRequestOmitsIt(t *testing.T) {
	store := &fakeStore{spendingData: SpendingSummaryData{Currency: "RUB", Total: "0"}}
	service, err := NewService(store, Limits{DefaultTimezone: "Europe/Moscow"})
	require.NoError(t, err)

	result, err := service.GetSpendingSummary(
		context.Background(),
		Principal{Subject: "user", UserIDs: []int64{1}},
		SpendingSummaryRequest{Period: PeriodRequest{From: "2026-08-01", To: "2026-08-01"}},
	)

	require.NoError(t, err)
	require.Equal(t, "Europe/Moscow", store.spendingQuery.Range.Timezone)
	require.Equal(t, "Europe/Moscow", result.Metadata.Period.Timezone)
	require.Equal(
		t,
		"Europe/Moscow",
		result.Metadata.NormalizedRequest.SpendingSummary.Period.Timezone,
	)
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

func TestFiltersAcceptEveryPostgresUUIDShape(t *testing.T) {
	store := &fakeStore{spendingData: SpendingSummaryData{Currency: "RUB", Total: "0"}}
	service := newTestService(t, store)
	request := SpendingSummaryRequest{
		Period: PeriodRequest{From: "2026-08-01", To: "2026-08-01"},
		Filters: Filters{AccountIDs: []string{
			"00000000-0000-0000-0000-000000000000",
			"ABCDEFAB-CDEF-0ABC-0ABC-ABCDEFABCDEF",
		}},
	}

	_, err := service.GetSpendingSummary(
		context.Background(), Principal{Subject: "user", UserIDs: []int64{1}}, request,
	)

	require.NoError(t, err)
	require.Equal(t, []string{
		"00000000-0000-0000-0000-000000000000",
		"abcdefab-cdef-0abc-0abc-abcdefabcdef",
	}, store.spendingQuery.Filters.AccountIDs)
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
		context.Background(), principal, BudgetProgressRequest{Period: PeriodRequest{
			From: "2026-08-01", To: "2026-08-31",
		}},
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

func TestDataFreshnessIsDatabaseScopedWithoutPublicRecordCounts(t *testing.T) {
	finished := time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC)
	store := &fakeStore{freshnessData: FreshnessData{LastCompleted: &SyncSnapshot{
		StartedAt: finished.Add(-time.Minute), FinishedAt: &finished, Status: "completed",
		SyncType: "partial", ServerTimestamp: 123, RecordsProcessed: 999,
	}}}
	service := newTestService(t, store)
	service.now = func() time.Time { return finished }

	result, err := service.GetDataFreshness(
		context.Background(),
		Principal{Subject: "user", UserIDs: []int64{1}},
		DataFreshnessRequest{},
	)
	require.NoError(t, err)
	require.Equal(t, FreshnessScopeDatabase, result.Scope)
	require.Equal(t, []AnalyticsUser{{UserID: 1, ID: "user:1", Currency: "RUB"}}, result.Users)
	require.Equal(t, result.Users, result.Metadata.Users)
	require.Contains(
		t,
		result.Metadata.Rules.Limitations,
		"sync_status has database-wide scope because the schema lacks per-user synchronization provenance",
	)
	for _, column := range result.Table.Columns {
		require.NotEqual(t, "records", column.Key)
	}
	for _, row := range result.Table.Rows {
		for _, cell := range row.Cells {
			require.NotEqual(t, "records", cell.Key)
		}
	}

	payload, err := json.Marshal(result)
	require.NoError(t, err)
	require.NotContains(t, string(payload), "recordsProcessed")
	var decoded map[string]any
	require.NoError(t, json.Unmarshal(payload, &decoded))
	require.Equal(t, "database", decoded["scope"])
	completed := decoded["lastCompleted"].(map[string]any)
	require.NotContains(t, completed, "recordsProcessed")
}

func TestDataFreshnessResolvesAllUsersToExplicitStoreScope(t *testing.T) {
	store := &fakeStore{userCatalog: []AnalyticsUser{
		{UserID: 2, ID: "user:2", Currency: "USD"},
		{UserID: 1, ID: "user:1", Currency: "RUB"},
	}}
	service := newTestService(t, store)

	result, err := service.GetDataFreshness(
		context.Background(), Principal{Subject: "owner", AllUsers: true}, DataFreshnessRequest{},
	)

	require.NoError(t, err)
	require.Equal(t, Principal{Subject: "owner", UserIDs: []int64{1, 2}}, store.freshnessPrincipal)
	require.Equal(t, []string{"user:1", "user:2"}, []string{
		result.Users[0].ID, result.Users[1].ID,
	})
}

func TestBudgetProgressRequiresCompleteCalendarMonths(t *testing.T) {
	store := &fakeStore{budgetData: BudgetProgressData{
		Currency: "RUB", Totals: BudgetTotals{Budget: "0", Spent: "0", Remaining: "0"},
	}}
	service := newTestService(t, store)
	principal := Principal{Subject: "user", UserIDs: []int64{1}}

	for _, period := range []PeriodRequest{
		{From: "2026-01-02", To: "2026-01-31"},
		{From: "2026-01-01", To: "2026-01-30"},
		{From: "2026-01-15", To: "2026-02-14"},
	} {
		_, err := service.GetBudgetProgress(
			context.Background(), principal, BudgetProgressRequest{Period: period},
		)
		require.ErrorContains(t, err, "complete calendar months")
	}

	result, err := service.GetBudgetProgress(
		context.Background(), principal, BudgetProgressRequest{Period: PeriodRequest{
			From: "2026-01-01", To: "2026-02-28",
		}},
	)
	require.NoError(t, err)
	require.Equal(t, "2026-03-01", store.budgetQuery.Range.To.Format(dateLayout))
	require.Equal(t, "2026-02-28", result.Metadata.Period.To)
}

func TestBudgetProgressRejectsUnsupportedFilters(t *testing.T) {
	store := &fakeStore{}
	service := newTestService(t, store)
	principal := Principal{Subject: "user", UserIDs: []int64{1}}
	period := PeriodRequest{From: "2026-08-01", To: "2026-08-31"}

	for name, filters := range map[string]Filters{
		"account":  {AccountIDs: []string{testAccountID}},
		"merchant": {MerchantIDs: []string{"33333333-3333-3333-3333-333333333333"}},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := service.GetBudgetProgress(
				context.Background(), principal,
				BudgetProgressRequest{Period: period, Filters: filters},
			)
			require.ErrorContains(t, err, "supports only category")
		})
	}

	store.budgetData = BudgetProgressData{
		Currency: "RUB", Totals: BudgetTotals{Budget: "0", Spent: "0", Remaining: "0"},
	}
	_, err := service.GetBudgetProgress(
		context.Background(), principal,
		BudgetProgressRequest{Period: period, Filters: Filters{
			CategoryIDs: []string{testCategoryID}, IncludeHold: true,
		}},
	)
	require.NoError(t, err)
}

func TestAggregateResultsExposeStoreTruncation(t *testing.T) {
	store := &fakeStore{
		spendingData: SpendingSummaryData{Currency: "RUB", Total: "0", HasMore: true},
		budgetData: BudgetProgressData{
			Currency: "RUB", HasMore: true,
			Totals: BudgetTotals{Budget: "0", Spent: "0", Remaining: "0"},
		},
	}
	service := newTestService(t, store)
	principal := Principal{Subject: "user", UserIDs: []int64{1}}

	spending, err := service.GetSpendingSummary(
		context.Background(), principal,
		SpendingSummaryRequest{Period: PeriodRequest{From: "2026-08-01", To: "2026-08-31"}},
	)
	require.NoError(t, err)
	require.True(t, spending.HasMore)
	require.True(t, spending.Truncated)

	budget, err := service.GetBudgetProgress(
		context.Background(), principal,
		BudgetProgressRequest{Period: PeriodRequest{From: "2026-08-01", To: "2026-08-31"}},
	)
	require.NoError(t, err)
	require.True(t, budget.HasMore)
	require.True(t, budget.Truncated)
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
	service, err := NewService(
		store,
		Limits{MaxPeriodDays: 365, MaxPageSize: 100, MaxChartPoints: 100},
	)
	require.NoError(t, err)
	return service
}
