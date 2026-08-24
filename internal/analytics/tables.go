package analytics

import (
	"strconv"
	"time"
)

func spendingTable(rows []SpendingCategory, currency string) TableFallback {
	table := TableFallback{
		Caption: "Spending by category (" + currency + ")",
		Columns: []TableColumn{
			{Key: "category", Label: "Category", Format: FormatText},
			{Key: "amount", Label: "Amount", Format: FormatCurrency},
			{Key: "share", Label: "Share", Format: FormatPercent},
			{Key: "transactions", Label: "Transactions", Format: FormatNumber},
		},
		Rows: make([]TableRow, 0, len(rows)),
	}
	for _, row := range rows {
		table.Rows = append(table.Rows, TableRow{ID: row.ID, Cells: []TableCell{
			{Key: "category", Value: row.Title}, {Key: "amount", Value: string(row.Amount)},
			{Key: "share", Value: string(row.SharePercent)},
			{Key: "transactions", Value: strconv.FormatInt(row.TransactionCount, 10)},
		}})
	}
	return table
}

func cashflowTable(points []CashflowPoint, currency string) TableFallback {
	table := TableFallback{
		Caption: "Cashflow (" + currency + ")",
		Columns: []TableColumn{
			{Key: "period", Label: "Period", Format: FormatText},
			{Key: "income", Label: "Income", Format: FormatCurrency},
			{Key: "outcome", Label: "Outcome", Format: FormatCurrency},
			{Key: "net", Label: "Net", Format: FormatCurrency},
		},
		Rows: make([]TableRow, 0, len(points)),
	}
	for _, point := range points {
		table.Rows = append(table.Rows, TableRow{ID: point.ID, Cells: []TableCell{
			{Key: "period", Value: point.Label}, {Key: "income", Value: string(point.Income)},
			{Key: "outcome", Value: string(point.Outcome)}, {Key: "net", Value: string(point.Net)},
		}})
	}
	return table
}

func budgetTable(rows []BudgetProgressRow, currency string) TableFallback {
	table := TableFallback{
		Caption: "Budget progress (" + currency + ")",
		Columns: []TableColumn{
			{Key: "category", Label: "Category", Format: FormatText},
			{Key: "budget", Label: "Budget", Format: FormatCurrency},
			{Key: "spent", Label: "Spent", Format: FormatCurrency},
			{Key: "remaining", Label: "Remaining", Format: FormatCurrency},
			{Key: "percent", Label: "Progress", Format: FormatPercent},
		},
		Rows: make([]TableRow, 0, len(rows)),
	}
	for _, row := range rows {
		percent := ""
		if row.Percent != nil {
			percent = string(*row.Percent)
		}
		table.Rows = append(table.Rows, TableRow{ID: row.ID, Cells: []TableCell{
			{Key: "category", Value: row.Title}, {Key: "budget", Value: string(row.Budget)},
			{Key: "spent", Value: string(row.Spent)},
			{Key: "remaining", Value: string(row.Remaining)}, {Key: "percent", Value: percent},
		}})
	}
	return table
}

func transactionsTable(items []TransactionItem, currency string) TableFallback {
	table := TableFallback{
		Caption: "Transactions (" + currency + ")",
		Columns: []TableColumn{
			{Key: "date", Label: "Date", Format: FormatDate},
			{Key: "direction", Label: "Direction", Format: FormatText},
			{Key: "amount", Label: "Net amount", Format: FormatCurrency},
			{Key: "account", Label: "Account", Format: FormatText},
			{Key: "merchant", Label: "Merchant", Format: FormatText},
		},
		Rows: make([]TableRow, 0, len(items)),
	}
	for _, item := range items {
		table.Rows = append(table.Rows, TableRow{ID: item.ID, Cells: []TableCell{
			{Key: "date", Value: item.Date}, {Key: "direction", Value: string(item.Direction)},
			{Key: "amount", Value: string(item.Amount)}, {Key: "account", Value: item.AccountTitle},
			{Key: "merchant", Value: item.MerchantTitle},
		}})
	}
	return table
}

func freshnessTable(data FreshnessData) TableFallback {
	table := TableFallback{
		Caption: "Data synchronization freshness",
		Columns: []TableColumn{
			{Key: "run", Label: "Run", Format: FormatText},
			{Key: "finished", Label: "Finished", Format: FormatText},
			{Key: "status", Label: "Status", Format: FormatText},
			{Key: "records", Label: "Records", Format: FormatNumber},
		},
		Rows: []TableRow{},
	}
	appendSnapshot := func(id, label string, snapshot *SyncSnapshot) {
		if snapshot == nil {
			return
		}
		finished := ""
		if snapshot.FinishedAt != nil {
			finished = snapshot.FinishedAt.Format(time.RFC3339)
		}
		table.Rows = append(table.Rows, TableRow{ID: id, Cells: []TableCell{
			{Key: "run", Value: label}, {Key: "finished", Value: finished},
			{Key: "status", Value: snapshot.Status},
			{Key: "records", Value: strconv.FormatInt(snapshot.RecordsProcessed, 10)},
		}})
	}
	appendSnapshot("sync:last_completed", "Last completed", data.LastCompleted)
	appendSnapshot("sync:last_attempt", "Last attempt", data.LastAttempt)
	return table
}
