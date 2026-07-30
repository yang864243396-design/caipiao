package schemes

import (
	"strings"
	"testing"

	"caipiao/backend/internal/db/sqlcdb"
)

func TestRandomZu3HonorsCounts2(t *testing.T) {
	t.Parallel()
	cfg := pickTestConfig(t, `{
		"runTypeId":"random_draw","playTemplate":"ssc_std",
		"playTypeId":"g002","subPlayId":"19","betMode":"zu3",
		"randomDraw":{"counts":[2],"strategy":"every"}
	}`)
	if !isZuxuanPoolRandom(cfg.Play) {
		t.Fatalf("want zuxuan pool, rule=%+v", cfg.Play)
	}
	if zuxuanPoolMinPick(cfg.Play) != 2 {
		t.Fatalf("zu3 minPick=%d want 2", zuxuanPoolMinPick(cfg.Play))
	}
	for i := 0; i < 20; i++ {
		dec := pickRandomDraw(cfg, sqlcdb.SchemeInstance{Kind: "custom"})
		toks := strings.Split(dec.Content, ",")
		if len(toks) != 2 {
			t.Fatalf("counts=[2] want 2 nums, got %d: %q", len(toks), dec.Content)
		}
	}
}

func TestHotColdZu3LocksConfiguredPool(t *testing.T) {
	t.Parallel()
	cfg := pickTestConfig(t, `{
		"runTypeId":"hot_cold_warm","playTemplate":"ssc_std",
		"playTypeId":"g002","subPlayId":"19","betMode":"zu3",
		"hotColdWarm":{
			"pickTypes":["cold"],"pool":["0,1,6,7,9"],
			"ranks":[[5,6,7,8,9]],"strategy":"after_miss","totalPeriods":20
		}
	}`)
	draws := [][]string{
		{"1", "0", "2", "3", "4"}, {"5", "6", "7", "8", "9"},
		{"0", "1", "6", "7", "9"}, {"2", "3", "4", "5", "0"},
		{"1", "1", "1", "1", "1"}, {"9", "9", "9", "9", "9"},
		{"6", "6", "6", "6", "6"}, {"7", "0", "1", "6", "7"},
		{"3", "3", "3", "3", "3"}, {"4", "4", "4", "4", "4"},
		{"8", "8", "8", "8", "8"}, {"0", "0", "0", "0", "0"},
		{"1", "6", "7", "9", "0"}, {"2", "2", "2", "2", "2"},
		{"5", "5", "5", "5", "5"}, {"6", "7", "9", "0", "1"},
		{"9", "0", "1", "6", "7"}, {"0", "1", "6", "7", "9"},
		{"1", "2", "3", "4", "5"}, {"6", "7", "8", "9", "0"},
	}
	got := buildHotColdPickContent(cfg, draws)
	if got == "" {
		t.Fatal("empty pick")
	}
	allow := map[string]bool{"0": true, "1": true, "6": true, "7": true, "9": true}
	for _, tok := range strings.Split(got, ",") {
		if !allow[tok] {
			t.Fatalf("pool-locked pick leaked %q in %q", tok, got)
		}
	}
}

func TestTriggerZu3MatchesAnySegmentDigit(t *testing.T) {
	t.Parallel()
	cfg := pickTestConfig(t, `{
		"runTypeId":"adv_trigger_bet","playTemplate":"ssc_std",
		"playTypeId":"g002","subPlayId":"19","betMode":"zu3",
		"triggerBet":{"mode":"alt_neg_first","rows":[
			{"enabled":true,"open":"0","pos":"0","neg":"0,1"},
			{"enabled":true,"open":"1","pos":"1","neg":"1,2"},
			{"enabled":true,"open":"2","pos":"2","neg":"2,3"},
			{"enabled":true,"open":"3","pos":"3","neg":"3,4"},
			{"enabled":true,"open":"4","pos":"4","neg":"4,5"},
			{"enabled":false,"open":"5"},{"enabled":false,"open":"6"},
			{"enabled":false,"open":"7"},{"enabled":false,"open":"8"},{"enabled":false,"open":"9"}
		]}
	}`)
	if !isZuxuanPoolTriggerPlay(cfg.Play) {
		t.Fatalf("zu3 should use zuxuan pool trigger, rule=%+v", cfg.Play)
	}
	// alt_neg_first 无历史投向 → 首局 neg
	dec := resolveTriggerBetDecision(cfg, []string{"5", "8", "0", "8", "8"}, "")
	if dec.Skip {
		t.Fatal("segment contains 0 (enabled) must not skip")
	}
	if dec.Direction != "neg" || dec.Content != "0,1" {
		t.Fatalf("want neg/0,1 got dir=%q content=%q", dec.Direction, dec.Content)
	}
	skip := resolveTriggerBetDecision(cfg, []string{"1", "5", "6", "7", "8"}, "")
	if !skip.Skip {
		t.Fatalf("no enabled open in segment must skip, got %q", skip.Content)
	}
}

