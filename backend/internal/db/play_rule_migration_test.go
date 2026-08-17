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
