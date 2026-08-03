package schemes

import "testing"

func TestNormalizeZuxuanDanshiDigitPool(t *testing.T) {
	rule := resolveSSCPlayRule("g004", "43", "zuxuan_ds", "前二组选单式")
	if rule.BetMode != "zuxuan_ds" || rule.SegmentLen != 2 {
		t.Fatalf("rule=%+v", rule)
	}
	if got := normalizeZhixuanDanshiContent(rule, "5,6"); got != "56" {
		t.Fatalf("comma pool → %q want 56", got)
	}
	if got := normalizeZhixuanDanshiContent(rule, "5\n6"); got != "56" {
		t.Fatalf("newline pool → %q want 56", got)
	}
	if got := normalizeZhixuanDanshiContent(rule, "11"); got != "" {
		t.Fatalf("duizi → %q want empty", got)
	}
	if got := normalizeZhixuanDanshiContent(rule, "12,21"); got != "12" {
		t.Fatalf("form dedup → %q want 12", got)
	}
}

func TestLegacySubModeZuxuanDanshi(t *testing.T) {
	if got := legacySubMode("43", "zuxuan_ds"); got != "zuxuan_ds" {
		t.Fatalf("legacySubMode=%q want zuxuan_ds", got)
	}
}

func TestHotColdDigitOverallIncludesZuxuanDanshi(t *testing.T) {
	rule := resolveSSCPlayRule("g004", "43", "zuxuan_ds", "前二组选单式")
	if !isHotColdDigitOverall(rule) {
		t.Fatal("前二组选单式冷热应走整体号频，勿按位出 5\\n6")
	}
}

func TestRandomWholeTicketsZuxuanDanshiNoDuizi(t *testing.T) {
	rule := resolveSSCPlayRule("g004", "43", "zuxuan_ds", "前二组选单式")
	for i := 0; i < 50; i++ {
		raw := randomWholeTickets(rule, 20)
		if raw == "" {
			t.Fatal("empty tickets")
		}
		for _, tok := range splitContentTokens(raw) {
			if len(tok) == 2 && tok[0] == tok[1] {
				t.Fatalf("got duizi %q in %q", tok, raw)
			}
		}
	}
}
