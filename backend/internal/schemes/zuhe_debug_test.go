package schemes

import "testing"

func TestZuheContent013Nested(t *testing.T) {
	rule := resolveSSCPlayRule("qianzhonghou3", "qzh3_zuhe", "zuhe", "前中后三组合")
	balls := []string{"9", "8", "7", "6", "0"}
	ev := evaluatePlayHit(rule, balls, "0,1,3\n0\n0", false, "", 0)
	if !ev.Hit {
		t.Fatalf("want hit, ev=%+v", ev)
	}
	if ev.BetUnits != 27 {
		t.Fatalf("BetUnits=%d want 27", ev.BetUnits)
	}
	amount := float64(ev.BetUnits)
	pnl := amount * ev.Odds
	if pnl < 9.60 || pnl > 9.70 {
		t.Fatalf("local pnl=%v want ~9.65 (odds=%v)", pnl, ev.Odds)
	}
}
