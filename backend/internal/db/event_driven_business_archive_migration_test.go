package db

import (
	"os"
	"strings"
	"testing"
)

func TestEventDrivenBusinessArchiveMigration(t *testing.T) {
	b, err := os.ReadFile("../../migrations/00164_scheme_betting_business_history_archive.sql")
	if err != nil {
		t.Fatal(err)
	}
	sql := string(b)
	for _, want := range []string{
		"scheme_betting_business_history_archive",
		"PARTITION BY RANGE",
		"archive_scheme_betting_business_history",
		"trg_wallet_ledger_append_only",
		"status IN ('win', 'lose', 'cancel')",
		"status IN ('hit', 'miss')",
	} {
		if !strings.Contains(sql, want) {
			t.Errorf("migration missing %q", want)
		}
	}
}
