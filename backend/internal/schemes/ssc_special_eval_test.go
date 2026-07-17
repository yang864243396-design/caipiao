package schemes

import "testing"

func TestEvaluateLonghuWanQian(t *testing.T) {
	rule := resolveSSCPlayRule("longhu", "lh_wanqian_dou", "longhu")
	ev := evaluatePlayHit(rule, []string{"8", "3", "1", "2", "9"}, "龙", false, "", 0)
	if !ev.Hit {
		t.Fatalf("want hit for 龙 when wan>qian, ev=%+v", ev)
	}
}

func TestEvaluateHezhiQian3(t *testing.T) {
	rule := resolveSSCPlayRule("qian3", "qian3_zhixuan_hz", "hezhi")
	// 1+2+3=6
	ev := evaluatePlayHit(rule, []string{"1", "2", "3", "0", "0"}, "6", false, "", 0)
	if !ev.Hit {
		t.Fatalf("want hezhi hit, ev=%+v", ev)
	}
}

func TestEvaluateRenxuan2(t *testing.T) {
	rule := resolveSSCPlayRule("renxuan", "ren2_zhixuan_fs", "fushi")
	content := "1,3\n\n\n\n"
	ev := evaluatePlayHit(rule, []string{"1", "0", "0", "0", "0"}, content, false, "", 0)
	if !ev.Hit {
		t.Fatalf("want ren2 hit on wan, ev=%+v", ev)
	}
}

func TestEvaluateRenxuanHezhiG011(t *testing.T) {
	// 线上 catalog typeId=g011、subId 为数字；须归一到任选和值专用评估
	rule := resolveSSCPlayRule("g011", "76", "hezhi", "任选 任二直选和值")
	if rule.PlayTypeID != "renxuan" {
		t.Fatalf("PlayTypeID=%q want renxuan", rule.PlayTypeID)
	}
	// 千=3 个=9 → 和值 12
	ev := evaluatePlayHit(rule, []string{"0", "3", "0", "0", "9"}, "千个|12", false, "", 0)
	if !ev.Hit {
		t.Fatalf("want ren2 hezhi hit, ev=%+v rule=%+v", ev, rule)
	}
	miss := evaluatePlayHit(rule, []string{"0", "3", "0", "0", "8"}, "千个|12", false, "", 0)
	if miss.Hit {
		t.Fatalf("want miss when sum!=12, ev=%+v", miss)
	}
}

func TestEvaluateZu3Qian3(t *testing.T) {
	rule := resolveSSCPlayRule("qian3", "qian3_zu3", "zu3")
	// 1,1,2 组三形态
	ev := evaluatePlayHit(rule, []string{"1", "1", "2", "0", "0"}, "1,2,3", false, "", 0)
	if !ev.Hit {
		t.Fatalf("want zu3 hit, ev=%+v", ev)
	}
	ev2 := evaluatePlayHit(rule, []string{"1", "2", "3", "0", "0"}, "1,2,3", false, "", 0)
	if ev2.Hit {
		t.Fatalf("want zu6 miss on zu3 play, ev=%+v", ev2)
	}
}

func TestEvaluateBaodanQian2(t *testing.T) {
	rule := resolveSSCPlayRule("qian2", "qian2_zuxuan_bd", "baodan")
	ev := evaluatePlayHit(rule, []string{"3", "7", "0", "0", "0"}, "3", false, "", 0)
	if !ev.Hit {
		t.Fatalf("want baodan hit, ev=%+v", ev)
	}
}

func TestEvaluateWeishuQian3(t *testing.T) {
	rule := resolveSSCPlayRule("qian3", "qian3_hz_weishu", "weishu")
	// 1+2+3=6, tail 6
	ev := evaluatePlayHit(rule, []string{"1", "2", "3", "0", "0"}, "6", false, "", 0)
	if !ev.Hit {
		t.Fatalf("want weishu hit, ev=%+v", ev)
	}
}

func TestEvaluateZuheQian3(t *testing.T) {
	rule := resolveSSCPlayRule("qian3", "qian3_zuhe", "zuhe")
	ev := evaluatePlayHit(rule, []string{"1", "2", "5", "0", "0"}, "1\n2\n5", false, "", 0)
	if !ev.Hit {
		t.Fatalf("want zuhe hit, ev=%+v", ev)
	}
	if ev.BetUnits != 3 {
		t.Fatalf("BetUnits=%d want 3 (1×1×1×3)", ev.BetUnits)
	}
	ev2 := evaluatePlayHit(rule, []string{"1", "2", "5", "0", "0"}, "0,1\n2\n5", false, "", 0)
	if !ev2.Hit || ev2.BetUnits != 6 {
		t.Fatalf("want hit units=6, ev=%+v", ev2)
	}
}

func TestEvaluateZuheQianzhonghou3MultiZone(t *testing.T) {
	rule := resolveSSCPlayRule("qianzhonghou3", "qzh3_zuhe", "zuhe")
	// 仅中三 2,5,0 命中，前三/后三不中
	ev := evaluatePlayHit(rule, []string{"9", "2", "5", "0", "1"}, "2\n5\n0", false, "", 0)
	if !ev.Hit {
		t.Fatalf("want mid-zone hit, ev=%+v", ev)
	}
	if ev.BetUnits != 9 {
		t.Fatalf("BetUnits=%d want 9 (1×1×1×3组合×3区)", ev.BetUnits)
	}
	// 1/3 区中：odds_eff = (三星赔率+1)/3 - 1
	wantOdds := (oddsZhixuan(3, 0)+1)/3 - 1
	if ev.Odds < wantOdds-0.01 || ev.Odds > wantOdds+0.01 {
		t.Fatalf("Odds=%v want ~%v", ev.Odds, wantOdds)
	}
	miss := evaluatePlayHit(rule, []string{"9", "8", "7", "6", "5"}, "2\n5\n0", false, "", 0)
	if miss.Hit {
		t.Fatalf("want miss, ev=%+v", miss)
	}
	if miss.BetUnits != 9 {
		t.Fatalf("miss BetUnits=%d want 9", miss.BetUnits)
	}
}

