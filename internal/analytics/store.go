package analytics

import "context"

// AnalyticsStore is the deliberately small, read-only persistence boundary.
// Every method receives an authenticated Principal. AnalyticsUsers resolves the
// authenticated catalog; report methods receive an explicit, service-validated
// UserIDs subset and must scope every query to it.
type AnalyticsStore interface {
	AnalyticsUsers(context.Context, Principal) ([]AnalyticsUser, error)
	SpendingSummary(context.Context, Principal, SpendingSummaryQuery) (SpendingSummaryData, error)
	Cashflow(context.Context, Principal, CashflowQuery) (CashflowData, error)
	BudgetProgress(context.Context, Principal, BudgetProgressQuery) (BudgetProgressData, error)
	SearchTransactions(context.Context, Principal, TransactionSearchQuery) (TransactionPage, error)
	DataFreshness(context.Context, Principal) (FreshnessData, error)
}
