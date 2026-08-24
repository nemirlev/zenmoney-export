package mcpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/nemirlev/zenmoney-export/v2/internal/analytics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeAnalyticsService struct {
	mu                 sync.Mutex
	principals         []analytics.Principal
	spendingContextErr chan error
}

func (f *fakeAnalyticsService) record(principal analytics.Principal) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.principals = append(f.principals, principal)
}

func (f *fakeAnalyticsService) GetSpendingSummary(
	ctx context.Context,
	principal analytics.Principal,
	_ analytics.SpendingSummaryRequest,
) (analytics.SpendingSummaryResult, error) {
	f.record(principal)
	if f.spendingContextErr != nil {
		<-ctx.Done()
		f.spendingContextErr <- ctx.Err()
		return analytics.SpendingSummaryResult{}, ctx.Err()
	}
	return analytics.SpendingSummaryResult{
		Metadata:         metadata(analytics.ReportSpendingSummary, "RUB"),
		Total:            "123.45",
		TransactionCount: 2,
		Categories: []analytics.SpendingCategory{
			{
				ID:               "category:food",
				CategoryID:       "food",
				Title:            "Food",
				Amount:           "123.45",
				SharePercent:     "100.00",
				TransactionCount: 2,
			},
		},
		Table: tableFallback(),
	}, nil
}

func (f *fakeAnalyticsService) GetCashflow(
	_ context.Context,
	principal analytics.Principal,
	_ analytics.CashflowRequest,
) (analytics.CashflowResult, error) {
	f.record(principal)
	return analytics.CashflowResult{
		Metadata: metadata(analytics.ReportCashflow, "RUB"),
		Points:   []analytics.CashflowPoint{},
		Table:    tableFallback(),
	}, nil
}

func (f *fakeAnalyticsService) GetBudgetProgress(
	_ context.Context,
	principal analytics.Principal,
	_ analytics.BudgetProgressRequest,
) (analytics.BudgetProgressResult, error) {
	f.record(principal)
	return analytics.BudgetProgressResult{
		Metadata: metadata(analytics.ReportBudgetProgress, "RUB"),
		Rows:     []analytics.BudgetProgressRow{},
		Table:    tableFallback(),
	}, nil
}

func (f *fakeAnalyticsService) SearchTransactions(
	_ context.Context,
	principal analytics.Principal,
	_ analytics.TransactionSearchRequest,
) (analytics.TransactionSearchResult, error) {
	f.record(principal)
	return analytics.TransactionSearchResult{
		Metadata: metadata(analytics.ReportTransactions, "RUB"),
		Items:    []analytics.TransactionItem{},
		Table:    tableFallback(),
	}, nil
}

func (f *fakeAnalyticsService) GetDataFreshness(
	_ context.Context,
	principal analytics.Principal,
	_ analytics.DataFreshnessRequest,
) (analytics.DataFreshnessResult, error) {
	f.record(principal)
	return analytics.DataFreshnessResult{
		Metadata: metadata(analytics.ReportDataFreshness, ""),
		Table:    tableFallback(),
	}, nil
}

func (f *fakeAnalyticsService) RenderFinanceChart(
	_ context.Context,
	principal analytics.Principal,
	_ analytics.RenderFinanceChartRequest,
) (analytics.RenderFinanceChartResult, error) {
	f.record(principal)
	return analytics.RenderFinanceChartResult{
		SchemaVersion: analytics.SchemaVersion,
		ReportKind:    analytics.ReportSpendingSummary,
		Chart: analytics.ChartSpec{
			Type:      analytics.ChartBar,
			Title:     "Spending",
			Dimension: analytics.DimensionCategory,
			Series: []analytics.ChartSeries{
				{Key: analytics.SeriesAmount, Label: "Amount", Format: analytics.FormatCurrency},
			},
			Stacking: analytics.StackingNone,
			Sort: analytics.ChartSort{
				By:        analytics.SortValue,
				Direction: analytics.SortDescending,
				Series:    analytics.SeriesAmount,
			},
			Other:        analytics.OtherBucket{},
			Legend:       true,
			Tooltip:      true,
			ShowNegative: true,
			Table:        analytics.ChartTableSpec{Enabled: true, Caption: "Spending"},
		},
		Data: []analytics.ChartDataPoint{
			{
				ID:     "category:food",
				Label:  "Food",
				Values: []analytics.ChartValue{{Series: analytics.SeriesAmount, Value: "123.45"}},
			},
		},
		Table:  tableFallback(),
		Report: analytics.ReportEnvelope{Kind: analytics.ReportSpendingSummary},
	}, nil
}

