package analytics

type ValueFormat string

const (
	FormatText     ValueFormat = "text"
	FormatDate     ValueFormat = "date"
	FormatCurrency ValueFormat = "currency"
	FormatNumber   ValueFormat = "number"
	FormatPercent  ValueFormat = "percent"
)

type TableFallback struct {
	Caption string        `json:"caption"`
	Columns []TableColumn `json:"columns"`
	Rows    []TableRow    `json:"rows"`
}

type TableColumn struct {
	Key    string      `json:"key"`
	Label  string      `json:"label"`
	Format ValueFormat `json:"format"`
}

type TableRow struct {
	ID    string      `json:"id"`
	Cells []TableCell `json:"cells"`
}

type TableCell struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}
