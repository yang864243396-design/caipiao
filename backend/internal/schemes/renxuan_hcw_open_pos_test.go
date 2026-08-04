package schemes

import (
	"strings"
	"testing"
)

func TestBuildRenxuanHcwOpenPosPickContent_UsesAbsoluteOpenPositions(t *testing.T) {
	// 任二直选单式：开奖选位万/个（0,4），ranks 各取最热 1 码
	cfg := parsedSchemeConfig{
		Play: playRule{
			PlayTypeID:   "g011",
			CatalogSubID: "75",
			SubPlayID:    "75",
			BetMode:      "danshi",
			SegmentLen:   2,
			PlayTemplate: "ssc_std",
		},
		HotCold: &hotColdWarmCfg{
			TotalPeriods:     20,
			Strategy:         "keep",
			OpenPositionIdxs: []int{0, 4},
			PositionIdxs:     []int{0, 1},
			Ranks:            [][]int{{0}, {0}},
		},
	}
	if !isRenxuanPerPosTriggerPlay(cfg.Play) {
		t.Fatalf("want renxuan per-pos play")
	}
	// 构造开奖：万位多 1、个位多 9
	draws := make([][]string, 10)
	for i := 0; i < 10; i++ {
		draws[i] = []string{"1", "2", "3", "4", "9"}
	}
	pool := []string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9"}
	content, ok := buildRenxuanHcwOpenPosPickContent(cfg, draws, pool)
	if !ok {
		t.Fatal("expected renxuan hcw open-pos branch")
	}
	lines := strings.Split(content, "\n")
	if len(lines) != 2 {
		t.Fatalf("lines=%v want 2", lines)
	}
	if !strings.Contains(lines[0], "1") {
		t.Fatalf("open万 freq pick=%q want contain 1", lines[0])
	}
	if !strings.Contains(lines[1], "9") {
		t.Fatalf("open个 freq pick=%q want contain 9", lines[1])
	}

	wrapped := applyRenxuanRunPositionWrap(cfg, content)
	if !strings.HasPrefix(wrapped, "万,千\n") {
		t.Fatalf("wrap=%q want prefix 万,千", wrapped)
	}
}

func TestNormalizeRenxuanHcwOpenPositionIdxs(t *testing.T) {
	got := normalizeRenxuanHcwOpenPositionIdxs([]int{4, 0, 4}, 2)
	if len(got) != 2 || got[0] != 0 || got[1] != 4 {
		t.Fatalf("got=%v want [0 4]", got)
	}
	def := normalizeRenxuanHcwOpenPositionIdxs(nil, 2)
	if len(def) != 2 || def[0] != 0 || def[1] != 1 {
		t.Fatalf("default=%v want [0 1]", def)
	}
}
