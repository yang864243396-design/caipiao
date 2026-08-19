package db

import (
	"os"
	"strings"
	"testing"
)

func TestEventDrivenAPIOutboxMigration(t *testing.T) {
	b, err := os.ReadFile("../../migrations/00160_scheme_betting_api_outbox.sql")
	if err != nil {
		t.Fatal(err)
	}
	sql := string(b)
	for _, want := range []string{
		"origin", "scheme_bet_outbox_origin_check", "DROP NOT NULL",
		"origin = 'scheme'", "origin = 'api'", "local_order_no IS NOT NULL",
		"scheme_betting_admin_actions", "scheme_bet_outbox_origin_shape_check",
	} {
		if !strings.Contains(sql, want) {
			t.Errorf("migration missing %q", want)
		}
	}
}
