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
	// 对齐 V8：运行类型与玩法正交、无门禁——任意组合均放行。
	runTypes := []string{
		RunTypeFixedRotate, RunTypeAdvFixedRotate, RunTypeAdvTriggerBet,
		RunTypeHotColdWarm, RunTypeRandomDraw, RunTypeBuiltinPlan, RunTypeFixedNumber,
	}
	plays := []struct{ typeID, subID, group, label string }{
		{"dingwei", "dingwei_ge", "一星", "定位胆"},
		{"longhu", "lh_1v10", "龙虎", "龙虎"},
		{"pc28_20", "teshu", "2.0模式", "特殊号"},
		{"hou4", "hou4_zhixuan_fs", "后四", "后四直选复式"},
		{"g001", "2", "前三码", "前三直选单式"},
		{"g010", "233", "2.0模式", "和值"},
		{"g007", "r2z", "任选", "任二组选复式"},
	}
	for _, rt := range runTypes {
		for _, p := range plays {
			if err := ValidateRunTypePlay(rt, p.typeID, p.subID, p.group, p.label); err != nil {
				t.Errorf("%s + %s/%s 应放行(全玩法), got %v", rt, p.group, p.label, err)
			}
		}
	}
}

func pickTestConfig(t *testing.T, raw string) parsedSchemeConfig {
	t.Helper()
	return parseSchemeConfig("custom", []byte(raw), 0, 0)
}

