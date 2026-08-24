package analytics

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateChartSpecAcceptsSupportedChartTypes(t *testing.T) {
	for _, chartType := range []ChartType{
		ChartBar, ChartHorizontalBar, ChartLine, ChartArea, ChartDonut, ChartStackedBar, ChartGroupedBar,
	} {
		t.Run(string(chartType), func(t *testing.T) {
			spec := validChartSpec(chartType)
			require.NoError(t, ValidateChartSpec(spec, 100))
		})
	}
}

func TestValidateChartSpecRejectsUnsupportedAndExecutableInputs(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*ChartSpec)
		want   string
	}{
		{name: "unsupported type", mutate: func(spec *ChartSpec) { spec.Type = "radar" }, want: "unsupported chart type"},
		{name: "HTML", mutate: func(spec *ChartSpec) { spec.Title = "<script>alert(1)</script>" }, want: "unsafe"},
		{name: "JavaScript URL", mutate: func(spec *ChartSpec) { spec.Subtitle = "javascript:alert(1)" }, want: "unsafe"},
		{name: "remote URL", mutate: func(spec *ChartSpec) { spec.Other.Label = "https://evil.example" }, want: "unsafe"},
		{name: "eval", mutate: func(spec *ChartSpec) { spec.Series[0].Label = "eval(payload)" }, want: "unsafe"},
		{name: "SQL", mutate: func(spec *ChartSpec) { spec.Table.Caption = "DROP TABLE transaction" }, want: "unsafe"},
		{name: "native config path", mutate: func(spec *ChartSpec) { spec.Series[0].Key = "options.plugins" }, want: "unsupported chart series"},
		{name: "Other without topN", mutate: func(spec *ChartSpec) { spec.Other.Enabled = true }, want: "requires topN"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			spec := validChartSpec(ChartBar)
			test.mutate(&spec)
			require.ErrorContains(t, ValidateChartSpec(spec, 100), test.want)
		})
	}
}

func TestNormalizeReportRequestRequiresMatchingSinglePayload(t *testing.T) {
	service := newTestService(t, &fakeStore{})
	period := PeriodRequest{From: "2026-08-01", To: "2026-08-24"}

	_, err := service.NormalizeReportRequest(ReportRequest{
		Kind:            ReportCashflow,
		SpendingSummary: &SpendingSummaryRequest{Period: period},
	})
	require.ErrorContains(t, err, "does not match")

	_, err = service.NormalizeReportRequest(ReportRequest{
		Kind:            ReportCashflow,
		SpendingSummary: &SpendingSummaryRequest{Period: period},
		Cashflow:        &CashflowRequest{Period: period},
	})
	require.ErrorContains(t, err, "exactly one payload")
}

func TestNormalizeRenderRequestAppliesSafeCanonicalDefaults(t *testing.T) {
	service := newTestService(t, &fakeStore{})
	request := RenderFinanceChartRequest{
		Report: ReportRequest{Kind: ReportSpendingSummary, SpendingSummary: &SpendingSummaryRequest{
			Period: PeriodRequest{From: "2026-08-01", To: "2026-08-24"},
		}},
		Chart: ChartSpec{
			Type: ChartBar, Title: " Spending ", Dimension: DimensionCategory,
			Series: []ChartSeries{{Key: SeriesAmount, Label: " Amount ", Format: FormatCurrency}},
		},
	}

	normalized, err := service.NormalizeRenderRequest(request)

	require.NoError(t, err)
	require.Equal(t, "Spending", normalized.Chart.Title)
	require.Equal(t, "Amount", normalized.Chart.Series[0].Label)
	require.Equal(t, StackingNone, normalized.Chart.Stacking)
	require.Equal(t, ChartSort{By: SortDimension, Direction: SortAscending}, normalized.Chart.Sort)
	require.Equal(t, OtherBucket{Label: "Other"}, normalized.Chart.Other)
	require.Equal(t, ChartTableSpec{Enabled: true, Caption: "Spending data"}, normalized.Chart.Table)
}

