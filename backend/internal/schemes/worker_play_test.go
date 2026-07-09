package schemes

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestEvaluateDingwei(t *testing.T) {
	rule := playRule{PlayTypeID: "dingwei", SegmentLen: 1, PositionIdx: 0}
	balls := []string{"3", "9", "2", "7", "5"}
	ev := evaluatePlayHit(rule, balls, "1,3,7", false, "", 0)
	if !ev.Hit || ev.BetUnits != 3 {
		t.Fatalf("got %+v", ev)
	}
}

func TestEvaluateDingweiMultiline(t *testing.T) {
	rule := playRule{PlayTypeID: "dingwei", BetMode: "dingwei", SegmentLen: 1, PositionIdx: 0}
	balls := []string{"3", "9", "2", "7", "5"}
	ev := evaluatePlayHit(rule, balls, "1,3\n9\n\n\n7,8", false, "", 0)
	if !ev.Hit || ev.BetUnits != 5 {
		t.Fatalf("got %+v want hit units=5", ev)
	}
}

func TestEvaluateHou4ZhixuanFushiMultiline(t *testing.T) {
	rule := playRule{PlayTypeID: "hou4", SubPlayID: "zhixuan_fs", SegmentStart: 1, SegmentLen: 4}
	balls := []string{"0", "9", "2", "7", "5"}
	content := "9\n2\n7\n5"
	ev := evaluatePlayHit(rule, balls, content, false, "", 0)
	if !ev.Hit {
		t.Fatal("expected hit")
	}
	if ev.BetUnits != 1 {
		t.Fatalf("units=%d", ev.BetUnits)
	}
	balls2 := []string{"0", "1", "2", "3", "4"}
	ev2 := evaluatePlayHit(rule, balls2, content, false, "", 0)
	if ev2.Hit {
		t.Fatal("expected miss")
	}
}

func TestEvaluateHou4ZhixuanFushiSinglePool(t *testing.T) {
	rule := playRule{PlayTypeID: "hou4", SubPlayID: "zhixuan_fs", SegmentStart: 1, SegmentLen: 4}
	balls := []string{"0", "9", "2", "7", "5"}
	ev := evaluatePlayHit(rule, balls, "9,2,7,5", false, "", 0)
	if !ev.Hit {
		t.Fatal("expected hit")
	}
	if ev.BetUnits != 4*4*4*4 {
		t.Fatalf("units=%d want 256", ev.BetUnits)
	}
}

func TestEvaluateQian3ZhixuanDanshi(t *testing.T) {
	rule := playRule{PlayTypeID: "qian3", SubPlayID: "zhixuan_ds", SegmentStart: 0, SegmentLen: 3}
	balls := []string{"3", "9", "2", "7", "5"}
	ev := evaluatePlayHit(rule, balls, "392,123,456", false, "", 0)
	if !ev.Hit || ev.BetUnits != 3 {
		t.Fatalf("got %+v", ev)
	}
}

func TestEvaluateZhong3ZuxuanPool(t *testing.T) {
	rule := playRule{PlayTypeID: "zhong3", SubPlayID: "zuxuan_fs", SegmentStart: 1, SegmentLen: 3}
	balls := []string{"0", "9", "2", "2", "5"}
	ev := evaluatePlayHit(rule, balls, "1,2,9", false, "", 0)
	if !ev.Hit {
		t.Fatal("expected 组三 hit")
	}
}

func TestEvaluateZuxuanSortedToken(t *testing.T) {
	rule := playRule{PlayTypeID: "qian3", SubPlayID: "zuxuan_fs", SegmentStart: 0, SegmentLen: 3}
	balls := []string{"3", "9", "2", "7", "5"}
	ev := evaluatePlayHit(rule, balls, "239,123", false, "", 0)
	if !ev.Hit {
		t.Fatal("expected sorted token hit")
	}
}

