package schemes

import (
	"strings"
	"testing"
)

func TestLHCWsDuipengAttributeUniverse(t *testing.T) {
	rule := resolveLHCPlayRule("g003", "282", "ws_dp")
	if !isAttributeRandom(rule) || !isHotColdAttributePlay(rule) {
		t.Fatal("ws_dp should be attribute random + HCW")
	}
	if universeKindForRule(rule) != UniverseAttribute {
		t.Fatalf("universeKind=%q want attribute", universeKindForRule(rule))
	}
	uni := attributeUniverse(rule)
	if len(uni) != 10 {
		t.Fatalf("universe len=%d want 10", len(uni))
	}
	if uni[0] != "0" || uni[1] != "1" {
		t.Fatalf("universe head=%v", uni[:2])
	}
	if max := randomDrawCountMax(rule); max != 2 {
		t.Fatalf("randomDrawCountMax=%d want 2", max)
	}
}

func TestLHCWsDuipengRandomDraw(t *testing.T) {
	cfgJSON := `{
		"subId":"282","typeId":"g003","betMode":"ws_dp","runTypeId":"random_draw",
		"subPlayId":"282","playTypeId":"g003","playTemplate":"lhc_std",
		"randomDraw":{"counts":[2],"strategy":"every"},
		"schemeGroups":["0|1"]
	}`
	cfg := pickTestConfig(t, cfgJSON)
	for i := 0; i < 40; i++ {
		got := randomDrawContentUnderMax(cfg)
		if strings.TrimSpace(got) == "" {
			t.Fatalf("empty undermax at i=%d", i)
		}
		if !strings.Contains(got, "|") {
			t.Fatalf("want 尾A|尾B, got %q", got)
		}
		parts := strings.Split(got, "|")
		if len(parts) != 2 {
			t.Fatalf("parts=%v", parts)
		}
		if !randomDrawContentAcceptable(cfg.Play, got) {
			t.Fatalf("unacceptable %q", got)
		}
		units := countPlayWireBetUnits(cfg.Play, got)
		// 0尾×其它=20；两非 0 尾=25
		if units != 20 && units != 25 {
			t.Fatalf("units=%d want 20 or 25 content=%q", units, got)
		}
	}
}

func TestLHCWsDuipengHotColdPick(t *testing.T) {
	rule := resolveLHCPlayRule("g003", "282", "ws_dp")
	// 0 尾 10/20；1 尾 01/11 — 两期都含
	draws := [][]string{
		{"10", "1", "12", "22", "32", "42", "20"},
		{"1", "11", "13", "23", "33", "43", "30"},
	}
	res := HotColdWarmAttributeTiers(rule, draws)
	if res.Mode != "attribute" || len(res.Universe) != 10 {
		t.Fatalf("mode=%q universe=%d", res.Mode, len(res.Universe))
	}
	if res.Counts["0"] < 1 || res.Counts["1"] < 1 {
		t.Fatalf("counts 0=%d 1=%d", res.Counts["0"], res.Counts["1"])
	}

	cfg := pickTestConfig(t, `{
		"subId":"282","typeId":"g003","betMode":"ws_dp","runTypeId":"hot_cold_warm",
		"subPlayId":"282","playTypeId":"g003","playTemplate":"lhc_std",
		"hotColdWarm":{"totalPeriods":20,"strategy":"every","ranks":[[0,1]],"pool":[""]},
		"schemeGroups":["0|1"]
	}`)
	got := buildHotColdPickContent(cfg, draws)
	if !strings.Contains(got, "|") {
		t.Fatalf("want 尾A|尾B, got %q", got)
	}
	if vs := validateLHCWsDuipengBetContent(got); len(vs) > 0 {
		t.Fatalf("invalid pick %q: %s", got, vs[0].Detail)
	}
}

func TestSupportsAdvTriggerBet_lhcWsDuipeng(t *testing.T) {
	if !SupportsAdvTriggerBet("g003", "282", "连码", "二全中尾数对碰") {
		t.Fatal("g003/282 should support adv trigger")
	}
	if !SupportsAdvTriggerBet("erquanzhong", "ws_dp", "二全中", "尾数对碰") {
		t.Fatal("erquanzhong/ws_dp should support adv trigger")
	}
}

func TestTriggerOpenMatches_lhcWsDuipeng(t *testing.T) {
	rule := resolveLHCPlayRule("g003", "282", "ws_dp")
	// 特码 25 → 5 尾
	balls := []string{"3", "12", "7", "33", "41", "8", "25"}
	if !triggerOpenMatches(rule, balls, "5") {
		t.Fatal("open 5 should match tema 25")
	}
	if triggerOpenMatches(rule, balls, "0") {
		t.Fatal("open 0 should not match")
	}
}
