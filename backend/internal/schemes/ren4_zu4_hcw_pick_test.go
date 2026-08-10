package schemes

import (
	"testing"
)

func TestBuildZu4HcwPickContent_sixing(t *testing.T) {
	t.Parallel()
	raw := []byte(`{
		"playTemplate":"ssc_std","playTypeId":"sixing","subPlayId":"zu4","catalogSubId":"zu4",
		"betMode":"zu4","playMethodLabel":"组选4","segmentLen":4,
		"runTypeId":"hot_cold_warm",
		"hotColdWarm":{
			"totalPeriods":20,
			"ranks":[[0],[1]]
		}
	}`)
	cfg := parseSchemeConfig("custom", raw, 0, 0)
	if cfg.Play.SegmentLen != 4 || cfg.Play.SegmentStart != 1 {
		t.Fatalf("sixing segment=%d+%d want start=1 len=4", cfg.Play.SegmentStart, cfg.Play.SegmentLen)
	}
	draws := [][]string{
		{"9", "1", "1", "1", "1"},
		{"9", "1", "1", "1", "1"},
		{"9", "1", "1", "1", "1"},
		{"9", "2", "2", "2", "2"},
		{"9", "2", "2", "2", "2"},
		{"9", "3", "3", "3", "3"},
		{"9", "4", "4", "4", "4"},
		{"9", "5", "5", "5", "5"},
		{"9", "6", "6", "6", "6"},
		{"9", "7", "7", "7", "7"},
	}
	pool := []string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9"}
	got, ok := buildZu4HcwPickContent(cfg, draws, pool)
	if !ok {
		t.Fatal("expected sixing zu4 hcw dual branch")
	}
	if got != "1,2" {
		t.Fatalf("content=%q want 1,2", got)
	}
	if isHotColdDigitOverall(cfg.Play) {
		t.Fatal("sixing zu4 must not use flat overall digit pool")
	}
}

func TestCountZu4DualZoneBetUnits(t *testing.T) {
	t.Parallel()
	cases := []struct {
		content string
		want    int
	}{
		{"1,2", 1},
		{"12,34", 4},
		{"1,12", 1},
		{"1,1", 0},
		{"1,234", 3},
	}
	for _, tc := range cases {
		if got := countZu4DualZoneBetUnits(tc.content); got != tc.want {
			t.Fatalf("%q units=%d want %d", tc.content, got, tc.want)
		}
	}
}

func TestShouldSkipZeroBetUnits_zu4(t *testing.T) {
	t.Parallel()
	rule := playRule{BetMode: "zu4", SubPlayID: "133", CatalogSubID: "133"}
	if !shouldSkipZeroBetUnits(rule) {
		t.Fatal("zu4 should skip zero units")
	}
}
