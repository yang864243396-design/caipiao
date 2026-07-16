package schemes

import "testing"

func TestZuheContent013NestedPnl(t *testing.T) {
	rule := resolveSSCPlayRule("qianzhonghou3", "qzh3_zuhe", "zuhe", "前中后三组合")
	// 选号 0/1/3 · 0 · 0；仅个位=0 嵌套一星
	balls := []string{"9", "8", "7", "6", "0"}
	ev := evaluatePlayHit(rule, balls, "0,1,3\n0\n0", false, "", 0)
	if !ev.Hit {
		t.Fatalf("want hit, ev=%+v", ev)
	}
	if ev.BetUnits != 27 {
		t.Fatalf("BetUnits=%d want 27 (3×1×1×3组合×3区)", ev.BetUnits)
	}
	amount := float64(ev.BetUnits)
	pnl := calcPnLWithOdds(amount, ev.Hit, ev.Odds)
	if pnl < 9.60 || pnl > 9.70 {
		t.Fatalf("pnl=%v want ~9.65 (not 9.65-27=%v)", pnl, 9.65-amount)
	}
}
