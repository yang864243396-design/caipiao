package member

import (
	"testing"

	"caipiao/backend/internal/db/sqlcdb"
)

func TestGroupAdminMemberCurrencyStatsKeepsCurrenciesSeparate(t *testing.T) {
	grouped := groupAdminMemberCurrencyStats([]sqlcdb.ListAdminMemberCurrencyStatsRow{
		{MemberID: 7, Currency: "USDT", TotalBetAmount: 120.5, TotalPnl: 20.5},
		{MemberID: 7, Currency: "TRX", TotalBetAmount: 80, TotalPnl: -8},
	})

	stats := grouped[7]
	if len(stats) != 2 {
		t.Fatalf("stats length = %d, want 2", len(stats))
	}
	if stats[0].Currency != "USDT" || stats[0].TotalBetAmount != 120.5 || stats[0].TotalPnl != 20.5 {
		t.Fatalf("USDT stat = %#v", stats[0])
	}
	if stats[1].Currency != "TRX" || stats[1].TotalBetAmount != 80 || stats[1].TotalPnl != -8 {
		t.Fatalf("TRX stat = %#v", stats[1])
	}
}
