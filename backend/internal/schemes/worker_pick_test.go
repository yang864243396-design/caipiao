package schemes

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"caipiao/backend/internal/db/sqlcdb"
)

func TestNormalizeRunTypeID(t *testing.T) {
	cases := map[string]string{
		"fixed_rotate":     RunTypeFixedRotate,
		"adv_fixed_rotate": RunTypeAdvFixedRotate,
		"adv_trigger_bet":  RunTypeAdvTriggerBet,
		"hot_cold_warm":    RunTypeHotColdWarm,
		"random_draw":      RunTypeRandomDraw,
		"builtin_plan":     RunTypeBuiltinPlan,
		"fixed_number":     RunTypeFixedNumber,
		// 废弃值映射（Q9=B）
		"batch_fixed":   RunTypeAdvFixedRotate,
		"dynamic_chase": RunTypeAdvFixedRotate,
		"plan_follow":   RunTypeAdvFixedRotate,
		// 兜底
		"":        RunTypeAdvFixedRotate,
		"unknown": RunTypeAdvFixedRotate,
	}
	for in, want := range cases {
		if got := NormalizeRunTypeID(in); got != want {
			t.Errorf("NormalizeRunTypeID(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestValidateRunTypePlayMatrix(t *testing.T) {
	if err := ValidateRunTypePlay(RunTypeAdvTriggerBet, "dingwei", "dingwei_ge", "", ""); err != nil {
		t.Errorf("trigger+dingwei should pass: %v", err)
	}
	if err := ValidateRunTypePlay(RunTypeAdvTriggerBet, "longhu", "lh_1v10", "", ""); err != nil {
		t.Errorf("trigger+longhu should pass: %v", err)
	}
	if err := ValidateRunTypePlay(RunTypeAdvTriggerBet, "pc28_20", "hezhi", "", ""); err != nil {
		t.Errorf("trigger+pc28 hezhi should pass: %v", err)
	}
	if err := ValidateRunTypePlay(RunTypeAdvTriggerBet, "pc28_28", "dxds", "", ""); err != nil {
		t.Errorf("trigger+pc28 dxds should pass: %v", err)
	}
	if err := ValidateRunTypePlay(RunTypeAdvTriggerBet, "pc28_20", "teshu", "", ""); err == nil {
		t.Error("trigger+pc28 teshu should fail")
	}
	if err := ValidateRunTypePlay(RunTypeAdvTriggerBet, "hou4", "hou4_zhixuan_fs", "", ""); err == nil {
		t.Error("trigger+hou4 should fail")
	}
	if err := ValidateRunTypePlay(RunTypeHotColdWarm, "longhu", "lh_1v10", "", ""); err == nil {
		t.Error("hotcold+longhu should fail")
	}
	if err := ValidateRunTypePlay(RunTypeHotColdWarm, "dingwei", "dingwei_ge", "", ""); err != nil {
		t.Errorf("hotcold+dingwei should pass: %v", err)
	}
	if err := ValidateRunTypePlay(RunTypeFixedRotate, "longhu", "lh_1v10", "", ""); err != nil {
		t.Errorf("fixed_rotate+longhu should pass: %v", err)
	}
	// rules/v2 同步后的 guajiGroup / label
	if err := ValidateRunTypePlay(RunTypeAdvTriggerBet, "g006", "13", "一星", "定位胆"); err != nil {
		t.Errorf("trigger+一星 should pass: %v", err)
	}
	if err := ValidateRunTypePlay(RunTypeAdvTriggerBet, "g010", "233", "2.0模式", "和值"); err != nil {
		t.Errorf("trigger+2.0模式和值 should pass: %v", err)
	}
	if err := ValidateRunTypePlay(RunTypeHotColdWarm, "g008", "x", "龙虎", "万位"); err == nil {
		t.Error("hotcold+龙虎 group should fail")
	}
	// 冷热温/随机：仅按位产号子玩法
	if err := ValidateRunTypePlay(RunTypeHotColdWarm, "g001", "1", "前三码", "前三直选复式"); err != nil {
		t.Errorf("hotcold+前三直选复式 should pass: %v", err)
	}
	if err := ValidateRunTypePlay(RunTypeRandomDraw, "g001", "5", "前三码", "前三组合"); err != nil {
		t.Errorf("random+前三组合 should pass: %v", err)
	}
	if err := ValidateRunTypePlay(RunTypeHotColdWarm, "g001", "2", "前三码", "前三直选单式"); err == nil {
		t.Error("hotcold+前三直选单式 should fail")
	}
	if err := ValidateRunTypePlay(RunTypeRandomDraw, "g001", "3", "前三码", "前三组三"); err == nil {
		t.Error("random+前三组三 should fail")
	}
	if err := ValidateRunTypePlay(RunTypeHotColdWarm, "g007", "r2", "任选", "任二直选复式"); err != nil {
		t.Errorf("hotcold+任二直选复式 should pass: %v", err)
	}
	if err := ValidateRunTypePlay(RunTypeHotColdWarm, "g007", "r2z", "任选", "任二组选复式"); err == nil {
		t.Error("hotcold+任二组选复式 should fail")
	}
}

func pickTestConfig(t *testing.T, raw string) parsedSchemeConfig {
	t.Helper()
	return parseSchemeConfig("custom", []byte(raw), 0, 0)
}

func TestPickFixedRotateEveryPeriod(t *testing.T) {
	cfg := pickTestConfig(t, `{"runTypeId":"fixed_rotate","playTypeId":"dingwei","subPlayId":"sub_ge","schemeGroups":["1,2","3,4","5,6"]}`)
	inst := sqlcdb.SchemeInstance{Kind: "custom", PickIndex: 0}

	dec := pickFixedRotate(cfg, inst)
	if dec.Content != "1,2" {
		t.Fatalf("first pick = %q, want 1,2", dec.Content)
	}
	// Q2=C：每期必换组（与命中无关）
	for _, hit := range []bool{true, false} {
		idx, _, _ := advancePickState(cfg, inst, dec, hit)
		if idx != 1 {
			t.Errorf("hit=%v next pick index = %d, want 1", hit, idx)
		}
	}
	inst.PickIndex = 2
	dec = pickFixedRotate(cfg, inst)
	if dec.Content != "5,6" {
		t.Fatalf("third pick = %q, want 5,6", dec.Content)
	}
	idx, _, _ := advancePickState(cfg, inst, dec, false)
	if idx != 0 {
		t.Errorf("wrap pick index = %d, want 0", idx)
	}
}

func TestPickJushuListNavigation(t *testing.T) {
	cfg := pickTestConfig(t, `{
		"runTypeId":"adv_fixed_rotate","playTypeId":"dingwei","subPlayId":"sub_ge",
		"jushuList":[
			{"ju":1,"content":"0,1","afterHit":1,"afterMiss":2},
			{"ju":2,"content":"2,3","afterHit":1,"afterMiss":9}
		]}`)
	inst := sqlcdb.SchemeInstance{Kind: "custom", PickIndex: 0}

	dec := pickJushuList(cfg, inst)
	if dec.Content != "0,1" {
		t.Fatalf("ju1 content = %q", dec.Content)
	}
	idx, _, _ := advancePickState(cfg, inst, dec, false)
	if idx != 2 {
		t.Fatalf("after miss → ju %d, want 2", idx)
	}

	inst.PickIndex = 2
	dec = pickJushuList(cfg, inst)
	if dec.Content != "2,3" {
		t.Fatalf("ju2 content = %q", dec.Content)
	}
	// 跳转到不存在的局（9）→ 回第 1 局【Q3】
	idx, _, _ = advancePickState(cfg, inst, dec, false)
	if idx != 1 {
		t.Fatalf("missing ju fallback = %d, want 1", idx)
	}
	// 中后回第 1 局
	idx, _, _ = advancePickState(cfg, inst, dec, true)
	if idx != 1 {
		t.Fatalf("after hit → ju %d, want 1", idx)
	}
}

func TestJushuLegacyDerivation(t *testing.T) {
	// 存量无 jushuList：由 schemeGroups + rounds 换形（v8 §8）
	cfg := pickTestConfig(t, `{
		"runTypeId":"batch_fixed","playTypeId":"dingwei","subPlayId":"sub_ge",
		"schemeGroups":["1","2","3"],
		"rounds":[{"mult":1,"afterHit":0,"afterMiss":1},{"mult":2,"afterHit":0,"afterMiss":2},{"mult":4,"afterHit":0,"afterMiss":2}]}`)
	if cfg.RunTypeID != RunTypeAdvFixedRotate {
		t.Fatalf("legacy runType = %q", cfg.RunTypeID)
	}
	if len(cfg.Jushu) != 3 {
		t.Fatalf("jushu rows = %d, want 3", len(cfg.Jushu))
	}
	if cfg.Jushu[0].AfterMiss != 2 || cfg.Jushu[0].AfterHit != 1 {
		t.Errorf("row0 jumps = hit %d miss %d, want 1/2", cfg.Jushu[0].AfterHit, cfg.Jushu[0].AfterMiss)
	}
}

func TestTriggerDirectionStateMachine(t *testing.T) {
	cases := []struct {
		mode string
		last string
		want string
	}{
		{"always_pos", "", "pos"},
		{"always_pos", "neg", "pos"},
		{"always_neg", "", "neg"},
		{"alt_pos_first", "", "pos"},
		{"alt_pos_first", "pos", "neg"},
		{"alt_pos_first", "neg", "pos"},
		{"alt_neg_first", "", "neg"},
		{"alt_neg_first", "neg", "pos"},
		{"alt_neg_first", "pos", "neg"},
	}
	for _, c := range cases {
		if got := nextTriggerDirection(c.mode, c.last); got != c.want {
			t.Errorf("mode=%s last=%s got %s want %s", c.mode, c.last, got, c.want)
		}
	}
}

func TestPickRandomDrawKeepsCurrentPick(t *testing.T) {
	cfg := pickTestConfig(t, `{"runTypeId":"random_draw","playTypeId":"dingwei","subPlayId":"sub_ge","randomDraw":{"counts":[3],"strategy":"keep"}}`)
	inst := sqlcdb.SchemeInstance{Kind: "custom", CurrentPick: "1,2,3"}
	dec := pickRandomDraw(cfg, inst)
	if dec.Content != "1,2,3" {
		t.Fatalf("keep strategy should reuse current pick, got %q", dec.Content)
	}

	// 无保号时生成 3 个去重号码
	inst.CurrentPick = ""
	dec = pickRandomDraw(cfg, inst)
	tokens := strings.Split(dec.Content, ",")
	if len(tokens) != 3 {
		t.Fatalf("random tokens = %v, want 3", tokens)
	}
	seen := map[string]bool{}
	for _, tk := range tokens {
		if seen[tk] {
			t.Fatalf("duplicate token in %v", tokens)
		}
		seen[tk] = true
	}

	// 换号策略推进
	for _, c := range []struct {
		strategy string
		hit      bool
		keep     bool
	}{
		{"every", true, false},
		{"every", false, false},
		{"keep", false, true},
		{"after_hit", true, false},
		{"after_hit", false, true},
		{"after_miss", false, false},
		{"after_miss", true, true},
	} {
		cfg.Random.Strategy = c.strategy
		_, cur, _ := advancePickState(cfg, inst, dec, c.hit)
		if c.keep && cur == "" {
			t.Errorf("strategy=%s hit=%v should keep pick", c.strategy, c.hit)
		}
		if !c.keep && cur != "" {
			t.Errorf("strategy=%s hit=%v should clear pick", c.strategy, c.hit)
		}
	}
}

func TestHotColdWarmPoolAndRotate(t *testing.T) {
	cfg := pickTestConfig(t, `{"runTypeId":"hot_cold_warm","playTypeId":"dingwei","subPlayId":"sub_ge","hotColdWarm":{"totalPeriods":50,"pool":["1,5,9"],"winRotate":true}}`)
	inst := sqlcdb.SchemeInstance{Kind: "custom"}
	dec := pickHotColdWarm(cfg, inst)
	if dec.Content != "1,5,9" {
		t.Fatalf("pool content = %q", dec.Content)
	}
	// 中奖轮换：池内号码 +1 循环并持久化
	_, cur, _ := advancePickState(cfg, inst, dec, true)
	if cur != "2,6,0" {
		t.Fatalf("rotated pool = %q, want 2,6,0", cur)
	}
	// 未中不持久化（保持空值 → 下期继续读配置池，运行中改池即时生效）
	_, cur, _ = advancePickState(cfg, inst, dec, false)
	if cur != "" {
		t.Fatalf("miss pool persisted = %q, want empty", cur)
	}
	// 已轮换过的池在未中时保持
	inst.CurrentPick = "2,6,0"
	_, cur, _ = advancePickState(cfg, inst, dec, false)
	if cur != "2,6,0" {
		t.Fatalf("rotated pool should keep on miss, got %q", cur)
	}
}

func TestHotColdWarmTiers(t *testing.T) {
	draws := [][]string{
		{"0", "0", "0", "0", "7"},
		{"0", "0", "0", "0", "7"},
		{"0", "0", "0", "0", "7"},
		{"0", "0", "0", "0", "3"},
		{"0", "0", "0", "0", "3"},
		{"0", "0", "0", "0", "1"},
	}
	pool := []string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9"}
	hot, warm, cold := hotColdWarmTiers(draws, 4, pool)
	if len(hot)+len(warm)+len(cold) != 10 {
		t.Fatalf("tiers total = %d", len(hot)+len(warm)+len(cold))
	}
	if hot[0] != "7" || hot[1] != "3" {
		t.Errorf("hot = %v, want 7,3 leading", hot)
	}
}

func TestCompileBetMultiplierRounds(t *testing.T) {
	var payload map[string]interface{}
	_ = json.Unmarshal([]byte(`{"kind":"2","simple":{"multiples":"1,2,4"}}`), &payload)
	rounds := compileBetMultiplierRounds(payload, nil)
	want := []schemeRound{
		{Mult: 1, AfterHit: 0, AfterMiss: 1},
		{Mult: 2, AfterHit: 0, AfterMiss: 2},
		{Mult: 4, AfterHit: 0, AfterMiss: 0},
	}
	if !reflect.DeepEqual(rounds, want) {
		t.Fatalf("compiled rounds = %+v, want %+v", rounds, want)
	}

	// P2：中翻倍 on_win
	payload = nil
	_ = json.Unmarshal([]byte(`{"kind":"2","simple":{"multiples":"2,4,8","advanceMode":"on_win"}}`), &payload)
	rounds = compileBetMultiplierRounds(payload, nil)
	wantWin := []schemeRound{
		{Mult: 2, AfterHit: 1, AfterMiss: 0},
		{Mult: 4, AfterHit: 2, AfterMiss: 0},
		{Mult: 8, AfterHit: 0, AfterMiss: 0},
	}
	if !reflect.DeepEqual(rounds, wantWin) {
		t.Fatalf("on_win rounds = %+v, want %+v", rounds, wantWin)
	}

	payload = nil
	_ = json.Unmarshal([]byte(`{"kind":"0","newbie":{"profitTable":[{"mult":"1"},{"mult":"3"},{"mult":"9"}]}}`), &payload)
	rounds = compileBetMultiplierRounds(payload, nil)
	if len(rounds) != 3 || rounds[2].Mult != 9 {
		t.Fatalf("newbie compiled = %+v", rounds)
	}

	// P1：kind=0 但已写入 simple.multiples → 优先简单表
	payload = nil
	_ = json.Unmarshal([]byte(`{"kind":"0","simple":{"multiples":"2,4,8"},"newbie":{"profitTable":[{"mult":"1"},{"mult":"3"}]}}`), &payload)
	rounds = compileBetMultiplierRounds(payload, nil)
	if len(rounds) != 3 || rounds[0].Mult != 2 || rounds[2].Mult != 8 {
		t.Fatalf("simple preferred over newbie = %+v", rounds)
	}

	// 高级倍投：首次选择模板 → 注入默认轮次（1-based 跳转）
	payload = nil
	_ = json.Unmarshal([]byte(`{"kind":"3","advanced":{"selectedId":"tpl_demo_plan_4"}}`), &payload)
	rounds = compileBetMultiplierRounds(payload, nil)
	wantAdv := defaultAdvancedBetMultiplierRounds()
	if !reflect.DeepEqual(rounds, wantAdv) {
		t.Fatalf("advanced default = %+v, want %+v", rounds, wantAdv)
	}

	// 已在高级倍投且已有轮次 → 不覆盖
	existing := map[string]interface{}{
		"betMultiplier": map[string]interface{}{"kind": "3"},
		"rounds": []interface{}{
			map[string]interface{}{"mult": 5.0, "afterHit": 1.0, "afterMiss": 2.0},
		},
	}
	if rounds := compileBetMultiplierRounds(payload, existing); rounds != nil {
		t.Fatalf("should preserve existing advanced rounds, got %+v", rounds)
	}

	// 从简单倍投切换 → 覆盖为高级默认
	existingSimple := map[string]interface{}{
		"betMultiplier": map[string]interface{}{"kind": "2"},
		"rounds": []interface{}{
			map[string]interface{}{"mult": 1.0, "afterHit": 0.0, "afterMiss": 1.0},
		},
	}
	rounds = compileBetMultiplierRounds(payload, existingSimple)
	if !reflect.DeepEqual(rounds, wantAdv) {
		t.Fatalf("switch to advanced = %+v, want %+v", rounds, wantAdv)
	}
}

func TestResolveEffectiveRunTypeBuiltinPlan(t *testing.T) {
	cfg := map[string]interface{}{
		"runTypeId": "builtin_plan",
		"builtinPlan": map[string]interface{}{
			"snapshotId": "SD10001",
			"runTypeId":  "fixed_number",
		},
	}
	if got := resolveEffectiveRunType("custom", cfg); got != RunTypeFixedNumber {
		t.Fatalf("builtin effective = %q, want fixed_number", got)
	}
	// 未物化 → 保留 builtin_plan，引擎按期跳过
	cfg["builtinPlan"] = map[string]interface{}{}
	if got := resolveEffectiveRunType("custom", cfg); got != RunTypeBuiltinPlan {
		t.Fatalf("unmaterialized effective = %q, want builtin_plan", got)
	}
	// 非 custom 不分发
	if got := resolveEffectiveRunType("contrary", cfg); got != "" {
		t.Fatalf("contrary effective = %q, want empty", got)
	}
}

func TestUnmaterializedBuiltinPlanSkips(t *testing.T) {
	w := &Worker{}
	cfg := pickTestConfig(t, `{"runTypeId":"builtin_plan"}`)
	dec := w.resolvePick(nil, cfg, sqlcdb.SchemeInstance{Kind: "custom"}, sqlcdb.LotteryDraw{})
	if !dec.Skip {
		t.Fatal("unmaterialized builtin_plan should skip the period")
	}
}

func TestLHCNumberPoolFallback(t *testing.T) {
	pool := playNumberPool(playRule{PlayTemplate: "lhc_std"})
	if len(pool) != 49 || pool[0] != "1" || pool[48] != "49" {
		t.Fatalf("lhc pool = len %d [%s..%s], want 1-49", len(pool), pool[0], pool[len(pool)-1])
	}
	pool = playNumberPool(playRule{})
	if len(pool) != 10 || pool[0] != "0" || pool[9] != "9" {
		t.Fatalf("default pool = %v, want 0-9", pool)
	}
}

func TestPickFixedNumber(t *testing.T) {
	cfg := pickTestConfig(t, `{"runTypeId":"fixed_number","playTypeId":"dingwei","subPlayId":"sub_ge","schemeGroups":["6,8"]}`)
	dec := pickFixedNumber(cfg)
	if dec.Content != "6,8" {
		t.Fatalf("fixed number content = %q", dec.Content)
	}
	inst := sqlcdb.SchemeInstance{Kind: "custom"}
	idx, cur, dir := advancePickState(cfg, inst, dec, true)
	if idx != 0 || cur != "" || dir != "" {
		t.Fatalf("fixed number state should stay zero: %d %q %q", idx, cur, dir)
	}
}
