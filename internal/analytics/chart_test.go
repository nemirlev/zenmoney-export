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
