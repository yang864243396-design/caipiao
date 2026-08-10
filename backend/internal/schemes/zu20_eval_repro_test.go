package schemes

import "testing"

func TestEvaluateZu20BetUnits_repro(t *testing.T) {
	raw := []byte(`{"playTemplate":"ssc_std","playTypeId":"g015","subPlayId":"159","catalogSubId":"159","betMode":"zu20","typeId":"g015","subId":"159"}`)
	cfg := parseSchemeConfig("custom", raw, 0, 0)
	t.Logf("BetMode=%q SegmentLen=%d SubPlayID=%q", cfg.Play.BetMode, cfg.Play.SegmentLen, cfg.Play.SubPlayID)
	balls := []string{"1", "2", "3", "4", "5"}
	ev := evaluatePlayHit(cfg.Play, balls, "123,321", false, "", 0)
	t.Logf("evaluate BetUnits=%d Hit=%v", ev.BetUnits, ev.Hit)
	wire := countPlayWireBetUnits(cfg.Play, "123,321")
	t.Logf("wire units=%d", wire)
	if wire != 3 {
		t.Fatalf("wire=%d want 3", wire)
	}
	if ev.BetUnits != 3 {
		t.Fatalf("evaluate units=%d want 3 (def-1-1786010363463 repro)", ev.BetUnits)
	}
	if !shouldSkipZeroBetUnits(cfg.Play) {
		t.Fatal("zu20 should skip zero units instead of pausing")
	}
}
