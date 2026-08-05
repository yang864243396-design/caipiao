package guajibet

import "testing"

func TestFormatCount_ren3Hunhe(t *testing.T) {
	meta := ParseRuleMeta("ssc_std", "g011", "87", "混合组选", "任选", nil, "87")
	meta.ForcedBetMode = "hunhe"
	wire := FormatBetContentForRule(meta, "万,千,个\n012,210,111")
	if wire != "万千个|012" {
		t.Fatalf("wire=%q want 万千个|012", wire)
	}
	if n := CountBetNums(meta, wire); n != 1 {
		t.Fatalf("bets=%d want 1", n)
	}
}

// def-1-1785836661744：任三混合组选 2 注须 solo=true（≤hunheSoloMaxBets）；4 注须 false。
func TestResolveSolo_ren3HunheByBets(t *testing.T) {
	meta := ParseRuleMeta("ssc_std", "g011", "87", "任三混合组选", "任选", nil, "87")
	meta.ForcedBetMode = "hunhe"
	cases := []struct {
		wire string
		want int
		solo bool
	}{
		{"万千个|183", 1, true},
		{"万千个|183,838", 2, true},
		{"万千个|183,838,012", 3, true},
		{"万千个|183,838,012,345", 4, false},
	}
	for _, c := range cases {
		n := CountBetNums(meta, c.wire)
		if n != c.want {
			t.Fatalf("%q bets=%d want %d", c.wire, n, c.want)
		}
		if got := ResolveSolo(meta, c.wire, n); got != c.solo {
			t.Fatalf("%q bets=%d ResolveSolo=%v want %v", c.wire, n, got, c.solo)
		}
	}
}
