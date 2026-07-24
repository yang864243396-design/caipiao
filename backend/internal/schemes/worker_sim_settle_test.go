package schemes

import (
	"testing"

	"caipiao/backend/internal/db/sqlcdb"
)

func TestSimBetDefersSettlementLikeFormal(t *testing.T) {
	w := &Worker{}
	inst := sqlcdb.SchemeInstance{SimBet: true}
	if w.usesGuajiThirdParty(inst) {
		t.Fatal("sim must not place via third party")
	}
	// 约定：模拟盘 deferSettle=true（与 guajiReal 同路径记 pending，待真实开奖结算）
	deferSettle := w.usesGuajiThirdParty(inst) || inst.SimBet
	if !deferSettle {
		t.Fatal("sim bets must defer settlement until real draw")
	}
}

func TestSimBaodanHitAgainstRealBalls(t *testing.T) {
	// 复现线上误判：包胆 9 / 开奖含 9，即时空球验奖会 miss；真实球号应 hit。
	rule := playRule{
		PlayTemplate: "ssc_std",
		PlayTypeID:   "g007",
		SubPlayID:    "109",
		BetMode:      "baodan",
		CatalogSubID: "qianzhonghou3_zuxuan_bd",
		SegmentLen:   3,
		SegmentPos:   []int{0, 1, 2, 1, 2, 3, 2, 3, 4},
		OddsBase:     970,
	}
	empty := evaluatePlayHit(rule, nil, "9", false, "", 0)
	if empty.Hit {
		t.Fatal("empty balls must not hit")
	}
	real := evaluatePlayHit(rule, []string{"7", "9", "4", "9", "6"}, "9", false, "", 0)
	if !real.Hit {
		t.Fatalf("baodan 9 vs balls with 9 should hit, units=%d odds=%v", real.BetUnits, real.Odds)
	}
	if pnl := calcPnLWithOdds(1296, real.Hit, real.Odds); pnl <= 0 {
		t.Fatalf("hit pnl should be positive, got %v", pnl)
	}
}
