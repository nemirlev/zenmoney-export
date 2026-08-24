package postgres

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

func TestInitialDownMigrationDropsCreatedEnumTypesAfterTables(t *testing.T) {
	t.Parallel()

	up := readMigration(t, "000001_init.up.sql")
	down := readMigration(t, "000001_init.down.sql")

	createTypePattern := regexp.MustCompile(
		`(?im)^\s*CREATE\s+TYPE\s+("?[a-z_][a-z0-9_]*"?)\s+AS\s+ENUM\b`,
	)
	createdTypes := createTypePattern.FindAllStringSubmatch(up, -1)
	if len(createdTypes) == 0 {
		t.Fatal("initial up migration does not create any enum types")
	}

	lastTableDrop := strings.LastIndex(strings.ToUpper(down), "DROP TABLE")
	if lastTableDrop == -1 {
		t.Fatal("initial down migration does not drop any tables")
	}

	for _, match := range createdTypes {
		typeName := match[1]
		dropTypePattern := regexp.MustCompile(
			`(?im)^\s*DROP\s+TYPE\s+IF\s+EXISTS\s+` + regexp.QuoteMeta(typeName) + `\s*;`,
		)
		dropLocation := dropTypePattern.FindStringIndex(down)
		if dropLocation == nil {
			t.Errorf("initial down migration does not safely drop enum type %s", typeName)
			continue
		}
		if dropLocation[0] < lastTableDrop {
			t.Errorf("enum type %s is dropped before all tables", typeName)
		}
	}
}

func TestAnalyticsTypesMigrationValidatesTextBeforeCasting(t *testing.T) {
	t.Parallel()

	up := normalizedSQL(readMigration(t, "000004_analytics_types.up.sql"))

	// Optional API fields deliberately normalize blanks to NULL.
	for _, conversion := range []string{
		"NULLIF(BTRIM(start_date), '')::DATE",
		"NULLIF(BTRIM(end_date), '')::DATE",
		"NULLIF(BTRIM(parent), '')::UUID",
		"NULLIF(BTRIM(outcome_account), '')::UUID",
	} {
		if !strings.Contains(up, normalizedSQL(conversion)) {
			t.Errorf("optional blank normalization is missing: %s", conversion)
		}
	}

	// Required SDK fields use a direct cast, so blank values fail instead of
	// silently becoming NULL.
	for _, conversion := range []string{
		"ALTER COLUMN date TYPE DATE USING date::DATE",
		"ALTER COLUMN start_date TYPE DATE USING start_date::DATE",
		"ALTER COLUMN income_account TYPE UUID USING income_account::UUID",
	} {
		if !strings.Contains(up, normalizedSQL(conversion)) {
			t.Errorf("required field does not use a strict cast: %s", conversion)
		}
	}

	for _, preflight := range []string{
		"RAISE EXCEPTION 'cannot migrate %. Invalid ISO date: %'",
		"date !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}$'",
		"RAISE EXCEPTION 'cannot migrate %. Invalid UUID: %'",
		"parent !~* '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$'",
	} {
		if !strings.Contains(up, normalizedSQL(preflight)) {
			t.Errorf("invalid non-empty value preflight is missing: %s", preflight)
		}
	}
}

func TestAnalyticsTypesMigrationCoversFinancialAndAnalyticalColumns(t *testing.T) {
	t.Parallel()

	up := normalizedSQL(readMigration(t, "000004_analytics_types.up.sql"))

	for _, conversion := range []string{
		"ALTER COLUMN rate TYPE NUMERIC",
		"ALTER COLUMN balance TYPE NUMERIC",
		"ALTER COLUMN start_balance TYPE NUMERIC",
		"ALTER COLUMN credit_limit TYPE NUMERIC",
		"ALTER COLUMN percent TYPE NUMERIC",
		"ALTER COLUMN op_income TYPE NUMERIC",
		"ALTER COLUMN op_outcome TYPE NUMERIC",
	} {
		if !strings.Contains(up, normalizedSQL(conversion)) {
			t.Errorf("financial conversion is missing: %s", conversion)
		}
	}

	for _, indexName := range []string{
		"idx_transaction_user_date_created",
		"idx_transaction_income_account_date",
		"idx_transaction_outcome_account_date",
		"idx_transaction_merchant_date",
		"idx_transaction_income_instrument_date",
		"idx_transaction_outcome_instrument_date",
		"idx_transaction_tags_gin",
	} {
		if !strings.Contains(up, indexName) {
			t.Errorf("transaction index is missing: %s", indexName)
		}
	}

	if strings.Contains(up, "ALTER COLUMN latitude") ||
		strings.Contains(up, "ALTER COLUMN longitude") {
		t.Error("geographic coordinates must remain floating point")
	}
	if strings.Contains(up, "deletion_history") {
		t.Error("mixed deletion_history.object_id must remain text")
	}
}

func TestAnalyticsTypesDownMigrationRestoresOriginalStorageTypes(t *testing.T) {
	t.Parallel()

	down := normalizedSQL(readMigration(t, "000004_analytics_types.down.sql"))

	for _, reversal := range []string{
		"TYPE DOUBLE PRECISION USING",
		"TYPE TEXT USING TO_CHAR(date, 'YYYY-MM-DD')",
		"TYPE TEXT USING outcome_account::TEXT",
		"DROP INDEX IF EXISTS idx_transaction_tags_gin",
		"DROP INDEX IF EXISTS idx_transaction_user_date_created",
	} {
		if !strings.Contains(down, normalizedSQL(reversal)) {
			t.Errorf("down migration reversal is missing: %s", reversal)
		}
	}
}

func normalizedSQL(sql string) string {
	return strings.Join(strings.Fields(sql), " ")
}

func readMigration(t *testing.T, name string) string {
	t.Helper()

	contents, err := os.ReadFile(name)
	if err != nil {
		t.Fatalf("read migration %s: %v", name, err)
	}

	return string(contents)
}