func metadata(kind analytics.ReportKind, currency string) analytics.ReportMetadata {
	return analytics.ReportMetadata{
		SchemaVersion: analytics.SchemaVersion,
		ReportKind:    kind,
		Currency:      currency,
		Filters: analytics.AppliedFilters{
			AccountIDs: []string{}, CategoryIDs: []string{}, MerchantIDs: []string{},
		},
		Rules: analytics.CalculationRules{Limitations: []string{}},
	}
}

func tableFallback() analytics.TableFallback {
	return analytics.TableFallback{Columns: []analytics.TableColumn{}, Rows: []analytics.TableRow{}}
}

func newTestHandlers(t *testing.T, service *fakeAnalyticsService) HTTPHandlers {
	t.Helper()
	server, err := New(service, ServerOptions{Name: "test", Version: "1.0.0"})
	require.NoError(t, err)
	handlers, err := NewHTTPHandlers(server, HTTPOptions{
		IdentityResolver: IdentityResolverFunc(
			func(_ context.Context, request *http.Request) (analytics.Principal, error) {
				subject := request.Header.Get("X-Test-Subject")
				if subject == "" {
					return analytics.Principal{}, ErrUnauthenticated
				}
				userID := int64(1)
				if subject == "bob" {
					userID = 2
				}
				return analytics.Principal{Subject: subject, UserIDs: []int64{userID}}, nil
			},
		),
		ProtectOrigin:  func(next http.Handler) http.Handler { return next },
		JSONResponse:   true,
		RequestTimeout: 30 * time.Second,
	})
	require.NoError(t, err)
	return handlers
}

func TestToolMetadataAndOutputSchemas(t *testing.T) {
	handlers := newTestHandlers(t, &fakeAnalyticsService{})
	response := postMCP(t, handlers.MCP, "alice", "tools/list", "", map[string]any{})
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())

	result := decodeResult(t, response)
	tools, ok := result["tools"].([]any)
	require.True(t, ok)
	require.Len(t, tools, 6)

	for _, value := range tools {
		tool := value.(map[string]any)
		assert.NotNil(t, tool["inputSchema"], tool["name"])
		assert.NotNil(t, tool["outputSchema"], tool["name"])
		inputSchema := tool["inputSchema"].(map[string]any)
		properties, _ := inputSchema["properties"].(map[string]any)
		assert.NotContains(
			t,
			properties,
			"currency",
			"%s must derive currency from the authenticated user",
			tool["name"],
		)
		meta, hasMeta := tool["_meta"].(map[string]any)
		if tool["name"] != ToolRenderFinanceChart {
			assert.False(t, hasMeta, "%s must not open an app", tool["name"])
			continue
		}
		require.True(t, hasMeta)
		assert.NotContains(t, meta, "ui/resourceUri")
		uiMeta := meta["ui"].(map[string]any)
		assert.Equal(t, FinanceChartResourceURI, uiMeta["resourceUri"])
	}
}

func TestToolReturnsStructuredContentAndTextFallback(t *testing.T) {
	handlers := newTestHandlers(t, &fakeAnalyticsService{})
	response := postMCP(
		t,
		handlers.MCP,
		"alice",
		"tools/call",
		ToolGetSpendingSummary,
		map[string]any{
			"name": ToolGetSpendingSummary,
			"arguments": map[string]any{
				"period": map[string]any{
					"from":     "2026-01-01",
					"to":       "2026-02-01",
					"timezone": "Europe/Moscow",
				},
			},
		},
	)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	result := decodeResult(t, response)
	structured := result["structuredContent"].(map[string]any)
	assert.Equal(t, "123.45", structured["total"])
	content := result["content"].([]any)
	require.NotEmpty(t, content)
	text := content[0].(map[string]any)
	assert.Equal(t, "text", text["type"])
	assert.Contains(t, text["text"], "123.45 RUB")
}

func TestCategoryTextFallbackDisclosesTruncation(t *testing.T) {
	spending := analytics.SpendingSummaryResult{
		Metadata:   metadata(analytics.ReportSpendingSummary, "RUB"),
		Categories: make([]analytics.SpendingCategory, 2),
		Truncated:  true,
	}
	assert.Contains(
		t,
		spendingSummaryText(spending),
		"Showing first 2 categories; more categories exist.",
	)

	budget := analytics.BudgetProgressResult{
		Metadata: metadata(analytics.ReportBudgetProgress, "RUB"),
		Rows:     make([]analytics.BudgetProgressRow, 3),
		HasMore:  true,
	}
	assert.Contains(
		t,
		budgetProgressText(budget),
		"Showing first 3 categories; more categories exist.",
	)
}