func TestResolvePlayRuleSegments(t *testing.T) {
	cfg := map[string]interface{}{"playTypeId": "hou4", "subPlayId": "zhixuan_fs"}
	r := resolvePlayRule(cfg, "后四直选复式")
	if r.SegmentStart != 1 || r.SegmentLen != 4 {
		t.Fatalf("hou4: %+v", r)
	}
	cfg["playTypeId"] = "qian3"
	r = resolvePlayRule(cfg, "")
	if r.SegmentStart != 0 || r.SegmentLen != 3 {
		t.Fatalf("qian3: %+v", r)
	}
	cfg["playTypeId"] = "zhong3"
	r = resolvePlayRule(cfg, "")
	if r.SegmentStart != 1 || r.SegmentLen != 3 {
		t.Fatalf("zhong3: %+v", r)
	}
}

func TestParseSchemeConfigPlayTypes(t *testing.T) {
	raw, _ := json.Marshal(map[string]interface{}{
		"playTypeId":   "hou4",
		"subPlayId":    "zhixuan_fs",
		"schemeGroups": []string{"1,2\n3,4\n5,6\n7,8"},
	})
	cfg := parseSchemeConfig("custom", raw, 0, 0)
	if cfg.Play.SegmentLen != 4 || cfg.GroupContent == "" {
		t.Fatalf("cfg=%+v", cfg)
	}
}

func TestParseSchemeConfigRounds(t *testing.T) {
	raw, _ := json.Marshal(map[string]interface{}{
		"playTypeId":   "dingwei",
		"subPlayId":    "zhixuan_fs",
		"playMethod":   "定位胆万位",
		"schemeGroups": []string{"1,3,7", "2,4,6"},
		"rounds": []map[string]interface{}{
			{"mult": 1, "afterHit": 0, "afterMiss": 1},
			{"mult": 2, "afterHit": 0, "afterMiss": 0},
		},
	})
	cfg := parseSchemeConfig("custom", raw, 1, 1)
	if cfg.Play.PositionIdx != 0 {
		t.Fatalf("position=%d", cfg.Play.PositionIdx)
	}
	if cfg.GroupContent != "2,4,6" {
		t.Fatalf("group=%q", cfg.GroupContent)
	}
	if len(cfg.Rounds) != 2 {
		t.Fatalf("rounds=%d", len(cfg.Rounds))
	}
}

func TestCalcBetAmount(t *testing.T) {
	if calcBetAmount(3, 2, baseBetUnitYuan) != 12 {
		t.Fatalf("amount=%v", calcBetAmount(3, 2, baseBetUnitYuan))
	}
}

func TestEffectiveBetMultiple(t *testing.T) {
	base := numericFromFloat(2)
	rounds := []schemeRound{
		{Mult: 1, AfterHit: 0, AfterMiss: 1},
		{Mult: 2, AfterHit: 0, AfterMiss: 2},
		{Mult: 4, AfterHit: 0, AfterMiss: 0},
	}
	if got := effectiveBetMultiple(instanceBaseCoef(base), rounds[0]); got != 2 {
		t.Fatalf("round1 mult=%v want 2", got)
	}
	if got := effectiveBetMultiple(instanceBaseCoef(base), rounds[1]); got != 4 {
		t.Fatalf("round2 mult=%v want 4", got)
	}
	if got := effectiveBetMultiple(instanceBaseCoef(base), rounds[2]); got != 8 {
		t.Fatalf("round3 mult=%v want 8", got)
	}
	if betMultipleAsInt(2.4) != 2 {
		t.Fatalf("betMultipleAsInt")
	}
}

func TestCombinedBaseCoefWithPlanMultiplier(t *testing.T) {
	inst := numericFromFloat(3)
	if got := combinedBaseCoef(inst, 2); got != 6 {
		t.Fatalf("combined base=%v want 6", got)
	}
	if got := effectiveBetMultiple(combinedBaseCoef(inst, 2), schemeRound{Mult: 4}); got != 24 {
		t.Fatalf("effective mult=%v want 24", got)
	}
	if got := planBaseCoef(0); got != 1 {
		t.Fatalf("plan default=%v", got)
	}
}

