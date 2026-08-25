package analytics

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"sort"
	"strings"
)

func (s *Service) NormalizeRenderRequest(
	request RenderFinanceChartRequest,
) (NormalizedRenderRequest, error) {
	reportRequest := request.Report
	if reportRequest.Kind == ReportCashflow && reportRequest.Cashflow != nil &&
		reportRequest.Cashflow.Granularity == "" && request.Chart.Granularity != "" {
		cashflow := *reportRequest.Cashflow
		cashflow.Granularity = request.Chart.Granularity
		reportRequest.Cashflow = &cashflow
	}
	report, err := s.NormalizeReportRequest(reportRequest)
	if err != nil {
		return NormalizedRenderRequest{}, err
	}
	chart, err := s.normalizeChartSpec(report, request.Chart)
	if err != nil {
		return NormalizedRenderRequest{}, err
	}
	if err := ValidateChartSpec(chart, s.limits.MaxChartPoints); err != nil {
		return NormalizedRenderRequest{}, err
	}
	if err := validateChartForReport(report.Kind, chart); err != nil {
		return NormalizedRenderRequest{}, err
	}
	normalized := NormalizedRenderRequest{Report: report, Chart: chart}
	if chart.ComparisonPeriods {
		previous, err := s.previousCashflowReport(report)
		if err != nil {
			return NormalizedRenderRequest{}, err
		}
		normalized.PreviousReport = &previous
	}
	return normalized, nil
}

func (s *Service) normalizeChartSpec(
	report NormalizedReportRequest,
	chart ChartSpec,
) (ChartSpec, error) {
	chart = normalizeChartDefaults(chart)
	if report.Kind != ReportCashflow {
		return validateNonCashflowChart(chart)
	}
	return normalizeCashflowChart(report, chart)
}

func normalizeChartDefaults(chart ChartSpec) ChartSpec {
	chart.Title = strings.TrimSpace(chart.Title)
	chart.Subtitle = strings.TrimSpace(chart.Subtitle)
	chart.Other.Label = strings.TrimSpace(chart.Other.Label)
	chart.Table.Caption = strings.TrimSpace(chart.Table.Caption)
	for index := range chart.Series {
		chart.Series[index].Label = strings.TrimSpace(chart.Series[index].Label)
	}
	if chart.Stacking == "" {
		chart.Stacking = StackingNone
		if chart.Type == ChartStackedBar {
			chart.Stacking = StackingNormal
		}
	}
	if chart.Sort.By == "" {
		chart.Sort.By = SortDimension
	}
	if chart.Sort.Direction == "" {
		chart.Sort.Direction = SortAscending
	}
	if chart.Sort.By == SortValue && chart.Sort.Series == "" && len(chart.Series) > 0 {
		chart.Sort.Series = chart.Series[0].Key
	}
	if chart.Other.Label == "" {
		chart.Other.Label = "Other"
	}
	chart.Table.Enabled = true
	if chart.Table.Caption == "" {
		chart.Table.Caption = chart.Title + " data"
	}
	return chart
}

func validateNonCashflowChart(chart ChartSpec) (ChartSpec, error) {
	if chart.Granularity != "" {
		return ChartSpec{}, errors.New(
			"chart granularity is only valid for cashflow period reports",
		)
	}
	if chart.ComparisonPeriods {
		return ChartSpec{}, errors.New(
			"comparison periods are only valid for cashflow period reports",
		)
	}
	return chart, nil
}

func normalizeCashflowChart(
	report NormalizedReportRequest,
	chart ChartSpec,
) (ChartSpec, error) {
	if report.Cashflow == nil {
		return ChartSpec{}, errors.New("normalized cashflow report is missing its payload")
	}
	if hasServerDerivedSeries(chart.Series) {
		return ChartSpec{}, errors.New("current_net and previous_net series are server-derived")
	}
	reportGranularity := report.Cashflow.Granularity
	if chart.Granularity != "" && chart.Granularity != reportGranularity {
		return ChartSpec{}, fmt.Errorf(
			"chart granularity %q does not match cashflow granularity %q",
			chart.Granularity,
			reportGranularity,
		)
	}
	chart.Granularity = reportGranularity
	if !chart.ComparisonPeriods {
		return chart, nil
	}
	if err := validateComparisonChart(chart, reportGranularity); err != nil {
		return ChartSpec{}, err
	}
	format := chart.Series[0].Format
	chart.Series = []ChartSeries{
		{Key: SeriesCurrentNet, Label: "Current net", Format: format},
		{Key: SeriesPreviousNet, Label: "Previous net", Format: format},
	}
	return chart, nil
}

