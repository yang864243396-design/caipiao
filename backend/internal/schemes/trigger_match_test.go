package schemes

import (
	"strings"
	"testing"

	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/guajibet"
)

func TestSupportsAdvTriggerBetPC28(t *testing.T) {
	cases := []struct {
		typeID, subID, group, label string
		want                        bool
	}{
		{"pc28_20", "hezhi", "", "", true},
		{"pc28_28", "dxds", "", "", true},
		{"pc28_20", "longhubao", "", "", true},
		{"pc28_20", "teshu", "", "", false},
		{"dingwei", "dingwei_ge", "", "", true},
		{"longhu", "lh_1v10", "", "", true},
		{"hou3", "hou3_zhixuan_fs", "", "", false},
		{"g006", "13", "一星", "定位胆", true},
		{"g010", "233", "2.0模式", "和值", true},
		{"g010", "999", "2.0模式", "特殊号", false},
	}
	for _, c := range cases {
		if got := SupportsAdvTriggerBet(c.typeID, c.subID, c.group, c.label); got != c.want {
			t.Errorf("SupportsAdvTriggerBet(%q,%q,%q,%q)=%v want %v", c.typeID, c.subID, c.group, c.label, got, c.want)
		}
	}
}

func TestTriggerOpenMatchesSSCHezhi(t *testing.T) {
	t.Parallel()
	// 中三直选和值：千+百+十，不是某一球号
	rule := playRule{
		PlayTemplate: "ssc_std", BetMode: "hezhi",
		PlayTypeID: "g002", SegmentStart: 1, SegmentLen: 3,
	}
	balls := []string{"9", "1", "2", "3", "0"} // 中三 1+2+3=6
	if !triggerOpenMatches(rule, balls, "6") {
		t.Fatal("zhong3 hezhi open 6 should match")
	}
	if !triggerOpenMatches(rule, balls, "06") {
		t.Fatal("padded open 06 should normalize-match")
	}
	if triggerOpenMatches(rule, balls, "1") {
		t.Fatal("must not match a single ball digit as hezhi open")
	}
}

func TestTriggerOpenMatchesSSCTeshu(t *testing.T) {
	t.Parallel()
	// 中三特殊号：开出=豹子/对子/顺子，按区位形态匹配（勿当球号比）
	rule := playRule{
		PlayTemplate: "ssc_std", BetMode: "teshu",
		PlayTypeID: "g002", SubPlayID: "25", CatalogSubID: "25",
		SegmentStart: 1, SegmentLen: 3,
	}
	// 万千百十个：9|8|2|2|0 → 中三 822 对子
	duizi := []string{"9", "8", "2", "2", "0"}
	if !triggerOpenMatches(rule, duizi, "对子") {
		t.Fatal("zhong3 822 should match open 对子")
	}
	if triggerOpenMatches(rule, duizi, "豹子") || triggerOpenMatches(rule, duizi, "顺子") {
		t.Fatal("zhong3 822 must not match 豹子/顺子")
	}
	// 中三 123 顺子
	shunzi := []string{"9", "1", "2", "3", "0"}
	if !triggerOpenMatches(rule, shunzi, "顺子") {
		t.Fatal("zhong3 123 should match open 顺子")
	}
	// 中三 888 豹子
	baozi := []string{"9", "8", "8", "8", "0"}
	if !triggerOpenMatches(rule, baozi, "豹子") {
		t.Fatal("zhong3 888 should match open 豹子")
	}
}

func TestTriggerOpenMatchesPC28(t *testing.T) {
	rule := playRule{PlayTemplate: "pc28_std", BetMode: "hezhi"}
	balls := []string{"3", "5", "7"} // sum=15
	if !triggerOpenMatches(rule, balls, "15") {
		t.Fatal("hezhi open 15 should match")
	}
	if triggerOpenMatches(rule, balls, "14") {
		t.Fatal("hezhi open 14 should not match")
	}

	rule.BetMode = "dxds"
	if !triggerOpenMatches(rule, balls, "大") || !triggerOpenMatches(rule, balls, "单") {
		t.Fatal("dxds 大/单 should match sum=15")
	}
	if triggerOpenMatches(rule, balls, "小") || triggerOpenMatches(rule, balls, "双") {
		t.Fatal("dxds 小/双 should not match sum=15")
	}

	rule.BetMode = "longhubao"
	balls = []string{"9", "2", "1"}
	if !triggerOpenMatches(rule, balls, "龙") {
		t.Fatal("9 vs 1 should match 龙")
	}
	if triggerOpenMatches(rule, balls, "虎") {
		t.Fatal("9 vs 1 should not match 虎")
	}
}