func TestNextRoundIndexMartingale(t *testing.T) {
	rounds := []schemeRound{
		{Mult: 1, AfterHit: 0, AfterMiss: 1},
		{Mult: 2, AfterHit: 0, AfterMiss: 2},
		{Mult: 4, AfterHit: 0, AfterMiss: 0},
	}
	if nextRoundIndex(rounds, 0, true) != 0 {
		t.Fatal("hit round1 should stay")
	}
	if nextRoundIndex(rounds, 0, false) != 1 {
		t.Fatal("miss round1 should go round2")
	}
	if nextRoundIndex(rounds, 1, true) != 0 {
		t.Fatal("hit round2 should reset")
	}
	if nextRoundIndex(rounds, 1, false) != 2 {
		t.Fatal("miss round2 should go round3")
	}
	if nextRoundIndex(rounds, 2, true) != 0 {
		t.Fatal("hit round3 should reset")
	}
	if nextRoundIndex(rounds, 2, false) != 0 {
		t.Fatal("miss round3 should reset")
	}
}

func TestNormalizeSchemeRoundsOneBased(t *testing.T) {
	raw := []schemeRound{
		{Mult: 0, AfterHit: 2, AfterMiss: 1},
		{Mult: 1, AfterHit: 2, AfterMiss: 3},
		{Mult: 3, AfterHit: 2, AfterMiss: 1},
	}
	norm := normalizeSchemeRounds(raw)
	if norm[0].AfterHit != 1 || norm[0].AfterMiss != 0 {
		t.Fatalf("row0 = %+v", norm[0])
	}
	if norm[1].AfterHit != 1 || norm[1].AfterMiss != 2 {
		t.Fatalf("row1 = %+v", norm[1])
	}

	// 归一化后引擎按自定义跳转推进
	if nextRoundIndex(norm, 0, true) != 1 {
		t.Fatal("adv hit round1 -> round2")
	}
	if nextRoundIndex(norm, 0, false) != 0 {
		t.Fatal("adv miss round1 -> round1")
	}
	if nextRoundIndex(norm, 1, false) != 2 {
		t.Fatal("adv miss round2 -> round3")
	}
	if nextRoundIndex(norm, 2, true) != 1 {
		t.Fatal("adv hit round3 -> round2")
	}

	// 0-based 编译结果不被二次减一
	compiled := []schemeRound{{Mult: 1, AfterHit: 0, AfterMiss: 1}}
	if got := normalizeSchemeRounds(compiled); !reflect.DeepEqual(got, compiled) {
		t.Fatalf("compiled should stay %+v, got %+v", compiled, got)
	}
}

func TestParseSchemeConfigAdvancedRounds(t *testing.T) {
	raw, _ := json.Marshal(map[string]interface{}{
		"rounds": []map[string]interface{}{
			{"mult": 2, "afterHit": 1, "afterMiss": 2},
			{"mult": 4, "afterHit": 1, "afterMiss": 1},
		},
	})
	cfg := parseSchemeConfig("custom", raw, 0, 0)
	if len(cfg.Rounds) != 2 {
		t.Fatalf("rounds=%d", len(cfg.Rounds))
	}
	if cfg.Rounds[0].AfterHit != 0 || cfg.Rounds[0].AfterMiss != 1 {
		t.Fatalf("round0 normalized = %+v", cfg.Rounds[0])
	}
	if effectiveBetMultiple(1, cfg.Rounds[1]) != 4 {
		t.Fatalf("round2 mult=%v", effectiveBetMultiple(1, cfg.Rounds[1]))
	}
}

func TestBumpIssueNo(t *testing.T) {
	if bumpIssueNo("20231103031") != "20231103032" {
		t.Fatal(bumpIssueNo("20231103031"))
	}
}

func TestSynthDrawBallsDeterministic(t *testing.T) {
	a := synthDrawBalls("tencent_ffc", "20231103099")
	b := synthDrawBalls("tencent_ffc", "20231103099")
	if len(a) != 5 || len(b) != 5 {
		t.Fatal("need 5 balls")
	}
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("not deterministic: %v vs %v", a, b)
		}
	}
}

func TestEvaluatePositionHit(t *testing.T) {
	balls := []string{"3", "9", "2", "7", "5"}
	if !evaluatePositionHit(balls, 0, []string{"3", "1"}) {
		t.Fatal("expected hit on 万位 3")
	}
}

func TestParseContraryPicks(t *testing.T) {
	picks := parseContraryPicks("1,3,7,2,5", 0)
	if len(picks) != 9 {
		t.Fatalf("expected 9 picks, got %d", len(picks))
	}
}
