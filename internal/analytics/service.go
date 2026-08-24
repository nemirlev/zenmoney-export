package analytics

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

const dateLayout = "2006-01-02"

type Limits struct {
	MaxPeriodDays   int
	DefaultPageSize int
	MaxPageSize     int
	MaxChartPoints  int
	MaxFilterValues int
	StaleAfter      time.Duration
}

func DefaultLimits() Limits {
	return Limits{
		MaxPeriodDays:   3660,
		DefaultPageSize: 50,
		MaxPageSize:     100,
		MaxChartPoints:  400,
		MaxFilterValues: 100,
		StaleAfter:      24 * time.Hour,
	}
}

type Service struct {
	store  AnalyticsStore
	limits Limits
	now    func() time.Time
}

func NewService(store AnalyticsStore, limits Limits) (*Service, error) {
	if store == nil {
		return nil, errors.New("analytics store is required")
	}
	limits = normalizeLimits(limits)
	return &Service{store: store, limits: limits, now: time.Now}, nil
}

func (s *Service) GetSpendingSummary(
	ctx context.Context,
	principal Principal,
	request SpendingSummaryRequest,
) (SpendingSummaryResult, error) {
	principal, err := normalizePrincipal(principal)
	if err != nil {
		return SpendingSummaryResult{}, err
	}
	canonical, query, period, err := s.normalizeSpendingRequest(request)
	if err != nil {
		return SpendingSummaryResult{}, err
	}
	data, err := s.store.SpendingSummary(ctx, principal, query)
	if err != nil {
		return SpendingSummaryResult{}, fmt.Errorf("get spending summary: %w", err)
	}
	if err := validateSpendingData(data); err != nil {
		return SpendingSummaryResult{}, fmt.Errorf("invalid spending data: %w", err)
	}
	normalized := NormalizedReportRequest{Kind: ReportSpendingSummary, SpendingSummary: &canonical}
	return SpendingSummaryResult{
		Metadata: reportMetadata(
			ReportSpendingSummary, &period, data.Currency, query.Filters, data.LastSyncAt, normalized,
		),
		Total: data.Total, TransactionCount: data.TransactionCount,
		Categories: nonNilSpendingRows(data.Categories),
		Table:      spendingTable(data.Categories, data.Currency),
	}, nil
}

func (s *Service) GetCashflow(
	ctx context.Context,
	principal Principal,
	request CashflowRequest,
) (CashflowResult, error) {
	principal, err := normalizePrincipal(principal)
	if err != nil {
		return CashflowResult{}, err
	}
	canonical, query, period, err := s.normalizeCashflowRequest(request)
	if err != nil {
		return CashflowResult{}, err
	}
	data, err := s.store.Cashflow(ctx, principal, query)
	if err != nil {
		return CashflowResult{}, fmt.Errorf("get cashflow: %w", err)
	}
	if err := validateCashflowData(data); err != nil {
		return CashflowResult{}, fmt.Errorf("invalid cashflow data: %w", err)
	}
	normalized := NormalizedReportRequest{Kind: ReportCashflow, Cashflow: &canonical}
	return CashflowResult{
		Metadata: reportMetadata(
			ReportCashflow, &period, data.Currency, query.Filters, data.LastSyncAt, normalized,
		),
		Points: nonNilCashflowPoints(data.Points), Totals: data.Totals,
		Table: cashflowTable(data.Points, data.Currency),
	}, nil
}

func (s *Service) GetBudgetProgress(
	ctx context.Context,
	principal Principal,
	request BudgetProgressRequest,
) (BudgetProgressResult, error) {
	principal, err := normalizePrincipal(principal)
	if err != nil {
		return BudgetProgressResult{}, err
	}
	canonical, query, period, err := s.normalizeBudgetRequest(request)
	if err != nil {
		return BudgetProgressResult{}, err
	}
	data, err := s.store.BudgetProgress(ctx, principal, query)
	if err != nil {
		return BudgetProgressResult{}, fmt.Errorf("get budget progress: %w", err)
	}
	if err := validateBudgetData(data); err != nil {
		return BudgetProgressResult{}, fmt.Errorf("invalid budget data: %w", err)
	}
	normalized := NormalizedReportRequest{Kind: ReportBudgetProgress, BudgetProgress: &canonical}
	return BudgetProgressResult{
		Metadata: reportMetadata(
			ReportBudgetProgress, &period, data.Currency, query.Filters, data.LastSyncAt, normalized,
		),
		Rows: nonNilBudgetRows(data.Rows), Totals: data.Totals,
		Table: budgetTable(data.Rows, data.Currency),
	}, nil
}

