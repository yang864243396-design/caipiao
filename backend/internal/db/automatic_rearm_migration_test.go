package db_test

import (
	"os"
	"strings"
	"testing"
)

func TestAutomaticRearmRecoveryMigrationContracts(t *testing.T) {
	b, err := os.ReadFile("../../migrations/00174_scheme_betting_automatic_rearm.sql")
	if err != nil {
		t.Fatal(err)
	}
	sql := string(b)
	for _, want := range []string{
		"-- +goose NO TRANSACTION",
		"CREATE INDEX CONCURRENTLY",
		"idx_scheme_instances_safe_rearm",
		"idx_scheme_bet_outbox_scheme_detail",
		"blocked_requires_rearm",
	} {
		if !strings.Contains(sql, want) {
			t.Errorf("migration missing %q", want)
		}
	}
}