func TestTriggerHunhePerPosLikeFushi(t *testing.T) {
	t.Parallel()
	cfg := pickTestConfig(t, `{
		"runTypeId":"adv_trigger_bet","playTemplate":"ssc_std",
		"playTypeId":"g002","subPlayId":"23","betMode":"hunhe",
		"triggerBet":{"mode":"always_pos","rows":[
			{"enabled":true,"open":"1","pos":"4\n5\n6","neg":""},
			{"enabled":true,"open":"2","pos":"4\n5\n6","neg":""},
			{"enabled":true,"open":"3","pos":"4\n5\n6","neg":""},
			{"enabled":false,"open":"0"},{"enabled":false,"open":"4"},
			{"enabled":false,"open":"5"},{"enabled":false,"open":"6"},
			{"enabled":false,"open":"7"},{"enabled":false,"open":"8"},{"enabled":false,"open":"9"}
		]}
	}`)
	if !triggerBetUsesPosition(cfg.Play) {
		t.Fatalf("hunhe should use per-pos trigger like fushi; play=%+v", cfg.Play)
	}
	// 上期千/百/十 = 1/2/3 → 各位取对应开出行 → 4\n5\n6，再展成整注 456
	dec := resolveTriggerBetDecision(cfg, []string{"9", "1", "2", "3", "0"}, "")
	want := "4\n5\n6"
	if dec.Skip || dec.Content != want {
		t.Fatalf("per-seg pick skip=%v content=%q want %q", dec.Skip, dec.Content, want)
	}
	expanded := normalizeZhixuanDanshiContent(cfg.Play, dec.Content)
	if expanded != "456" {
		t.Fatalf("expanded=%q want 456", expanded)
	}
}

func TestHotColdHunheUsesPerPosNotOverall(t *testing.T) {
	t.Parallel()
	rule := playRule{
		PlayTemplate: "ssc_std",
		PlayTypeID:   "g002",
		SubPlayID:    "23",
		BetMode:      "hunhe",
		SegmentLen:   3,
		SegmentStart: 1,
	}
	if isHotColdDigitOverall(rule) {
		t.Fatal("混合组选冷热应按千/百/十分列，不应走整体号码池")
	}
	cfg := pickTestConfig(t, `{
		"runTypeId":"hot_cold_warm","playTemplate":"ssc_std",
		"playTypeId":"g002","subPlayId":"23","betMode":"hunhe",
		"hotColdWarm":{"pickTypes":["hot"],"totalPeriods":20,"strategy":"every"}
	}`)
	draws := [][]string{
		{"1", "2", "3", "4", "5"},
		{"1", "2", "3", "4", "5"},
		{"6", "7", "8", "9", "0"},
	}
	got := buildHotColdPickContent(cfg, draws)
	lines := strings.Split(got, "\n")
	if len(lines) != 3 {
		t.Fatalf("hunhe hot/cold want 3 lines (千/百/十), got %d content=%q", len(lines), got)
	}
}

func TestHunheNormalizeFiltersBaoziAndEmpty(t *testing.T) {
	t.Parallel()
	rule := playRule{
		PlayTemplate: "ssc_std",
		PlayTypeID:   "g002",
		SubPlayID:    "23",
		BetMode:      "hunhe",
		SegmentLen:   3,
		SegmentStart: 1,
	}
	// 含豹子：保留非豹子，形态去重
	got := normalizeZhixuanDanshiContent(rule, "123,111,321,222")
	if got != "123" {
		t.Fatalf("filter baozi+shape got %q want 123", got)
	}
	// 按位号池仅能展成豹子 → 空串（本期不投）
	empty := normalizeZhixuanDanshiContent(rule, "5\n5\n5")
	if empty != "" {
		t.Fatalf("solo baozi pool got %q want empty", empty)
	}
	ev := evaluateHunhe(rule, []string{"1", "5", "5", "5", "0"}, empty)
	if ev.BetUnits != 0 {
		t.Fatalf("evaluate empty hunhe BetUnits=%d want 0", ev.BetUnits)
	}
}

func TestAdvancePickAfterMissKeepsCurrentPickOnHit(t *testing.T) {
	t.Parallel()
	raw := `{"runTypeId":"hot_cold_warm","playTypeId":"g002","subPlayId":"19","betMode":"zu3",
		"hotColdWarm":{"strategy":"after_miss","pool":["0,1,6,7,9"],"pickTypes":["cold"]}}`
	inst := sqlcdb.SchemeInstance{Kind: "custom", PickIndex: 0}
	_, cur, _ := AdvancePickAfterFormalSettlement("custom", []byte(raw), inst, "0,1,6", true)
	if cur != "0,1,6" {
		t.Fatalf("after_miss hit should lock current_pick, got %q", cur)
	}
	_, cur2, _ := AdvancePickAfterFormalSettlement("custom", []byte(raw), inst, "0,1,6", false)
	if cur2 != "" {
		t.Fatalf("after_miss miss should clear current_pick, got %q", cur2)
	}
}