func TestNormalizeRenderRequestReconcilesCashflowGranularity(t *testing.T) {
	service := newTestService(t, &fakeStore{})
	request := cashflowRenderRequest("2026-08-01", "2026-08-03", GranularityDay)

	normalized, err := service.NormalizeRenderRequest(request)
	require.NoError(t, err)
	require.Equal(t, GranularityDay, normalized.Chart.Granularity)

	request.Chart.Granularity = GranularityWeek
	_, err = service.NormalizeRenderRequest(request)
	require.ErrorContains(t, err, "does not match cashflow granularity")

	request.Report.Cashflow.Granularity = ""
	normalized, err = service.NormalizeRenderRequest(request)
	require.NoError(t, err)
	require.Equal(t, GranularityWeek, normalized.Report.Cashflow.Granularity)
	require.Equal(t, GranularityWeek, normalized.Chart.Granularity)

	spending := RenderFinanceChartRequest{
		Report: ReportRequest{Kind: ReportSpendingSummary, SpendingSummary: &SpendingSummaryRequest{
			Period: PeriodRequest{From: "2026-08-01", To: "2026-08-03"},
		}},
		Chart: validChartSpec(ChartBar),
	}
	spending.Chart.Granularity = GranularityDay
	_, err = service.NormalizeRenderRequest(spending)
	require.ErrorContains(t, err, "only valid for cashflow")
}

func TestComparisonPeriodsPreservesAuthoritativeBucketValues(t *testing.T) {
	store := &fakeStore{cashflowDataSet: []CashflowData{
		{
			Currency: "RUB",
			Points: []CashflowPoint{
				{ID: "period:2026-08-01", From: "2026-08-01", To: "2026-08-02", Label: "2026-08-01", Income: "10.50", Outcome: "2.25", Net: "8.25"},
				{ID: "period:2026-08-02", From: "2026-08-02", To: "2026-08-03", Label: "2026-08-02", Income: "4", Outcome: "5", Net: "-1"},
				{ID: "period:2026-08-03", From: "2026-08-03", To: "2026-08-04", Label: "2026-08-03", Income: "0", Outcome: "1", Net: "-1"},
			},
			Totals: CashflowTotals{Income: "14.50", Outcome: "8.25", Net: "6.25"},
		},
		{
			Currency: "RUB",
			Points: []CashflowPoint{
				{ID: "period:2026-07-29", From: "2026-07-29", To: "2026-07-30", Label: "2026-07-29", Income: "7", Outcome: "2", Net: "5"},
				{ID: "period:2026-07-30", From: "2026-07-30", To: "2026-07-31", Label: "2026-07-30", Income: "1", Outcome: "4.75", Net: "-3.75"},
				{ID: "period:2026-07-31", From: "2026-07-31", To: "2026-08-01", Label: "2026-07-31", Income: "2", Outcome: "2", Net: "0"},
			},
			Totals: CashflowTotals{Income: "10", Outcome: "8.75", Net: "1.25"},
		},
	}}
	service := newTestService(t, store)
	request := cashflowRenderRequest("2026-08-01", "2026-08-03", GranularityDay)
	request.Chart.ComparisonPeriods = true

	result, err := service.RenderFinanceChart(
		context.Background(), Principal{Subject: "user", UserIDs: []int64{1}}, request,
	)

	require.NoError(t, err)
	require.Equal(t, 2, store.cashflowCalls, "comparison must execute both authoritative reports")
	require.Equal(t, "2026-08-01", store.cashflowQueries[0].Range.From.Format(dateLayout))
	require.Equal(t, "2026-07-29", store.cashflowQueries[1].Range.From.Format(dateLayout))
	require.Equal(t, "2026-08-01", store.cashflowQueries[1].Range.To.Format(dateLayout))
	require.True(t, result.Chart.ComparisonPeriods)
	require.Equal(t, GranularityDay, result.Chart.Granularity)
	require.Equal(t, []ChartSeriesKey{SeriesCurrentNet, SeriesPreviousNet}, []ChartSeriesKey{
		result.Chart.Series[0].Key, result.Chart.Series[1].Key,
	})
	require.Equal(t, []Decimal{"8.25", "-1", "-1"}, []Decimal{
		result.Data[0].Values[0].Value,
		result.Data[1].Values[0].Value,
		result.Data[2].Values[0].Value,
	})
	require.Equal(t, []Decimal{"5", "-3.75", "0"}, []Decimal{
		result.Data[0].Values[1].Value,
		result.Data[1].Values[1].Value,
		result.Data[2].Values[1].Value,
	})
	require.Equal(t, []string{"comparison:bucket:0000", "comparison:bucket:0001", "comparison:bucket:0002"},
		[]string{result.Data[0].ID, result.Data[1].ID, result.Data[2].ID})
	require.NotNil(t, result.PreviousReport)
	require.NotNil(t, result.Comparison)
	require.Equal(t, "2026-08-01", result.Comparison.CurrentPeriod.From)
	require.Equal(t, "2026-07-29", result.Comparison.PreviousPeriod.From)
	require.Equal(t, "2026-07-31", result.Comparison.PreviousPeriod.To)
	require.Equal(t, "calendar_day_index", result.Comparison.Alignment)
	require.Equal(t, 3, result.Comparison.BucketCount)
	require.Equal(t, "2026-08-01", result.Table.Rows[0].Cells[1].Value)
	require.Equal(t, "8.25", result.Table.Rows[0].Cells[2].Value)
	require.Equal(t, "2026-07-29", result.Table.Rows[0].Cells[3].Value)
	require.Equal(t, "5", result.Table.Rows[0].Cells[4].Value)
}