func hasServerDerivedSeries(seriesList []ChartSeries) bool {
	for _, series := range seriesList {
		if series.Key == SeriesCurrentNet || series.Key == SeriesPreviousNet {
			return true
		}
	}
	return false
}

func validateComparisonChart(chart ChartSpec, reportGranularity Granularity) error {
	if chart.TopN != 0 || chart.Other.Enabled {
		return errors.New("comparison periods cannot use topN or an Other bucket")
	}
	if chart.Sort.By != SortDimension || chart.Sort.Direction != SortAscending {
		return errors.New("comparison periods require ascending chronological sorting")
	}
	if chart.Type != ChartLine {
		return errors.New("comparison periods currently support only line charts")
	}
	if reportGranularity != GranularityDay {
		return errors.New(
			"comparison periods currently support only daily granularity",
		)
	}
	if len(chart.Series) != 1 || chart.Series[0].Key != SeriesNet {
		return errors.New(
			"comparison periods currently require exactly the net series",
		)
	}
	return nil
}

func (s *Service) previousCashflowReport(
	current NormalizedReportRequest,
) (NormalizedReportRequest, error) {
	if current.Kind != ReportCashflow || current.Cashflow == nil {
		return NormalizedReportRequest{}, errors.New("comparison periods require a cashflow report")
	}
	dateRange, _, _, err := s.normalizePeriod(current.Cashflow.Period)
	if err != nil {
		return NormalizedReportRequest{}, fmt.Errorf("normalize comparison period: %w", err)
	}
	days := calendarDayCount(dateRange)
	previousTo := dateRange.From.AddDate(0, 0, -1)
	previousFrom := dateRange.From
	for range days {
		previousFrom = previousFrom.AddDate(0, 0, -1)
	}
	request := CashflowRequest{
		Period: PeriodRequest{
			From:     previousFrom.Format(dateLayout),
			To:       previousTo.Format(dateLayout),
			Timezone: current.Cashflow.Period.Timezone,
		},
		Filters:     current.Cashflow.Filters,
		Granularity: current.Cashflow.Granularity,
		Users:       current.Cashflow.Users,
	}
	canonical, _, _, err := s.normalizeCashflowRequest(request)
	if err != nil {
		return NormalizedReportRequest{}, fmt.Errorf(
			"normalize previous comparison period: %w",
			err,
		)
	}
	return NormalizedReportRequest{Kind: ReportCashflow, Cashflow: &canonical}, nil
}

func calendarDayCount(dateRange DateRange) int {
	days := 0
	for day := dateRange.From; day.Before(dateRange.To); day = day.AddDate(0, 0, 1) {
		days++
	}
	return days
}

func (s *Service) RenderFinanceChart(
	ctx context.Context,
	principal Principal,
	request RenderFinanceChartRequest,
) (RenderFinanceChartResult, error) {
	normalized, err := s.NormalizeRenderRequest(request)
	if err != nil {
		return RenderFinanceChartResult{}, err
	}
	report, err := s.ExecuteNormalizedReport(ctx, principal, normalized.Report)
	if err != nil {
		return RenderFinanceChartResult{}, err
	}
	if normalized.PreviousReport != nil {
		previous, err := s.ExecuteNormalizedReport(ctx, principal, *normalized.PreviousReport)
		if err != nil {
			return RenderFinanceChartResult{}, err
		}
		points, table, comparison, err := s.comparisonChartData(normalized, report, previous)
		if err != nil {
			return RenderFinanceChartResult{}, err
		}
		points, err = applyPresentation(points, normalized.Chart)
		if err != nil {
			return RenderFinanceChartResult{}, err
		}
		if normalized.Chart.Table.Caption != "" {
			table.Caption = normalized.Chart.Table.Caption
		}
		return RenderFinanceChartResult{
			SchemaVersion: SchemaVersion, ReportKind: normalized.Report.Kind,
			Chart: normalized.Chart, Data: points, Table: table, Report: report,
			PreviousReport: &previous, Comparison: &comparison,
		}, nil
	}
	points, table, err := chartData(report)
	if err != nil {
		return RenderFinanceChartResult{}, err
	}
	points, err = applyPresentation(points, normalized.Chart)
	if err != nil {
		return RenderFinanceChartResult{}, err
	}
	if normalized.Chart.Table.Caption != "" {
		table.Caption = normalized.Chart.Table.Caption
	}
	return RenderFinanceChartResult{
		SchemaVersion: SchemaVersion, ReportKind: normalized.Report.Kind,
		Chart: normalized.Chart, Data: points, Table: table, Report: report,
	}, nil
}

