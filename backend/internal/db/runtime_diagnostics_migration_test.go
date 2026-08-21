package db_test

import (
	"os"
	"strings"
	"testing"
)

func TestRuntimeDiagnosticsMigrationUsesConcurrentIndexContract(t *testing.T) {
	b, err := os.ReadFile("../../migrations/00178_scheme_runtime_diagnostics_index.sql")
	if err != nil {
		t.Fatal(err)
	}
	sql := string(b)
	for _, want := range []string{
		"-- +goose NO TRANSACTION",
		"CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_scheme_period_decisions_scheme_latest",
		"DROP INDEX CONCURRENTLY IF EXISTS idx_scheme_period_decisions_scheme_latest",
	} {
		if !strings.Contains(sql, want) {
			t.Errorf("migration missing %q", want)
		}
	}
}
