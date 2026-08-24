package analytics

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

type ChartType string

const (
	ChartBar           ChartType = "bar"
	ChartHorizontalBar ChartType = "horizontal_bar"
	ChartLine          ChartType = "line"
	ChartArea          ChartType = "area"
	ChartDonut         ChartType = "donut"
	ChartStackedBar    ChartType = "stacked_bar"
	ChartGroupedBar    ChartType = "grouped_bar"
)

type ChartDimension string

const (
	DimensionCategory    ChartDimension = "category"
	DimensionPeriod      ChartDimension = "period"
	DimensionBudget      ChartDimension = "budget_category"
	DimensionTransaction ChartDimension = "transaction"
)

type ChartSeriesKey string

const (
	SeriesAmount    ChartSeriesKey = "amount"
	SeriesIncome    ChartSeriesKey = "income"
	SeriesOutcome   ChartSeriesKey = "outcome"
	SeriesNet       ChartSeriesKey = "net"
	SeriesBudget    ChartSeriesKey = "budget"
	SeriesSpent     ChartSeriesKey = "spent"
	SeriesRemaining ChartSeriesKey = "remaining"
	SeriesPercent   ChartSeriesKey = "percent"
)

type StackingMode string

const (
	StackingNone   StackingMode = "none"
	StackingNormal StackingMode = "normal"
)

type SortBy string

const (
	SortDimension SortBy = "dimension"
	SortValue     SortBy = "value"
)

type SortDirection string

const (
	SortAscending  SortDirection = "asc"
	SortDescending SortDirection = "desc"
)

type ChartSpec struct {
	Type              ChartType      `json:"type"`
	Title             string         `json:"title"`
	Subtitle          string         `json:"subtitle,omitempty"`
	Dimension         ChartDimension `json:"dimension"`
	Series            []ChartSeries  `json:"series"`
	Stacking          StackingMode   `json:"stacking,omitempty"`
	Sort              ChartSort      `json:"sort"`
	TopN              int            `json:"topN,omitempty"`
	Other             OtherBucket    `json:"other"`
	Legend            bool           `json:"legend"`
	Tooltip           bool           `json:"tooltip"`
	ShowNegative      bool           `json:"showNegative"`
	ComparisonPeriods bool           `json:"comparisonPeriods"`
	Granularity       Granularity    `json:"granularity,omitempty"`
	Table             ChartTableSpec `json:"table"`
}

type ChartSeries struct {
	Key    ChartSeriesKey `json:"key"`
	Label  string         `json:"label"`
	Format ValueFormat    `json:"format"`
}

type ChartSort struct {
	By        SortBy         `json:"by"`
	Direction SortDirection  `json:"direction"`
	Series    ChartSeriesKey `json:"series,omitempty"`
}

type OtherBucket struct {
	Enabled bool   `json:"enabled"`
	Label   string `json:"label,omitempty"`
}

type ChartTableSpec struct {
	Enabled bool   `json:"enabled"`
	Caption string `json:"caption,omitempty"`
}

type RenderFinanceChartRequest struct {
	Report ReportRequest `json:"report"`
	Chart  ChartSpec     `json:"chart"`
}

type NormalizedRenderRequest struct {
	Report NormalizedReportRequest `json:"report"`
	Chart  ChartSpec               `json:"chart"`
}

type ChartDataPoint struct {
	ID     string       `json:"id"`
	Label  string       `json:"label"`
	Values []ChartValue `json:"values"`
}

type ChartValue struct {
	Series ChartSeriesKey `json:"series"`
	Value  Decimal        `json:"value"`
}

type RenderFinanceChartResult struct {
	SchemaVersion string           `json:"schemaVersion"`
	ReportKind    ReportKind       `json:"reportKind"`
	Chart         ChartSpec        `json:"chart"`
	Data          []ChartDataPoint `json:"data"`
	Table         TableFallback    `json:"table"`
	Report        ReportEnvelope   `json:"report"`
}

var dangerousPresentationPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)<\s*/?\s*(script|iframe|object|embed|style|link|meta)\b`),
	regexp.MustCompile(`(?i)\b(javascript|vbscript|data\s*:\s*text/html)\s*:`),
	regexp.MustCompile(`(?i)\bhttps?\s*://|\bwww\.`),
	regexp.MustCompile(`(?i)\b(eval|function|settimeout|setinterval)\s*\(`),
	regexp.MustCompile(`(?i)\bon[a-z]+\s*=`),
	regexp.MustCompile(`(?i)\b(select|insert|update|delete|drop|alter|create|truncate|grant|revoke|union)\b`),
	regexp.MustCompile(`(?i)(--|/\*|\*/|=>)`),
}

func ValidateChartSpec(spec ChartSpec, maxPoints int) error {
	if !validChartType(spec.Type) {
		return fmt.Errorf("unsupported chart type %q", spec.Type)
	}
	if err := validatePresentationText("title", spec.Title, true); err != nil {
		return err
	}
	for name, value := range map[string]string{
		"subtitle":      spec.Subtitle,
		"other label":   spec.Other.Label,
		"table caption": spec.Table.Caption,
	} {
		if err := validatePresentationText(name, value, false); err != nil {
			return err
		}
	}
	if !validDimension(spec.Dimension) {
		return fmt.Errorf("unsupported chart dimension %q", spec.Dimension)
	}
	if len(spec.Series) == 0 || len(spec.Series) > 4 {
		return fmt.Errorf("chart series count must be between 1 and 4")
	}
	if spec.Type == ChartDonut && len(spec.Series) != 1 {
		return fmt.Errorf("donut requires exactly one series")
	}
	if (spec.Type == ChartLine || spec.Type == ChartArea) && spec.Dimension != DimensionPeriod {
		return fmt.Errorf("%s requires period dimension", spec.Type)
	}
	if (spec.Type == ChartStackedBar || spec.Type == ChartGroupedBar) && len(spec.Series) < 2 {
		return fmt.Errorf("%s requires at least two series", spec.Type)
	}
	seen := make(map[ChartSeriesKey]struct{}, len(spec.Series))
	for _, series := range spec.Series {
		if !validSeries(series.Key) {
			return fmt.Errorf("unsupported chart series %q", series.Key)
		}
		if _, exists := seen[series.Key]; exists {
			return fmt.Errorf("duplicate chart series %q", series.Key)
		}
		seen[series.Key] = struct{}{}
		if err := validatePresentationText("series label", series.Label, true); err != nil {
			return err
		}
		if !validValueFormat(series.Format) {
			return fmt.Errorf("unsupported series format %q", series.Format)
		}
	}
	if spec.Stacking == "" {
		spec.Stacking = StackingNone
	}
	if spec.Stacking != StackingNone && spec.Stacking != StackingNormal {
		return fmt.Errorf("unsupported stacking mode %q", spec.Stacking)
	}
	if spec.Type == ChartStackedBar && spec.Stacking != StackingNormal {
		return fmt.Errorf("stacked_bar requires normal stacking")
	}
	if spec.Type != ChartStackedBar && spec.Stacking == StackingNormal {
		return fmt.Errorf("normal stacking is only valid for stacked_bar")
	}
	if spec.Sort.By != SortDimension && spec.Sort.By != SortValue {
		return fmt.Errorf("unsupported sort field %q", spec.Sort.By)
	}
	if spec.Sort.Direction != SortAscending && spec.Sort.Direction != SortDescending {
		return fmt.Errorf("unsupported sort direction %q", spec.Sort.Direction)
	}
	if spec.Sort.By == SortValue {
		if _, ok := seen[spec.Sort.Series]; !ok {
			return fmt.Errorf("sort series %q is not present in chart series", spec.Sort.Series)
		}
	}
	if spec.TopN < 0 || (maxPoints > 0 && spec.TopN > maxPoints) {
		return fmt.Errorf("topN must be between 0 and %d", maxPoints)
	}
	if spec.Other.Enabled && spec.TopN == 0 {
		return fmt.Errorf("Other bucket requires topN")
	}
	if spec.ComparisonPeriods && spec.Dimension != DimensionPeriod {
		return fmt.Errorf("comparison periods require period dimension")
	}
	if spec.Granularity != "" && !validGranularity(spec.Granularity) {
		return fmt.Errorf("unsupported granularity %q", spec.Granularity)
	}
	if spec.Granularity != "" && spec.Dimension != DimensionPeriod {
		return fmt.Errorf("granularity requires period dimension")
	}
	if spec.ComparisonPeriods {
		if spec.Granularity == "" {
			return fmt.Errorf("comparison periods require explicit granularity")
		}
		if spec.TopN != 0 || spec.Other.Enabled {
			return fmt.Errorf("comparison periods cannot use topN or an Other bucket")
		}
		if spec.Sort.By != SortDimension || spec.Sort.Direction != SortAscending {
			return fmt.Errorf("comparison periods require ascending chronological sorting")
		}
	}
	return nil
}

