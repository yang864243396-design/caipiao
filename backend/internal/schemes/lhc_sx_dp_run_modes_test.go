package schemes

import (
	"strings"
	"testing"
)

func TestLHCSxDuipengAttributeUniverse(t *testing.T) {
	rule := resolveLHCPlayRule("g003", "281", "sx_dp")
	if !isAttributeRandom(rule) || !isHotColdAttributePlay(rule) {
		t.Fatal("sx_dp should be attribute random + HCW")
	}
	if universeKindForRule(rule) != UniverseAttribute {
		t.Fatalf("universeKind=%q want attribute", universeKindForRule(rule))
	}
	uni := attributeUniverse(rule)
	if len(uni) != 12 {
		t.Fatalf("universe len=%d want 12", len(uni))
	}
	if uni[0] != "马" || uni[1] != "蛇" {
		t.Fatalf("universe head=%v", uni[:2])
	}
	if max := randomDrawCountMax(rule); max != 2 {
		t.Fatalf("randomDrawCountMax=%d want 2", max)
	}
}

func TestLHCSxDuipengRandomDraw(t *testing.T) {
	cfgJSON := `{
		"subId":"281","typeId":"g003","betMode":"sx_dp","runTypeId":"random_draw",
		"subPlayId":"281","playTypeId":"g003","playTemplate":"lhc_std",
		"randomDraw":{"counts":[2],"strategy":"every"},
		"schemeGroups":["马|蛇"]
	}`
	cfg := pickTestConfig(t, cfgJSON)
	for i := 0; i < 40; i++ {
		got := randomDrawContentUnderMax(cfg)
		if strings.TrimSpace(got) == "" {
			t.Fatalf("empty undermax at i=%d", i)
		}
		if !strings.Contains(got, "|") {
			t.Fatalf("want 肖A|肖B, got %q", got)
		}
		parts := strings.Split(got, "|")
		if len(parts) != 2 {
			t.Fatalf("parts=%v", parts)
		}
		if !randomDrawContentAcceptable(cfg.Play, got) {
			t.Fatalf("unacceptable %q", got)
		}
		units := countPlayWireBetUnits(cfg.Play, got)
		if units != 16 && units != 20 {
			t.Fatalf("units=%d want 16 or 20 content=%q", units, got)
		}
	}
}

func TestLHCSxDuipengHotColdPick(t *testing.T) {
	rule := resolveLHCPlayRule("g003", "281", "sx_dp")
	// 马号码 01/13/25；蛇 02/14；龙 03 — 两期都含马+蛇
	draws := [][]string{
		{"1", "2", "10", "20", "30", "40", "13"},
		{"1", "14", "11", "21", "31", "41", "25"},
	}
	res := HotColdWarmAttributeTiers(rule, draws)
	if res.Mode != "attribute" || len(res.Universe) != 12 {
		t.Fatalf("mode=%q universe=%d", res.Mode, len(res.Universe))
	}
	if res.Counts["马"] < 1 || res.Counts["蛇"] < 1 {
		t.Fatalf("counts 马=%d 蛇=%d", res.Counts["马"], res.Counts["蛇"])
	}

	cfg := pickTestConfig(t, `{
		"subId":"281","typeId":"g003","betMode":"sx_dp","runTypeId":"hot_cold_warm",
		"subPlayId":"281","playTypeId":"g003","playTemplate":"lhc_std",
		"hotColdWarm":{"totalPeriods":20,"strategy":"every","ranks":[[0,1]],"pool":[""]},
		"schemeGroups":["马|蛇"]
	}`)
	got := buildHotColdPickContent(cfg, draws)
	if !strings.Contains(got, "|") {
		t.Fatalf("want 肖A|肖B, got %q", got)
	}
	if vs := validateLHCSxDuipengBetContent(got); len(vs) > 0 {
		t.Fatalf("invalid pick %q: %s", got, vs[0].Detail)
	}
}

func TestSupportsAdvTriggerBet_lhcSxDuipeng(t *testing.T) {
	if !SupportsAdvTriggerBet("g003", "281", "连码", "二全中生肖对碰") {
		t.Fatal("g003/281 should support adv trigger")
	}
	if !SupportsAdvTriggerBet("erquanzhong", "sx_dp", "二全中", "生肖对碰") {
		t.Fatal("erquanzhong/sx_dp should support adv trigger")
	}
	if SupportsAdvTriggerBet("erquanzhong", "tuotou", "二全中", "拖头") {
		t.Fatal("tuotou should stay closed")
	}
}

func TestTriggerOpenMatches_lhcSxDuipeng(t *testing.T) {
	rule := resolveLHCPlayRule("g003", "281", "sx_dp")
	// 特码 25 → 马
	balls := []string{"3", "12", "7", "33", "41", "8", "25"}
	if !triggerOpenMatches(rule, balls, "马") {
		t.Fatal("open 马 should match tema 25")
	}
	if triggerOpenMatches(rule, balls, "蛇") {
		t.Fatal("open 蛇 should not match")
	}
}
