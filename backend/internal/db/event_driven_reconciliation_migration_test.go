package db

import (
	"os"
	"strings"
	"testing"
)

func TestEventDrivenReconciliationMigration(t *testing.T) {
	b, err := os.ReadFile("../../migrations/00159_scheme_betting_reconciliation.sql")
	if err != nil {
		t.Fatal(err)
	}
	sql := string(b)
	for _, want := range []string{"reconciliation_evidence", "sent_unknown", "external_acceptance_unknown"} {
		if !strings.Contains(sql, want) {
			t.Errorf("migration missing %q", want)
		}
	}
}
