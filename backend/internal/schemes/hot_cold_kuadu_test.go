package schemes

import "testing"

func TestHotColdKuaduQianhou3NotAllZero(t *testing.T) {
	rule := resolveSSCPlayRule("g012", "97", "kuadu", "直选跨度")
	if rule.SegmentLen != 3 {
		t.Fatalf("SegmentLen=%d want 3", rule.SegmentLen)
	}
	polluted := rule
	polluted.SegmentLen = 1

	draws := [][]string{
		{"1", "2", "9", "0", "5"}, // 前三跨度 8；后三跨度 5
		{"3", "3", "3", "7", "1"}, // 前三 0；后三 6
		{"0", "9", "1", "2", "8"}, // 前三 9；后三 7
		{"4", "5", "6", "4", "5"}, // 前三 2；后三 1
	}
	bad := HotColdWarmAttributeTiers(polluted, draws)
	if bad.Counts["0"] != len(draws) {
		t.Fatalf("polluted SegmentLen=1 should pile on 0, counts=%+v", bad.Counts)
	}

	good := HotColdWarmAttributeTiers(rule, draws)
	if good.Mode != "attribute" {
		t.Fatalf("mode=%s", good.Mode)
	}
	if good.Counts["0"] == len(draws) && good.Counts["8"] == 0 {
		t.Fatalf("counts all on 0: %+v", good.Counts)
	}
	if good.Counts["8"] < 1 || good.Counts["9"] < 1 {
		t.Fatalf("want spans 8/9 counted, counts=%+v", good.Counts)
	}
}

// 回归：请求体 segmentLen=1 时，SSC 跨度仍按 resolve 的 3 位计频。
func TestApplyHotColdWarmInput_sscKuaduIgnoresSegmentLen1(t *testing.T) {
	in := HotColdWarmTiersInput{
		PlayTypeID:      "g012",
		SubPlayID:       "97",
		PlayTemplate:    "ssc_std",
		BetMode:         "kuadu",
		PlayMethodLabel: "直选跨度",
		NumberPoolMin:   0,
		NumberPoolMax:   9,
		SegmentLen:      1,
	}
	rule := resolveSSCPlayRule(in.PlayTypeID, in.SubPlayID, in.BetMode, in.PlayMethodLabel)
	rule = applyHotColdWarmInputOverrides(rule, in)
	if rule.SegmentLen != 3 {
		t.Fatalf("SegmentLen=%d want 3 (request segmentLen=1 must not override SSC kuadu)", rule.SegmentLen)
	}
}
