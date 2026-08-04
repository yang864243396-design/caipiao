package schemes

import "testing"

// 任二直选和值冷热：选项频次须随投注选位变化（对齐 evaluateRenxuanHezhi 的 C(选位,2)）。
func TestHotColdWarmAttributeTiers_ren2HezhiFollowsPositionIdxs(t *testing.T) {
	cfg := pickTestConfig(t, `{
		"runTypeId":"hot_cold_warm",
		"playTemplate":"ssc_std",
		"playTypeId":"g011",
		"subPlayId":"76",
		"betMode":"hezhi",
		"playMethodLabel":"任二直选和值",
		"playTypeLabel":"任选",
		"guajiGroup":"任选"
	}`)
	rule := cfg.Play
	if !isRenxuanNeedsPositionRule(rule) {
		t.Fatalf("want renxuan needs position, rule=%+v", rule)
	}
	// 开奖：万千和=9，千个和=1（万=5 千=4 百=0 十=0 个=7 → 万千=9，千个=11）
	// 用多期拉开：万千常出和值 9，千个常出和值 1
	draws := [][]string{
		{"5", "4", "0", "0", "0"}, // 万千=9，千个=4
		{"5", "4", "1", "1", "1"}, // 万千=9，千个=5
		{"5", "4", "2", "2", "2"}, // 万千=9，千个=6
		{"0", "0", "0", "0", "1"}, // 万千=0，千个=1
		{"1", "0", "0", "0", "1"}, // 万千=1，千个=1
		{"2", "0", "0", "0", "1"}, // 万千=2，千个=1
	}
	wanQian := HotColdWarmAttributeTiersForPositions(rule, draws, []int{0, 1})
	qianGe := HotColdWarmAttributeTiersForPositions(rule, draws, []int{1, 4})
	if wanQian.Mode != "attribute" || qianGe.Mode != "attribute" {
		t.Fatalf("mode wanqian=%s qiange=%s", wanQian.Mode, qianGe.Mode)
	}
	if wanQian.Counts["9"] < 3 {
		t.Fatalf("万千选位：和值9 count=%d want >=3, counts=%v", wanQian.Counts["9"], wanQian.Counts)
	}
	if qianGe.Counts["1"] < 3 {
		t.Fatalf("千个选位：和值1 count=%d want >=3, counts=%v", qianGe.Counts["1"], qianGe.Counts)
	}
	// 同一批开奖、不同选位，频次分布应不同
	if wanQian.Counts["9"] == qianGe.Counts["9"] && wanQian.Counts["1"] == qianGe.Counts["1"] {
		t.Fatalf("不同投注选位频次应不同: wanqian=%v qiange=%v", wanQian.Counts, qianGe.Counts)
	}
	if attributeHitContent(rule, "9", []int{0, 1}) != "万,千\n9" {
		t.Fatalf("hit content=%q", attributeHitContent(rule, "9", []int{0, 1}))
	}
}
