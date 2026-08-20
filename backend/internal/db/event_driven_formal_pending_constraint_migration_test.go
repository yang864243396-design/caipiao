package db_test

import (
	"os"
	"strings"
	"testing"
)

func TestEventDrivenFormalPendingConstraintMigrationContracts(t *testing.T) {
	b, err := os.ReadFile("../../migrations/00173_drop_legacy_formal_pending_outbox_check.sql")
	if err != nil {
		t.Fatal(err)
	}
	sql := string(b)
	for _, want := range []string{
		"-- +goose StatementBegin",
		"-- +goose StatementEnd",
		"pg_constraint",
		"scheme_bet_outbox",
		"state <> 'pending'",
		"DROP CONSTRAINT",
	} {
		if !strings.Contains(sql, want) {
			t.Errorf("migration missing %q", want)
		}
	}
}
