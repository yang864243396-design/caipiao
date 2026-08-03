package schemes

import "testing"

func TestEvaluateRenxuanDanshiMultiPositions(t *testing.T) {
	rule := resolveSSCPlayRule("g011", "75", "danshi", "任选 任二直选单式")
	// 开奖 1 2 3 4 5；选万千百 + 号码 12 → C(3,2)=3 注，万千=12 应中
	balls := []string{"1", "2", "3", "4", "5"}
	ev := evaluatePlayHit(rule, balls, "万,千,百\n12,99", false, "", 0)
	if !ev.Hit {
		t.Fatalf("want hit, got %+v", ev)
	}
	if ev.BetUnits != 6 {
		t.Fatalf("BetUnits=%d want 6 (C(3,2)*2)", ev.BetUnits)
	}
	miss := evaluatePlayHit(rule, balls, "万,千,百\n99,88", false, "", 0)
	if miss.Hit {
		t.Fatalf("want miss, got %+v", miss)
	}
	if miss.BetUnits != 6 {
		t.Fatalf("miss BetUnits=%d want 6", miss.BetUnits)
	}
}
