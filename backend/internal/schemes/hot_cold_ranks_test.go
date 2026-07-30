package schemes

import (
	"strings"
	"testing"
)

func TestBuildHotColdPickContent_byRanks(t *testing.T) {
	t.Parallel()
	// 人为构造频次：千位（pos=1）开奖序列使排序可预期
	// 球序：万千百十个 → 中三看 1,2,3
	draws := [][]string{
		{"0", "9", "0", "0", "0"},
		{"0", "9", "1", "1", "0"},
		{"0", "8", "0", "2", "0"},
		{"0", "7", "1", "0", "0"},
		{"0", "9", "2", "1", "0"},
	}
	// 千位频次：9×3, 8×1, 7×1 → 最热=9，其次 7/8（并列按号码小优先 7 再 8）
	cfg := pickTestConfig(t, `{
		"runTypeId":"hot_cold_warm",
		"playTemplate":"ssc_std",
		"playTypeId":"g002",
		"subPlayId":"2",
		"betMode":"danshi",
		"hotColdWarm":{
			"totalPeriods":20,
			"strategy":"every",
			"ranks":[[0,2],[0],[1]],
			"pool":["","",""]
		}
	}`)
	got := buildHotColdPickContent(cfg, draws)
	lines := strings.Split(got, "\n")
	if len(lines) != 3 {
		t.Fatalf("lines=%d content=%q", len(lines), got)
	}
	// 千位全序以 9 为首；名次 0,2 → 含 9 与第三名
	if !strings.Contains(lines[0], "9") {
		t.Fatalf("千位应含最热 9, got %q", lines[0])
	}
	// 换一批 draws：最热变了，名次 0 应对应新最热
	draws2 := [][]string{
		{"0", "1", "0", "0", "0"},
		{"0", "1", "0", "0", "0"},
		{"0", "1", "0", "0", "0"},
		{"0", "2", "0", "0", "0"},
	}
	got2 := buildHotColdPickContent(cfg, draws2)
	lines2 := strings.Split(got2, "\n")
	if len(lines2) < 1 || !strings.HasPrefix(lines2[0], "1") && !strings.Contains(","+lines2[0]+",", ",1,") {
		// 名次 0 应为 1
		toks := strings.Split(lines2[0], ",")
		if len(toks) == 0 || toks[0] != "1" && !containsToken(toks, "1") {
			t.Fatalf("draws 变更后千位名次0应为1, got %q", lines2[0])
		}
	}
}

func containsToken(toks []string, want string) bool {
	for _, t := range toks {
		if t == want {
			return true
		}
	}
	return false
}

func TestResolveHotColdRanks_legacyPickTypes(t *testing.T) {
	t.Parallel()
	hc := &hotColdWarmCfg{PickTypes: []string{"hot"}}
	ranks := resolveHotColdRanks(hc, 0, 10)
	if len(ranks) != 5 {
		t.Fatalf("hot half of 10 want 5 ranks, got %v", ranks)
	}
	for i, r := range ranks {
		if r != i {
			t.Fatalf("ranks[%d]=%d want %d", i, r, i)
		}
	}
	hc2 := &hotColdWarmCfg{Ranks: [][]int{{0, 9, 3}}}
	got := resolveHotColdRanks(hc2, 0, 10)
	if len(got) != 3 || got[0] != 0 || got[1] != 9 || got[2] != 3 {
		t.Fatalf("explicit ranks=%v", got)
	}
}

func TestResolveHotColdWarm_parsesRanksKeepsEmptyPool(t *testing.T) {
	t.Parallel()
	cfg := pickTestConfig(t, `{
		"runTypeId":"hot_cold_warm",
		"playTypeId":"g002",
		"subPlayId":"2",
		"betMode":"danshi",
		"hotColdWarm":{
			"totalPeriods":20,
			"ranks":[[0,1],[],[2]],
			"pool":["x","","y"]
		}
	}`)
	if cfg.HotCold == nil {
		t.Fatal("HotCold nil")
	}
	if len(cfg.HotCold.Ranks) != 3 || len(cfg.HotCold.Ranks[0]) != 2 || len(cfg.HotCold.Ranks[1]) != 0 || len(cfg.HotCold.Ranks[2]) != 1 {
		t.Fatalf("ranks=%v", cfg.HotCold.Ranks)
	}
	if len(cfg.HotCold.Pool) != 3 || cfg.HotCold.Pool[1] != "" {
		t.Fatalf("pool should keep empty slot: %v", cfg.HotCold.Pool)
	}
	enabled := hotColdPositionEnabled(cfg.HotCold, 3)
	if !enabled[0] || enabled[1] || !enabled[2] {
		t.Fatalf("enabled=%v", enabled)
	}
}
