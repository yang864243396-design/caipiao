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
		"lottery_code, shard_no, id",
		"cannot roll back migration 177 while contiguous decisions remain",
	} {
		if !strings.Contains(sql, want) {
			t.Errorf("migration missing %q", want)
		}
	}
}

func TestContiguousTargetListingIsWorkBoundedByShardScope(t *testing.T) {
	migration := readContiguousTargetMigration(t)
	for _, want := range []string{
		"idx_scheme_period_decisions_awaiting_target_shard",
		"(lottery_code, shard_no, id)",
		"WHERE status = 'awaiting_target'",
	} {
		if !strings.Contains(migration, want) {
			t.Errorf("migration missing work-bounded shard index %q", want)
		}
	}
	if strings.Contains(migration, "(shard_no, lottery_code, target_deadline_at, id)") {
		t.Error("migration retains the non-keyset shard recovery index")
	}

	query := readContiguousTargetQuery(t)
	for _, want := range []string{
		"WITH scopes AS",
		"CROSS JOIN LATERAL",
		"d.lottery_code = scope.lottery_code",
		"d.shard_no = scope.shard_no",
		"ORDER BY d.id",
		"LIMIT 32",
		"ORDER BY decision_id",
		"AND clock_timestamp() < d.target_deadline_at",
		"AND clock_timestamp() >= d.target_deadline_at",
	} {
		if !strings.Contains(query, want) {
			t.Errorf("contiguous target query missing %q", want)
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

func readContiguousTargetQuery(t *testing.T) string {
	t.Helper()
	const path = "sqlcdb/scheme_contiguous_target_ext.go"
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read query %s: %v", path, err)
	}
	return string(b)
}
