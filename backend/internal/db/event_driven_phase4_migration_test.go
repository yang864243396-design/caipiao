package db_test

import (
	"os"
	"strings"
	"testing"
)

func TestEventDrivenFinanceMigrationContracts(t *testing.T) {
	b, err := os.ReadFile("../../migrations/00156_event_driven_scheme_betting_finance.sql")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"provider_account_id", "provider_amount", "local_order_no", "local_cloud_record_id", "financial_finalized_at", "state = 'accepted'"} {
		if !strings.Contains(string(b), want) {
			t.Errorf("migration missing %q", want)
		}
	}
}
