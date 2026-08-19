package db

import (
	"os"
	"strings"
	"testing"
)

func TestEventDrivenEventPublishBackoffMigration(t *testing.T) {
	b, err := os.ReadFile("../../migrations/00163_scheme_betting_event_publish_backoff.sql")
	if err != nil {
		t.Fatal(err)
	}
	sql := string(b)
	for _, want := range []string{
		"ready_next_attempt_at", "reconcile_next_attempt_at",
		"state NOT IN ('pending', 'leased')", "idx_scheme_bet_outbox_ready_retry",
	} {
		if !strings.Contains(sql, want) {
			t.Errorf("migration missing %q", want)
		}
	}
}
