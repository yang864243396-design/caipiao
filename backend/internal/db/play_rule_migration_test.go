package db

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPlayRuleStrategyMigrationDefinesRequiredPersistence(t *testing.T) {
	path := filepath.Join("..", "..", "migrations", "00149_play_rule_specs.sql")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read play rule migration: %v", err)
	}
	ddl := string(data)
	for _, want := range []string{
		"CREATE TABLE IF NOT EXISTS play_rule_specs",
		"CREATE TABLE IF NOT EXISTS play_rule_spec_revisions",
		"CREATE TABLE IF NOT EXISTS scheme_strategy_evaluations",
		"UNIQUE NULLS NOT DISTINCT (template_code, type_id, sub_id, lottery_code)",
		"UNIQUE (instance_id, period_no)",
		"ADD COLUMN IF NOT EXISTS rule_snapshot JSONB",
		"ADD COLUMN IF NOT EXISTS rule_version INTEGER",
		"ADD COLUMN IF NOT EXISTS rule_snapshot_hash TEXT",
		"idx_scheme_strategy_evaluations_recovery",
	} {
		if !strings.Contains(ddl, want) {
			t.Errorf("migration missing %q", want)
		}
	}
}

func TestFastSSCHashTailBigSmallRuleIsPublishedByMigration(t *testing.T) {
	path := filepath.Join("..", "..", "migrations", "00150_publish_fast_ssc_hash_tail_big_small.sql")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read hash tail rule migration: %v", err)
	}
	sql := string(data)
	for _, want := range []string{
		"'fast_ssc_std'",
		"'g017'",
		"'390'",
		"'ssc.attribute'",
		`"betMode":"daxiao"`,
		"strategy_enabled",
		"TRUE",
		"'published'",
	} {
		if !strings.Contains(sql, want) {
			t.Errorf("hash tail rule migration missing %q", want)
		}
	}
}

func TestFastSSCHashTailOddEvenRuleIsPublishedByMigration(t *testing.T) {
	path := filepath.Join("..", "..", "migrations", "00153_publish_fast_ssc_hash_tail_odd_even.sql")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read hash tail odd/even rule migration: %v", err)
	}
	sql := string(data)
	for _, want := range []string{
		"'fast_ssc_std'",
		"'g017'",
		"'387'",
		"'ssc.attribute'",
		`"betMode":"danshuang"`,
		`"semantic":"final_digit_odd_even"`,
		"strategy_enabled",
		"TRUE",
		"'published'",
	} {
		if !strings.Contains(sql, want) {
			t.Errorf("hash tail odd/even rule migration missing %q", want)
		}
	}
}
