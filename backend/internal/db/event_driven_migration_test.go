package db_test

import (
	"os"
	"strings"
	"testing"
)

func TestEventDrivenSchemeBettingMigrationContracts(t *testing.T) {
	b, err := os.ReadFile("../../migrations/00154_event_driven_scheme_betting_shadow.sql")
	if err != nil {
		t.Fatal(err)
	}
	sql := string(b)
	wants := []string{
		"CREATE TABLE IF NOT EXISTS provider_period_snapshots",
		"CREATE TABLE IF NOT EXISTS scheme_period_decisions",
		"CREATE TABLE IF NOT EXISTS scheme_bet_outbox",
		"CREATE TABLE IF NOT EXISTS scheme_bet_attempts",
		"draw_hash",
		"provider_event_id",
		"safe_deadline_at",
		"lease_owner",
		"lease_fencing_token",
		"sent_unknown",
		"UNIQUE (scheme_id, source_period_no)",
		"UNIQUE (scheme_id, target_period_no)",
		"mode VARCHAR(16) NOT NULL DEFAULT 'shadow'",
	}
	for _, want := range wants {
		if !strings.Contains(sql, want) {
			t.Errorf("migration missing %q", want)
		}
	}
}
