package guajibet

import "testing"

func TestFormatCount_ren3Zu6Danshi(t *testing.T) {
	meta := ParseRuleMeta("ssc_std", "g011", "86", "组六单式", "任选", nil, "86")
	if !isZu6DanshiMeta(meta) {
		t.Fatal("expected zu6 danshi meta")
	}
	meta.ForcedBetMode = "zuxuan_ds"
	wire := FormatBetContentForRule(meta, "万,千,个\n012,112,111,210")
	if wire != "万千个|012" {
		t.Fatalf("wire=%q want 万千个|012", wire)
	}
	if n := CountBetNums(meta, wire); n != 1 {
		t.Fatalf("bets=%d want 1", n)
	}
	if formatSSCZu6DanshiDigits("012,210,112,111") != "012" {
		t.Fatalf("filter got %q want 012", formatSSCZu6DanshiDigits("012,210,112,111"))
	}
}