func (s *Service) SearchTransactions(
	ctx context.Context,
	principal Principal,
	request TransactionSearchRequest,
) (TransactionSearchResult, error) {
	principal, err := normalizePrincipal(principal)
	if err != nil {
		return TransactionSearchResult{}, err
	}
	canonical, query, period, err := s.normalizeSearchRequest(request)
	if err != nil {
		return TransactionSearchResult{}, err
	}
	page, err := s.store.SearchTransactions(ctx, principal, query)
	if err != nil {
		return TransactionSearchResult{}, fmt.Errorf("search transactions: %w", err)
	}
	if err := validateTransactionPage(page); err != nil {
		return TransactionSearchResult{}, fmt.Errorf("invalid transaction data: %w", err)
	}
	nextCursor, err := encodeCursor(page.NextCursor)
	if err != nil {
		return TransactionSearchResult{}, err
	}
	normalized := NormalizedReportRequest{Kind: ReportTransactions, Transactions: &canonical}
	filters := query.Filters
	filters.Search = canonical.Text
	return TransactionSearchResult{
		Metadata: reportMetadata(
			ReportTransactions, &period, page.Currency, filters, page.LastSyncAt, normalized,
		),
		Items: nonNilTransactions(page.Items), NextCursor: nextCursor,
		Table: transactionsTable(page.Items, page.Currency),
	}, nil
}

func (s *Service) GetDataFreshness(
	ctx context.Context,
	principal Principal,
	_ DataFreshnessRequest,
) (DataFreshnessResult, error) {
	principal, err := normalizePrincipal(principal)
	if err != nil {
		return DataFreshnessResult{}, err
	}
	data, err := s.store.DataFreshness(ctx, principal)
	if err != nil {
		return DataFreshnessResult{}, fmt.Errorf("get data freshness: %w", err)
	}
	var age *int64
	stale := true
	var lastSync *time.Time
	if data.LastCompleted != nil && data.LastCompleted.FinishedAt != nil {
		seconds := int64(s.now().Sub(*data.LastCompleted.FinishedAt).Seconds())
		if seconds < 0 {
			seconds = 0
		}
		age = &seconds
		stale = time.Duration(seconds)*time.Second > s.limits.StaleAfter
		lastSync = data.LastCompleted.FinishedAt
	}
	normalized := NormalizedReportRequest{
		Kind: ReportDataFreshness, DataFreshness: &DataFreshnessRequest{},
	}
	return DataFreshnessResult{
		Metadata: reportMetadata(
			ReportDataFreshness, nil, "", AppliedFilters{}, lastSync, normalized,
		),
		LastCompleted: data.LastCompleted, LastAttempt: data.LastAttempt,
		AgeSeconds: age, Stale: stale, Table: freshnessTable(data),
	}, nil
}

func (s *Service) NormalizeReportRequest(request ReportRequest) (NormalizedReportRequest, error) {
	if reportPayloadCount(request) != 1 {
		return NormalizedReportRequest{}, errors.New("report request must contain exactly one payload")
	}
	switch request.Kind {
	case ReportSpendingSummary:
		if request.SpendingSummary == nil {
			break
		}
		canonical, _, _, err := s.normalizeSpendingRequest(*request.SpendingSummary)
		return NormalizedReportRequest{Kind: request.Kind, SpendingSummary: &canonical}, err
	case ReportCashflow:
		if request.Cashflow == nil {
			break
		}
		canonical, _, _, err := s.normalizeCashflowRequest(*request.Cashflow)
		return NormalizedReportRequest{Kind: request.Kind, Cashflow: &canonical}, err
	case ReportBudgetProgress:
		if request.BudgetProgress == nil {
			break
		}
		canonical, _, _, err := s.normalizeBudgetRequest(*request.BudgetProgress)
		return NormalizedReportRequest{Kind: request.Kind, BudgetProgress: &canonical}, err
	case ReportTransactions:
		if request.Transactions == nil {
			break
		}
		canonical, _, _, err := s.normalizeSearchRequest(*request.Transactions)
		return NormalizedReportRequest{Kind: request.Kind, Transactions: &canonical}, err
	case ReportDataFreshness:
		if request.DataFreshness != nil {
			return NormalizedReportRequest{
				Kind: request.Kind, DataFreshness: &DataFreshnessRequest{},
			}, nil
		}
	default:
		return NormalizedReportRequest{}, fmt.Errorf("unsupported report kind %q", request.Kind)
	}
	return NormalizedReportRequest{}, fmt.Errorf("report kind %q does not match its payload", request.Kind)
}

