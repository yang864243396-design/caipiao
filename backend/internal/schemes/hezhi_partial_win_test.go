package schemes

import "testing"

func TestEvaluateHezhiZuxuanPartialWinNotFullZhixuanOdds(t *testing.T) {
	rule := resolveSSCPlayRule("g004", "44", "hezhi", "前二组选和值")
	if !rule.HezhiZuxuan || rule.SegmentLen != 2 {
		t.Fatalf("rule=%+v", rule)
	}
	// 全包 1–17 = 45 注；开奖 8,1 → 和值 9 = 5 注中
	content := "1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17"
	ev := evaluateHezhi(rule, []string{"8", "1", "1", "1", "1"}, content)
	if !ev.Hit || ev.BetUnits != 45 {
		t.Fatalf("ev=%+v", ev)
	}
	// unitNet(组选二星)=9 → net = 5*9 - 40 = 5；odds = 5/45
	wantOdds := 5.0 / 45.0
	if ev.Odds < wantOdds-1e-9 || ev.Odds > wantOdds+1e-9 {
		t.Fatalf("Odds=%v want %v (旧逻辑整票×97)", ev.Odds, wantOdds)
	}
	if ev.PrizeNet < 5-1e-9 || ev.PrizeNet > 5+1e-9 {
		t.Fatalf("PrizeNet=%v want 5", ev.PrizeNet)
	}
	amount := 180.0 // 2 元 × 倍数 2 × 45 注
	pnl := calcPnLWithOdds(amount, ev.Hit, ev.Odds)
	if pnl < 20-0.01 || pnl > 20+0.01 {
		t.Fatalf("pnl=%v want 20（旧逻辑 17460）", pnl)
	}
}

func TestEvaluateHezhiZuxuanSingleSumFullOdds(t *testing.T) {
	rule := resolveSSCPlayRule("g004", "44", "hezhi", "前二组选和值")
	ev := evaluateHezhi(rule, []string{"8", "1", "1", "1", "1"}, "9")
	if !ev.Hit || ev.BetUnits != 5 {
		t.Fatalf("ev=%+v", ev)
	}
	// 全中：odds = 组选二星 9
	if ev.Odds < 9-0.01 || ev.Odds > 9+0.01 {
		t.Fatalf("Odds=%v want 9", ev.Odds)
	}
}

func TestEvaluateHezhiZhixuanPartialWin(t *testing.T) {
	rule := resolveSSCPlayRule("g004", "40", "hezhi", "前二直选和值")
	if rule.HezhiZuxuan {
		t.Fatalf("直选和值不应 HezhiZuxuan, rule=%+v", rule)
	}
	// 和值 9 = 10 注；全包 0–18 = 100 注
	content := "0,1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18"
	ev := evaluateHezhi(rule, []string{"8", "1", "1", "1", "1"}, content)
	if !ev.Hit || ev.BetUnits != 100 {
		t.Fatalf("ev=%+v want units 100", ev)
	}
	// unitNet=97 → net = 10*97 - 90 = 880；odds = 880/100
	wantOdds := 880.0 / 100.0
	if ev.Odds < wantOdds-1e-9 || ev.Odds > wantOdds+1e-9 {
		t.Fatalf("Odds=%v want %v", ev.Odds, wantOdds)
	}
}

func TestEvaluateBetPayloadHezhiZuxuanFromPlayMethod(t *testing.T) {
	raw := []byte(`{"playTemplate":"ssc_std","typeId":"g004","subId":"hezhi","betMode":"hezhi","groupContent":"1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17","playMethod":"前二码 前二组选和值"}`)
	hit, odds := EvaluateBetPayload(raw, []string{"8", "1", "1", "1", "1"})
	if !hit {
		t.Fatal("want hit")
	}
	want := 5.0 / 45.0
	if odds < want-1e-9 || odds > want+1e-9 {
		t.Fatalf("odds=%v want %v (payload 须认组选和值)", odds, want)
	}
}
