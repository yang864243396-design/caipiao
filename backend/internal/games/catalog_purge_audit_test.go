package games

import (
	"testing"

	"caipiao/backend/internal/db/sqlcdb"
)

func TestLegacyCatalogPurgePreservesFinancialAndAuditFacts(t *testing.T) {
	steps := legacyCatalogPurgeSteps(&sqlcdb.Queries{})
	protected := map[string]bool{
		"wallet_ledger(bet)":     true,
		"wallet_ledger(chase)":   true,
		"bet_orders":             true,
		"cloud_bet_records":      true,
		"admin_audit_logs":       true,
		"chase_orders":           true,
		"scheme_definitions":     true,
		"scheme_share_snapshots": true,
		"lottery_draws":          true,
	}
	for _, step := range steps {
		if protected[step.name] {
			t.Fatalf("legacy purge includes protected audit table %q", step.name)
		}
	}
}
