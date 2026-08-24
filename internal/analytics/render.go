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
	return NormalizedRenderRequest{Report: report, Chart: chart}, nil
}

func (s *Service) normalizeChartSpec(
	report NormalizedReportRequest,
	chart ChartSpec,
) (ChartSpec, error) {
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

	if report.Kind != ReportCashflow {
		if chart.Granularity != "" {
			return ChartSpec{}, errors.New("chart granularity is only valid for cashflow period reports")
		}
		if chart.ComparisonPeriods {
			return ChartSpec{}, errors.New("comparison periods are only valid for cashflow period reports")
		}
		return chart, nil
	}
	if report.Cashflow == nil {
		return ChartSpec{}, errors.New("normalized cashflow report is missing its payload")
	}
	reportGranularity := report.Cashflow.Granularity
	if chart.Granularity == "" {
		chart.Granularity = reportGranularity
	} else if chart.Granularity != reportGranularity {
		return ChartSpec{}, fmt.Errorf(
			"chart granularity %q does not match cashflow granularity %q",
			chart.Granularity,
			reportGranularity,
		)
	}
	if !chart.ComparisonPeriods {
		return chart, nil
	}
	if chart.TopN != 0 || chart.Other.Enabled {
		return ChartSpec{}, errors.New("comparison periods cannot use topN or an Other bucket")
	}
	if chart.Sort.By != SortDimension || chart.Sort.Direction != SortAscending {
		return ChartSpec{}, errors.New("comparison periods require ascending chronological sorting")
	}
	dateRange, _, _, err := s.normalizePeriod(report.Cashflow.Period)
	if err != nil {
		return ChartSpec{}, fmt.Errorf("normalize comparison period: %w", err)
	}
	if estimatePoints(dateRange, reportGranularity) < 2 {
		return ChartSpec{}, errors.New("comparison periods require at least two authoritative time buckets")
	}
	return chart, nil
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
	points, table, err := chartData(report)
	if err != nil {
		return RenderFinanceChartResult{}, err
	}
	if normalized.Chart.ComparisonPeriods && len(points) < 2 {
		return RenderFinanceChartResult{}, errors.New(
			"comparison periods require at least two authoritative result buckets",
		)
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

func chartData(report ReportEnvelope) ([]ChartDataPoint, TableFallback, error) {
	switch report.Kind {
	case ReportSpendingSummary:
		if report.SpendingSummary == nil {
			break
		}
		points := make([]ChartDataPoint, 0, len(report.SpendingSummary.Categories))
		for _, row := range report.SpendingSummary.Categories {
			points = append(points, ChartDataPoint{ID: row.ID, Label: row.Title, Values: []ChartValue{
				{Series: SeriesAmount, Value: row.Amount},
			}})
		}
		return points, report.SpendingSummary.Table, nil
	case ReportCashflow:
		if report.Cashflow == nil {
			break
		}
		points := make([]ChartDataPoint, 0, len(report.Cashflow.Points))
		for _, row := range report.Cashflow.Points {
			points = append(points, ChartDataPoint{ID: row.ID, Label: row.Label, Values: []ChartValue{
				{Series: SeriesIncome, Value: row.Income},
				{Series: SeriesOutcome, Value: row.Outcome},
				{Series: SeriesNet, Value: row.Net},
			}})
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
			points = append(points, ChartDataPoint{ID: row.ID, Label: row.Title, Values: []ChartValue{
				{Series: SeriesBudget, Value: row.Budget}, {Series: SeriesSpent, Value: row.Spent},
				{Series: SeriesRemaining, Value: row.Remaining}, {Series: SeriesPercent, Value: percent},
			}})
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
	other := ChartDataPoint{ID: "presentation:other", Label: label, Values: make([]ChartValue, 0, len(spec.Series))}
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
