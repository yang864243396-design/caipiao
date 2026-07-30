package schemes

import "testing"

func TestPickTriggerBetZuxuanPool_skipsBelowMin(t *testing.T) {
	t.Parallel()
	cfg := parsedSchemeConfig{
		Play: playRule{
			PlayTemplate: "ssc_std",
			PlayTypeID:   "g002",
			SubPlayID:    "261",
			BetMode:      "zu6",
			SegmentStart: 1,
			SegmentLen:   3,
		},
		Trigger: &triggerBetCfg{
			Mode: "alt_neg_first",
			Rows: []triggerRow{
				{Open: "0", Pos: "0", Neg: "0,1", Enabled: true},
				{Open: "1", Pos: "1", Neg: "1,2", Enabled: true},
				{Open: "3", Pos: "1,2,3", Neg: "3,4,5", Enabled: true},
			},
		},
	}
	enabled := cfg.Trigger.Rows

	// 上期中三含 0 → 命中第一行，正投 "0" 不足 3 码 → Skip
	dec := pickTriggerBetZuxuanPool(cfg, enabled, []string{"9", "0", "8"}, "pos")
	if !dec.Skip {
		t.Fatalf("pos 1-digit pool want Skip, got content=%q", dec.Content)
	}
	// 反投 "0,1" 仍不足 → Skip
	dec = pickTriggerBetZuxuanPool(cfg, enabled, []string{"9", "0", "8"}, "neg")
	if !dec.Skip {
		t.Fatalf("neg 2-digit pool want Skip, got content=%q", dec.Content)
	}
	// 开出 3 → 正投 1,2,3 合法
	dec = pickTriggerBetZuxuanPool(cfg, enabled, []string{"1", "3", "5"}, "pos")
	if dec.Skip || dec.Content != "1,2,3" {
		t.Fatalf("valid pool want content=1,2,3, got skip=%v content=%q", dec.Skip, dec.Content)
	}
}

func TestEvaluateZu6_zeroUnitsBelowMin(t *testing.T) {
	t.Parallel()
	rule := playRule{PlayTemplate: "ssc_std", PlayTypeID: "g002", SubPlayID: "261", BetMode: "zu6", SegmentStart: 1, SegmentLen: 3}
	ev := evaluateZu6(rule, []string{"1", "2", "3", "4", "5"}, "3,4")
	if ev.BetUnits != 0 {
		t.Fatalf("2-digit zu6 units=%d want 0", ev.BetUnits)
	}
	ev = evaluateZu6(rule, []string{"1", "2", "3", "4", "5"}, "3,4,5")
	if ev.BetUnits != 1 {
		t.Fatalf("3-digit zu6 units=%d want 1", ev.BetUnits)
	}
}
