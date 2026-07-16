package schemes

import "testing"

func TestEvaluateRenxuanDanshiPipeAndNewline(t *testing.T) {
	rule := resolveSSCPlayRule("g011", "75", "danshi", "任选 任二直选单式")
	// 千=1 个=2 → 应中 12
	balls := []string{"0", "1", "0", "0", "2"}
	for _, content := range []string{"千,个\n12,34", "千个|12,34", "千,个\n12"} {
		ev := evaluatePlayHit(rule, balls, content, false, "", 0)
		t.Logf("content=%q hit=%v units=%v odds=%v rule=%+v", content, ev.Hit, ev.BetUnits, ev.Odds, rule)
		if !ev.Hit {
			t.Errorf("want hit content=%q", content)
		}
	}
	miss := evaluatePlayHit(rule, []string{"0", "9", "0", "0", "8"}, "千,个\n12,34", false, "", 0)
	if miss.Hit {
		t.Fatalf("want miss, got %+v", miss)
	}
}

func TestEvaluateRenxuanZhixuanFsComma(t *testing.T) {
	rule := resolveSSCPlayRule("g011", "74", "fushi", "任选 任二直选复式")
	// wire: 千位1 个位2
	ev := evaluatePlayHit(rule, []string{"0", "1", "0", "0", "2"}, ",1,,,2", false, "", 0)
	t.Logf("hit=%v units=%v", ev.Hit, ev.BetUnits)
	if !ev.Hit {
		t.Fatal("want hit")
	}
}

func TestEvaluateRenxuanZuxuanDanshi(t *testing.T) {
	rule := resolveSSCPlayRule("g011", "78", "zuxuan_ds", "任选 任二组选单式")
	// 千=2 个=1 → 组选中 12
	ev := evaluatePlayHit(rule, []string{"0", "2", "0", "0", "1"}, "千,个\n12,34", false, "", 0)
	t.Logf("hit=%v", ev.Hit)
	if !ev.Hit {
		t.Fatal("want zuxuan hit on reverse order")
	}
	// 对子不应误中？12 对 11
	miss := evaluatePlayHit(rule, []string{"0", "1", "0", "0", "1"}, "千,个\n12,34", false, "", 0)
	t.Logf("duizi miss=%v", miss.Hit)
}
