package analytics

import "time"

const SchemaVersion = "1.0"

// Decimal is a base-10, non-exponential financial value. It is deliberately
// serialized as a JSON string so tool clients never lose NUMERIC precision.
type Decimal string

type Principal struct {
	Subject  string
	AllUsers bool
	UserIDs  []int64
}

// AnalyticsUser is a user visible inside an authenticated analytics scope.
// UserID is store-internal; ID is the stable public key "user:<UserID>".
type AnalyticsUser struct {
	UserID   int64  `json:"-"`
	ID       string `json:"id"`
	Label    string `json:"label,omitempty"`
	Currency string `json:"currency"`
}

// UserSelection narrows an authenticated principal by stable analytics user
// keys. An empty selection means every user in the principal's catalog.
type UserSelection struct {
	UserIDs []string `json:"userIds,omitempty"`
}

type PeriodRequest struct {
	From     string `json:"from"`
	To       string `json:"to"`
	Timezone string `json:"timezone,omitempty"`
}

// DateRange is the normalized half-open range used by stores: [From, To).
type DateRange struct {
	From     time.Time
	To       time.Time
	Timezone string
}

type Period struct {
	From          string `json:"from"`
	To            string `json:"to"`
	FromInclusive bool   `json:"fromInclusive"`
	ToInclusive   bool   `json:"toInclusive"`
	Timezone      string `json:"timezone"`
}

type Filters struct {
	AccountIDs  []string `json:"accountIds,omitempty"`
	CategoryIDs []string `json:"categoryIds,omitempty"`
	MerchantIDs []string `json:"merchantIds,omitempty"`
	IncludeHold bool     `json:"includeHold,omitempty"`
}

type AppliedFilters struct {
	AccountIDs  []string `json:"accountIds"`
	CategoryIDs []string `json:"categoryIds"`
	MerchantIDs []string `json:"merchantIds"`
	IncludeHold bool     `json:"includeHold"`
	Search      string   `json:"search,omitempty"`
}

type CalculationRules struct {
	DeletedTransactions string   `json:"deletedTransactions"`
	HoldTransactions    string   `json:"holdTransactions"`
	Transfers           string   `json:"transfers"`
	MixedFlows          string   `json:"mixedFlows"`
	Currencies          string   `json:"currencies"`
	MultiTag            string   `json:"multiTag"`
	CategoryHierarchy   string   `json:"categoryHierarchy"`
	Accounts            string   `json:"accounts"`
	Merchants           string   `json:"merchants"`
	Dates               string   `json:"dates"`
	Rounding            string   `json:"rounding"`
	Limitations         []string `json:"limitations"`
}

type ReportKind string

const (
	ReportSpendingSummary ReportKind = "spending_summary"
	ReportCashflow        ReportKind = "cashflow"
	ReportBudgetProgress  ReportKind = "budget_progress"
	ReportTransactions    ReportKind = "transactions"
	ReportDataFreshness   ReportKind = "data_freshness"
)

type ReportMetadata struct {
	SchemaVersion     string                   `json:"schemaVersion"`
	ReportKind        ReportKind               `json:"reportKind"`
	Period            *Period                  `json:"period,omitempty"`
	Currency          string                   `json:"currency,omitempty"`
	Users             []AnalyticsUser          `json:"users"`
	Filters           AppliedFilters           `json:"filters"`
	Rules             CalculationRules         `json:"calculationRules"`
	LastSyncAt        *time.Time               `json:"lastSyncAt,omitempty"`
	NormalizedRequest *NormalizedReportRequest `json:"normalizedRequest,omitempty"`
}

type Granularity string

const (
	GranularityDay   Granularity = "day"
	GranularityWeek  Granularity = "week"
	GranularityMonth Granularity = "month"
)

type SpendingSummaryRequest struct {
	Period  PeriodRequest `json:"period"`
	Filters Filters       `json:"filters,omitempty"`
	Limit   int           `json:"limit,omitempty"`
	Users   UserSelection `json:"users,omitempty"`
}

type SpendingSummaryQuery struct {
	Range   DateRange
	Filters AppliedFilters
	Limit   int
}

type SpendingCategory struct {
	ID               string  `json:"id"`
	CategoryID       string  `json:"categoryId"`
	ParentCategoryID *string `json:"parentCategoryId,omitempty"`
	Title            string  `json:"title"`
	Amount           Decimal `json:"amount"`
	SharePercent     Decimal `json:"sharePercent"`
	TransactionCount int64   `json:"transactionCount"`
}