func TestEvaluateZuheNestedYixingMultiZone(t *testing.T) {
	rule := resolveSSCPlayRule("qianzhonghou3", "qzh3_zuhe", "zuhe")
	// 选号 2/5/0；开奖 9,8,7,6,0 —— 仅后三的「后一(个位=0)」中，三星/后二都不中
	ev := evaluatePlayHit(rule, []string{"9", "8", "7", "6", "0"}, "2\n5\n0", false, "", 0)
	if !ev.Hit {
		t.Fatalf("want nested 一星 hit, ev=%+v", ev)
	}
	if ev.BetUnits != 9 {
		t.Fatalf("BetUnits=%d want 9", ev.BetUnits)
	}
	// 小奖净额 9.65 被其它区位亏损淹没 → 按第三方口径记 PrizeNet，odds=9.65/9
	want := 9.65 / 9
	if ev.Odds < want-0.02 || ev.Odds > want+0.02 {
		t.Fatalf("Odds=%v want ~%v", ev.Odds, want)
	}
	amount := 9.0
	pnl := amount * ev.Odds
	if pnl < 9.60 || pnl > 9.70 {
		t.Fatalf("pnl=%v want ~9.65", pnl)
	}
	miss := evaluatePlayHit(rule, []string{"9", "8", "7", "6", "1"}, "2\n5\n0", false, "", 0)
	if miss.Hit {
		t.Fatalf("want miss, ev=%+v", miss)
	}
}

func TestEvaluateBudingweiBetUnits(t *testing.T) {
	// 二码：选 2 号 → C(2,2)=1 注
	rule2 := resolveSSCPlayRule("budingwei", "qian3_2ma", "budingwei", "前三二码不定位")
	ev2 := evaluateBudingwei(rule2, []string{"1", "2", "3", "4", "5"}, "1,2")
	if ev2.BetUnits != 1 {
		t.Fatalf("二码 C(2,2) BetUnits=%d want 1", ev2.BetUnits)
	}
	if !ev2.Hit {
		t.Fatalf("want 二码 hit")
	}
	// 三码：选 3 号 → C(3,3)=1 注
	rule3 := resolveSSCPlayRule("budingwei", "hou3_3ma", "budingwei", "后三三码不定位")
	ev3 := evaluateBudingwei(rule3, []string{"0", "0", "7", "8", "9"}, "7,8,9")
	if ev3.BetUnits != 1 {
		t.Fatalf("三码 C(3,3) BetUnits=%d want 1", ev3.BetUnits)
	}
	if !ev3.Hit {
		t.Fatalf("want 三码 hit")
	}
	// 一码：选 2 号 → 2 注
	rule1 := resolveSSCPlayRule("budingwei", "qian4_1ma", "budingwei", "前四一码不定位")
	ev1 := evaluateBudingwei(rule1, []string{"1", "2", "3", "4", "5"}, "1,9")
	if ev1.BetUnits != 2 {
		t.Fatalf("一码 BetUnits=%d want 2", ev1.BetUnits)
	}
	if rule1.SegmentStart != 0 || rule1.SegmentLen != 4 {
		t.Fatalf("前四 segment=%d,%d want 0,4", rule1.SegmentStart, rule1.SegmentLen)
	}
	ruleHou4 := resolveSSCPlayRule("budingwei", "99", "budingwei", "后四一码不定位")
	if ruleHou4.SegmentStart != 1 || ruleHou4.SegmentLen != 4 {
		t.Fatalf("后四 segment=%d,%d want 1,4", ruleHou4.SegmentStart, ruleHou4.SegmentLen)
	}
	// 五星二码：4 号 C(4,2)=6；仅 1 组合中 → net≈10.95-5=5.95，勿整单×16
	ruleW := resolveSSCPlayRule("budingwei", "wuxing_2ma", "budingwei", "五星二码不定位")
	evW := evaluateBudingwei(ruleW, []string{"1", "2", "0", "0", "0"}, "1,2,3,4")
	if evW.BetUnits != 6 {
		t.Fatalf("五星二码 BetUnits=%d want 6", evW.BetUnits)
	}
	if !evW.Hit {
		t.Fatalf("want 五星二码 hit")
	}
	pnl := calcPnLWithOdds(float64(evW.BetUnits), evW.Hit, evW.Odds)
	if pnl < 5.5 || pnl > 6.5 {
		t.Fatalf("五星二码 pnl=%v want ~5.95 (not 96)", pnl)
	}
}

func TestEvaluateTeshuWuxingYifan(t *testing.T) {
	rule := resolveSSCPlayRule("wuxing", "wuxing_yifan", "teshu")
	ev := evaluatePlayHit(rule, []string{"8", "8", "8", "8", "8"}, "一帆风顺", false, "", 0)
	if !ev.Hit {
		t.Fatalf("want yifan hit, ev=%+v", ev)
	}
}

func TestEvaluateSixingZu24(t *testing.T) {
	rule := resolveSSCPlayRule("sixing", "sixing_zu24", "zu24")
	ev := evaluatePlayHit(rule, []string{"0", "1", "2", "3", "9"}, "0,1,2,3,4,5,9", false, "", 0)
	if !ev.Hit {
		t.Fatalf("want zu24 hit, ev=%+v", ev)
	}
}
