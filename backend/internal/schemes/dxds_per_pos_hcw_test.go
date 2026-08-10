package schemes

import (
	"strings"
	"testing"
)

func TestBuildHotColdPickContent_hou2DxdsPerPos(t *testing.T) {
	rule := resolveSSCPlayRule("g016", "266", "dxds", "后二大小单双")
	if !isPerPosDxdsRandom(rule) {
		t.Fatal("后二大小单双应为按位 dxds")
	}
	if isHotColdAttributePlay(rule) {
		t.Fatal("按位大小单双不应走扁选属性冷热")
	}
	cfg := parsedSchemeConfig{
		Play: rule,
		HotCold: &hotColdWarmCfg{
			TotalPeriods: 20,
			Ranks:        [][]int{{0}, {0}},
		},
	}
	// 十位多为大(6+)，个位多为小(3-)
	draws := [][]string{
		{"1", "2", "3", "8", "1"},
		{"1", "2", "3", "9", "2"},
		{"1", "2", "3", "7", "0"},
		{"1", "2", "3", "1", "8"},
	}
	got := buildHotColdPickContent(cfg, draws)
	lines := splitGroupLines(got)
	if len(lines) != 2 {
		t.Fatalf("content=%q want 2 lines", got)
	}
	if lines[0] == "" || lines[1] == "" {
		t.Fatalf("每位应有 1 选项 content=%q", got)
	}
	if strings.Contains(lines[0], ",") || strings.Contains(lines[1], ",") {
		t.Fatalf("每位仅 1 选项 content=%q", got)
	}
}
