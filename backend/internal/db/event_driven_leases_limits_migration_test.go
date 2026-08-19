package db

import (
	"os"
	"strings"
	"testing"
)

func TestEventDrivenLeasesAndLimitsMigration(t *testing.T) {
	b, err := os.ReadFile("../../migrations/00161_scheme_betting_leases_limits.sql")
	if err != nil {
		t.Fatal(err)
	}
	sql := string(b)
	for _, want := range []string{
		"lease_kind", "scheme_betting_draw_leases", "scheme_betting_dispatch_rate_buckets",
		"max_account_dispatch_per_second", "max_global_dispatch_per_second",
	} {
		if !strings.Contains(sql, want) {
			t.Errorf("migration missing %q", want)
		}
	}
}