func TestComparisonPeriodsRejectsNoOpConfigurations(t *testing.T) {
	service := newTestService(t, &fakeStore{})
	request := cashflowRenderRequest("2026-08-01", "2026-08-03", GranularityDay)
	request.Chart.ComparisonPeriods = true
	request.Chart.TopN = 2
	_, err := service.NormalizeRenderRequest(request)
	require.ErrorContains(t, err, "cannot use topN")

	request = cashflowRenderRequest("2026-08-01", "2026-08-03", GranularityDay)
	request.Chart.ComparisonPeriods = true
	request.Chart.Sort = ChartSort{By: SortValue, Direction: SortDescending, Series: SeriesNet}
	_, err = service.NormalizeRenderRequest(request)
	require.ErrorContains(t, err, "ascending chronological sorting")

	request = cashflowRenderRequest("2026-08-01", "2026-08-03", GranularityWeek)
	request.Chart.ComparisonPeriods = true
	_, err = service.NormalizeRenderRequest(request)
	require.ErrorContains(t, err, "only daily granularity")

	request = cashflowRenderRequest("2026-08-01", "2026-08-03", GranularityDay)
	request.Chart.ComparisonPeriods = true
	request.Chart.Type = ChartArea
	_, err = service.NormalizeRenderRequest(request)
	require.ErrorContains(t, err, "only line charts")

	request = cashflowRenderRequest("2026-08-01", "2026-08-03", GranularityDay)
	request.Chart.ComparisonPeriods = true
	request.Chart.Series = []ChartSeries{{Key: SeriesIncome, Label: "Income", Format: FormatCurrency}}
	_, err = service.NormalizeRenderRequest(request)
	require.ErrorContains(t, err, "exactly the net series")
}

func TestRenderRejectsCallerSuppliedComparisonSeries(t *testing.T) {
	service := newTestService(t, &fakeStore{})
	request := cashflowRenderRequest("2026-08-01", "2026-08-03", GranularityDay)
	request.Chart.Series = []ChartSeries{
		{Key: SeriesCurrentNet, Label: "Current net", Format: FormatCurrency},
		{Key: SeriesPreviousNet, Label: "Previous net", Format: FormatCurrency},
	}

	_, err := service.NormalizeRenderRequest(request)
	require.ErrorContains(t, err, "server-derived")
}