func TestCategoryTextFallbackDisclosesCompleteRows(t *testing.T) {
	spending := analytics.SpendingSummaryResult{
		Metadata:   metadata(analytics.ReportSpendingSummary, "RUB"),
		Categories: make([]analytics.SpendingCategory, 2),
	}
	assert.Contains(t, spendingSummaryText(spending), "Showing all 2 categories.")
	assert.NotContains(t, spendingSummaryText(spending), "more categories exist")

	budget := analytics.BudgetProgressResult{
		Metadata: metadata(analytics.ReportBudgetProgress, "RUB"),
		Rows:     make([]analytics.BudgetProgressRow, 3),
	}
	assert.Contains(t, budgetProgressText(budget), "Showing all 3 categories.")
	assert.NotContains(t, budgetProgressText(budget), "more categories exist")
}

func TestUIResourceUsesPortableAppsContract(t *testing.T) {
	handlers := newTestHandlers(t, &fakeAnalyticsService{})
	response := postMCP(
		t,
		handlers.MCP,
		"alice",
		"resources/read",
		FinanceChartResourceURI,
		map[string]any{"uri": FinanceChartResourceURI},
	)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	result := decodeResult(t, response)
	contents := result["contents"].([]any)
	require.Len(t, contents, 1)
	resource := contents[0].(map[string]any)
	assert.Equal(t, FinanceChartMIMEType, resource["mimeType"])
	html := resource["text"].(string)
	assert.Contains(t, html, "ui/initialize")
	assert.Contains(t, html, "ui/notifications/tool-result")
	assert.Contains(t, html, "tools/call")
	assert.NotContains(t, html, "window.openai")
	assert.NotContains(t, html, "https://")
	uiMeta := resource["_meta"].(map[string]any)["ui"].(map[string]any)
	assert.NotNil(t, uiMeta["csp"])
}

func TestStatelessRequestsResolveIdentityIndependently(t *testing.T) {
	service := &fakeAnalyticsService{}
	handlers := newTestHandlers(t, service)
	for _, subject := range []string{"alice", "bob"} {
		response := postMCP(
			t,
			handlers.MCP,
			subject,
			"tools/call",
			ToolGetSpendingSummary,
			map[string]any{
				"name": ToolGetSpendingSummary,
				"arguments": map[string]any{
					"period": map[string]any{"from": "2026-01-01", "to": "2026-02-01"},
				},
			},
		)
		require.Equal(t, http.StatusOK, response.Code, response.Body.String())
		assert.Empty(t, response.Header().Get("Mcp-Session-Id"))
	}

	service.mu.Lock()
	defer service.mu.Unlock()
	require.Len(t, service.principals, 2)
	assert.Equal(
		t,
		analytics.Principal{Subject: "alice", UserIDs: []int64{1}},
		service.principals[0],
	)
	assert.Equal(t, analytics.Principal{Subject: "bob", UserIDs: []int64{2}}, service.principals[1])
}

func TestUnauthenticatedRequestCannotReachService(t *testing.T) {
	service := &fakeAnalyticsService{}
	handlers := newTestHandlers(t, service)
	response := postMCP(t, handlers.MCP, "", "tools/list", "", map[string]any{})
	assert.Equal(t, http.StatusUnauthorized, response.Code)
	service.mu.Lock()
	defer service.mu.Unlock()
	assert.Empty(t, service.principals)
}

func TestOriginProtectionRunsBeforeIdentityResolution(t *testing.T) {
	server, err := New(&fakeAnalyticsService{}, ServerOptions{})
	require.NoError(t, err)
	resolved := false
	handlers, err := NewHTTPHandlers(server, HTTPOptions{
		IdentityResolver: IdentityResolverFunc(
			func(context.Context, *http.Request) (analytics.Principal, error) {
				resolved = true
				return analytics.Principal{Subject: "alice", UserIDs: []int64{1}}, nil
			},
		),
		ProtectOrigin: func(http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				http.Error(w, "origin rejected", http.StatusForbidden)
			})
		},
		RequestTimeout: 30 * time.Second,
	})
	require.NoError(t, err)

	response := postMCP(t, handlers.MCP, "alice", "tools/list", "", map[string]any{})
	assert.Equal(t, http.StatusForbidden, response.Code)
	assert.False(t, resolved)
}

