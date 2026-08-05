package schemes

import (
	"testing"
)

func TestBuildRenxuanZu12HcwPickContent(t *testing.T) {
	t.Parallel()
	raw := []byte(`{
		"playTemplate":"ssc_std","playTypeId":"g011","subPlayId":"144","catalogSubId":"144",
		"betMode":"zu12","playMethodLabel":"任四组选12","renPositionCount":4,"segmentLen":1,
		"runTypeId":"hot_cold_warm",
		"hotColdWarm":{
			"totalPeriods":20,
			"positionIdxs":[0,1,2,3],
			"ranks":[[0],[1,2]]
		}
	}`)
	cfg := parseSchemeConfig("custom", raw, 0, 0)
	// 投注选位万千百十合并计频：各位同号，1 最热、其次 2/3 → full 序约 1,2,3,...
	draws := [][]string{
		{"1", "1", "1", "1", "9"},
		{"1", "1", "1", "1", "9"},
		{"1", "1", "1", "1", "9"},
		{"2", "2", "2", "2", "9"},
		{"2", "2", "2", "2", "9"},
		{"3", "3", "3", "3", "9"},
		{"4", "4", "4", "4", "9"},
		{"5", "5", "5", "5", "9"},
		{"6", "6", "6", "6", "9"},
		{"7", "7", "7", "7", "9"},
	}
	pool := []string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9"}
	got, ok := buildRenxuanZu12HcwPickContent(cfg, draws, pool)
	if !ok {
		t.Fatal("expected zu12 hcw branch")
	}
	if got == "" {
		t.Fatal("expected non-empty dual zone")
	}
	if n := countZu12DualZoneBetUnits(got); n <= 0 {
		t.Fatalf("units=%d content=%q want >0", n, got)
	}
	// ranks[0]=[0] → 最热 1；ranks[1]=[1,2] → 2,3 → "1,23"
	if got != "1,23" {
		t.Fatalf("content=%q want 1,23", got)
	}
}

func TestBuildRenxuanZu12HcwPickContent_BetPosChangesFreq(t *testing.T) {
	t.Parallel()
	// 任四投注位须 ≥4；用两组合法选位对比合并频次差异。
	// 开奖：万=8、千=7、百=6、十=5、个=1（各期相同）→ 各位计数相等时按号码升序，最热取较小码。
	rawWQBS := []byte(`{
		"playTemplate":"ssc_std","playTypeId":"g011","subPlayId":"144","catalogSubId":"144",
		"betMode":"zu12","playMethodLabel":"任四组选12","renPositionCount":4,"segmentLen":1,
		"runTypeId":"hot_cold_warm",
		"hotColdWarm":{
			"positionIdxs":[0,1,2,3],
			"ranks":[[0],[1,2]]
		}
	}`)
	rawQBSG := []byte(`{
		"playTemplate":"ssc_std","playTypeId":"g011","subPlayId":"144","catalogSubId":"144",
		"betMode":"zu12","playMethodLabel":"任四组选12","renPositionCount":4,"segmentLen":1,
		"runTypeId":"hot_cold_warm",
		"hotColdWarm":{
			"positionIdxs":[1,2,3,4],
			"ranks":[[0],[1,2]]
		}
	}`)
	draws := [][]string{
		{"8", "7", "6", "5", "1"},
		{"8", "7", "6", "5", "1"},
		{"8", "7", "6", "5", "1"},
		{"8", "7", "6", "5", "1"},
		{"8", "7", "6", "5", "1"},
	}
	pool := []string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9"}
	cfgWQBS := parseSchemeConfig("custom", rawWQBS, 0, 0)
	gotWQBS, ok := buildRenxuanZu12HcwPickContent(cfgWQBS, draws, pool)
	if !ok || gotWQBS == "" {
		t.Fatalf("万千百十: ok=%v got=%q", ok, gotWQBS)
	}
	if gotWQBS[:1] != "5" {
		t.Fatalf("选万千百十时最热应为 5, got %q", gotWQBS)
	}
	cfgQBSG := parseSchemeConfig("custom", rawQBSG, 0, 0)
	gotQBSG, ok := buildRenxuanZu12HcwPickContent(cfgQBSG, draws, pool)
	if !ok || gotQBSG == "" {
		t.Fatalf("千百十个: ok=%v got=%q", ok, gotQBSG)
	}
	if gotQBSG[:1] != "1" {
		t.Fatalf("选千百十个时最热应为 1, got %q", gotQBSG)
	}
}

func TestShouldSkipZeroBetUnits_zu12(t *testing.T) {
	t.Parallel()
	rule := playRule{BetMode: "zu12", SubPlayID: "144", CatalogSubID: "144"}
	if !shouldSkipZeroBetUnits(rule) {
		t.Fatal("zu12 should skip zero units")
	}
}