func TestIsLonghuPlayExcludesLonghubao(t *testing.T) {
	if isLonghuPlay(playRule{BetMode: "longhubao"}) {
		t.Fatal("longhubao must not be treated as longhu")
	}
	if !isLonghuPlay(playRule{BetMode: "longhu"}) {
		t.Fatal("longhu bet mode should match")
	}
}

// TestTriggerBetDingweiFivePanelIgnoresLegacyPositionIdx 一星五位面板已取消投注位：
// 旧 positionIdx 被忽略，始终按万～个全位匹配/出号。
func TestTriggerBetDingweiFivePanelIgnoresLegacyPositionIdx(t *testing.T) {
	t.Parallel()
	raw := `{
		"runTypeId":"adv_trigger_bet",
		"playTemplate":"ssc_std",
		"playTypeId":"g006",
		"subPlayId":"13",
		"betMode":"dingwei",
		"triggerBet":{
			"mode":"always_pos",
			"positionIdx":2,
			"rows":[
				{"enabled":true,"open":"4","pos":"4\n4\n4\n4\n4","neg":"9\n9\n9\n9\n9"},
				{"enabled":true,"open":"1","pos":"1\n1\n1\n1\n1","neg":"0\n0\n0\n0\n0"}
			]
		}
	}`
	cfg := parseSchemeConfig("custom", []byte(raw), 0, 0)
	if cfg.Trigger == nil || cfg.Trigger.HasPosition {
		t.Fatalf("五位面板应忽略旧投注位: %+v", cfg.Trigger)
	}
	if got := cfg.Trigger.PositionIdxs; len(got) != 5 {
		t.Fatalf("PositionIdxs=%v want 5 位全选", got)
	}
	if !isDingweiFivePanelPlay(cfg.Play) {
		t.Fatalf("want five-panel play, got %+v", cfg.Play)
	}
	// 上期 12456：万=1 → 各位正投 1
	// 上期各位均为 1 → 全用 open=1 的正投行
	dec := pickTriggerBetPreview(cfg, sqlcdb.SchemeInstance{}, []string{"1", "1", "1", "1", "1"})
	if dec.Skip {
		t.Fatal("should not skip")
	}
	want := "1\n1\n1\n1\n1"
	if dec.Content != want {
		t.Fatalf("pick content=%q want %q", dec.Content, want)
	}
}

// TestTriggerBetDingweiFivePanelWatchesAllPositions 旧 positionIdxs 多选亦展开为五位全位。
func TestTriggerBetDingweiFivePanelWatchesAllPositions(t *testing.T) {
	t.Parallel()
	raw := `{
		"runTypeId":"adv_trigger_bet",
		"playTemplate":"ssc_std",
		"playTypeId":"g006",
		"subPlayId":"13",
		"betMode":"dingwei",
		"triggerBet":{
			"mode":"always_pos",
			"positionIdxs":[0,2],
			"rows":[{"enabled":true,"open":"6","pos":"8\n8\n8\n8\n8","neg":"1\n1\n1\n1\n1"}]
		}
	}`
	cfg := parseSchemeConfig("custom", []byte(raw), 0, 0)
	if cfg.Trigger == nil || len(cfg.Trigger.PositionIdxs) != 5 {
		t.Fatalf("PositionIdxs=%v want 5", cfg.Trigger)
	}
	watch := cfg.Trigger.PositionIdxs
	balls := []string{"7", "3", "6", "0", "2"}
	for _, open := range []string{"7", "3", "6", "0", "2"} {
		if !triggerOpenMatches(cfg.Play, balls, open, watch) {
			t.Fatalf("五位全位应匹配开出 %s", open)
		}
	}
}

