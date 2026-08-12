package sqlcdb

import (
	"strings"
	"testing"
)

func TestListAdminFundRecordsPagedLimitsLedgerBeforeSchemeLookup(t *testing.T) {
	if !strings.Contains(listAdminFundRecordsPaged, "WITH ledger_page AS MATERIALIZED") {
		t.Fatal("admin ledger query must page wallet_ledger before looking up scheme names")
	}
	if !strings.Contains(listAdminFundRecordsPaged, "c.bet_order_no = NULLIF(TRIM(p.order_ref), '')") {
		t.Fatal("admin ledger query must use the indexed order-reference scheme lookup")
	}
	if strings.Contains(listAdminFundRecordsPaged, "ABS(EXTRACT(EPOCH FROM (c.placed_at - l.created_at)))") {
		t.Fatal("admin ledger query must not use non-sargable ABS time matching")
	}
}
