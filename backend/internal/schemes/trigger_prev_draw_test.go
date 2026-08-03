package schemes

import "testing"

// 回归：上期开奖未入库时不可用上上期映射。
// 实盘：0356 中三和值=15，但 0357 误用 0355（和值=8）投出了 "8"。
func TestTriggerBetSkipsWhenPrevBallsEmpty(t *testing.T) {
	t.Parallel()
	raw := []byte(`{
		"runTypeId":"adv_trigger_bet","playTemplate":"ssc_std",
		"playTypeId":"g002","subPlayId":"16","betMode":"hezhi",
		"triggerBet":{"mode":"alt_pos_first","rows":[
			{"enabled":true,"open":"8","pos":"8","neg":"8,9"},
			{"enabled":true,"open":"15","pos":"15","neg":"15,16"}
		]}
	}`)
	cfg := parseSchemeConfig("custom", raw, 0, 0)
	dec := resolveTriggerBetDecision(cfg, nil, "neg")
	if !dec.Skip {
		t.Fatalf("empty prev balls must skip, got content=%q", dec.Content)
	}
}

func TestTriggerBetZhong3HezhiUsesSegmentSum(t *testing.T) {
	t.Parallel()
	raw := []byte(`{
		"runTypeId":"adv_trigger_bet","playTemplate":"ssc_std",
		"playTypeId":"g002","typeId":"g002","subPlayId":"16","subId":"16","betMode":"hezhi",
		"triggerBet":{"mode":"alt_pos_first","rows":[
			{"enabled":true,"open":"8","pos":"8","neg":"8,9"},
			{"enabled":true,"open":"15","pos":"15","neg":"15,16"}
		]}
	}`)
	cfg := parseSchemeConfig("custom", raw, 0, 0)
	// 0356: 5,4,3,8,2 → 中三 4+3+8=15；上期投向 neg → 本期 pos → "15"
	balls0356 := []string{"5", "4", "3", "8", "2"}
	dec := resolveTriggerBetDecision(cfg, balls0356, "neg")
	if dec.Skip || dec.Content != "15" || dec.Direction != "pos" {
		t.Fatalf("want content=15 dir=pos, got skip=%v content=%q dir=%q play=%+v",
			dec.Skip, dec.Content, dec.Direction, cfg.Play)
	}
	// 若误把上上期 0355 (9,5,3,0,4 中三=8) 当上期，会得到 "8" —— 这是本次线上错投形态
	balls0355 := []string{"9", "5", "3", "0", "4"}
	wrong := resolveTriggerBetDecision(cfg, balls0355, "neg")
	if wrong.Content != "8" {
		t.Fatalf("stale prev sanity: want 8, got %q", wrong.Content)
	}
}

func TestHotColdAdjacentPrevMissing_triggerScenario(t *testing.T) {
	t.Parallel()
	// 当期 0357，期望上期 0356；库里最新仍是 0355 → 必须判定为缺相邻上期
	if !hotColdAdjacentPrevMissing("1014147800356", "1014147800355") {
		t.Fatal("0356 missing while latest=0355 must block")
	}
	if hotColdAdjacentPrevMissing("1014147800356", "1014147800356") {
		t.Fatal("exact prev ready must not block")
	}
	// 今日实盘：投 0109 时期望上期 0108；若库止于 0107 必须阻塞
	if !hotColdAdjacentPrevMissing("1014150300108", "1014150300107") {
		t.Fatal("0108 missing while latest=0107 must block")
	}
	// 投 0215 期望 0214；库止于 0212（缺两期）也必须阻塞，禁止回退用 0212 后三 719
	if !hotColdAdjacentPrevMissing("1014157300214", "1014157300212") {
		t.Fatal("0214 missing while latest=0212 must block")
	}
}