// TestTriggerBetQian3FushiPerPosition 前三直选复式：按万/千/百各自开出映射，
// 且取该行对应位的正投（pos 换行分位）。
func TestTriggerBetQian3FushiPerPosition(t *testing.T) {
	t.Parallel()
	rows := make([]string, 0, 10)
	for i := 0; i <= 9; i++ {
		d := string(rune('0' + i))
		// 万/千/百正投分别为 d / (d+1)%10 / (d+2)%10
		pos := d + `\n` + string(rune('0'+((i+1)%10))) + `\n` + string(rune('0'+((i+2)%10)))
		neg := string(rune('0'+(9-i))) + `\n` + string(rune('0'+((8-i+10)%10))) + `\n` + string(rune('0'+((7-i+10)%10)))
		rows = append(rows, `{"enabled":true,"open":"`+d+`","pos":"`+pos+`","neg":"`+neg+`"}`)
	}
	raw := `{
		"runTypeId":"adv_trigger_bet",
		"playTemplate":"ssc_std",
		"playTypeId":"g001",
		"subPlayId":"1",
		"betMode":"fushi",
		"triggerBet":{
			"mode":"always_pos",
			"rows":[` + strings.Join(rows, ",") + `]
		}
	}`
	cfg := parseSchemeConfig("custom", []byte(raw), 0, 0)
	if cfg.Play.SegmentLen != 3 || cfg.Play.SegmentStart != 0 {
		t.Fatalf("segment=%d,%d want 0,3", cfg.Play.SegmentStart, cfg.Play.SegmentLen)
	}
	if !triggerBetUsesPosition(cfg.Play) {
		t.Fatal("前三直选复式应按位出号")
	}
	if isDingweiTriggerPlay(cfg.Play) {
		t.Fatal("前三复式不应当作定位胆改写段")
	}
	// 上期 172xx：万开1→取行1万位正投1；千开7→行7千位正投8；百开2→行2百位正投4
	dec := pickTriggerBetPreview(cfg, sqlcdb.SchemeInstance{}, []string{"1", "7", "2", "3", "2"})
	if dec.Skip {
		t.Fatal("should not skip")
	}
	want := "1\n8\n4"
	if dec.Content != want {
		t.Fatalf("content=%q want %q", dec.Content, want)
	}
	meta := guajibet.ParseRuleMeta("ssc_std", "g001", "1", "前三直选复式", "前三码", nil, "1")
	wire := guajibet.FormatBetContentForRule(meta, dec.Content)
	if wire != "1,8,4" {
		t.Fatalf("wire=%q want 1,8,4", wire)
	}
}

// TestLayoutTriggerBetDingweiMultiNumbers 正投多号「1,3,5」应按位编排，不能误判为五段 wire。
// TestLayoutTriggerBetDingweiFivePanelKeepsMultiline 五位面板出号已是多行时原样保留；
// 稀疏 wire 亦原样保留。
func TestLayoutTriggerBetDingweiFivePanelKeepsMultiline(t *testing.T) {
	t.Parallel()
	raw := `{
		"runTypeId":"adv_trigger_bet",
		"playTemplate":"ssc_std",
		"playTypeId":"g006",
		"subPlayId":"13",
		"betMode":"dingwei",
		"triggerBet":{
			"mode":"always_pos",
			"rows":[{"enabled":true,"open":"6","pos":"1,3,5\n\n1,3,5\n\n1,3,5","neg":"0"}]
		}
	}`
	cfg := parseSchemeConfig("custom", []byte(raw), 0, 0)
	multi := "1,3,5\n\n1,3,5\n\n1,3,5"
	if got := layoutTriggerBetDingweiContent(cfg, multi); got != multi {
		t.Fatalf("multiline layout=%q want unchanged", got)
	}
	sparse := "8,,,,"
	if got := layoutTriggerBetDingweiContent(cfg, sparse); got != sparse {
		t.Fatalf("sparse wire layout=%q want %q", got, sparse)
	}
	meta := guajibet.ParseRuleMeta("ssc_std", "g006", "13", "一星定位胆", "一星", nil, "13")
	wire := guajibet.FormatBetContentForRule(meta, multi)
	if wire != "135,,135,,135" {
		t.Fatalf("wire=%q want 135,,135,,135", wire)
	}
}

