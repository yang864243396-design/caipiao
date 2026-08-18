package schemes

import (
	"encoding/json"
	"strings"
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

func TestCloudCenterStatsGeneratedAtJSONContract(t *testing.T) {
	const generatedAt = "2026-08-18T12:34:56.123456789Z"
	raw, err := json.Marshal(CloudCenterStats{GeneratedAt: generatedAt})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"generatedAt":"`+generatedAt+`"`) {
		t.Fatalf("json=%s", raw)
	}

	empty, err := json.Marshal(CloudCenterStats{})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(empty), `"generatedAt"`) {
		t.Fatalf("optional generatedAt serialized when empty: %s", empty)
	}
}