// TestTriggerBetHou3FushiStalePrevShape 回归 inst-1-1785503619886：
// 后三直选复式 + 前反后正；若误用 0212=07719 后三 719 会得到 789/123/901，
// 正确上期 0214=49427 后三 427 应为 456/234/789。
func TestTriggerBetHou3FushiStalePrevShape(t *testing.T) {
	t.Parallel()
	raw := []byte(`{
		"runTypeId":"adv_trigger_bet","playTemplate":"ssc_std",
		"playTypeId":"g003","subId":"26","betMode":"fushi",
		"triggerBet":{"mode":"alt_neg_first","rows":[
			{"enabled":true,"open":"0","pos":"0,1\n0,1\n0,1","neg":"0,1,2\n0,1,2\n0,1,2"},
			{"enabled":true,"open":"1","pos":"1,2\n1,2\n1,2","neg":"1,2,3\n1,2,3\n1,2,3"},
			{"enabled":true,"open":"2","pos":"2,3\n2,3\n2,3","neg":"2,3,4\n2,3,4\n2,3,4"},
			{"enabled":true,"open":"3","pos":"3,4\n3,4\n3,4","neg":"3,4,5\n3,4,5\n3,4,5"},
			{"enabled":true,"open":"4","pos":"4,5\n4,5\n4,5","neg":"4,5,6\n4,5,6\n4,5,6"},
			{"enabled":true,"open":"5","pos":"5,6\n5,6\n5,6","neg":"5,6,7\n5,6,7\n5,6,7"},
			{"enabled":true,"open":"6","pos":"6,7\n6,7\n6,7","neg":"6,7,8\n6,7,8\n6,7,8"},
			{"enabled":true,"open":"7","pos":"7,8\n7,8\n7,8","neg":"7,8,9\n7,8,9\n7,8,9"},
			{"enabled":true,"open":"8","pos":"8,9\n8,9\n8,9","neg":"8,9,0\n8,9,0\n8,9,0"},
			{"enabled":true,"open":"9","pos":"9,0\n9,0\n9,0","neg":"9,0,1\n9,0,1\n9,0,1"}
		]}
	}`)
	cfg := parseSchemeConfig("custom", raw, 0, 0)
	if cfg.Play.SegmentStart != 2 || cfg.Play.SegmentLen != 3 {
		t.Fatalf("hou3 segment want start=2 len=3, got start=%d len=%d", cfg.Play.SegmentStart, cfg.Play.SegmentLen)
	}
	// 正确：0214 → 后三 427 → 首局 neg
	want := resolveTriggerBetDecision(cfg, []string{"4", "9", "4", "2", "7"}, "")
	if want.Skip || want.Direction != "neg" || want.Content != "4,5,6\n2,3,4\n7,8,9" {
		t.Fatalf("correct prev: want 4,5,6\\n2,3,4\\n7,8,9 neg, got skip=%v content=%q dir=%q",
			want.Skip, want.Content, want.Direction)
	}
	// 错票形态：误用 0212 → 后三 719 → 789/123/901
	stale := resolveTriggerBetDecision(cfg, []string{"0", "7", "7", "1", "9"}, "")
	if stale.Content != "7,8,9\n1,2,3\n9,0,1" {
		t.Fatalf("stale prev sanity: want 7,8,9\\n1,2,3\\n9,0,1, got %q", stale.Content)
	}
}

// TestTriggerBetZhong3ZuheSkipsDisabledOpen 回归：中三组合开投重，
// 上期中三含未启用开出（8）时应 Skip；若误用上上期 0/4/2 会错投。
func TestTriggerBetZhong3ZuheSkipsDisabledOpen(t *testing.T) {
	t.Parallel()
	cfg := pickTestConfig(t, `{
		"runTypeId":"adv_trigger_bet","playTemplate":"ssc_std",
		"playTypeId":"g002","subPlayId":"18","betMode":"zuhe",
		"triggerBet":{"mode":"alt_pos_first","rows":[
			{"enabled":true,"open":"0","pos":"0\n0\n0","neg":"0,1\n0,1\n0,1"},
			{"enabled":true,"open":"1","pos":"1\n1\n1","neg":"1,2\n1,2\n1,2"},
			{"enabled":true,"open":"2","pos":"2\n2\n2","neg":"2,3\n2,3\n2,3"},
			{"enabled":true,"open":"3","pos":"3\n3\n3","neg":"3,4\n3,4\n3,4"},
			{"enabled":true,"open":"4","pos":"4\n4\n4","neg":"4,5\n4,5\n4,5"},
			{"enabled":false,"open":"5"},{"enabled":false,"open":"6"},
			{"enabled":false,"open":"7"},{"enabled":false,"open":"8"},{"enabled":false,"open":"9"}
		]}
	}`)
	// 正确上期 0108：中三 8,0,8 → 含未启用 8 → Skip
	got := resolveTriggerBetDecision(cfg, []string{"5", "8", "0", "8", "8"}, "")
	if !got.Skip {
		t.Fatalf("prev 8,0,8 must skip, got content=%q", got.Content)
	}
	// 误用上上期 0107：中三 0,4,2 → 正投 0|4|2（今日错票形态）
	stale := resolveTriggerBetDecision(cfg, []string{"4", "0", "4", "2", "2"}, "")
	if stale.Skip || stale.Content != "0\n4\n2" {
		t.Fatalf("stale prev sanity: want 0\\n4\\n2, got skip=%v content=%q", stale.Skip, stale.Content)
	}
	// 无上期球号（相邻缺库）必须 Skip，禁止回退
	empty := resolveTriggerBetDecision(cfg, nil, "")
	if !empty.Skip {
		t.Fatalf("empty prev must skip, got %q", empty.Content)
	}
}
