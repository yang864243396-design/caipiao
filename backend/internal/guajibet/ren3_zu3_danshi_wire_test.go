package guajibet

import "testing"

func TestFormatCount_ren3Zu3Danshi(t *testing.T) {
	t.Parallel()
	meta := ParseRuleMeta("ssc_std", "g011", "84", "组三单式", "任选", nil, "84")
	if !isZu3DanshiMeta(meta) {
		t.Fatalf("meta should be zu3 danshi: %+v", meta)
	}
	wire := FormatBetContentForRule(meta, "万,千,个\n112")
	if wire == "" || countRenxuanZu3DanshiWire(wire, 3) != 1 {
		t.Fatalf("112 wire=%q units=%d", wire, countRenxuanZu3DanshiWire(wire, 3))
	}
	wireBad := FormatBetContentForRule(meta, "万,千,个\n012")
	if countRenxuanZu3DanshiWire(wireBad, 3) != 0 {
		t.Fatalf("012 should count 0, wire=%q", wireBad)
	}
	if formatSSCZu3DanshiDigits("112,121,012,111") != "112" {
		t.Fatalf("filter got %q want 112", formatSSCZu3DanshiDigits("112,121,012,111"))
	}
}