func TestHealthAndReadiness(t *testing.T) {
	server, err := New(&fakeAnalyticsService{}, ServerOptions{})
	require.NoError(t, err)
	handlers, err := NewHTTPHandlers(server, HTTPOptions{
		IdentityResolver: StaticIdentityResolver{
			Principal: analytics.Principal{Subject: "local", UserIDs: []int64{1}},
		},
		ProtectOrigin:  func(next http.Handler) http.Handler { return next },
		ReadinessCheck: func(*http.Request) error { return errors.New("database unavailable") },
		RequestTimeout: 30 * time.Second,
	})
	require.NoError(t, err)

	health := httptest.NewRecorder()
	handlers.Health.ServeHTTP(health, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	assert.Equal(t, http.StatusOK, health.Code)
	ready := httptest.NewRecorder()
	handlers.Readiness.ServeHTTP(ready, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	assert.Equal(t, http.StatusServiceUnavailable, ready.Code)
}

func TestMCPRequestTimeoutCancelsServiceContext(t *testing.T) {
	canceled := make(chan error, 1)
	service := &fakeAnalyticsService{spendingContextErr: canceled}
	server, err := New(service, ServerOptions{Name: "test", Version: "1.0.0"})
	require.NoError(t, err)
	handlers, err := NewHTTPHandlers(server, HTTPOptions{
		IdentityResolver: StaticIdentityResolver{Principal: analytics.Principal{
			Subject: "local", UserIDs: []int64{1},
		}},
		ProtectOrigin:  func(next http.Handler) http.Handler { return next },
		JSONResponse:   true,
		RequestTimeout: 20 * time.Millisecond,
	})
	require.NoError(t, err)

	response := postMCP(
		t,
		handlers.MCP,
		"local",
		"tools/call",
		ToolGetSpendingSummary,
		map[string]any{
			"name": ToolGetSpendingSummary,
			"arguments": map[string]any{
				"period": map[string]any{"from": "2026-01-01", "to": "2026-01-31"},
			},
		},
	)

	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	require.ErrorIs(t, <-canceled, context.DeadlineExceeded)
}

func TestNewHTTPHandlersRequiresPositiveRequestTimeout(t *testing.T) {
	server, err := New(&fakeAnalyticsService{}, ServerOptions{})
	require.NoError(t, err)

	_, err = NewHTTPHandlers(server, HTTPOptions{
		IdentityResolver: StaticIdentityResolver{Principal: analytics.Principal{
			Subject: "local", UserIDs: []int64{1},
		}},
	})
	require.ErrorContains(t, err, "timeout must be greater than zero")
}

func postMCP(
	t *testing.T,
	handler http.Handler,
	subject, method, name string,
	params map[string]any,
) *httptest.ResponseRecorder {
	t.Helper()
	params["_meta"] = map[string]any{
		"io.modelcontextprotocol/protocolVersion":    "2026-07-28",
		"io.modelcontextprotocol/clientCapabilities": map[string]any{},
		"io.modelcontextprotocol/clientInfo": map[string]any{
			"name":    "test-client",
			"version": "1.0.0",
		},
	}
	body, err := json.Marshal(
		map[string]any{"jsonrpc": "2.0", "id": 1, "method": method, "params": params},
	)
	require.NoError(t, err)
	request := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json, text/event-stream")
	request.Header.Set("MCP-Protocol-Version", "2026-07-28")
	request.Header.Set("Mcp-Method", method)
	if name != "" {
		request.Header.Set("Mcp-Name", name)
	}
	if subject != "" {
		request.Header.Set("X-Test-Subject", subject)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func decodeResult(t *testing.T, response *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var envelope map[string]any
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &envelope), response.Body.String())
	if rpcError := envelope["error"]; rpcError != nil {
		t.Fatalf("MCP error: %v", rpcError)
	}
	result, ok := envelope["result"].(map[string]any)
	require.True(t, ok, response.Body.String())
	return result
}

func TestEmbeddedUIIsSelfContained(t *testing.T) {
	assert.True(t, strings.HasPrefix(strings.TrimSpace(financeChartHTML), "<!doctype html>"))
	assert.NotContains(t, financeChartHTML, "<script src=")
	assert.NotContains(t, financeChartHTML, "<link rel=")
}

func TestEmbeddedUIHonorsPresentationFlags(t *testing.T) {
	for _, contract := range []string{
		"legend: spec.legend !== false",
		"tooltip: spec.tooltip !== false",
		"showNegative: spec.showNegative !== false",
		"tableEnabled: Boolean(spec.table && spec.table.enabled)",
		"authoritativeTable: payload.table",
		"formatExactValue(value, column.format",
		"valueFormat === \"percent\" ? formatted + \"%\"",
		"negative values remain available in the table",
	} {
		assert.Contains(t, financeChartHTML, contract)
	}
	assert.NotContains(t, financeChartHTML, `options.style = "percent"`)
}

func TestEmbeddedUITableUsesAuthoritativeStringCells(t *testing.T) {
	start := strings.Index(financeChartHTML, "function renderTable(model)")
	require.GreaterOrEqual(t, start, 0)
	end := strings.Index(financeChartHTML[start:], "function svg(")
	require.Greater(t, end, 0)
	tableRenderer := financeChartHTML[start : start+end]

	assert.Contains(t, tableRenderer, "model.authoritativeTable")
	assert.Contains(t, tableRenderer, "formatExactValue(value, column.format")
	assert.NotContains(t, tableRenderer, "number(")
}