type SpendingSummaryData struct {
	Currency         string             `json:"currency"`
	Total            Decimal            `json:"total"`
	TransactionCount int64              `json:"transactionCount"`
	Categories       []SpendingCategory `json:"categories"`
	HasMore          bool               `json:"hasMore"`
	LastSyncAt       *time.Time         `json:"lastSyncAt,omitempty"`
}

type SpendingSummaryResult struct {
	Metadata         ReportMetadata     `json:"metadata"`
	Total            Decimal            `json:"total"`
	TransactionCount int64              `json:"transactionCount"`
	Categories       []SpendingCategory `json:"categories"`
	HasMore          bool               `json:"hasMore"`
	Truncated        bool               `json:"truncated"`
	Table            TableFallback      `json:"table"`
}

type CashflowRequest struct {
	Period      PeriodRequest `json:"period"`
	Filters     Filters       `json:"filters,omitempty"`
	Granularity Granularity   `json:"granularity,omitempty"`
	Users       UserSelection `json:"users,omitempty"`
}

type CashflowQuery struct {
	Range       DateRange
	Filters     AppliedFilters
	Granularity Granularity
	MaxPoints   int
}

type CashflowPoint struct {
	ID      string  `json:"id"`
	From    string  `json:"from"`
	To      string  `json:"to"`
	Label   string  `json:"label"`
	Income  Decimal `json:"income"`
	Outcome Decimal `json:"outcome"`
	Net     Decimal `json:"net"`
}

type CashflowTotals struct {
	Income  Decimal `json:"income"`
	Outcome Decimal `json:"outcome"`
	Net     Decimal `json:"net"`
}

type CashflowData struct {
	Currency   string          `json:"currency"`
	Points     []CashflowPoint `json:"points"`
	Totals     CashflowTotals  `json:"totals"`
	LastSyncAt *time.Time      `json:"lastSyncAt,omitempty"`
}

type CashflowResult struct {
	Metadata ReportMetadata  `json:"metadata"`
	Points   []CashflowPoint `json:"points"`
	Totals   CashflowTotals  `json:"totals"`
	Table    TableFallback   `json:"table"`
}

type BudgetProgressRequest struct {
	Period  PeriodRequest `json:"period"`
	Filters Filters       `json:"filters,omitempty"`
	Limit   int           `json:"limit,omitempty"`
	Users   UserSelection `json:"users,omitempty"`
}

type BudgetProgressQuery struct {
	Range   DateRange
	Filters AppliedFilters
	Limit   int
}

type BudgetProgressRow struct {
	ID               string   `json:"id"`
	CategoryID       string   `json:"categoryId"`
	Title            string   `json:"title"`
	Budget           Decimal  `json:"budget"`
	Spent            Decimal  `json:"spent"`
	Remaining        Decimal  `json:"remaining"`
	Percent          *Decimal `json:"percent,omitempty"`
	TransactionCount int64    `json:"transactionCount"`
}

type BudgetTotals struct {
	Budget    Decimal  `json:"budget"`
	Spent     Decimal  `json:"spent"`
	Remaining Decimal  `json:"remaining"`
	Percent   *Decimal `json:"percent,omitempty"`
}

type BudgetProgressData struct {
	Currency   string              `json:"currency"`
	Rows       []BudgetProgressRow `json:"rows"`
	Totals     BudgetTotals        `json:"totals"`
	HasMore    bool                `json:"hasMore"`
	LastSyncAt *time.Time          `json:"lastSyncAt,omitempty"`
}

type BudgetProgressResult struct {
	Metadata  ReportMetadata      `json:"metadata"`
	Rows      []BudgetProgressRow `json:"rows"`
	Totals    BudgetTotals        `json:"totals"`
	HasMore   bool                `json:"hasMore"`
	Truncated bool                `json:"truncated"`
	Table     TableFallback       `json:"table"`
}

type TransactionDirection string

const (
	DirectionIncome      TransactionDirection = "income"
	DirectionOutcome     TransactionDirection = "outcome"
	DirectionTransfer    TransactionDirection = "transfer"
	DirectionDebtIncome  TransactionDirection = "debt_income"
	DirectionDebtOutcome TransactionDirection = "debt_outcome"
	DirectionMixed       TransactionDirection = "mixed"
	DirectionInvalid     TransactionDirection = "invalid"
)

type TransactionSearchRequest struct {
	Period   PeriodRequest `json:"period"`
	Filters  Filters       `json:"filters,omitempty"`
	Text     string        `json:"text,omitempty"`
	Cursor   string        `json:"cursor,omitempty"`
	PageSize int           `json:"pageSize,omitempty"`
	Users    UserSelection `json:"users,omitempty"`
}

