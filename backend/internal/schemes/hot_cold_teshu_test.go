package schemes

import "testing"

func TestHotColdWarmAttributeTiers_qian3Teshu(t *testing.T) {
	cfg := pickTestConfig(t, `{"runTypeId":"hot_cold_warm","playTypeId":"g001","subPlayId":"12","betMode":"teshu","playTemplate":"ssc_std","playMethodLabel":"前三特殊号"}`)
	cfg.Play.SegmentLen = 3
	cfg.Play.SegmentStart = 0
	draws := [][]string{
		{"1", "1", "1", "2", "3"},
		{"1", "2", "2", "3", "4"},
		{"1", "2", "3", "4", "5"},
		{"9", "8", "7", "6", "5"},
	}
	res := HotColdWarmAttributeTiers(cfg.Play, draws)
	t.Logf("mode=%s universe=%v counts=%v hot=%v cold=%v counted=%d", res.Mode, res.Universe, res.Counts, res.Hot, res.Cold, res.Counted)
	if res.Mode != "attribute" {
		t.Fatalf("mode=%s", res.Mode)
	}
	if len(res.Universe) != 3 {
		t.Fatalf("universe=%v want 豹子/对子/顺子", res.Universe)
	}
	if res.Counts["豹子"] < 1 {
		t.Fatalf("豹子 count=%d want >=1", res.Counts["豹子"])
	}
}

func TestHotColdWarmTiersServiceResolve_qian3Teshu(t *testing.T) {
	rule := resolveSSCPlayRule("g001", "12", "teshu", "前三特殊号")
	t.Logf("rule=%+v universe=%v", rule, attributeUniverse(rule))
	if rule.SegmentStart != 0 || rule.SegmentLen != 3 {
		t.Fatalf("segment=%d,%d want 0,3", rule.SegmentStart, rule.SegmentLen)
	}
	if len(attributeUniverse(rule)) != 3 {
		t.Fatalf("universe empty/wrong betMode=%q", rule.BetMode)
	}
	if got := inferAttributeBetModeFromLabel("前三特殊号"); got != "teshu" {
		t.Fatalf("infer betMode=%q want teshu", got)
	}
}

// 前端特殊号 UI segmentLen=1 不得覆盖前三=3，否则冷热次数全 0。
func TestAttributeUsesInputSegmentLen(t *testing.T) {
	if attributeUsesInputSegmentLen("teshu") {
		t.Fatal("teshu must keep resolved SegmentLen")
	}
	if !attributeUsesInputSegmentLen("hezhi") {
		t.Fatal("hezhi may use request SegmentLen for universe bounds")
	}
}
