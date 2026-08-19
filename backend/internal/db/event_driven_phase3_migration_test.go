package db_test

import (
	"os"
	"strings"
	"testing"
)

func TestEventDrivenPhase3MigrationContracts(t *testing.T) {
	b, err := os.ReadFile("../../migrations/00155_event_driven_scheme_betting_dispatch.sql")
	if err != nil {
		t.Fatal(err)
	}
	sql := string(b)
	wants := []string{
		"accepted_wrong_period",
		"external_acceptance_unknown",
		"blocked_requires_rearm",
		"betting_owner",
		"frozen_request",
		"frozen_request_hash",
		"command_frozen_at",
		"dispatch_started_at",
		"CREATE TABLE IF NOT EXISTS scheme_betting_shard_leases",
		"CREATE TABLE IF NOT EXISTS scheme_betting_admin_actions",
		"CREATE TABLE IF NOT EXISTS scheme_betting_capacity_limits",
		"mode = 'shadow' OR",
	}
	for _, want := range wants {
		if !strings.Contains(sql, want) {
			t.Errorf("migration missing %q", want)
		}
	}
}