type TransactionCursor struct {
	Date    time.Time
	Created int64
	ID      string
}

type TransactionSearchQuery struct {
	Range   DateRange
	Filters AppliedFilters
	Text    string
	Cursor  *TransactionCursor
	Limit   int
}

type TransactionItem struct {
	ID               string               `json:"id"`
	Date             string               `json:"date"`
	Direction        TransactionDirection `json:"direction"`
	Amount           Decimal              `json:"amount"`
	Income           Decimal              `json:"income"`
	Outcome          Decimal              `json:"outcome"`
	Currency         string               `json:"currency"`
	AccountID        string               `json:"accountId"`
	AccountTitle     string               `json:"accountTitle"`
	IncomeAccountID  *string              `json:"incomeAccountId,omitempty"`
	OutcomeAccountID *string              `json:"outcomeAccountId,omitempty"`
	CategoryIDs      []string             `json:"categoryIds"`
	CategoryTitles   []string             `json:"categoryTitles"`
	MerchantID       *string              `json:"merchantId,omitempty"`
	MerchantTitle    string               `json:"merchantTitle"`
	Hold             bool                 `json:"hold"`
}

type TransactionPage struct {
	Currency   string
	Items      []TransactionItem
	NextCursor *TransactionCursor
	LastSyncAt *time.Time
}

type TransactionSearchResult struct {
	Metadata   ReportMetadata    `json:"metadata"`
	Items      []TransactionItem `json:"items"`
	NextCursor string            `json:"nextCursor,omitempty"`
	Table      TableFallback     `json:"table"`
}

type DataFreshnessRequest struct{}

type SyncSnapshot struct {
	StartedAt        time.Time  `json:"startedAt"`
	FinishedAt       *time.Time `json:"finishedAt,omitempty"`
	Status           string     `json:"status"`
	SyncType         string     `json:"syncType"`
	ServerTimestamp  int64      `json:"serverTimestamp"`
	RecordsProcessed int64      `json:"-"`
}

type FreshnessData struct {
	LastCompleted *SyncSnapshot `json:"lastCompleted,omitempty"`
	LastAttempt   *SyncSnapshot `json:"lastAttempt,omitempty"`
}

type FreshnessScope string

const FreshnessScopeDatabase FreshnessScope = "database"

type DataFreshnessResult struct {
	Metadata      ReportMetadata  `json:"metadata"`
	Scope         FreshnessScope  `json:"scope"`
	LastCompleted *SyncSnapshot   `json:"lastCompleted,omitempty"`
	LastAttempt   *SyncSnapshot   `json:"lastAttempt,omitempty"`
	AgeSeconds    *int64          `json:"ageSeconds,omitempty"`
	Stale         bool            `json:"stale"`
	Users         []AnalyticsUser `json:"users"`
	Table         TableFallback   `json:"table"`
}

type ReportRequest struct {
	Kind            ReportKind                `json:"kind"`
	SpendingSummary *SpendingSummaryRequest   `json:"spendingSummary,omitempty"`
	Cashflow        *CashflowRequest          `json:"cashflow,omitempty"`
	BudgetProgress  *BudgetProgressRequest    `json:"budgetProgress,omitempty"`
	Transactions    *TransactionSearchRequest `json:"transactions,omitempty"`
	DataFreshness   *DataFreshnessRequest     `json:"dataFreshness,omitempty"`
}

type NormalizedReportRequest struct {
	Kind            ReportKind                `json:"kind"`
	SpendingSummary *SpendingSummaryRequest   `json:"spendingSummary,omitempty"`
	Cashflow        *CashflowRequest          `json:"cashflow,omitempty"`
	BudgetProgress  *BudgetProgressRequest    `json:"budgetProgress,omitempty"`
	Transactions    *TransactionSearchRequest `json:"transactions,omitempty"`
	DataFreshness   *DataFreshnessRequest     `json:"dataFreshness,omitempty"`
}

type ReportEnvelope struct {
	Kind            ReportKind               `json:"kind"`
	SpendingSummary *SpendingSummaryResult   `json:"spendingSummary,omitempty"`
	Cashflow        *CashflowResult          `json:"cashflow,omitempty"`
	BudgetProgress  *BudgetProgressResult    `json:"budgetProgress,omitempty"`
	Transactions    *TransactionSearchResult `json:"transactions,omitempty"`
	DataFreshness   *DataFreshnessResult     `json:"dataFreshness,omitempty"`
}
