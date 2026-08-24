package mcpserver

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/nemirlev/zenmoney-export/v2/internal/analytics"
)

const (
	FinanceChartResourceURI = "ui://zenmoney/finance-chart"
	FinanceChartMIMEType    = "text/html;profile=mcp-app"
)

const (
	ToolGetSpendingSummary = "get_spending_summary"
	ToolGetCashflow        = "get_cashflow"
	ToolGetBudgetProgress  = "get_budget_progress"
	ToolSearchTransactions = "search_transactions"
	ToolGetDataFreshness   = "get_data_freshness"
	ToolRenderFinanceChart = "render_finance_chart"
)

//go:embed ui/chart.html
var financeChartHTML string

// AnalyticsService keeps MCP transport independent from persistence. The
// concrete analytics.Service satisfies this interface; tests can use fakes.
type AnalyticsService interface {
	GetSpendingSummary(
		context.Context,
		analytics.Principal,
		analytics.SpendingSummaryRequest,
	) (analytics.SpendingSummaryResult, error)
	GetCashflow(
		context.Context,
		analytics.Principal,
		analytics.CashflowRequest,
	) (analytics.CashflowResult, error)
	GetBudgetProgress(
		context.Context,
		analytics.Principal,
		analytics.BudgetProgressRequest,
	) (analytics.BudgetProgressResult, error)
	SearchTransactions(
		context.Context,
		analytics.Principal,
		analytics.TransactionSearchRequest,
	) (analytics.TransactionSearchResult, error)
	GetDataFreshness(
		context.Context,
		analytics.Principal,
		analytics.DataFreshnessRequest,
	) (analytics.DataFreshnessResult, error)
	RenderFinanceChart(
		context.Context,
		analytics.Principal,
		analytics.RenderFinanceChartRequest,
	) (analytics.RenderFinanceChartResult, error)
}

var _ AnalyticsService = (*analytics.Service)(nil)

type ServerOptions struct {
	Name    string
	Version string
}

type Server struct {
	core *mcp.Server
}

func New(service AnalyticsService, options ServerOptions) (*Server, error) {
	if service == nil {
		return nil, errors.New("analytics service is required")
	}
	if strings.TrimSpace(options.Name) == "" {
		options.Name = "zenmcp"
	}
	if strings.TrimSpace(options.Version) == "" {
		options.Version = "dev"
	}

	core := mcp.NewServer(&mcp.Implementation{Name: options.Name, Version: options.Version}, nil)
	server := &Server{core: core}
	server.registerTools(service)
	server.registerUIResource()
	return server, nil
}

func (s *Server) registerTools(service AnalyticsService) {
	mcp.AddTool(s.core, &mcp.Tool{
		Name: ToolGetSpendingSummary,
		Description: "Summarize expenses by category for an inclusive from/to date period. " +
			"Returns authoritative Decimal amounts as strings, applied filters, calculation rules, stable category IDs, and a compact table fallback.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, input analytics.SpendingSummaryRequest) (*mcp.CallToolResult, analytics.SpendingSummaryResult, error) {
		principal, err := principalFromContext(ctx)
		if err != nil {
			return nil, analytics.SpendingSummaryResult{}, err
		}
		output, err := service.GetSpendingSummary(ctx, principal, input)
		return textResult(spendingSummaryText(output)), output, err
	})

	mcp.AddTool(s.core, &mcp.Tool{
		Name: ToolGetCashflow,
		Description: "Return aggregated income, outcome, and net cashflow over an inclusive from/to date period with day, week, or month granularity. " +
			"Transfers and currency conversion follow the calculation rules included in the result.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, input analytics.CashflowRequest) (*mcp.CallToolResult, analytics.CashflowResult, error) {
		principal, err := principalFromContext(ctx)
		if err != nil {
			return nil, analytics.CashflowResult{}, err
		}
		output, err := service.GetCashflow(ctx, principal, input)
		return textResult(cashflowText(output)), output, err
	})

	mcp.AddTool(s.core, &mcp.Tool{
		Name: ToolGetBudgetProgress,
		Description: "Compare category spending with configured budgets for an inclusive from/to date period. " +
			"Returns budget, spent, remaining, and percentage values with stable category IDs.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, input analytics.BudgetProgressRequest) (*mcp.CallToolResult, analytics.BudgetProgressResult, error) {
		principal, err := principalFromContext(ctx)
		if err != nil {
			return nil, analytics.BudgetProgressResult{}, err
		}
		output, err := service.GetBudgetProgress(ctx, principal, input)
		return textResult(budgetProgressText(output)), output, err
	})

	mcp.AddTool(s.core, &mcp.Tool{
		Name: ToolSearchTransactions,
		Description: "Search a bounded, cursor-paginated set of transactions visible to the authenticated identity. " +
			"Use aggregated tools for broad analysis; this tool intentionally limits personal transaction detail.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, input analytics.TransactionSearchRequest) (*mcp.CallToolResult, analytics.TransactionSearchResult, error) {
		principal, err := principalFromContext(ctx)
		if err != nil {
			return nil, analytics.TransactionSearchResult{}, err
		}
		output, err := service.SearchTransactions(ctx, principal, input)
		return textResult(transactionSearchText(output)), output, err
	})

	mcp.AddTool(s.core, &mcp.Tool{
		Name:        ToolGetDataFreshness,
		Description: "Report the latest completed and attempted ZenMoney synchronization and whether the analytical data is stale.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, input analytics.DataFreshnessRequest) (*mcp.CallToolResult, analytics.DataFreshnessResult, error) {
		principal, err := principalFromContext(ctx)
		if err != nil {
			return nil, analytics.DataFreshnessResult{}, err
		}
		output, err := service.GetDataFreshness(ctx, principal, input)
		return textResult(dataFreshnessText(output)), output, err
	})

	mcp.AddTool(s.core, &mcp.Tool{
		Name: ToolRenderFinanceChart,
		Description: "Re-run a normalized authoritative finance report and render it using a validated declarative ChartSpec. " +
			"The input accepts no SQL, HTML, JavaScript, URLs, executable expressions, or caller-supplied financial rows.",
		Meta: mcp.Meta{
			"ui": map[string]any{
				"resourceUri": FinanceChartResourceURI,
				"visibility":  []string{"model", "app"},
			},
		},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, input analytics.RenderFinanceChartRequest) (*mcp.CallToolResult, analytics.RenderFinanceChartResult, error) {
		principal, err := principalFromContext(ctx)
		if err != nil {
			return nil, analytics.RenderFinanceChartResult{}, err
		}
		output, err := service.RenderFinanceChart(ctx, principal, input)
		return textResult(renderChartText(output)), output, err
	})
}

