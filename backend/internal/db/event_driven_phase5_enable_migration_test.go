package db_test

import (
	"os"
	"strings"
	"testing"
)

func TestEventDrivenEnableActionIsAudited(t *testing.T) {
	b, err := os.ReadFile("../../migrations/00158_scheme_betting_enable_event_audit.sql")
	if err != nil {
		t.Fatal(err)
	}
	sql := string(b)
	for _, want := range []string{"enable_event", "scheme_betting_admin_actions_action_check", "NOT VALID"} {
		if !strings.Contains(sql, want) {
			t.Errorf("migration missing %q", want)
		}
	}
}