func (s *Service) ExecuteNormalizedReport(
	ctx context.Context,
	principal Principal,
	request NormalizedReportRequest,
) (ReportEnvelope, error) {
	switch request.Kind {
	case ReportSpendingSummary:
		if request.SpendingSummary != nil {
			result, err := s.GetSpendingSummary(ctx, principal, *request.SpendingSummary)
			return ReportEnvelope{Kind: request.Kind, SpendingSummary: &result}, err
		}
	case ReportCashflow:
		if request.Cashflow != nil {
			result, err := s.GetCashflow(ctx, principal, *request.Cashflow)
			return ReportEnvelope{Kind: request.Kind, Cashflow: &result}, err
		}
	case ReportBudgetProgress:
		if request.BudgetProgress != nil {
			result, err := s.GetBudgetProgress(ctx, principal, *request.BudgetProgress)
			return ReportEnvelope{Kind: request.Kind, BudgetProgress: &result}, err
		}
	case ReportTransactions:
		if request.Transactions != nil {
			result, err := s.SearchTransactions(ctx, principal, *request.Transactions)
			return ReportEnvelope{Kind: request.Kind, Transactions: &result}, err
		}
	case ReportDataFreshness:
		if request.DataFreshness != nil {
			result, err := s.GetDataFreshness(ctx, principal, *request.DataFreshness)
			return ReportEnvelope{Kind: request.Kind, DataFreshness: &result}, err
		}
	}
	return ReportEnvelope{}, errors.New("normalized report request is incomplete")
}

func (s *Service) normalizeSpendingRequest(
	request SpendingSummaryRequest,
) (SpendingSummaryRequest, SpendingSummaryQuery, Period, error) {
	rangeValue, period, canonicalPeriod, err := s.normalizePeriod(request.Period)
	if err != nil {
		return SpendingSummaryRequest{}, SpendingSummaryQuery{}, Period{}, err
	}
	filters, err := s.normalizeFilters(request.Filters)
	if err != nil {
		return SpendingSummaryRequest{}, SpendingSummaryQuery{}, Period{}, err
	}
	limit, err := s.normalizePageSize(request.Limit)
	if err != nil {
		return SpendingSummaryRequest{}, SpendingSummaryQuery{}, Period{}, err
	}
	canonical := SpendingSummaryRequest{
		Period: canonicalPeriod, Filters: filtersToRequest(filters), Limit: limit,
	}
	return canonical, SpendingSummaryQuery{
		Range: rangeValue, Filters: filters, Limit: limit,
	}, period, nil
}

func (s *Service) normalizeCashflowRequest(
	request CashflowRequest,
) (CashflowRequest, CashflowQuery, Period, error) {
	rangeValue, period, canonicalPeriod, err := s.normalizePeriod(request.Period)
	if err != nil {
		return CashflowRequest{}, CashflowQuery{}, Period{}, err
	}
	filters, err := s.normalizeFilters(request.Filters)
	if err != nil {
		return CashflowRequest{}, CashflowQuery{}, Period{}, err
	}
	granularity := request.Granularity
	if granularity == "" {
		granularity = chooseGranularity(rangeValue, s.limits.MaxChartPoints)
	}
	if !validGranularity(granularity) {
		return CashflowRequest{}, CashflowQuery{}, Period{}, fmt.Errorf(
			"unsupported granularity %q", granularity,
		)
	}
	if estimatePoints(rangeValue, granularity) > s.limits.MaxChartPoints {
		return CashflowRequest{}, CashflowQuery{}, Period{}, fmt.Errorf(
			"granularity %q exceeds the maximum of %d chart points",
			granularity,
			s.limits.MaxChartPoints,
		)
	}
	canonical := CashflowRequest{
		Period: canonicalPeriod, Filters: filtersToRequest(filters),
		Granularity: granularity,
	}
	return canonical, CashflowQuery{
		Range: rangeValue, Filters: filters,
		Granularity: granularity, MaxPoints: s.limits.MaxChartPoints,
	}, period, nil
}

