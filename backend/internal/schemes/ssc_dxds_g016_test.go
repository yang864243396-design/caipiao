package schemes

import "testing"

func TestResolveSSCPlayRule_g016WuxingSumDaxiao(t *testing.T) {
	rule := resolveSSCPlayRule("g016", "264", "daxiao", "大小单双 五星和值大小")
	if rule.SegmentLen != 5 {
		t.Fatalf("SegmentLen=%d want 5 (got start=%d)", rule.SegmentLen, rule.SegmentStart)
	}
	if rule.BetMode != "daxiao" && rule.BetMode != "dxds" && rule.BetMode != "danshuang" {
		t.Fatalf("BetMode=%q want daxiao/dxds", rule.BetMode)
	}
	// 和=1+2+3+4+5=15 → 小；和=9+9+9+9+9=45 → 大
	evSmall := evaluatePlayHit(rule, []string{"1", "2", "3", "4", "5"}, "小", false, "", 0)
	if !evSmall.Hit {
		t.Fatalf("sum15 小 want hit, got %+v", evSmall)
	}
	evBigMiss := evaluatePlayHit(rule, []string{"1", "2", "3", "4", "5"}, "大", false, "", 0)
	if evBigMiss.Hit {
		t.Fatalf("sum15 大 want miss")
	}
	evBig := evaluatePlayHit(rule, []string{"9", "9", "9", "9", "9"}, "大", false, "", 0)
	if !evBig.Hit {
		t.Fatalf("sum45 大 want hit, got %+v", evBig)
	}
}

func TestResolveSSCPlayRule_g016Hou2Dxds(t *testing.T) {
	rule := resolveSSCPlayRule("g016", "261", "dxds", "大小单双 后二大小单双")
	if rule.SegmentStart != 3 || rule.SegmentLen != 2 {
		t.Fatalf("seg=%d+%d want 3+2", rule.SegmentStart, rule.SegmentLen)
	}
}

// 线上 rule id=266、无中文 playMethod 时，仍须解析为十位+个位（start=3,len=2）
func TestResolveSSCPlayRule_g016Hou2ByRuleID266(t *testing.T) {
	rule := resolveSSCPlayRule("g016", "266", "dxds")
	if rule.SegmentStart != 3 || rule.SegmentLen != 2 {
		t.Fatalf("seg=%d+%d want 3+2 (十/个)", rule.SegmentStart, rule.SegmentLen)
	}
	// 十=6(大) 个=3(小) → 大\n小 命中；大\n大 不中
	hit := evaluatePlayHit(rule, []string{"1", "2", "3", "6", "3"}, "大\n小", false, "", 0)
	if !hit.Hit || hit.BetUnits != 1 {
		t.Fatalf("want hit units=1, got %+v", hit)
	}
	miss := evaluatePlayHit(rule, []string{"1", "2", "3", "6", "3"}, "大\n大", false, "", 0)
	if miss.Hit {
		t.Fatalf("个位小不应命中 大\\n大")
	}
}
