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
}