func (s *Service) normalizeBudgetRequest(
	request BudgetProgressRequest,
) (BudgetProgressRequest, BudgetProgressQuery, Period, error) {
	rangeValue, period, canonicalPeriod, err := s.normalizePeriod(request.Period)
	if err != nil {
		return BudgetProgressRequest{}, BudgetProgressQuery{}, Period{}, err
	}
	filters, err := s.normalizeFilters(request.Filters)
	if err != nil {
		return BudgetProgressRequest{}, BudgetProgressQuery{}, Period{}, err
	}
	limit, err := s.normalizePageSize(request.Limit)
	if err != nil {
		return BudgetProgressRequest{}, BudgetProgressQuery{}, Period{}, err
	}
	canonical := BudgetProgressRequest{
		Period: canonicalPeriod, Filters: filtersToRequest(filters), Limit: limit,
	}
	return canonical, BudgetProgressQuery{
		Range: rangeValue, Filters: filters, Limit: limit,
	}, period, nil
}

func (s *Service) normalizeSearchRequest(
	request TransactionSearchRequest,
) (TransactionSearchRequest, TransactionSearchQuery, Period, error) {
	rangeValue, period, canonicalPeriod, err := s.normalizePeriod(request.Period)
	if err != nil {
		return TransactionSearchRequest{}, TransactionSearchQuery{}, Period{}, err
	}
	filters, err := s.normalizeFilters(request.Filters)
	if err != nil {
		return TransactionSearchRequest{}, TransactionSearchQuery{}, Period{}, err
	}
	text, err := normalizeSearchText(request.Text)
	if err != nil {
		return TransactionSearchRequest{}, TransactionSearchQuery{}, Period{}, err
	}
	pageSize, err := s.normalizePageSize(request.PageSize)
	if err != nil {
		return TransactionSearchRequest{}, TransactionSearchQuery{}, Period{}, err
	}
	cursor, err := decodeCursor(request.Cursor)
	if err != nil {
		return TransactionSearchRequest{}, TransactionSearchQuery{}, Period{}, err
	}
	canonicalCursor, err := encodeCursor(cursor)
	if err != nil {
		return TransactionSearchRequest{}, TransactionSearchQuery{}, Period{}, err
	}
	canonical := TransactionSearchRequest{
		Period: canonicalPeriod, Filters: filtersToRequest(filters),
		Text: text, Cursor: canonicalCursor, PageSize: pageSize,
	}
	return canonical, TransactionSearchQuery{
		Range: rangeValue, Filters: filters, Text: text,
		Cursor: cursor, Limit: pageSize,
	}, period, nil
}

func (s *Service) normalizePeriod(
	request PeriodRequest,
) (DateRange, Period, PeriodRequest, error) {
	timezone := strings.TrimSpace(request.Timezone)
	if timezone == "" {
		timezone = "UTC"
	}
	location, err := time.LoadLocation(timezone)
	if err != nil {
		return DateRange{}, Period{}, PeriodRequest{}, fmt.Errorf("invalid timezone %q", timezone)
	}
	from, err := time.ParseInLocation(dateLayout, request.From, location)
	if err != nil || from.Format(dateLayout) != request.From {
		return DateRange{}, Period{}, PeriodRequest{}, errors.New("period.from must be YYYY-MM-DD")
	}
	toInclusive, err := time.ParseInLocation(dateLayout, request.To, location)
	if err != nil || toInclusive.Format(dateLayout) != request.To {
		return DateRange{}, Period{}, PeriodRequest{}, errors.New("period.to must be YYYY-MM-DD")
	}
	if toInclusive.Before(from) {
		return DateRange{}, Period{}, PeriodRequest{}, errors.New("period.to must not precede period.from")
	}
	days := int(toInclusive.Sub(from).Hours()/24) + 1
	if days > s.limits.MaxPeriodDays {
		return DateRange{}, Period{}, PeriodRequest{}, fmt.Errorf(
			"report period exceeds the maximum of %d days", s.limits.MaxPeriodDays,
		)
	}
	toExclusive := toInclusive.AddDate(0, 0, 1)
	period := Period{
		From: request.From, To: request.To, FromInclusive: true, ToInclusive: true,
		Timezone: timezone,
	}
	canonical := PeriodRequest{From: request.From, To: request.To, Timezone: timezone}
	return DateRange{From: from, To: toExclusive, Timezone: timezone}, period, canonical, nil
}

