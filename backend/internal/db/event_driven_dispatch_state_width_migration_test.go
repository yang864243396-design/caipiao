package db_test

import (
	"os"
	"strings"
	"testing"
)

func TestEventDrivenDispatchStateWidthMigrationContracts(t *testing.T) {
	b, err := os.ReadFile("../../migrations/00172_fix_scheme_betting_state_width.sql")
	if err != nil {
		t.Fatal(err)
	}
	sql := string(b)
	for _, want := range []string{
		"ALTER TABLE scheme_bet_outbox",
		"ALTER COLUMN state TYPE VARCHAR(32)",
		"ALTER TABLE scheme_bet_attempts",
		"ALTER COLUMN outcome TYPE VARCHAR(32)",
	} {
		if !strings.Contains(sql, want) {
			t.Errorf("migration missing %q", want)
		}
	}
}
