package schemes

import (
	"strings"
	"testing"
)

func TestLHCSwDuipengAttributeUniverse(t *testing.T) {
	rule := resolveLHCPlayRule("g003", "283", "sw_dp")
	if !isAttributeRandom(rule) || !isHotColdAttributePlay(rule) {
		t.Fatal("sw_dp should be attribute random + HCW")
	}
	if universeKindForRule(rule) != UniverseAttribute {
		t.Fatalf("universeKind=%q want attribute", universeKindForRule(rule))
	}
	uni := attributeUniverse(rule)
	if len(uni) != 22 {
		t.Fatalf("universe len=%d want 22 (12肖+10尾)", len(uni))
	}
	if uni[0] != "马" || uni[12] != "0" {
		t.Fatalf("universe head/mid=%v / %v", uni[:2], uni[12:14])
	}
	if max := randomDrawCountMax(rule); max != 2 {
		t.Fatalf("randomDrawCountMax=%d want 2", max)
	}
}

func TestLHCSwDuipengRandomDraw(t *testing.T) {
	cfgJSON := `{
		"subId":"283","typeId":"g003","betMode":"sw_dp","runTypeId":"random_draw",
		"subPlayId":"283","playTypeId":"g003","playTemplate":"lhc_std",
		"randomDraw":{"counts":[2],"strategy":"every"},
		"schemeGroups":["马|0"]
	}`
	cfg := pickTestConfig(t, cfgJSON)
	for i := 0; i < 40; i++ {
		got := randomDrawContentUnderMax(cfg)
		if strings.TrimSpace(got) == "" {
			t.Fatalf("empty undermax at i=%d", i)
		}
		if vs := validateLHCSwDuipengBetContent(got); len(vs) > 0 {
			t.Fatalf("invalid %q: %s", got, vs[0].Detail)
		}
		if !randomDrawContentAcceptable(cfg.Play, got) {
			t.Fatalf("unacceptable %q", got)
		}
		units := countPlayWireBetUnits(cfg.Play, got)
		want := countLHCSwDuipengBetUnits(got)
		if want <= 0 || units != want {
			t.Fatalf("units=%d want %d content=%q", units, want, got)
		}
	}
}

func TestLHCSwDuipengHotColdPick(t *testing.T) {
	rule := resolveLHCPlayRule("g003", "283", "sw_dp")
	draws := [][]string{
		{"10", "1", "12", "22", "32", "42", "20"}, // 0尾 + 羊(12)等
		{"1", "13", "25", "37", "49", "11", "30"}, // 马 + 0/1 尾
	}
	res := HotColdWarmAttributeTiers(rule, draws)
	if res.Mode != "attribute" || len(res.Universe) != 22 {
		t.Fatalf("mode=%q universe=%d", res.Mode, len(res.Universe))
	}

	cfg := pickTestConfig(t, `{
		"subId":"283","typeId":"g003","betMode":"sw_dp","runTypeId":"hot_cold_warm",
		"subPlayId":"283","playTypeId":"g003","playTemplate":"lhc_std",
		"hotColdWarm":{"totalPeriods":20,"strategy":"every","ranks":[[0,1,2,3]],"pool":[""]},
		"schemeGroups":["马|0"]
	}`)
	got := buildHotColdPickContent(cfg, draws)
	if vs := validateLHCSwDuipengBetContent(got); len(vs) > 0 {
		t.Fatalf("invalid pick %q: %s", got, vs[0].Detail)
	}
}

func TestSupportsAdvTriggerBet_lhcSwDuipeng(t *testing.T) {
	if !SupportsAdvTriggerBet("g003", "283", "连码", "二全中生尾对碰") {
		t.Fatal("g003/283 should support adv trigger")
	}
	if !SupportsAdvTriggerBet("erquanzhong", "sw_dp", "二全中", "生尾对碰") {
		t.Fatal("erquanzhong/sw_dp should support adv trigger")
	}
}

func TestTriggerOpenMatches_lhcSwDuipeng(t *testing.T) {
	rule := resolveLHCPlayRule("g003", "283", "sw_dp")
	// 特码 25 → 马 + 5 尾
	balls := []string{"3", "12", "7", "33", "41", "8", "25"}
	if !triggerOpenMatches(rule, balls, "马") {
		t.Fatal("open 马 should match tema 25")
	}
	if !triggerOpenMatches(rule, balls, "5") {
		t.Fatal("open 5 should match tema 25")
	}
	if triggerOpenMatches(rule, balls, "蛇") {
		t.Fatal("open 蛇 should not match")
	}
}
