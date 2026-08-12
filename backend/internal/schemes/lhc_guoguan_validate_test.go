package schemes

import (
	"strings"
	"testing"
)

func TestValidateSchemeBetContent_lhcGuoguanRequiresSixFixedPositions(t *testing.T) {
	cfg := []byte(`{"playTemplate":"lhc_std","typeId":"g004","subId":"guoguan","betMode":"guoguan","playMethodLabel":"过关"}`)
	valid := "大,单,,红波,,绿波"
	if vs := ValidateSchemeBetContent("custom", cfg, valid, 0); len(vs) > 0 {
		t.Fatalf("six-position guoguan %q should pass: %+v", valid, vs)
	}
	if !validateLHCGroupContent(resolveLHCPlayRule("g004", "guoguan", "guoguan"), valid) {
		t.Fatalf("direct bet validation should accept %q", valid)
	}

	for _, content := range []string{"大,,,,,", "大,单,,豹子,,双", "大,单,,红波,,绿波,小"} {
		vs := ValidateSchemeBetContent("custom", cfg, content, 0)
		if len(vs) == 0 {
			t.Fatalf("invalid guoguan %q should be rejected", content)
		}
		if !strings.Contains(vs[0].Detail, "正码") {
			t.Fatalf("invalid guoguan %q should explain position rule, got %+v", content, vs)
		}
	}
}

func TestGenerateRandomDrawContent_lhcGuoguanKeepsSixPositionWireFormat(t *testing.T) {
	cfg := parsedSchemeConfig{
		Play:   resolveLHCPlayRule("g004", "guoguan", "guoguan"),
		Random: &randomDrawCfg{Counts: []int{1}},
	}
	content := generateRandomDrawContent(cfg, 0)
	positions, ok := parseLHCGuoguanPositions(content)
	if !ok {
		t.Fatalf("random guoguan should generate a valid six-position content, got %q", content)
	}
	selected := 0
	for _, pick := range positions {
		if pick != "" {
			selected++
		}
	}
	if selected != 2 {
		t.Fatalf("random guoguan count=1 must clamp to two selected positions, got %q", content)
	}
}

func TestBuildHotColdPickContent_lhcGuoguanKeepsSixPositionWireFormat(t *testing.T) {
	cfg := parsedSchemeConfig{
		Play: resolveLHCPlayRule("g004", "guoguan", "guoguan"),
		HotCold: &hotColdWarmCfg{Ranks: [][]int{
			{0}, {0}, {}, {}, {}, {0},
		}},
	}
	content := buildHotColdPickContent(cfg, [][]string{
		{"01", "02", "03", "04", "05", "06", "07"},
		{"49", "48", "47", "46", "45", "44", "43"},
	})
	positions, ok := parseLHCGuoguanPositions(content)
	if !ok {
		t.Fatalf("hot/cold guoguan should generate valid six-position content, got %q", content)
	}
	selected := 0
	for _, pick := range positions {
		if pick != "" {
			selected++
		}
	}
	if selected != 3 {
		t.Fatalf("expected the configured three positions, got %q", content)
	}
}

func TestTriggerOpenMatches_lhcGuoguanUsesZhengmaOneNumber(t *testing.T) {
	rule := resolveLHCPlayRule("g004", "guoguan", "guoguan")
	balls := []string{"26", "01", "02", "03", "04", "05", "06"}
	if !triggerOpenMatches(rule, balls, "26") {
		t.Fatal("正码1=26 should match 开出 26")
	}
	if triggerOpenMatches(rule, balls, "27") {
		t.Fatal("正码1=26 must not match 开出 27")
	}
	if triggerOpenMatches(rule, balls, "大") {
		t.Fatal("正码1=26 must not match an attribute open condition")
	}
}
