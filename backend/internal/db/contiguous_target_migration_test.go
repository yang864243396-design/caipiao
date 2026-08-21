package db_test

import (
	"os"
	"strings"
	"testing"
)

func TestContiguousTargetMigrationContracts(t *testing.T) {
	sql := readContiguousTargetMigration(t)
	for _, want := range []string{
		"awaiting_target",
		"missed_contiguous_period",
		"target_deadline_at",
		"target_period_no",
		"failure_reason",
		"chain_block_reason",
		"lottery_code, status, target_deadline_at, id",
		"shard_no INTEGER",
		"CHECK (shard_no >= 0 AND shard_no <= 63)",
		"shard_no, lottery_code, target_deadline_at, id",
		"cannot roll back migration 177 while contiguous decisions remain",
	} {
		if !strings.Contains(sql, want) {
			t.Errorf("migration missing %q", want)
		}
	}
}

func readContiguousTargetMigration(t *testing.T) string {
	t.Helper()
	const path = "../../migrations/00177_scheme_contiguous_target_wait.sql"
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read migration %s: %v", path, err)
	}
	return string(b)
}
