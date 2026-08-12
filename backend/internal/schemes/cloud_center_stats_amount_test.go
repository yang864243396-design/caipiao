package schemes

import (
	"testing"

	"github.com/jackc/pgx/v5/pgtype"

	"caipiao/backend/internal/db/sqlcdb"
)

func TestMapCloudCenterChannelStats_truncatesTurnoverToGuajiCents(t *testing.T) {
	var turnover, pnl pgtype.Numeric
	if err := turnover.Scan("0.176"); err != nil {
		t.Fatal(err)
	}
	if err := pnl.Scan("0.176"); err != nil {
		t.Fatal(err)
	}
	got := mapCloudCenterChannelStats(sqlcdb.MemberCloudCenterStatsRow{
		TotalTurnover:   turnover,
		TotalSessionPnl: pnl,
	})
	if got.TotalTurnover != 0.17 {
		t.Fatalf("turnover=%v want 0.17", got.TotalTurnover)
	}
	if got.TotalSessionPnl != 0.2 {
		t.Fatalf("sessionPnl=%v want 0.2", got.TotalSessionPnl)
	}
}