func (s *Service) comparisonChartData(
	normalized NormalizedRenderRequest,
	current ReportEnvelope,
	previous ReportEnvelope,
) ([]ChartDataPoint, TableFallback, ChartComparisonMetadata, error) {
	if normalized.Report.Cashflow == nil || normalized.PreviousReport == nil ||
		normalized.PreviousReport.Cashflow == nil || current.Cashflow == nil || previous.Cashflow == nil {
		return nil, TableFallback{}, ChartComparisonMetadata{}, errors.New(
			"comparison result does not contain both cashflow reports",
		)
	}
	if current.Cashflow.Metadata.Period == nil || previous.Cashflow.Metadata.Period == nil {
		return nil, TableFallback{}, ChartComparisonMetadata{}, errors.New(
			"comparison result is missing period metadata",
		)
	}
	currentRange, _, _, err := s.normalizePeriod(normalized.Report.Cashflow.Period)
	if err != nil {
		return nil, TableFallback{}, ChartComparisonMetadata{}, err
	}
	previousRange, _, _, err := s.normalizePeriod(normalized.PreviousReport.Cashflow.Period)
	if err != nil {
		return nil, TableFallback{}, ChartComparisonMetadata{}, err
	}
	currentValues := cashflowNetByDate(current.Cashflow.Points)
	previousValues := cashflowNetByDate(previous.Cashflow.Points)
	bucketCount := calendarDayCount(currentRange)
	points := make([]ChartDataPoint, 0, bucketCount)
	rows := make([]TableRow, 0, bucketCount)
	currentDay, previousDay := currentRange.From, previousRange.From
	for index := range bucketCount {
		currentDate := currentDay.Format(dateLayout)
		previousDate := previousDay.Format(dateLayout)
		currentNet := currentValues[currentDate]
		previousNet := previousValues[previousDate]
		if currentNet == "" {
			currentNet = "0"
		}
		if previousNet == "" {
			previousNet = "0"
		}
		id := fmt.Sprintf("comparison:bucket:%04d", index)
		points = append(points, ChartDataPoint{
			ID: id, Label: currentDate + " vs " + previousDate,
			Values: []ChartValue{
				{Series: SeriesCurrentNet, Value: currentNet},
				{Series: SeriesPreviousNet, Value: previousNet},
			},
		})
		rows = append(rows, TableRow{ID: id, Cells: []TableCell{
			{Key: "bucket", Value: fmt.Sprintf("%d", index)},
			{Key: "current_period", Value: currentDate},
			{Key: "current_net", Value: string(currentNet)},
			{Key: "previous_period", Value: previousDate},
			{Key: "previous_net", Value: string(previousNet)},
		}})
		currentDay = currentDay.AddDate(0, 0, 1)
		previousDay = previousDay.AddDate(0, 0, 1)
	}
	table := TableFallback{Columns: []TableColumn{
		{Key: "bucket", Label: "Bucket index", Format: FormatNumber},
		{Key: "current_period", Label: "Current period", Format: FormatDate},
		{Key: "current_net", Label: "Current net", Format: FormatCurrency},
		{Key: "previous_period", Label: "Previous period", Format: FormatDate},
		{Key: "previous_net", Label: "Previous net", Format: FormatCurrency},
	}, Rows: rows}
	metadata := ChartComparisonMetadata{
		CurrentPeriod:  *current.Cashflow.Metadata.Period,
		PreviousPeriod: *previous.Cashflow.Metadata.Period,
		Granularity:    GranularityDay,
		Alignment:      "calendar_day_index",
		BucketCount:    bucketCount,
	}
	return points, table, metadata, nil
}

func cashflowNetByDate(points []CashflowPoint) map[string]Decimal {
	values := make(map[string]Decimal, len(points))
	for _, point := range points {
		values[point.From] = point.Net
	}
	return values
}

