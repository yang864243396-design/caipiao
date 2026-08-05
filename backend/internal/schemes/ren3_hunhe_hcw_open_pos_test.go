package schemes

import (
	"strings"
	"testing"
)

func TestRen3HunheHcwUsesOpenPositions(t *testing.T) {
	t.Parallel()
	raw := []byte(`{
		"playTemplate":"ssc_std","playTypeId":"g011","subPlayId":"87","catalogSubId":"87",
		"betMode":"hunhe","playMethodLabel":"任三混合组选","playTypeLabel":"任选","guajiGroup":"任选",
		"runTypeId":"hot_cold_warm",
		"hotColdWarm":{
			"totalPeriods":20,
			"openPositionIdxs":[0,1,4],
			"positionIdxs":[0,1,4],
			"ranks":[[0],[0],[0]]
		}
	}`)
	cfg := parseSchemeConfig("custom", raw, 0, 0)
	if !isRenxuanHcwOpenPosPlay(cfg.Play) {
		t.Fatalf("want renxuan hunhe hcw open-pos, play=%+v", cfg.Play)
	}
	if isRenxuanPerPosTriggerPlay(cfg.Play) {
		t.Fatal("任三混合组选开某投某不应按位分列")
	}
	// 万位多 1、千位多 2、个位多 9；开奖选位万/千/个 → 最热 1/2/9
	draws := make([][]string, 8)
	for i := 0; i < 8; i++ {
		draws[i] = []string{"1", "2", "3", "4", "9"}
	}
	pool := []string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9"}
	content, ok := buildRenxuanHcwOpenPosPickContent(cfg, draws, pool)
	if !ok {
		t.Fatal("expected ren3 hunhe open-pos branch")
	}
	lines := strings.Split(content, "\n")
	if len(lines) != 3 {
		t.Fatalf("lines=%v want 3", lines)
	}
	if !strings.Contains(lines[0], "1") {
		t.Fatalf("open万 pick=%q want contain 1", lines[0])
	}
	if !strings.Contains(lines[1], "2") {
		t.Fatalf("open千 pick=%q want contain 2", lines[1])
	}
	if !strings.Contains(lines[2], "9") {
		t.Fatalf("open个 pick=%q want contain 9", lines[2])
	}
	wrapped := applyRenxuanRunPositionWrap(cfg, content)
	if !strings.HasPrefix(wrapped, "万,千,个\n") {
		t.Fatalf("wrap=%q want prefix 万,千,个", wrapped)
	}
}

func TestIsRenxuanHcwOpenPosPlay_zhong3HunheFalse(t *testing.T) {
	t.Parallel()
	rule := playRule{
		PlayTemplate: "ssc_std",
		PlayTypeID:   "g002",
		SubPlayID:    "23",
		BetMode:      "hunhe",
		SegmentLen:   3,
		SegmentStart: 1,
	}
	if isRenxuanHcwOpenPosPlay(rule) {
		t.Fatal("中三混合组选不应走任选开奖选位")
	}
}