// TestTriggerBetDingweiFivePanelPerPos 一星五位：上期 17232，开出 N→正投 N，
// 万～个全位出号 1\n7\n2\n3\n2（同前三码按位预备号）。
func TestTriggerBetDingweiFivePanelPerPos(t *testing.T) {
	t.Parallel()
	rows := make([]string, 0, 10)
	for i := 0; i <= 9; i++ {
		d := string(rune('0' + i))
		rows = append(rows, `{"enabled":true,"open":"`+d+`","pos":"`+d+`","neg":"`+string(rune('0'+(9-i)))+`"}`)
	}
	raw := `{
		"runTypeId":"adv_trigger_bet",
		"playTemplate":"ssc_std",
		"playTypeId":"g006",
		"subPlayId":"13",
		"betMode":"dingwei",
		"triggerBet":{
			"mode":"always_pos",
			"rows":[` + strings.Join(rows, ",") + `]
		}
	}`
	cfg := parseSchemeConfig("custom", []byte(raw), 0, 0)
	if !isDingweiFivePanelPlay(cfg.Play) {
		t.Fatalf("want five-panel, play=%+v", cfg.Play)
	}
	dec := pickTriggerBetPreview(cfg, sqlcdb.SchemeInstance{}, []string{"1", "7", "2", "3", "2"})
	if dec.Skip {
		t.Fatal("should not skip")
	}
	wantLines := "1\n7\n2\n3\n2"
	if dec.Content != wantLines {
		t.Fatalf("content=%q want %q", dec.Content, wantLines)
	}
	meta := guajibet.ParseRuleMeta("ssc_std", "g006", "13", "一星定位胆", "一星", nil, "13")
	wire := guajibet.FormatBetContentForRule(meta, dec.Content)
	if wire != "1,7,2,3,2" {
		t.Fatalf("wire=%q want 1,7,2,3,2", wire)
	}
}

// TestTriggerBetNoMatchSkipsPeriod Q4c：开出未命中任何启用行时本期跳过，不回退启用第 1 行。
func TestTriggerBetNoMatchSkipsPeriod(t *testing.T) {
	t.Parallel()

	// 整期玩法（龙虎）：开出未命中 → Skip
	longhuRaw := `{
		"runTypeId":"adv_trigger_bet",
		"playTemplate":"ssc_std",
		"playTypeId":"longhu",
		"subPlayId":"lh_1v10",
		"betMode":"longhu",
		"triggerBet":{
			"mode":"always_pos",
			"rows":[
				{"enabled":true,"open":"龙","pos":"龙","neg":"虎"},
				{"enabled":true,"open":"和","pos":"和","neg":"龙"}
			]
		}
	}`
	longhuCfg := parseSchemeConfig("custom", []byte(longhuRaw), 0, 0)
	// 万=1 个=9 → 虎，映射无「虎」
	dec := resolveTriggerBetDecision(longhuCfg, []string{"1", "2", "3", "4", "9"}, "")
	if !dec.Skip {
		t.Fatalf("龙虎未命中应 Skip，got content=%q", dec.Content)
	}

	// 按位（一星）：仅启用 open=0，上期各位为 1 → Skip
	dingweiRaw := `{
		"runTypeId":"adv_trigger_bet",
		"playTemplate":"ssc_std",
		"playTypeId":"g006",
		"subPlayId":"13",
		"betMode":"dingwei",
		"triggerBet":{
			"mode":"always_pos",
			"rows":[{"enabled":true,"open":"0","pos":"0\n0\n0\n0\n0","neg":"9\n9\n9\n9\n9"}]
		}
	}`
	dingweiCfg := parseSchemeConfig("custom", []byte(dingweiRaw), 0, 0)
	dec = resolveTriggerBetDecision(dingweiCfg, []string{"1", "1", "1", "1", "1"}, "")
	if !dec.Skip {
		t.Fatalf("一星未命中应 Skip，got content=%q", dec.Content)
	}

	// 前三复式：段内任一位未命中 → Skip（不回退第一行）
	fushiRaw := `{
		"runTypeId":"adv_trigger_bet",
		"playTemplate":"ssc_std",
		"playTypeId":"g001",
		"subPlayId":"1",
		"betMode":"fushi",
		"triggerBet":{
			"mode":"always_pos",
			"rows":[
				{"enabled":true,"open":"1","pos":"1\n2\n3","neg":"0\n0\n0"},
				{"enabled":true,"open":"2","pos":"2\n3\n4","neg":"0\n0\n0"}
			]
		}
	}`
	fushiCfg := parseSchemeConfig("custom", []byte(fushiRaw), 0, 0)
	// 万=1 命中，千=7 未命中
	dec = resolveTriggerBetDecision(fushiCfg, []string{"1", "7", "2", "3", "2"}, "")
	if !dec.Skip {
		t.Fatalf("前三段内未命中应 Skip，got content=%q", dec.Content)
	}
}
