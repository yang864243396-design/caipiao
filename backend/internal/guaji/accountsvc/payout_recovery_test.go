package accountsvc

import "testing"

// This catches a recovery scan that restarts at page 1 after every payout
// tick, which can never reach a historical accepted order behind new bets.
func TestNextHistoricalSettlementPageAdvancesAndWraps(t *testing.T) {
	if got := nextHistoricalSettlementPage(4, 7, false); got != 7 {
		t.Fatalf("continued scan page=%d, want 7", got)
	}
	if got := nextHistoricalSettlementPage(4, 9, true); got != 4 {
		t.Fatalf("exhausted scan page=%d, want 4", got)
	}
	if got := nextHistoricalSettlementPage(4, 0, false); got != 4 {
		t.Fatalf("invalid scan page=%d, want 4", got)
	}
}
