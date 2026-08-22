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

	createTypePattern := regexp.MustCompile(`(?im)^\s*CREATE\s+TYPE\s+("?[a-z_][a-z0-9_]*"?)\s+AS\s+ENUM\b`)
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
		dropTypePattern := regexp.MustCompile(`(?im)^\s*DROP\s+TYPE\s+IF\s+EXISTS\s+` + regexp.QuoteMeta(typeName) + `\s*;`)
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

func readMigration(t *testing.T, name string) string {
	t.Helper()

	contents, err := os.ReadFile(name)
	if err != nil {
		t.Fatalf("read migration %s: %v", name, err)
	}

	return string(contents)
}