func (s *Server) registerUIResource() {
	s.core.AddResource(&mcp.Resource{
		URI:         FinanceChartResourceURI,
		Name:        "zenmoney_finance_chart",
		Title:       "ZenMoney finance chart",
		Description: "Self-contained accessible renderer for validated finance ChartSpec results.",
		MIMEType:    FinanceChartMIMEType,
	}, func(_ context.Context, request *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		if request.Params.URI != FinanceChartResourceURI {
			return nil, mcp.ResourceNotFoundError(request.Params.URI)
		}
		return &mcp.ReadResourceResult{Contents: []*mcp.ResourceContents{{
			URI:      FinanceChartResourceURI,
			MIMEType: FinanceChartMIMEType,
			Text:     financeChartHTML,
			Meta: mcp.Meta{"ui": map[string]any{
				"csp": map[string]any{
					"connectDomains":  []string{},
					"resourceDomains": []string{},
					"frameDomains":    []string{},
					"baseUriDomains":  []string{},
				},
				"prefersBorder": true,
			}},
		}}}, nil
	})
}

func textResult(text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: text}}}
}

func spendingSummaryText(output analytics.SpendingSummaryResult) string {
	return fmt.Sprintf(
		"Spending total: %s %s across %d transactions. %s",
		output.Total,
		output.Metadata.Currency,
		output.TransactionCount,
		categoryRowsText(len(output.Categories), output.HasMore || output.Truncated),
	)
}

func cashflowText(output analytics.CashflowResult) string {
	return fmt.Sprintf(
		"Cashflow: income %s, outcome %s, net %s %s across %d periods.",
		output.Totals.Income,
		output.Totals.Outcome,
		output.Totals.Net,
		output.Metadata.Currency,
		len(output.Points),
	)
}

func budgetProgressText(output analytics.BudgetProgressResult) string {
	return fmt.Sprintf(
		"Budget progress: spent %s of %s %s. %s",
		output.Totals.Spent,
		output.Totals.Budget,
		output.Metadata.Currency,
		categoryRowsText(len(output.Rows), output.HasMore || output.Truncated),
	)
}

func categoryRowsText(count int, truncated bool) string {
	if truncated {
		return fmt.Sprintf("Showing first %d categories; more categories exist.", count)
	}
	return fmt.Sprintf("Showing all %d categories.", count)
}

func transactionSearchText(output analytics.TransactionSearchResult) string {
	continuation := ""
	if output.NextCursor != "" {
		continuation = " More results are available with nextCursor."
	}
	return fmt.Sprintf(
		"Found %d transactions in this bounded page.%s",
		len(output.Items),
		continuation,
	)
}

func dataFreshnessText(output analytics.DataFreshnessResult) string {
	if output.LastCompleted == nil {
		return "No completed synchronization is available."
	}
	age := "unknown"
	if output.AgeSeconds != nil {
		age = fmt.Sprintf("%d seconds", *output.AgeSeconds)
	}
	return fmt.Sprintf("Latest completed synchronization is %s old; stale=%t.", age, output.Stale)
}

func renderChartText(output analytics.RenderFinanceChartResult) string {
	return fmt.Sprintf(
		"Prepared validated %s chart %q from %d authoritative report rows. An accessible table fallback is included.",
		output.Chart.Type,
		output.Chart.Title,
		len(output.Data),
	)
}
