package schemes

import (
	"testing"

	"caipiao/backend/internal/guajibet"
)

func TestLHCTemaHcwUniverse(t *testing.T) {
	uni := lhcTemaHcwUniverse()
	if len(uni) != 68 {
		t.Fatalf("universe len=%d want 68", len(uni))
	}
	if uni[0] != "01" || uni[48] != "49" {
		t.Fatalf("nums head/tail=%q/%q", uni[0], uni[48])
	}
	if uni[49] != "尾双" || uni[64] != "总分双" {
		t.Fatalf("attrs=%q…%q", uni[49], uni[64])
	}
	if uni[65] != "红波" || uni[67] != "绿波" {
		t.Fatalf("waves=%v", uni[65:])
	}
}

func TestIsHotColdAttributePlayTema(t *testing.T) {
	rule := resolveLHCPlayRule("tema", "tema_a", "tema")
	if !isHotColdAttributePlay(rule) {
		t.Fatal("tema should be attribute HCW")
	}
	if len(attributeUniverse(rule)) != 68 {
		t.Fatalf("attributeUniverse len=%d", len(attributeUniverse(rule)))
	}
}

func TestLHCTemaRandomDrawMax68(t *testing.T) {
	rule := resolveLHCPlayRule("tema", "tema_a", "tema")
	if !isAttributeRandom(rule) {
		t.Fatal("tema should be attribute random")
	}
	if max := randomDrawCountMax(rule); max != 68 {
		t.Fatalf("randomDrawCountMax=%d want 68", max)
	}
	content := randomAttributeContent(rule, 68)
	nums, attrs, waves := guajibet.ParseLHCTemaParts(content)
	if n := len(nums) + len(attrs) + len(waves); n != 68 {
		t.Fatalf("full pick parts=%d (nums=%d attrs=%d waves=%d) content=%q", n, len(nums), len(attrs), len(waves), content)
	}
	// 目录数字 id（无 betMode）也应推断为 tema，上限 68
	inferred := resolveLHCPlayRule("g001", "272", "")
	if inferred.BetMode != "tema" {
		t.Fatalf("infer betMode=%q want tema", inferred.BetMode)
	}
	if max := randomDrawCountMax(inferred); max != 68 {
		t.Fatalf("g001/272 max=%d want 68", max)
	}
}

func TestEvaluateLHCTemaAttrWave(t *testing.T) {
	rule := resolveLHCPlayRule("tema", "tema_a", "tema")
	// 特码 25：大/单/合大(2+5=7)/尾大/蓝波；总分 3+12+25+33+41+7+25=146 → 总分小
	balls := []string{"3", "12", "25", "33", "41", "7", "25"}
	for _, tc := range []struct {
		content string
		hit     bool
	}{
		{"25||", true},
		{"大", true},
		{"小", false},
		{"蓝波", true},
		{"红波", false},
		{"合大", true},
		{"尾大", true},
		{"尾小", false},
		{"总分小", true},
		{"总分大", false},
		{"01|大|蓝波", true},
		{"01|小|红波", false},
	} {
		ev := evaluatePlayHit(rule, balls, tc.content, false, "", 0)
		if ev.Hit != tc.hit {
			t.Fatalf("content=%q hit=%v want %v", tc.content, ev.Hit, tc.hit)
		}
	}
}

func TestHotColdWarmAttributeTiersTema(t *testing.T) {
	rule := resolveLHCPlayRule("tema", "tema_a", "tema")
	draws := [][]string{
		{"1", "2", "3", "4", "5", "6", "07"}, // 特码 7：小/单/红
		{"1", "2", "3", "4", "5", "6", "08"}, // 特码 8：小/双/红
	}
	res := HotColdWarmAttributeTiers(rule, draws)
	if res.Mode != "attribute" {
		t.Fatalf("mode=%q", res.Mode)
	}
	if len(res.Universe) != 68 {
		t.Fatalf("universe=%d", len(res.Universe))
	}
	if res.Counts["07"] != 1 || res.Counts["08"] != 1 {
		t.Fatalf("num counts 07=%d 08=%d", res.Counts["07"], res.Counts["08"])
	}
	if res.Counts["小"] != 2 {
		t.Fatalf("小 count=%d want 2", res.Counts["小"])
	}
	if res.Counts["红波"] != 2 {
		t.Fatalf("红波 count=%d want 2", res.Counts["红波"])
	}
	if res.Counts["大"] != 0 {
		t.Fatalf("大 count=%d want 0", res.Counts["大"])
	}
}

func TestEvaluateLHCTema49Tie(t *testing.T) {
	rule := resolveLHCPlayRule("tema", "tema_a", "tema")
	balls := []string{"3", "12", "25", "33", "41", "7", "49"}
	evBig := evaluatePlayHit(rule, balls, "大", false, "", 0)
	if evBig.Hit {
		t.Fatal("49 should not hit 大 (和局)")
	}
	evNum := evaluatePlayHit(rule, balls, "49", false, "", 0)
	if !evNum.Hit {
		t.Fatal("49 number should hit")
	}
	evWave := evaluatePlayHit(rule, balls, "绿波", false, "", 0)
	if !evWave.Hit {
		t.Fatal("49 is green wave")
	}
}