func TestRenderFinanceChartRerunsAuthoritativeReportAndBuildsOther(t *testing.T) {
	store := &fakeStore{spendingData: SpendingSummaryData{
		Currency: "RUB", Total: "16", TransactionCount: 3,
		Categories: []SpendingCategory{
			{ID: "category:a", CategoryID: "a", Title: "A", Amount: "10", SharePercent: "62.5"},
			{ID: "category:b", CategoryID: "b", Title: "B", Amount: "4.5", SharePercent: "28.125"},
			{ID: "category:c", CategoryID: "c", Title: "C", Amount: "1.5", SharePercent: "9.375"},
		},
	}}
	service := newTestService(t, store)
	request := RenderFinanceChartRequest{
		Report: ReportRequest{Kind: ReportSpendingSummary, SpendingSummary: &SpendingSummaryRequest{
			Period: PeriodRequest{From: "2026-08-01", To: "2026-08-24"},
		}},
		Chart: validChartSpec(ChartBar),
	}
	request.Chart.TopN = 1
	request.Chart.Other = OtherBucket{Enabled: true, Label: "Rest"}
	request.Chart.Sort = ChartSort{By: SortValue, Direction: SortDescending, Series: SeriesAmount}
	principal := Principal{Subject: "user", UserIDs: []int64{1}}

	first, err := service.RenderFinanceChart(context.Background(), principal, request)
	require.NoError(t, err)
	second, err := service.RenderFinanceChart(context.Background(), principal, request)
	require.NoError(t, err)
	require.Equal(t, 2, store.spendingCalls, "render must rerun the authoritative report")
	require.Equal(t, first.Data, second.Data)
	require.Len(t, first.Data, 2)
	require.Equal(t, "category:a", first.Data[0].ID)
	require.Equal(t, "presentation:other", first.Data[1].ID)
	require.Equal(t, Decimal("6"), first.Data[1].Values[0].Value)
	require.NotEmpty(t, first.Table.Columns, "an accessible table fallback is always returned")
}

func TestRenderRejectsReportSeriesMismatchBeforeStore(t *testing.T) {
	store := &fakeStore{}
	service := newTestService(t, store)
	request := RenderFinanceChartRequest{
		Report: ReportRequest{Kind: ReportSpendingSummary, SpendingSummary: &SpendingSummaryRequest{
			Period: PeriodRequest{From: "2026-08-01", To: "2026-08-24"},
		}},
		Chart: validChartSpec(ChartBar),
	}
	request.Chart.Series = []ChartSeries{{Key: SeriesIncome, Label: "Income", Format: FormatCurrency}}

	_, err := service.RenderFinanceChart(
		context.Background(), Principal{Subject: "user", UserIDs: []int64{1}}, request,
	)

	require.ErrorContains(t, err, "not available")
	require.Zero(t, store.spendingCalls)
}

func validChartSpec(chartType ChartType) ChartSpec {
	dimension := DimensionCategory
	series := []ChartSeries{{Key: SeriesAmount, Label: "Amount", Format: FormatCurrency}}
	stacking := StackingNone
	if chartType == ChartLine || chartType == ChartArea {
		dimension = DimensionPeriod
		series = []ChartSeries{{Key: SeriesNet, Label: "Net", Format: FormatCurrency}}
	}
	if chartType == ChartStackedBar || chartType == ChartGroupedBar {
		dimension = DimensionPeriod
		series = []ChartSeries{
			{Key: SeriesIncome, Label: "Income", Format: FormatCurrency},
			{Key: SeriesOutcome, Label: "Outcome", Format: FormatCurrency},
		}
	}
	if chartType == ChartStackedBar {
		stacking = StackingNormal
	}
	return ChartSpec{
		Type: chartType, Title: "Finance chart", Dimension: dimension, Series: series,
		Stacking: stacking, Sort: ChartSort{By: SortDimension, Direction: SortAscending},
		Legend: true, Tooltip: true, ShowNegative: true,
		Table: ChartTableSpec{Enabled: true, Caption: "Accessible data"},
	}
}

func cashflowRenderRequest(from, to string, granularity Granularity) RenderFinanceChartRequest {
	return RenderFinanceChartRequest{
		Report: ReportRequest{Kind: ReportCashflow, Cashflow: &CashflowRequest{
			Period: PeriodRequest{From: from, To: to}, Granularity: granularity,
		}},
		Chart: validChartSpec(ChartLine),
	}
}
