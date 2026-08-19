package db

import (
	"os"
	"strings"
	"testing"
)

func TestEventDrivenEventsAndRetentionMigration(t *testing.T) {
	b, err := os.ReadFile("../../migrations/00162_scheme_betting_events_retention.sql")
	if err != nil {
		t.Fatal(err)
	}
	sql := string(b)
	for _, want := range []string{
		"ready_published_at", "reconcile_published_state",
		"scheme_betting_retention_policies", "PARTITION BY RANGE",
		"archive_terminal_scheme_bets", "idx_wallet_ledger_scheme_bet_audit",
	} {
		if !strings.Contains(sql, want) {
			t.Errorf("migration missing %q", want)
		}
	}
}
