package schemes

import "testing"

// 复现：线上 g010 + 数字 subId=54、文案「万千」——冷热若只传数字 id，位解析失败会全 0。
func TestHotColdWarmAttributeTiers_longhuGuajiNumeric(t *testing.T) {
	rule := resolveSSCPlayRule("g010", "54", "longhu", "万千")
	if p1, p2, _ := longhuPositions(rule.CatalogSubID); p1 != 0 || p2 != 1 {
		t.Fatalf("CatalogSubID=%q positions=%d,%d want 0,1 (万千)", rule.CatalogSubID, p1, p2)
	}
	draws := [][]string{
		{"8", "3", "1", "2", "9"}, // 龙
		{"1", "5", "2", "3", "4"}, // 虎
		{"4", "4", "1", "2", "3"}, // 和（龙虎斗不计和选项）
		{"9", "0", "1", "2", "3"}, // 龙
	}
	res := HotColdWarmAttributeTiers(rule, draws)
	if res.Mode != "attribute" {
		t.Fatalf("mode=%s", res.Mode)
	}
	if res.Counts["龙"] < 1 || res.Counts["虎"] < 1 {
		t.Fatalf("counts=%v want 龙/虎 >0", res.Counts)
	}
	if res.Counted < 2 {
		t.Fatalf("counted=%d want >=2", res.Counted)
	}
}

func TestHotColdWarmAttributeTiers_longhuheGuaji(t *testing.T) {
	rule := resolveSSCPlayRule("g010", "55", "longhuhe", "万千和")
	draws := [][]string{
		{"8", "3", "1", "2", "9"}, // 龙
		{"1", "5", "2", "3", "4"}, // 虎
		{"4", "4", "1", "2", "3"}, // 和
	}
	res := HotColdWarmAttributeTiers(rule, draws)
	if res.Mode != "attribute" {
		t.Fatalf("mode=%s universe=%v", res.Mode, res.Universe)
	}
	if len(res.Universe) != 3 {
		t.Fatalf("universe=%v want 龙/虎/和", res.Universe)
	}
	if res.Counts["龙"] != 1 || res.Counts["虎"] != 1 || res.Counts["和"] != 1 {
		t.Fatalf("counts=%v want 龙=1 虎=1 和=1", res.Counts)
	}
}

func TestResolveHotColdCatalogSubID_longhuPrefersParsableHint(t *testing.T) {
	rule := playRule{BetMode: "longhu", CatalogSubID: "54"}
	in := HotColdWarmTiersInput{
		PlayTypeID:      "g010",
		SubPlayID:       "54",
		CatalogSubID:    "54",
		PlayMethodLabel: "万千",
		BetMode:         "longhu",
	}
	got := resolveHotColdCatalogSubID(rule, in)
	if p1, p2, _ := longhuPositions(got); p1 != 0 || p2 != 1 {
		t.Fatalf("got=%q positions=%d,%d want 万千", got, p1, p2)
	}
}

func TestEvaluateLonghu_hePlayKeepsDragonTigerHits(t *testing.T) {
	rule := resolveSSCPlayRule("longhu", "lh_wanqian_he", "longhuhe")
	ev := evaluatePlayHit(rule, []string{"8", "3", "1", "2", "9"}, "龙", false, "", 0)
	if !ev.Hit {
		t.Fatalf("龙虎和玩法投龙且万>千应命中, ev=%+v cat=%q", ev, rule.CatalogSubID)
	}
	tie := evaluatePlayHit(rule, []string{"4", "4", "1", "2", "3"}, "和", false, "", 0)
	if !tie.Hit {
		t.Fatalf("和应命中, ev=%+v", tie)
	}
}