func (s *Service) normalizeFilters(filters Filters) (AppliedFilters, error) {
	accounts, err := normalizeIDs("account", filters.AccountIDs, s.limits.MaxFilterValues)
	if err != nil {
		return AppliedFilters{}, err
	}
	categories, err := normalizeIDs("category", filters.CategoryIDs, s.limits.MaxFilterValues)
	if err != nil {
		return AppliedFilters{}, err
	}
	merchants, err := normalizeIDs("merchant", filters.MerchantIDs, s.limits.MaxFilterValues)
	if err != nil {
		return AppliedFilters{}, err
	}
	return AppliedFilters{
		AccountIDs: accounts, CategoryIDs: categories, MerchantIDs: merchants,
		IncludeHold: filters.IncludeHold,
	}, nil
}

func (s *Service) normalizePageSize(value int) (int, error) {
	if value == 0 {
		return s.limits.DefaultPageSize, nil
	}
	if value < 1 || value > s.limits.MaxPageSize {
		return 0, fmt.Errorf("page size must be between 1 and %d", s.limits.MaxPageSize)
	}
	return value, nil
}

func normalizeLimits(limits Limits) Limits {
	defaults := DefaultLimits()
	if limits.MaxPeriodDays <= 0 {
		limits.MaxPeriodDays = defaults.MaxPeriodDays
	}
	if limits.DefaultPageSize <= 0 {
		limits.DefaultPageSize = defaults.DefaultPageSize
	}
	if limits.MaxPageSize <= 0 {
		limits.MaxPageSize = defaults.MaxPageSize
	}
	if limits.DefaultPageSize > limits.MaxPageSize {
		limits.DefaultPageSize = limits.MaxPageSize
	}
	if limits.MaxChartPoints <= 0 {
		limits.MaxChartPoints = defaults.MaxChartPoints
	}
	if limits.MaxFilterValues <= 0 {
		limits.MaxFilterValues = defaults.MaxFilterValues
	}
	if limits.StaleAfter <= 0 {
		limits.StaleAfter = defaults.StaleAfter
	}
	return limits
}

func normalizePrincipal(principal Principal) (Principal, error) {
	principal.Subject = strings.TrimSpace(principal.Subject)
	if principal.Subject == "" {
		return Principal{}, errors.New("authenticated principal subject is required")
	}
	seen := make(map[int64]struct{}, len(principal.UserIDs))
	users := make([]int64, 0, len(principal.UserIDs))
	for _, userID := range principal.UserIDs {
		if userID <= 0 {
			return Principal{}, errors.New("authenticated principal contains an invalid user ID")
		}
		if _, exists := seen[userID]; exists {
			continue
		}
		seen[userID] = struct{}{}
		users = append(users, userID)
	}
	if len(users) == 0 {
		return Principal{}, errors.New("authenticated principal has no authorized users")
	}
	sort.Slice(users, func(i, j int) bool { return users[i] < users[j] })
	principal.UserIDs = users
	return principal, nil
}

var (
	currencyPattern = regexp.MustCompile(`^[A-Z0-9]{2,12}$`)
	uuidPattern     = regexp.MustCompile(
		`^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`,
	)
	decimalPattern = regexp.MustCompile(`^-?(0|[1-9][0-9]*)(\.[0-9]+)?$`)
)

func normalizeCurrency(currency string) (string, error) {
	currency = strings.ToUpper(strings.TrimSpace(currency))
	if !currencyPattern.MatchString(currency) {
		return "", errors.New("currency must be a 2-12 character instrument code")
	}
	return currency, nil
}