func chartData(report ReportEnvelope) ([]ChartDataPoint, TableFallback, error) {
	switch report.Kind {
	case ReportSpendingSummary:
		if report.SpendingSummary == nil {
			break
		}
		points := make([]ChartDataPoint, 0, len(report.SpendingSummary.Categories))
		for _, row := range report.SpendingSummary.Categories {
			points = append(
				points,
				ChartDataPoint{ID: row.ID, Label: row.Title, Values: []ChartValue{
					{Series: SeriesAmount, Value: row.Amount},
				}},
			)
		}
		return points, report.SpendingSummary.Table, nil
	case ReportCashflow:
		if report.Cashflow == nil {
			break
		}
		points := make([]ChartDataPoint, 0, len(report.Cashflow.Points))
		for _, row := range report.Cashflow.Points {
			points = append(
				points,
				ChartDataPoint{ID: row.ID, Label: row.Label, Values: []ChartValue{
					{Series: SeriesIncome, Value: row.Income},
					{Series: SeriesOutcome, Value: row.Outcome},
					{Series: SeriesNet, Value: row.Net},
				}},
			)
		}
		return points, report.Cashflow.Table, nil
	case ReportBudgetProgress:
		if report.BudgetProgress == nil {
			break
		}
		points := make([]ChartDataPoint, 0, len(report.BudgetProgress.Rows))
		for _, row := range report.BudgetProgress.Rows {
			percent := Decimal("0")
			if row.Percent != nil {
				percent = *row.Percent
			}
			points = append(
				points,
				ChartDataPoint{ID: row.ID, Label: row.Title, Values: []ChartValue{
					{Series: SeriesBudget, Value: row.Budget},
					{Series: SeriesSpent, Value: row.Spent},
					{Series: SeriesRemaining, Value: row.Remaining},
					{Series: SeriesPercent, Value: percent},
				}},
			)
		}
		return points, report.BudgetProgress.Table, nil
	case ReportTransactions:
		if report.Transactions == nil {
			break
		}
		points := make([]ChartDataPoint, 0, len(report.Transactions.Items))
		for _, row := range report.Transactions.Items {
			label := row.Date + " · " + row.MerchantTitle
			points = append(points, ChartDataPoint{ID: row.ID, Label: label, Values: []ChartValue{
				{Series: SeriesAmount, Value: row.Amount},
			}})
		}
		return points, report.Transactions.Table, nil
	}
	return nil, TableFallback{}, errors.New("report result does not contain renderable data")
}

func applyPresentation(points []ChartDataPoint, spec ChartSpec) ([]ChartDataPoint, error) {
	selected := make(map[ChartSeriesKey]bool, len(spec.Series))
	for _, series := range spec.Series {
		selected[series.Key] = true
	}
	result := make([]ChartDataPoint, 0, len(points))
	for _, point := range points {
		values := make([]ChartValue, 0, len(spec.Series))
		for _, value := range point.Values {
			if selected[value.Series] {
				values = append(values, value)
			}
		}
		result = append(result, ChartDataPoint{ID: point.ID, Label: point.Label, Values: values})
	}
	sort.SliceStable(result, func(i, j int) bool {
		comparison := strings.Compare(result[i].Label, result[j].Label)
		if spec.Sort.By == SortValue {
			comparison = compareDecimal(
				valueForSeries(result[i], spec.Sort.Series),
				valueForSeries(result[j], spec.Sort.Series),
			)
		}
		if spec.Sort.Direction == SortDescending {
			return comparison > 0
		}
		return comparison < 0
	})
	if spec.TopN == 0 || len(result) <= spec.TopN {
		return result, nil
	}
	remainder := result[spec.TopN:]
	result = result[:spec.TopN]
	if !spec.Other.Enabled {
		return result, nil
	}
	label := spec.Other.Label
	if label == "" {
		label = "Other"
	}
	other := ChartDataPoint{
		ID:     "presentation:other",
		Label:  label,
		Values: make([]ChartValue, 0, len(spec.Series)),
	}
	for _, series := range spec.Series {
		values := make([]Decimal, 0, len(remainder))
		for _, point := range remainder {
			values = append(values, valueForSeries(point, series.Key))
		}
		total, err := addDecimals(values)
		if err != nil {
			return nil, fmt.Errorf("aggregate Other bucket: %w", err)
		}
		other.Values = append(other.Values, ChartValue{Series: series.Key, Value: total})
	}
	return append(result, other), nil
}

func valueForSeries(point ChartDataPoint, series ChartSeriesKey) Decimal {
	for _, value := range point.Values {
		if value.Series == series {
			return value.Value
		}
	}
	return "0"
}

func compareDecimal(left, right Decimal) int {
	leftValue, leftOK := new(big.Rat).SetString(string(left))
	rightValue, rightOK := new(big.Rat).SetString(string(right))
	if !leftOK || !rightOK {
		return strings.Compare(string(left), string(right))
	}
	return leftValue.Cmp(rightValue)
}

func addDecimals(values []Decimal) (Decimal, error) {
	maxScale := 0
	total := new(big.Rat)
	for _, value := range values {
		text := string(value)
		if dot := strings.IndexByte(text, '.'); dot >= 0 && len(text)-dot-1 > maxScale {
			maxScale = len(text) - dot - 1
		}
		parsed, ok := new(big.Rat).SetString(text)
		if !ok {
			return "", fmt.Errorf("invalid decimal %q", value)
		}
		total.Add(total, parsed)
	}
	text := total.FloatString(maxScale)
	if strings.Contains(text, ".") {
		text = strings.TrimRight(strings.TrimRight(text, "0"), ".")
	}
	if text == "-0" || text == "" {
		text = "0"
	}
	return Decimal(text), nil
}
