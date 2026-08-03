package schemes

import (
	"strings"
	"testing"

	"caipiao/backend/internal/db/sqlcdb"
)

func TestHotColdBaodanSingleDanFromMultiRanks(t *testing.T) {
	// 与 def-1-1785593157388：ranks 勾了 5 档，不得拼出多胆撞「投注数字不合规」
	raw := `{
		"runTypeId":"hot_cold_warm","playTemplate":"ssc_std",
		"playTypeId":"g004","subPlayId":"45","betMode":"baodan",
		"playMethodLabel":"前二组选包胆",
		"hotColdWarm":{"totalPeriods":20,"ranks":[[0,1,2,3,4]],"strategy":"keep"}
	}`
	cfg := pickTestConfig(t, raw)
	if !isBaodanPlayRule(cfg.Play) {
		t.Fatalf("want baodan, rule=%+v", cfg.Play)
	}
	draws := [][]string{
		{"1", "3", "5", "7", "9"},
		{"1", "2", "4", "6", "8"},
		{"0", "1", "3", "5", "7"},
		{"1", "3", "5", "0", "2"},
		{"9", "1", "3", "4", "5"},
	}
	dec := pickHotColdWarmFromDraws(cfg, sqlcdb.SchemeInstance{Kind: "custom"}, draws)
	content := strings.TrimSpace(dec.Content)
	if content == "" {
		t.Fatal("empty baodan pick")
	}
	if strings.Contains(content, ",") || strings.Contains(content, "\n") || len(content) != 1 {
		t.Fatalf("包胆冷热须单胆, got %q", content)
	}
	if content[0] < '0' || content[0] > '9' {
		t.Fatalf("invalid dan %q", content)
	}
}
