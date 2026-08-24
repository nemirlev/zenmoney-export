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
	report, err := s.NormalizeReportRequest(request.Report)
	if err != nil {
		return NormalizedRenderRequest{}, err
	}
	if err := ValidateChartSpec(request.Chart, s.limits.MaxChartPoints); err != nil {
		return NormalizedRenderRequest{}, err
	}
	if err := validateChartForReport(report.Kind, request.Chart); err != nil {
		return NormalizedRenderRequest{}, err
	}
	return NormalizedRenderRequest{Report: report, Chart: request.Chart}, nil
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
