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

func TestTriggerBetPositionIdxBaiWei(t *testing.T) {
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
				{"enabled":true,"open":"4","pos":"4","neg":"9"},
				{"enabled":true,"open":"7","pos":"7","neg":"0"}
			]
		}
	}`
	cfg := parseSchemeConfig("custom", []byte(raw), 0, 0)
	if cfg.Trigger == nil || !cfg.Trigger.HasPosition || cfg.Trigger.PositionIdx != 2 {
		t.Fatalf("trigger position: %+v", cfg.Trigger)
	}
	if cfg.Play.PositionIdx != 2 {
		t.Fatalf("Play.PositionIdx=%d want 2", cfg.Play.PositionIdx)
	}
	watch := cfg.Trigger.PositionIdxs
	// 上期 73602：百位=6，不应命中 open=7（万位）
	ballsWan := []string{"7", "3", "6", "0", "2"}
	if triggerOpenMatches(cfg.Play, ballsWan, "7", watch) {
		t.Fatal("百位方案不应按万位 7 匹配")
	}
	if !triggerOpenMatches(cfg.Play, ballsWan, "6", watch) {
		t.Fatal("百位方案应按百位 6 匹配")
	}
	// 出号应按百位编排为多行
	laid := layoutTriggerBetDingweiContent(cfg, "4")
	want := "\n\n4\n\n"
	if laid != want {
		t.Fatalf("layout=%q want %q", laid, want)
	}
	dec := pickTriggerBetPreview(cfg, sqlcdb.SchemeInstance{}, []string{"1", "2", "4", "5", "6"})
	if dec.Skip {
		t.Fatal("should not skip")
	}
	if dec.Content != want {
		t.Fatalf("pick content=%q want %q", dec.Content, want)
	}
}

func TestTriggerBetPositionIdxsMulti(t *testing.T) {
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
			"rows":[{"enabled":true,"open":"6","pos":"8","neg":"1"}]
		}
	}`
	cfg := parseSchemeConfig("custom", []byte(raw), 0, 0)
	if cfg.Trigger == nil || len(cfg.Trigger.PositionIdxs) != 2 {
		t.Fatalf("PositionIdxs=%v", cfg.Trigger)
	}
	watch := cfg.Trigger.PositionIdxs
	balls := []string{"7", "3", "6", "0", "2"}
	if !triggerOpenMatches(cfg.Play, balls, "6", watch) {
		t.Fatal("多选含百位时应匹配开出 6")
	}
	if !triggerOpenMatches(cfg.Play, balls, "7", watch) {
		t.Fatal("多选含万位时应匹配开出 7")
	}
	if triggerOpenMatches(cfg.Play, balls, "3", watch) {
		t.Fatal("未选中的千位不应参与匹配")
	}
	laid := layoutTriggerBetDingweiContent(cfg, "8")
	want := "8\n\n8\n\n"
	if laid != want {
		t.Fatalf("layout=%q want %q", laid, want)
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
func TestLayoutTriggerBetDingweiMultiNumbers(t *testing.T) {
	t.Parallel()
	raw := `{
		"runTypeId":"adv_trigger_bet",
		"playTemplate":"ssc_std",
		"playTypeId":"g006",
		"subPlayId":"13",
		"betMode":"dingwei",
		"triggerBet":{
			"mode":"always_pos",
			"positionIdxs":[2],
			"rows":[{"enabled":true,"open":"6","pos":"1,3,5","neg":"0,2,4"}]
		}
	}`
	cfg := parseSchemeConfig("custom", []byte(raw), 0, 0)
	laid := layoutTriggerBetDingweiContent(cfg, "1,3,5")
	want := "\n\n1,3,5\n\n"
	if laid != want {
		t.Fatalf("layout=%q want %q", laid, want)
	}
	// 稀疏 wire 仍原样保留
	sparse := "8,,,,"
	if got := layoutTriggerBetDingweiContent(cfg, sparse); got != sparse {
		t.Fatalf("sparse wire layout=%q want %q", got, sparse)
	}
	meta := guajibet.ParseRuleMeta("ssc_std", "g006", "13", "一星定位胆", "一星", nil, "13")
	wire := guajibet.FormatBetContentForRule(meta, laid)
	if wire != ",,135,," {
		t.Fatalf("wire=%q want ,,135,,", wire)
	}
}

// TestTriggerBetPerPositionWanBaiGe 上期 17232、选万/百/个、开出 N→正投 N：
// 应得 1,,2,,2，而不是把万位 1 复制成 1,,1,,1。
func TestTriggerBetPerPositionWanBaiGe(t *testing.T) {
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
			"positionIdxs":[0,2,4],
			"rows":[` + strings.Join(rows, ",") + `]
		}
	}`
	cfg := parseSchemeConfig("custom", []byte(raw), 0, 0)
	dec := pickTriggerBetPreview(cfg, sqlcdb.SchemeInstance{}, []string{"1", "7", "2", "3", "2"})
	if dec.Skip {
		t.Fatal("should not skip")
	}
	wantLines := "1\n\n2\n\n2"
	if dec.Content != wantLines {
		t.Fatalf("content=%q want %q", dec.Content, wantLines)
	}
	meta := guajibet.ParseRuleMeta("ssc_std", "g006", "13", "一星定位胆", "一星", nil, "13")
	wire := guajibet.FormatBetContentForRule(meta, dec.Content)
	if wire != "1,,2,,2" {
		t.Fatalf("wire=%q want 1,,2,,2", wire)
	}
}