func TestExtractSchemeGroupsKeepsDingweiLeadingSlots(t *testing.T) {
	// 个/十/百/千：前导换行表示空位，TrimSpace 会全部压成万位 "1,2"
	cfg := pickTestConfig(t, `{
		"runTypeId":"fixed_rotate","playTypeId":"g006","subPlayId":"13","betMode":"dingwei",
		"schemeGroups":["\n\n\n\n1,2","\n\n\n1,2\n","\n\n1,2\n\n","\n1,2\n\n\n"]
	}`)
	want := []string{"\n\n\n\n1,2", "\n\n\n1,2\n", "\n\n1,2\n\n", "\n1,2\n\n\n"}
	if len(cfg.Groups) != len(want) {
		t.Fatalf("groups len=%d want %d: %#v", len(cfg.Groups), len(want), cfg.Groups)
	}
	for i := range want {
		if cfg.Groups[i] != want[i] {
			t.Fatalf("groups[%d]=%q want %q", i, cfg.Groups[i], want[i])
		}
	}
	dec := pickFixedRotate(cfg, sqlcdb.SchemeInstance{Kind: "custom", PickIndex: 0})
	if dec.Content != want[0] {
		t.Fatalf("pick=%q want %q", dec.Content, want[0])
	}
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

func TestAdvanceJushuUsesBetContentWhenPickStale(t *testing.T) {
	// pick_index 被回头复位打成 0，但仍应按本期实际内容（第 1 局）中后跳到第 2 局
	raw := `{"runTypeId":"adv_fixed_rotate","playTypeId":"dingwei","subPlayId":"sub_ge",
		"jushuList":[
			{"ju":1,"content":"1\n\n1\n\n1","afterHit":2,"afterMiss":3},
			{"ju":2,"content":"2\n\n2\n\n","afterHit":1,"afterMiss":1},
			{"ju":3,"content":"\n\n3\n\n3","afterHit":1,"afterMiss":1}
		]}`
	inst := sqlcdb.SchemeInstance{Kind: "custom", PickIndex: 0}
	idx, _, _ := AdvancePickAfterFormalSettlement("custom", []byte(raw), inst, "1\n\n1\n\n1", true)
	if idx != 2 {
		t.Fatalf("after hit ju1 with stale pick=0 → %d, want 2", idx)
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

func TestPickRandomDrawWholeTicketDanshi(t *testing.T) {
	// SSC 前三直选单式：整注随机应产出 N 个完整 3 位组合（逗号分隔），可被 evaluateZhixuanDanshi 解析。
	cfg := pickTestConfig(t, `{"runTypeId":"random_draw","playTypeId":"qian3","subPlayId":"zhixuan_ds","randomDraw":{"counts":[5],"strategy":"every"}}`)
	if !isWholeTicketRandom(cfg.Play) {
		t.Fatalf("qian3 直选单式 should be whole-ticket random, rule=%+v", cfg.Play)
	}
	dec := pickRandomDraw(cfg, sqlcdb.SchemeInstance{Kind: "custom"})
	tokens := strings.Split(dec.Content, ",")
	if len(tokens) != 5 {
		t.Fatalf("want 5 注, got %d: %q", len(tokens), dec.Content)
	}
	seen := map[string]bool{}
	for _, tk := range tokens {
		if len(tk) != 3 {
			t.Fatalf("每注应为 3 位组合, got %q in %q", tk, dec.Content)
		}
		for _, r := range tk {
			if r < '0' || r > '9' {
				t.Fatalf("非法号码 token %q", tk)
			}
		}
		if seen[tk] {
			t.Fatalf("整注随机应去重, dup %q in %q", tk, dec.Content)
		}
		seen[tk] = true
	}

	// 产出的单式内容应能命中：构造一注与开奖一致
	balls := []string{"3", "9", "2", "7", "5"} // 前三=392
	ev := evaluatePlayHit(cfg.Play, balls, "392,111,222", cfg.Contrary, cfg.ContraryPlan, cfg.Play.PositionIdx)
	if !ev.Hit || ev.BetUnits != 3 {
		t.Fatalf("单式命中评估异常: hit=%v units=%d", ev.Hit, ev.BetUnits)
	}

	// 矩阵放宽：随机出号支持单式，冷热温不支持
	if !SupportsRandomDrawSubPlay("前三码", "前三直选单式") {
		t.Fatal("随机出号应支持前三直选单式")
	}
	if SupportsPositionSourceSubPlay("前三码", "前三直选单式") {
		t.Fatal("冷热温(按位)不应支持直选单式")
	}
	if !SupportsRandomDrawSubPlay("2.0模式", "和值") {
		t.Fatal("随机出号现应支持和值（属性家族已放开）")
	}
}

func TestPickRandomDrawZuxuanPool(t *testing.T) {
	// 前三组选复式：随机选 K 个号组成号码池（升序、逗号分隔、去重），可被组选评估解析。
	cfg := pickTestConfig(t, `{"runTypeId":"random_draw","playTypeId":"qian3","subPlayId":"zuxuan_fs","betMode":"zu6","randomDraw":{"counts":[4],"strategy":"every"}}`)
	if !isZuxuanPoolRandom(cfg.Play) {
		t.Fatalf("前三组六 should be zuxuan-pool random, rule=%+v", cfg.Play)
	}
	if isWholeTicketRandom(cfg.Play) {
		t.Fatal("组选复式不应走整注单式路径")
	}
	dec := pickRandomDraw(cfg, sqlcdb.SchemeInstance{Kind: "custom"})
	toks := strings.Split(dec.Content, ",")
	if len(toks) != 4 {
		t.Fatalf("want 4 个号, got %d: %q", len(toks), dec.Content)
	}
	seen := map[string]bool{}
	prev := -1
	for _, tk := range toks {
		if seen[tk] {
			t.Fatalf("号码池应去重, dup %q in %q", tk, dec.Content)
		}
		seen[tk] = true
		n := 0
		for _, r := range tk {
			if r < '0' || r > '9' {
				t.Fatalf("非法号码 %q", tk)
			}
			n = n*10 + int(r-'0')
		}
		if n <= prev {
			t.Fatalf("号码池应升序, got %q", dec.Content)
		}
		prev = n
	}
	// 矩阵：随机出号放行组选家族，冷热温不放行
	if !SupportsRandomDrawSubPlay("前三码", "前三组选复式") {
		t.Fatal("随机出号应支持前三组选复式")
	}
	if !SupportsRandomDrawSubPlay("四星", "组选24") {
		t.Fatal("随机出号应支持组选24")
	}
	if SupportsPositionSourceSubPlay("前三码", "前三组选复式") {
		t.Fatal("冷热温不应支持组选复式")
	}
}

func TestPickRandomDrawAttributeFamily(t *testing.T) {
	// 各属性/聚合玩法：随机产出的内容应落在合法选项宇宙内，且能被对应评估解析。
	cases := []struct {
		name    string
		raw     string
		balls   []string
		options []string // 合法选项集（nil=数字池，另行校验）
	}{
		{"大小单双", `{"runTypeId":"random_draw","playTypeId":"dxds","subPlayId":"qian2_dxds","betMode":"dxds","randomDraw":{"counts":[2]}}`, []string{"3", "9", "2", "7", "5"}, []string{"大", "小", "单", "双"}},
		{"龙虎", `{"runTypeId":"random_draw","playTypeId":"longhu","subPlayId":"lh_wanqian_dou","betMode":"longhu","randomDraw":{"counts":[1]}}`, []string{"3", "9", "2", "7", "5"}, []string{"龙", "虎"}},
		{"和值", `{"runTypeId":"random_draw","playTypeId":"qian3","subPlayId":"qian3_zhixuan_hz","betMode":"hezhi","randomDraw":{"counts":[3]}}`, []string{"3", "9", "2", "7", "5"}, nil},
		{"跨度", `{"runTypeId":"random_draw","playTypeId":"qian3","subPlayId":"qian3_kuadu","betMode":"kuadu","randomDraw":{"counts":[2]}}`, []string{"3", "9", "2", "7", "5"}, nil},
		{"不定位", `{"runTypeId":"random_draw","playTypeId":"budingwei","subPlayId":"qian3_2ma","betMode":"budingwei","randomDraw":{"counts":[4]}}`, []string{"3", "9", "2", "7", "5"}, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cfg := pickTestConfig(t, c.raw)
			if !isAttributeRandom(cfg.Play) {
				t.Fatalf("%s should be attribute random, rule=%+v", c.name, cfg.Play)
			}
			dec := pickRandomDraw(cfg, sqlcdb.SchemeInstance{Kind: "custom"})
			if strings.TrimSpace(dec.Content) == "" {
				t.Fatalf("%s produced empty content", c.name)
			}
			// 内容应可被评估（注数>0）
			ev := evaluatePlayHit(cfg.Play, c.balls, dec.Content, cfg.Contrary, cfg.ContraryPlan, cfg.Play.PositionIdx)
			if ev.BetUnits <= 0 {
				t.Fatalf("%s content %q not bettable (units=%d)", c.name, dec.Content, ev.BetUnits)
			}
			if c.options != nil {
				for _, tk := range strings.Split(dec.Content, ",") {
					ok := false
					for _, o := range c.options {
						if tk == o {
							ok = true
							break
						}
					}
					if !ok {
						t.Fatalf("%s token %q not in universe %v", c.name, tk, c.options)
					}
				}
			}
			t.Logf("%s → %q (units=%d)", c.name, dec.Content, ev.BetUnits)
		})
	}

	// 矩阵：随机出号放行属性家族；冷热温仍不支持
	for _, sub := range []string{"前二大小单双", "龙虎", "前三和值", "前三跨度", "前三二码不定位"} {
		if !SupportsRandomDrawSubPlay("", sub) {
			t.Errorf("随机出号应支持 %s", sub)
		}
	}
}

func TestHotColdWarmAttributeTiers(t *testing.T) {
	// 属性家族按选项命中频次分档：复用 evaluatePlayHit 独立复算，校验分档为宇宙完整划分且降序。
	cases := []string{
		`{"runTypeId":"hot_cold_warm","playTypeId":"dxds","subPlayId":"qian2_dxds","betMode":"dxds","playMethodLabel":"大小单双"}`,
		`{"runTypeId":"hot_cold_warm","playTypeId":"longhu","subPlayId":"lh_wanqian_dou","betMode":"longhu","playMethodLabel":"龙虎"}`,
		`{"runTypeId":"hot_cold_warm","playTypeId":"qian3","subPlayId":"qian3_zhixuan_hz","betMode":"hezhi","playMethodLabel":"和值"}`,
	}
	draws := [][]string{
		{"9", "8", "7", "6", "1"},
		{"3", "5", "2", "4", "7"},
		{"1", "1", "1", "1", "1"},
		{"8", "6", "4", "2", "0"},
		{"5", "5", "5", "5", "5"},
		{"2", "9", "0", "3", "8"},
	}
	for _, raw := range cases {
		cfg := pickTestConfig(t, raw)
		res := HotColdWarmAttributeTiers(cfg.Play, draws)
		if res.Mode != "attribute" {
			t.Fatalf("mode=%s raw=%s", res.Mode, raw)
		}
		if len(res.Universe) == 0 {
			t.Fatalf("empty universe raw=%s", raw)
		}
		// 分档为宇宙完整划分（不重不漏）
		ordered := append(append(append([]string{}, res.Hot...), res.Warm...), res.Cold...)
		if len(ordered) != len(res.Universe) {
			t.Fatalf("partition size %d != universe %d raw=%s", len(ordered), len(res.Universe), raw)
		}
		seen := map[string]bool{}
		for _, o := range ordered {
			if seen[o] {
				t.Fatalf("duplicated tier option %q raw=%s", o, raw)
			}
			seen[o] = true
		}
		// 独立复算命中频次，校验热→冷非递增（温档为空）
		if len(res.Warm) != 0 {
			t.Fatalf("warm should be empty, got %v raw=%s", res.Warm, raw)
		}
		counts := map[string]int{}
		for _, balls := range draws {
			for _, opt := range res.Universe {
				if evaluatePlayHit(cfg.Play, balls, opt, false, "", cfg.Play.PositionIdx).Hit {
					counts[opt]++
				}
			}
		}
		for i := 1; i < len(ordered); i++ {
			if counts[ordered[i-1]] < counts[ordered[i]] {
				t.Fatalf("tiers not sorted desc: %v counts=%v raw=%s", ordered, counts, raw)
			}
		}
	}
}

func TestPickRandomDrawHunhe(t *testing.T) {
	// 混合组选单式：整注随机应产出 N 个组选组合（升序去重），且排除豹子（全同号），可被 evaluateHunhe 解析。
	cfg := pickTestConfig(t, `{"runTypeId":"random_draw","playTypeId":"qian3","subPlayId":"qian3_hunhe","betMode":"hunhe","randomDraw":{"counts":[8]}}`)
	if !isWholeTicketRandom(cfg.Play) {
		t.Fatalf("混合 should be whole-ticket random, rule=%+v", cfg.Play)
	}
	dec := pickRandomDraw(cfg, sqlcdb.SchemeInstance{Kind: "custom"})
	toks := strings.Split(dec.Content, ",")
	if len(toks) == 0 || dec.Content == "" {
		t.Fatalf("混合 produced empty content")
	}
	seen := map[string]bool{}
	for _, tk := range toks {
		if len(tk) != 3 {
			t.Fatalf("每注应 3 位, got %q", tk)
		}
		// 排除豹子（三位全同）
		if tk[0] == tk[1] && tk[1] == tk[2] {
			t.Fatalf("混合应排除豹子, got %q in %q", tk, dec.Content)
		}
		// 升序归一去重
		if seen[tk] {
			t.Fatalf("应按组合去重, dup %q", tk)
		}
		seen[tk] = true
		if !(tk[0] <= tk[1] && tk[1] <= tk[2]) {
			t.Fatalf("组选应升序, got %q", tk)
		}
	}
	ev := evaluatePlayHit(cfg.Play, []string{"1", "2", "3", "7", "5"}, dec.Content, cfg.Contrary, cfg.ContraryPlan, cfg.Play.PositionIdx)
	if ev.BetUnits <= 0 {
		t.Fatalf("混合内容 %q 不可评估 (units=%d)", dec.Content, ev.BetUnits)
	}
	if !SupportsRandomDrawSubPlay("前三码", "前三混合组选单式") {
		t.Fatal("随机出号应支持前三混合组选单式")
	}
}

// TestSimRandomDrawAllFamiliesMultiPeriod 随机出号全家族多期闭环终验：
// 逐期 pickRandomDraw → evaluatePlayHit → advancePickState（换号策略），
// 验证每期产号合法可下注（注数>0），且「每期换」策略每期重随、「不换号」跨期保号。
func TestSimRandomDrawAllFamiliesMultiPeriod(t *testing.T) {
	balls := []string{"3", "9", "2", "7", "5"}
	cases := []struct {
		name string
		raw  string
	}{
		{"直选单式", `{"runTypeId":"random_draw","playTypeId":"qian3","subPlayId":"zhixuan_ds","randomDraw":{"counts":[5],"strategy":"every"}}`},
		{"组选复式", `{"runTypeId":"random_draw","playTypeId":"qian3","subPlayId":"zuxuan_fs","betMode":"zu6","randomDraw":{"counts":[4],"strategy":"every"}}`},
		{"大小单双", `{"runTypeId":"random_draw","playTypeId":"dxds","subPlayId":"qian2_dxds","betMode":"dxds","randomDraw":{"counts":[2],"strategy":"every"}}`},
		{"龙虎", `{"runTypeId":"random_draw","playTypeId":"longhu","subPlayId":"lh_wanqian_dou","betMode":"longhu","randomDraw":{"counts":[1],"strategy":"every"}}`},
		{"和值", `{"runTypeId":"random_draw","playTypeId":"qian3","subPlayId":"qian3_zhixuan_hz","betMode":"hezhi","randomDraw":{"counts":[3],"strategy":"every"}}`},
		{"跨度", `{"runTypeId":"random_draw","playTypeId":"qian3","subPlayId":"qian3_kuadu","betMode":"kuadu","randomDraw":{"counts":[2],"strategy":"every"}}`},
		{"不定位", `{"runTypeId":"random_draw","playTypeId":"budingwei","subPlayId":"qian3_2ma","betMode":"budingwei","randomDraw":{"counts":[4],"strategy":"every"}}`},
		{"包胆", `{"runTypeId":"random_draw","playTypeId":"baodan","subPlayId":"qian3_baodan","betMode":"baodan","randomDraw":{"counts":[2],"strategy":"every"}}`},
		{"混合组选单式", `{"runTypeId":"random_draw","playTypeId":"qian3","subPlayId":"qian3_hunhe","betMode":"hunhe","randomDraw":{"counts":[6],"strategy":"every"}}`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cfg := pickTestConfig(t, c.raw)
			inst := sqlcdb.SchemeInstance{Kind: "custom", LotteryCode: "tron_ffc_1m"}
			for period := 0; period < 4; period++ {
				dec := pickRandomDraw(cfg, inst)
				if strings.TrimSpace(dec.Content) == "" {
					t.Fatalf("期%d 产号为空", period)
				}
				ev := evaluatePlayHit(cfg.Play, balls, dec.Content, cfg.Contrary, cfg.ContraryPlan, cfg.Play.PositionIdx)
				if ev.BetUnits <= 0 {
					t.Fatalf("期%d 内容 %q 不可下注 (units=%d)", period, dec.Content, ev.BetUnits)
				}
				_, cur, _ := advancePickState(cfg, inst, dec, ev.Hit)
				// every 策略：每期清空重随
				if cur != "" {
					t.Fatalf("期%d every 策略应清空保号, got %q", period, cur)
				}
				inst.CurrentPick = cur
				t.Logf("%s 期%d → %q (注数=%d, %s)", c.name, period, dec.Content, ev.BetUnits, hitLabel(ev.Hit))
			}
		})
	}

	// 不换号策略：跨期保号（以直选单式为例）
	t.Run("不换号保号", func(t *testing.T) {
		cfg := pickTestConfig(t, `{"runTypeId":"random_draw","playTypeId":"qian3","subPlayId":"zhixuan_ds","randomDraw":{"counts":[3],"strategy":"keep"}}`)
		inst := sqlcdb.SchemeInstance{Kind: "custom", LotteryCode: "tron_ffc_1m"}
		first := ""
		for period := 0; period < 3; period++ {
			dec := pickRandomDraw(cfg, inst)
			if period == 0 {
				first = dec.Content
			} else if dec.Content != first {
				t.Fatalf("期%d 不换号应保持 %q, got %q", period, first, dec.Content)
			}
			ev := evaluatePlayHit(cfg.Play, balls, dec.Content, cfg.Contrary, cfg.ContraryPlan, cfg.Play.PositionIdx)
			_, cur, _ := advancePickState(cfg, inst, dec, ev.Hit)
			inst.CurrentPick = cur
		}
	})
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
	cfg := pickTestConfig(t, `{"runTypeId":"hot_cold_warm","playTypeId":"dingwei","subPlayId":"sub_ge","hotColdWarm":{"totalPeriods":50,"pool":["1,5,9"],"pickTypes":["hot"],"faultCount":1,"strategy":"after_hit"}}`)
	inst := sqlcdb.SchemeInstance{Kind: "custom"}
	// 无开奖序列时回退配置池
	dec := pickHotColdWarmFromDraws(cfg, inst, nil)
	if dec.Content != "1,5,9" {
		t.Fatalf("pool content = %q", dec.Content)
	}
	// 中后换：命中清空 → 下期按冷热重取；未中锁定本期内容
	_, cur, _ := advancePickState(cfg, inst, dec, true)
	if cur != "" {
		t.Fatalf("after_hit on hit should clear pick, got %q", cur)
	}
	_, cur, _ = advancePickState(cfg, inst, dec, false)
	if cur != "1,5,9" {
		t.Fatalf("after_hit on miss should keep content, got %q", cur)
	}
}

func TestBuildHotColdPickContentByRankAndOffset(t *testing.T) {
	// 名次+起点偏移模型：容错=偏移，pickCount=每位取几个名次。
	cfg := pickTestConfig(t, `{"runTypeId":"hot_cold_warm","playTypeId":"dingwei","subPlayId":"sub_ge","hotColdWarm":{"totalPeriods":20,"pickTypes":["hot"],"faultCount":0,"pickCount":1}}`)
	// 个位（PositionIdx=4）频次：7×3 > 1×1、3×1 > 其余0
	// 全序（频次降序，平手按号码升序）：7,1,3,0,2,4,5,6,8,9
	draws := [][]string{
		{"0", "0", "0", "0", "7"},
		{"0", "0", "0", "0", "7"},
		{"0", "0", "0", "0", "7"},
		{"0", "0", "0", "0", "3"},
		{"0", "0", "0", "0", "1"},
	}
	// 热 offset0 count1 → 最热 7
	if got := buildHotColdPickContent(cfg, draws); got != "7" {
		t.Fatalf("hot offset0 count1 => %q, want 7", got)
	}
	// 热 offset1 count1 → 第2热（平手按号码升序：freq1 中 1<3）→ 1
	cfg.HotCold.FaultCount = 1
	if got := buildHotColdPickContent(cfg, draws); got != "1" {
		t.Fatalf("hot offset1 count1 => %q, want 1", got)
	}
	// 热 offset0 count2 → 第1、2热 → 7,1
	cfg.HotCold.FaultCount = 0
	cfg.HotCold.PickCount = 2
	if got := buildHotColdPickContent(cfg, draws); got != "7,1" {
		t.Fatalf("hot offset0 count2 => %q, want 7,1", got)
	}
	// 冷 offset0 count1 → 最冷（全序末位，freq0 中号码最大 9）→ 9
	cfg.HotCold.PickTypes = []string{"cold"}
	cfg.HotCold.PickCount = 1
	if got := buildHotColdPickContent(cfg, draws); got != "9" {
		t.Fatalf("cold offset0 count1 => %q, want 9", got)
	}
}

func TestBuildHotColdPickContentManualOverride(t *testing.T) {
	// 混合模式：某位手选号码覆盖，其余位按名次自动取号。
	cfg := pickTestConfig(t, `{"runTypeId":"hot_cold_warm","playTemplate":"ssc_std","playTypeId":"g006","subPlayId":"13","typeId":"g006","subId":"13","betMode":"dingwei","hotColdWarm":{"totalPeriods":20,"pickTypes":["hot"],"faultCount":0,"pickCount":1,"pool":["8","","","",""]}}`)
	draws := [][]string{
		{"1", "2", "3", "4", "5"},
		{"1", "2", "3", "4", "5"},
		{"6", "7", "8", "9", "0"},
	}
	got := buildHotColdPickContent(cfg, draws)
	lines := strings.Split(got, "\n")
	if len(lines) != 5 {
		t.Fatalf("lines=%d content=%q want 5", len(lines), got)
	}
	if lines[0] != "8" {
		t.Fatalf("万位应被手选覆盖为 8, got %q", lines[0])
	}
	// 千位（idx1）频次：2×2 > 7×1 → 最热 2
	if lines[1] != "2" {
		t.Fatalf("千位自动取号应为最热 2, got %q", lines[1])
	}
}

func TestBuildHotColdPickContentEveryPeriodResort(t *testing.T) {
	// 每期换：不同开奖窗口重排后取号应变化。
	cfg := pickTestConfig(t, `{"runTypeId":"hot_cold_warm","playTypeId":"dingwei","subPlayId":"sub_ge","hotColdWarm":{"totalPeriods":20,"pickTypes":["hot"],"faultCount":0,"pickCount":1,"strategy":"every"}}`)
	a := buildHotColdPickContent(cfg, [][]string{{"0", "0", "0", "0", "7"}, {"0", "0", "0", "0", "7"}})
	b := buildHotColdPickContent(cfg, [][]string{{"0", "0", "0", "0", "3"}, {"0", "0", "0", "0", "3"}})
	if a != "7" || b != "3" {
		t.Fatalf("每期重排应随窗口变化: a=%q(want 7) b=%q(want 3)", a, b)
	}
	// 每期换策略：结算后清空 current_pick，下期重取
	inst := sqlcdb.SchemeInstance{Kind: "custom", CurrentPick: "7"}
	if _, cur, _ := advancePickState(cfg, inst, pickDecision{Content: "7"}, false); cur != "" {
		t.Fatalf("every 策略应清空 current_pick, got %q", cur)
	}
}

func TestBuildHotColdPickContentG006FivePosition(t *testing.T) {
	// 统一定位胆 subId=13：五位各取 pickCount 个热号名次 → 五行内容
	cfg := pickTestConfig(t, `{"runTypeId":"hot_cold_warm","playTemplate":"ssc_std","playTypeId":"g006","subPlayId":"13","typeId":"g006","subId":"13","betMode":"dingwei","hotColdWarm":{"totalPeriods":20,"pickTypes":["hot"],"faultCount":0,"pickCount":5}}`)
	if playPositionCount(cfg.Play) != 5 {
		t.Fatalf("playPositionCount=%d want 5", playPositionCount(cfg.Play))
	}
	draws := [][]string{
		{"1", "2", "3", "4", "5"},
		{"1", "2", "3", "4", "5"},
		{"1", "2", "3", "4", "5"},
		{"1", "2", "3", "4", "5"},
		{"6", "7", "8", "9", "0"},
		{"6", "7", "8", "9", "0"},
	}
	got := buildHotColdPickContent(cfg, draws)
	lines := strings.Split(got, "\n")
	if len(lines) != 5 {
		t.Fatalf("lines=%d content=%q want 5", len(lines), got)
	}
	units := 0
	for _, line := range lines {
		n := len(strings.Split(strings.TrimSpace(line), ","))
		if n != 5 {
			t.Fatalf("line %q token count=%d want 5", line, n)
		}
		units += n
	}
	if units != 25 {
		t.Fatalf("units=%d want 25", units)
	}
	// 旧单位保号应被多位面板强制重取
	inst := sqlcdb.SchemeInstance{Kind: "custom", CurrentPick: "5"}
	dec := pickHotColdWarmFromDraws(cfg, inst, draws)
	if !strings.Contains(dec.Content, "\n") {
		t.Fatalf("stale single-digit current_pick should rebuild, got %q", dec.Content)
	}
}

func TestHotColdWarmDigitPoolFamilies(t *testing.T) {
	// 冷热温放开组选/不定位/包胆：号码整体频次分档选号池（单行数字集），可评估、可中奖轮换。
	cases := []struct {
		name  string
		raw   string
		balls []string
	}{
		{"组选复式", `{"runTypeId":"hot_cold_warm","playTypeId":"qian3","subPlayId":"zuxuan_fs","betMode":"zu6","hotColdWarm":{"totalPeriods":50,"pool":["1,3,5,7"],"winRotate":true}}`, []string{"1", "3", "5", "7", "9"}},
		{"不定位", `{"runTypeId":"hot_cold_warm","playTypeId":"budingwei","subPlayId":"qian3_2ma","betMode":"budingwei","hotColdWarm":{"totalPeriods":50,"pool":["1,3,5,7"],"winRotate":false}}`, []string{"1", "3", "5", "7", "9"}},
		{"包胆", `{"runTypeId":"hot_cold_warm","playTypeId":"baodan","subPlayId":"qian3_baodan","betMode":"baodan","hotColdWarm":{"totalPeriods":50,"pool":["3,6"],"winRotate":false}}`, []string{"1", "3", "5", "7", "9"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cfg := pickTestConfig(t, c.raw)
			inst := sqlcdb.SchemeInstance{Kind: "custom", LotteryCode: "tron_ffc_1m"}
			dec := pickHotColdWarmFromDraws(cfg, inst, nil)
			if strings.TrimSpace(dec.Content) == "" {
				t.Fatalf("%s 池为空", c.name)
			}
			ev := evaluatePlayHit(cfg.Play, c.balls, dec.Content, cfg.Contrary, cfg.ContraryPlan, cfg.Play.PositionIdx)
			if ev.BetUnits <= 0 {
				t.Fatalf("%s 池 %q 不可下注 (units=%d)", c.name, dec.Content, ev.BetUnits)
			}
			// 中后换：命中清空，下期按冷热重取
			_, cur, _ := advancePickState(cfg, inst, dec, true)
			rotateOnHit := cfg.HotCold != nil && (cfg.HotCold.Strategy == "after_hit" || cfg.HotCold.WinRotate)
			if rotateOnHit && strings.TrimSpace(cur) != "" {
				t.Fatalf("%s 中后换命中应清空 current_pick, got %q", c.name, cur)
			}
			t.Logf("%s → %q (units=%d) 换号后=%q", c.name, dec.Content, ev.BetUnits, cur)
		})
	}
	// 矩阵：冷热温放行组选/不定位/包胆 + 属性/聚合；单式仍不放行
	for _, sub := range []string{"前三组选复式", "前三组三", "前三二码不定位", "前三包胆", "前三特殊号", "和值", "龙虎", "大小单双"} {
		if !SupportsHotColdWarmSubPlay("前三码", sub) {
			t.Errorf("冷热温应支持 %s", sub)
		}
	}
	for _, sub := range []string{"前三直选单式", "前三组选单式"} {
		if SupportsHotColdWarmSubPlay("", sub) {
			t.Errorf("冷热温不应支持 %s", sub)
		}
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
	if len(warm) != 0 {
		t.Fatalf("warm should be empty, got %v", warm)
	}
	if len(hot)+len(cold) != 10 {
		t.Fatalf("tiers total = %d", len(hot)+len(cold))
	}
	if len(hot) != 5 || len(cold) != 5 {
		t.Fatalf("hot/cold sizes = %d/%d, want 5/5", len(hot), len(cold))
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

func TestAdvancePickAfterFormalSettlement(t *testing.T) {
	// 定码轮换：派奖后每期都应推进游标（与命中无关），修复“一直用第一组下注”。
	fixedCfg := []byte(`{"runTypeId":"fixed_rotate","playTypeId":"dingwei","subPlayId":"sub_ge","schemeGroups":["12","24"]}`)
	inst := sqlcdb.SchemeInstance{Kind: "custom", PickIndex: 0}
	for _, hit := range []bool{true, false} {
		idx, _, _ := AdvancePickAfterFormalSettlement("custom", fixedCfg, inst, "12", hit)
		if idx != 1 {
			t.Fatalf("fixed_rotate hit=%v next index = %d, want 1", hit, idx)
		}
	}
	inst.PickIndex = 1
	idx, _, _ := AdvancePickAfterFormalSettlement("custom", fixedCfg, inst, "24", false)
	if idx != 0 {
		t.Fatalf("fixed_rotate wrap index = %d, want 0", idx)
	}

	// 高级定码轮换：按中/未中跳转局号。
	advCfg := []byte(`{
		"runTypeId":"adv_fixed_rotate","playTypeId":"dingwei","subPlayId":"sub_ge",
		"jushuList":[
			{"ju":1,"content":"12","afterHit":1,"afterMiss":2},
			{"ju":2,"content":"24","afterHit":1,"afterMiss":2}
		]}`)
	inst = sqlcdb.SchemeInstance{Kind: "custom", PickIndex: 1}
	if idx, _, _ := AdvancePickAfterFormalSettlement("custom", advCfg, inst, "12", false); idx != 2 {
		t.Fatalf("adv_fixed_rotate after miss = %d, want 2", idx)
	}
	if idx, _, _ := AdvancePickAfterFormalSettlement("custom", advCfg, inst, "12", true); idx != 1 {
		t.Fatalf("adv_fixed_rotate after hit = %d, want 1", idx)
	}

	// 开某投某：派奖后推进投向状态机（下单时冻结，此处按同一起点重算）。
	trigCfg := []byte(`{"runTypeId":"adv_trigger_bet","playTypeId":"dingwei","subPlayId":"sub_ge",
		"triggerBet":{"mode":"alt_pos_first","rows":[{"enabled":true,"open":"1","pos":"2","neg":"3"}]}}`)
	inst = sqlcdb.SchemeInstance{Kind: "custom", LastDirection: "pos"}
	if _, _, dir := AdvancePickAfterFormalSettlement("custom", trigCfg, inst, "3", false); dir != "neg" {
		t.Fatalf("trigger after last=pos should flip to neg, got %q", dir)
	}
}

// TestSimFixedRotateMultiPeriod 定码轮换 sim 模式多期闭环：
// 复用引擎真实链路（pickFixedRotate 出号 → evaluatePlayHit 判中 → advancePickState 推进游标），
// 逐期把 pick_index 回传下一期（模拟 scheme_instance 持久化），验证「每期换下一组」——
// 无论命中/未中，下注组都按 组数 循环切换（12→24→…），命中与否不影响换组节奏。
func TestSimFixedRotateMultiPeriod(t *testing.T) {
	// SSC 定位胆·个位（sub_ge → 位索引 4）；三组号码轮换。
	cfg := pickTestConfig(t, `{"runTypeId":"fixed_rotate","playTypeId":"dingwei","subPlayId":"sub_ge","schemeGroups":["1,2","3,4","5,6"]}`)

	// 每期个位开奖数字（前 4 位填 0，个位=digit），构造命中/未中混合序列。
	geDigits := []string{"1", "0", "5", "9", "4", "0", "2", "3"}
	// 期望：内容严格按 组[期号 % 3] 每期切换；命中由开奖个位是否落在该组决定。
	wantContent := []string{"1,2", "3,4", "5,6", "1,2", "3,4", "5,6", "1,2", "3,4"}
	wantIndex := []int32{0, 1, 2, 0, 1, 2, 0, 1}
	wantHit := []bool{true, false, true, false, true, false, true, true}

	inst := sqlcdb.SchemeInstance{Kind: "custom", PickIndex: 0}
	for i, ge := range geDigits {
		if inst.PickIndex != wantIndex[i] {
			t.Fatalf("period %d: pick_index=%d, want %d", i, inst.PickIndex, wantIndex[i])
		}
		dec := pickFixedRotate(cfg, inst)
		if dec.Content != wantContent[i] {
			t.Fatalf("period %d: content=%q, want %q", i, dec.Content, wantContent[i])
		}
		balls := []string{"0", "0", "0", "0", ge}
		eval := evaluatePlayHit(cfg.Play, balls, dec.Content, cfg.Contrary, cfg.ContraryPlan, cfg.Play.PositionIdx)
		if eval.Hit != wantHit[i] {
			t.Fatalf("period %d: 个位=%s content=%q hit=%v, want %v", i, ge, dec.Content, eval.Hit, wantHit[i])
		}
		next, _, _ := advancePickState(cfg, inst, dec, eval.Hit)
		t.Logf("期%d 下注组[%d]=%q 个位=%s 判定=%s → 下期组游标=%d",
			i, inst.PickIndex, dec.Content, ge, hitLabel(eval.Hit), next)
		inst.PickIndex = next
	}

	// 复投一整轮后应回到起点（3 组 × 循环）。
	if inst.PickIndex != int32(len(geDigits)%3) {
		t.Fatalf("final pick_index=%d, want %d", inst.PickIndex, len(geDigits)%3)
	}
}

func hitLabel(hit bool) string {
	if hit {
		return "中"
	}
	return "挂"
}

// TestSimFormalPayoutAllRunTypes 正式盘派奖推进多期仿真（7 类型）。
//
// 正式盘下单时出号游标被冻结，派奖后由 ProcessFormalAfterSettlement →
// AdvancePickAfterFormalSettlement 补推进。本用例直接驱动该函数逐期闭环，
// 把 pick_index/current_pick/last_direction 回传下一期（模拟实例持久化），
// 验证「派奖后各类型游标不掉队」。无需连库/第三方，确定性可回归。
func TestSimFormalPayoutAllRunTypes(t *testing.T) {
	base := sqlcdb.SchemeInstance{Kind: "custom", LotteryCode: "tron_ffc_1m"}

	// 1) 定码轮换：每期换（与命中无关）
	t.Run("fixed_rotate", func(t *testing.T) {
		raw := `{"runTypeId":"fixed_rotate","playTypeId":"dingwei","subPlayId":"sub_ge","schemeGroups":["1,2","3,4","5,6"]}`
		cfg := pickTestConfig(t, raw)
		inst := base
		hits := []bool{true, false, false, true, false, true}
		want := []int32{1, 2, 0, 1, 2, 0}
		for i, hit := range hits {
			content := pickFixedRotate(cfg, inst).Content
			idx, _, _ := AdvancePickAfterFormalSettlement(inst.Kind, []byte(raw), inst, content, hit)
			t.Logf("期%d 组[%d]=%q %s → 下期组=%d", i, inst.PickIndex, content, hitLabel(hit), idx)
			if idx != want[i] {
				t.Fatalf("period %d: index=%d want %d", i, idx, want[i])
			}
			inst.PickIndex = idx
		}
	})

	// 2) 高级定码轮换：局数列表按中后/挂后跳局
	t.Run("adv_fixed_rotate", func(t *testing.T) {
		raw := `{"runTypeId":"adv_fixed_rotate","playTypeId":"dingwei","subPlayId":"sub_ge",
			"jushuList":[{"ju":1,"content":"0,1","afterHit":1,"afterMiss":2},
			             {"ju":2,"content":"2,3","afterHit":1,"afterMiss":2}]}`
		cfg := pickTestConfig(t, raw)
		inst := base
		inst.PickIndex = 1
		hits := []bool{false, false, true, true}
		want := []int32{2, 2, 1, 1}
		for i, hit := range hits {
			content := pickJushuList(cfg, inst).Content
			idx, _, _ := AdvancePickAfterFormalSettlement(inst.Kind, []byte(raw), inst, content, hit)
			t.Logf("期%d 局[%d]=%q %s → 下期局=%d", i, inst.PickIndex, content, hitLabel(hit), idx)
			if idx != want[i] {
				t.Fatalf("period %d: ju=%d want %d", i, idx, want[i])
			}
			inst.PickIndex = idx
		}
	})

	// 3) 高级开某投某：投向状态机逐期交替（前正后反）
	t.Run("adv_trigger_bet", func(t *testing.T) {
		raw := `{"runTypeId":"adv_trigger_bet","playTypeId":"dingwei","subPlayId":"sub_ge",
			"triggerBet":{"mode":"alt_pos_first","rows":[{"enabled":true,"open":"1","pos":"2","neg":"3"}]}}`
		inst := base
		want := []string{"pos", "neg", "pos", "neg"}
		hits := []bool{false, true, false, true}
		for i, hit := range hits {
			_, _, dir := AdvancePickAfterFormalSettlement(inst.Kind, []byte(raw), inst, "2", hit)
			t.Logf("期%d 上局投向=%q %s → 本局投向=%q", i, inst.LastDirection, hitLabel(hit), dir)
			if dir != want[i] {
				t.Fatalf("period %d: dir=%q want %q", i, dir, want[i])
			}
			inst.LastDirection = dir
		}
	})

	// 4) 随机出号·不换号：保号跨期不变
	t.Run("random_draw_keep", func(t *testing.T) {
		raw := `{"runTypeId":"random_draw","playTypeId":"dingwei","subPlayId":"sub_ge","randomDraw":{"counts":[3],"strategy":"keep"}}`
		cfg := pickTestConfig(t, raw)
		inst := base
		inst.CurrentPick = "1,2,3" // 首期已随机产出并落库
		hits := []bool{false, true, false}
		for i, hit := range hits {
			content := pickRandomDraw(cfg, inst).Content
			_, cur, _ := AdvancePickAfterFormalSettlement(inst.Kind, []byte(raw), inst, content, hit)
			t.Logf("期%d 下注=%q %s → 保号=%q", i, content, hitLabel(hit), cur)
			if content != "1,2,3" || cur != "1,2,3" {
				t.Fatalf("period %d: content=%q cur=%q want 保持 1,2,3", i, content, cur)
			}
			inst.CurrentPick = cur
		}
	})

	// 5) 随机出号·每期换：每期清空重随
	t.Run("random_draw_every", func(t *testing.T) {
		raw := `{"runTypeId":"random_draw","playTypeId":"dingwei","subPlayId":"sub_ge","randomDraw":{"counts":[2],"strategy":"every"}}`
		inst := base
		inst.CurrentPick = "4,5"
		for i, hit := range []bool{true, false} {
			_, cur, _ := AdvancePickAfterFormalSettlement(inst.Kind, []byte(raw), inst, "4,5", hit)
			if cur != "" {
				t.Fatalf("period %d: every 策略应清空保号, got %q", i, cur)
			}
			inst.CurrentPick = cur
		}
	})

	// 6) 冷热·中后换：命中清空重取，未中锁定内容
	t.Run("hot_cold_warm_rotate", func(t *testing.T) {
		raw := `{"runTypeId":"hot_cold_warm","playTypeId":"dingwei","subPlayId":"sub_ge","hotColdWarm":{"totalPeriods":50,"pool":["1,5,9"],"pickTypes":["hot"],"faultCount":1,"strategy":"after_hit"}}`
		inst := base
		content := "1,5,9"
		// 命中 → 清空
		_, cur, _ := AdvancePickAfterFormalSettlement(inst.Kind, []byte(raw), inst, content, true)
		if cur != "" {
			t.Fatalf("hit should clear, got %q", cur)
		}
		inst.CurrentPick = content
		// 未中 → 锁定
		_, cur, _ = AdvancePickAfterFormalSettlement(inst.Kind, []byte(raw), inst, content, false)
		if cur != content {
			t.Fatalf("miss should keep %q, got %q", content, cur)
		}
	})

	// 7) 固定号码：无游标推进，每期复投同一注
	t.Run("fixed_number", func(t *testing.T) {
		raw := `{"runTypeId":"fixed_number","playTypeId":"dingwei","subPlayId":"sub_ge","schemeGroups":["6,8"]}`
		inst := base
		for i, hit := range []bool{true, false, true} {
			idx, cur, dir := AdvancePickAfterFormalSettlement(inst.Kind, []byte(raw), inst, "6,8", hit)
			if idx != 0 || cur != "" || dir != "" {
				t.Fatalf("period %d: fixed_number 不应推进, got %d/%q/%q", i, idx, cur, dir)
			}
		}
	})

	// 8) 内置计画（物化为定码轮换）：按内嵌类型推进
	t.Run("builtin_plan_materialized_fixed_rotate", func(t *testing.T) {
		raw := `{"runTypeId":"builtin_plan","builtinPlan":{"runTypeId":"fixed_rotate"},
			"playTypeId":"dingwei","subPlayId":"sub_ge","schemeGroups":["1,2","3,4"]}`
		inst := base
		want := []int32{1, 0, 1}
		for i, hit := range []bool{true, false, true} {
			idx, _, _ := AdvancePickAfterFormalSettlement(inst.Kind, []byte(raw), inst, "1,2", hit)
			if idx != want[i] {
				t.Fatalf("period %d: builtin materialized index=%d want %d", i, idx, want[i])
			}
			inst.PickIndex = idx
		}
	})
}

func TestPickFixedNumberIgnoresLegacyFixedPick(t *testing.T) {
	// 存量 fixedPick 条件规则忽略，固定取码只认 schemeGroups
	cfg := pickTestConfig(t, `{"runTypeId":"fixed_number","playTypeId":"dingwei","subPlayId":"sub_ge",
		"schemeGroups":["6,8"],
		"fixedPick":{"rules":[{"posStart":0,"posEnd":4,"codeMin":0,"codeMax":2,"numbers":"1,2,3"}]}}`)
	if dec := pickFixedNumber(cfg); dec.Content != "6,8" {
		t.Fatalf("应投 schemeGroups 固定号 6,8, got %+v", dec)
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