func validateChartForReport(kind ReportKind, spec ChartSpec) error {
	expectedDimension := map[ReportKind]ChartDimension{
		ReportSpendingSummary: DimensionCategory,
		ReportCashflow:        DimensionPeriod,
		ReportBudgetProgress:  DimensionBudget,
		ReportTransactions:    DimensionTransaction,
	}[kind]
	if expectedDimension == "" {
		return fmt.Errorf("report %q cannot be rendered as a finance chart", kind)
	}
	if spec.Dimension != expectedDimension {
		return fmt.Errorf("report %q requires dimension %q", kind, expectedDimension)
	}
	allowed := map[ReportKind]map[ChartSeriesKey]bool{
		ReportSpendingSummary: {SeriesAmount: true},
		ReportCashflow:        {SeriesIncome: true, SeriesOutcome: true, SeriesNet: true},
		ReportBudgetProgress: {
			SeriesBudget: true, SeriesSpent: true, SeriesRemaining: true, SeriesPercent: true,
		},
		ReportTransactions: {SeriesAmount: true},
	}[kind]
	for _, series := range spec.Series {
		if !allowed[series.Key] {
			return fmt.Errorf("series %q is not available for report %q", series.Key, kind)
		}
	}
	return nil
}

func validatePresentationText(name, value string, required bool) error {
	value = strings.TrimSpace(value)
	if required && value == "" {
		return fmt.Errorf("%s is required", name)
	}
	if value == "" {
		return nil
	}
	if !utf8.ValidString(value) || utf8.RuneCountInString(value) > 160 {
		return fmt.Errorf("%s must be valid UTF-8 and at most 160 characters", name)
	}
	for _, r := range value {
		if unicode.IsControl(r) {
			return fmt.Errorf("%s contains control characters", name)
		}
	}
	if strings.ContainsAny(value, "<>;`") {
		return fmt.Errorf("%s contains unsafe markup or executable syntax", name)
	}
	for _, pattern := range dangerousPresentationPatterns {
		if pattern.MatchString(value) {
			return fmt.Errorf("%s contains unsafe markup or executable syntax", name)
		}
	}
	return nil
}

func validChartType(value ChartType) bool {
	switch value {
	case ChartBar, ChartHorizontalBar, ChartLine, ChartArea, ChartDonut, ChartStackedBar, ChartGroupedBar:
		return true
	default:
		return false
	}
}

func validDimension(value ChartDimension) bool {
	switch value {
	case DimensionCategory, DimensionPeriod, DimensionBudget, DimensionTransaction:
		return true
	default:
		return false
	}
}

func validSeries(value ChartSeriesKey) bool {
	switch value {
	case SeriesAmount, SeriesIncome, SeriesOutcome, SeriesNet, SeriesBudget, SeriesSpent,
		SeriesRemaining, SeriesPercent:
		return true
	default:
		return false
	}
}

func validValueFormat(value ValueFormat) bool {
	switch value {
	case FormatText, FormatDate, FormatCurrency, FormatNumber, FormatPercent:
		return true
	default:
		return false
	}
}

func validGranularity(value Granularity) bool {
	return value == GranularityDay || value == GranularityWeek || value == GranularityMonth
}
