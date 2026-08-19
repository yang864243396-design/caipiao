package db_test

import (
	"os"
	"strings"
	"testing"
)

func TestEventDrivenAdminAuditMigrationIsAppendOnly(t *testing.T) {
	b, err := os.ReadFile("../../migrations/00157_event_driven_scheme_betting_admin_audit.sql")
	if err != nil {
		t.Fatal(err)
	}
	sql := string(b)
	for _, want := range []string{"actor_account", "append-only", "BEFORE UPDATE OR DELETE"} {
		if !strings.Contains(sql, want) {
			t.Errorf("migration missing %q", want)
		}
	}
}