func normalizeIDs(name string, values []string, maxValues int) ([]string, error) {
	if len(values) > maxValues {
		return nil, fmt.Errorf("%s filters exceed the maximum of %d", name, maxValues)
	}
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.ToLower(strings.TrimSpace(value))
		if !uuidPattern.MatchString(value) {
			return nil, fmt.Errorf("invalid %s ID %q", name, value)
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	sort.Strings(result)
	return result, nil
}

func normalizeSearchText(value string) (string, error) {
	value = strings.TrimSpace(value)
	if !utf8.ValidString(value) || utf8.RuneCountInString(value) > 200 {
		return "", errors.New("search text must be valid UTF-8 and at most 200 characters")
	}
	for _, r := range value {
		if unicode.IsControl(r) {
			return "", errors.New("search text contains control characters")
		}
	}
	return value, nil
}

type cursorPayload struct {
	Version int    `json:"v"`
	Date    string `json:"date"`
	Created int64  `json:"created"`
	ID      string `json:"id"`
}

func encodeCursor(cursor *TransactionCursor) (string, error) {
	if cursor == nil {
		return "", nil
	}
	payload, err := json.Marshal(cursorPayload{
		Version: 1, Date: cursor.Date.Format(dateLayout), Created: cursor.Created,
		ID: strings.ToLower(cursor.ID),
	})
	if err != nil {
		return "", fmt.Errorf("encode transaction cursor: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(payload), nil
}

func decodeCursor(value string) (*TransactionCursor, error) {
	if value == "" {
		return nil, nil
	}
	if len(value) > 512 {
		return nil, errors.New("transaction cursor is too long")
	}
	payload, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return nil, errors.New("invalid transaction cursor")
	}
	var decoded cursorPayload
	decoder := json.NewDecoder(strings.NewReader(string(payload)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&decoded); err != nil || decoded.Version != 1 {
		return nil, errors.New("invalid transaction cursor")
	}
	date, err := time.Parse(dateLayout, decoded.Date)
	if err != nil || !uuidPattern.MatchString(strings.ToLower(decoded.ID)) || decoded.Created < 0 {
		return nil, errors.New("invalid transaction cursor")
	}
	return &TransactionCursor{Date: date, Created: decoded.Created, ID: strings.ToLower(decoded.ID)}, nil
}

func reportPayloadCount(request ReportRequest) int {
	count := 0
	for _, present := range []bool{
		request.SpendingSummary != nil,
		request.Cashflow != nil,
		request.BudgetProgress != nil,
		request.Transactions != nil,
		request.DataFreshness != nil,
	} {
		if present {
			count++
		}
	}
	return count
}

func reportMetadata(
	kind ReportKind,
	period *Period,
	currency string,
	filters AppliedFilters,
	lastSyncAt *time.Time,
	normalized NormalizedReportRequest,
) ReportMetadata {
	return ReportMetadata{
		SchemaVersion: SchemaVersion, ReportKind: kind, Period: period, Currency: currency,
		Filters: filters, Rules: calculationRules(filters.IncludeHold), LastSyncAt: lastSyncAt,
		NormalizedRequest: &normalized,
	}
}

func calculationRules(includeHold bool) CalculationRules {
	hold := "excluded because pending amounts are not authoritative"
	if includeHold {
		hold = "included only because the normalized request explicitly opted in"
	}
	return CalculationRules{
		DeletedTransactions: "always excluded; hard deletions are absent from the source table",
		HoldTransactions:    hold,
		Transfers:           "transfers between own accounts are excluded from spending and cashflow totals",
		MixedFlows:          "same-account income and outcome are separate legs; different-account flows are transfers",
		Currencies:          "amount * current source instrument RUB rate / current report instrument RUB rate",
		MultiTag:            "a transaction is counted once; its amount is split equally across distinct canonical root categories",
		CategoryHierarchy:   "a child category rolls up to its valid parent; otherwise the child is its own root",
		Accounts:            "only accounts included in balance participate in aggregate reports",
		Merchants:           "missing or blank merchants use the stable unknown-merchant bucket",
		Dates:               "public from and to dates are inclusive; store queries use an equivalent half-open range",
		Rounding:            "PostgreSQL NUMERIC values are not rounded until client display formatting",
		Limitations: []string{
			"instrument rates are current snapshots, not historical exchange rates",
			"the database stores calendar dates but no original transaction timezone",
			"operational op_income and op_outcome fields are not used for report valuation",
		},
	}
}

func filtersToRequest(filters AppliedFilters) Filters {
	return Filters{
		AccountIDs: filters.AccountIDs, CategoryIDs: filters.CategoryIDs,
		MerchantIDs: filters.MerchantIDs, IncludeHold: filters.IncludeHold,
	}
}

func chooseGranularity(dateRange DateRange, maxPoints int) Granularity {
	days := int(dateRange.To.Sub(dateRange.From).Hours() / 24)
	if days <= maxPoints {
		return GranularityDay
	}
	if (days+6)/7 <= maxPoints {
		return GranularityWeek
	}
	return GranularityMonth
}

func estimatePoints(dateRange DateRange, granularity Granularity) int {
	days := int(dateRange.To.Sub(dateRange.From).Hours() / 24)
	switch granularity {
	case GranularityDay:
		return days
	case GranularityWeek:
		return (days + 6) / 7
	default:
		return (days + 27) / 28
	}
}

func validDecimal(value Decimal) bool { return decimalPattern.MatchString(string(value)) }

func validateSpendingData(data SpendingSummaryData) error {
	if err := validateStoreCurrency(data.Currency); err != nil {
		return err
	}
	if !validDecimal(data.Total) {
		return errors.New("total is not a decimal string")
	}
	for _, row := range data.Categories {
		if row.ID == "" || row.CategoryID == "" || !validDecimal(row.Amount) ||
			!validDecimal(row.SharePercent) {
			return fmt.Errorf("invalid spending row %q", row.ID)
		}
	}
	return nil
}

func validateCashflowData(data CashflowData) error {
	if err := validateStoreCurrency(data.Currency); err != nil {
		return err
	}
	for _, value := range []Decimal{data.Totals.Income, data.Totals.Outcome, data.Totals.Net} {
		if !validDecimal(value) {
			return errors.New("cashflow total is not a decimal string")
		}
	}
	for _, point := range data.Points {
		if point.ID == "" || !validDecimal(point.Income) || !validDecimal(point.Outcome) ||
			!validDecimal(point.Net) {
			return fmt.Errorf("invalid cashflow point %q", point.ID)
		}
	}
	return nil
}

func validateBudgetData(data BudgetProgressData) error {
	if err := validateStoreCurrency(data.Currency); err != nil {
		return err
	}
	for _, value := range []Decimal{data.Totals.Budget, data.Totals.Spent, data.Totals.Remaining} {
		if !validDecimal(value) {
			return errors.New("budget total is not a decimal string")
		}
	}
	if data.Totals.Percent != nil && !validDecimal(*data.Totals.Percent) {
		return errors.New("budget percentage is not a decimal string")
	}
	for _, row := range data.Rows {
		if row.ID == "" || row.CategoryID == "" || !validDecimal(row.Budget) ||
			!validDecimal(row.Spent) || !validDecimal(row.Remaining) ||
			(row.Percent != nil && !validDecimal(*row.Percent)) {
			return fmt.Errorf("invalid budget row %q", row.ID)
		}
	}
	return nil
}

func validateTransactionPage(page TransactionPage) error {
	if err := validateStoreCurrency(page.Currency); err != nil {
		return err
	}
	for _, item := range page.Items {
		if item.ID == "" || !validDirection(item.Direction) || !validDecimal(item.Amount) ||
			!validDecimal(item.Income) || !validDecimal(item.Outcome) {
			return fmt.Errorf("invalid transaction item %q", item.ID)
		}
	}
	return nil
}

func validDirection(direction TransactionDirection) bool {
	switch direction {
	case DirectionIncome, DirectionOutcome, DirectionTransfer, DirectionDebtIncome,
		DirectionDebtOutcome, DirectionMixed, DirectionInvalid:
		return true
	default:
		return false
	}
}

func validateStoreCurrency(currency string) error {
	normalized, err := normalizeCurrency(currency)
	if err != nil || normalized != currency {
		return errors.New("store returned an invalid report currency")
	}
	return nil
}

func nonNilSpendingRows(rows []SpendingCategory) []SpendingCategory {
	if rows == nil {
		return []SpendingCategory{}
	}
	return rows
}

func nonNilCashflowPoints(rows []CashflowPoint) []CashflowPoint {
	if rows == nil {
		return []CashflowPoint{}
	}
	return rows
}

func nonNilBudgetRows(rows []BudgetProgressRow) []BudgetProgressRow {
	if rows == nil {
		return []BudgetProgressRow{}
	}
	return rows
}

func nonNilTransactions(rows []TransactionItem) []TransactionItem {
	if rows == nil {
		return []TransactionItem{}
	}
	return rows
}
