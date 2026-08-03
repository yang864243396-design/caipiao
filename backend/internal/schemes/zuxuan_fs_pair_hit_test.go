package schemes

import "testing"

// 前二组选复式号池含开出对子数字时仍不应计中（C(n,2) 无对子；与第三方一致）。
func TestQian2ZuxuanFsPoolMissOnPair(t *testing.T) {
	rule := resolveSSCPlayRule("g004", "42", "zuxuan_fs", "前二组选复式")
	ev := evaluateZuxuanFushi(rule, []string{"7", "7", "1", "9", "2"}, "0,1,2,3,4,5,6,7,8,9")
	if ev.Hit {
		t.Fatalf("开出 77 对子应挂，got hit units=%d odds=%v", ev.BetUnits, ev.Odds)
	}
	if ev.BetUnits != 45 {
		t.Fatalf("units=%d want 45", ev.BetUnits)
	}
	ev2 := evaluateZuxuanFushi(rule, []string{"7", "3", "1", "9", "2"}, "0,1,2,3,4,5,6,7,8,9")
	if !ev2.Hit {
		t.Fatal("开出 73 应中")
	}
}
