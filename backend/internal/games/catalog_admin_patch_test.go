package games

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"

	"caipiao/backend/internal/db/sqlcdb"
)

func TestNormalizeMaintenancePatchDefaults(t *testing.T) {
	current := sqlcdb.LotteryCatalogRow{
		Code:                "tron_ffc_1m",
		DisplayName:         "波场1分彩",
		SortOrder:           1,
		SaleStatus:          "maintenance",
		OutboundLotteryCode: pgtype.Text{String: "tron_ffc_1m", Valid: true},
	}
	got, err := normalizeMaintenancePatch(current, PatchCatalogInput{})
	if err != nil {
		t.Fatal(err)
	}
	if got.DisplayName != "波场1分彩" || got.OutboundLotteryCode != "tron_ffc_1m" || got.SortOrder != 1 {
		t.Fatalf("got=%+v", got)
	}
	if got.SaleStatus != "maintenance" {
		t.Fatalf("saleStatus=%s", got.SaleStatus)
	}
}

func TestPublicLotteryRouteStatusLegacy(t *testing.T) {
	s := &Service{}
	st := s.PublicLotteryRouteStatus(context.Background(), "tencent_ffc")
	if !st.Legacy || st.Exists {
		t.Fatalf("legacy=%+v", st)
	}
}
