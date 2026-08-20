package db_test

import (
	"os"
	"strings"
	"testing"
)

func TestGrayTronFFC6SCapacityMigrationContracts(t *testing.T) {
	b, err := os.ReadFile("../../migrations/00171_seed_tron_ffc_6s_gray_capacity.sql")
	if err != nil {
		t.Fatal(err)
	}
	sql := string(b)
	for _, want := range []string{
		"INSERT INTO scheme_betting_capacity_limits",
		"'tron_ffc_6s'",
		"ON CONFLICT (lottery_code) DO NOTHING",
		"max_account_dispatch_per_second",
		"max_global_dispatch_per_second",
	} {
		if !strings.Contains(sql, want) {
			t.Errorf("migration missing %q", want)
		}
	}
}
