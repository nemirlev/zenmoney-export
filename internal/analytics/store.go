package analytics

import "context"

// AnalyticsStore is the deliberately small, read-only persistence boundary.
// Every method receives an authenticated Principal; implementations must scope
// every query to Principal.UserIDs and must never accept a user ID from tool input.
type AnalyticsStore interface {
	SpendingSummary(context.Context, Principal, SpendingSummaryQuery) (SpendingSummaryData, error)
	Cashflow(context.Context, Principal, CashflowQuery) (CashflowData, error)
	BudgetProgress(context.Context, Principal, BudgetProgressQuery) (BudgetProgressData, error)
	SearchTransactions(context.Context, Principal, TransactionSearchQuery) (TransactionPage, error)
	DataFreshness(context.Context, Principal) (FreshnessData, error)
}
