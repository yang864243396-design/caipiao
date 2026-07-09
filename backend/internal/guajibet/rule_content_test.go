package guajibet

import (
	"encoding/json"
	"testing"
)

func qian3FushiMeta() RuleMeta {
	seg, _ := json.Marshal(map[string]string{
		"guajiGroup":    "前三码",
		"guajiTeam":     "前三直选",
		"guajiFullName": "前三直选复式",
		"guajiRuleId":   "1",
	})
	return ParseRuleMeta("ssc_std", "g001", "1", "前三直选复式", "前三码", seg, "1")
}

func dingweiMeta() RuleMeta {
	seg, _ := json.Marshal(map[string]string{
		"guajiGroup":    "一星",
		"guajiTeam":     "定位胆",
		"guajiFullName": "定位胆 · 万位",
		"guajiRuleId":   "13",
	})
	return ParseRuleMeta("ssc_std", "g006", "13", "定位胆 · 万位", "一星", seg, "13")
}

func TestFormatBetContentForRule_qian3Fushi(t *testing.T) {
	meta := qian3FushiMeta()
	got := FormatBetContentForRule(meta, "1\n1\n1")
	if got != "1,1,1" {
		t.Fatalf("wire=%q want 1,1,1", got)
	}
	if !NeedsSoloForRule(meta, got) {
		t.Fatal("直选复式单注应 solo")
	}
	if n := CountBetNums(meta, got); n != 1 {
		t.Fatalf("betsNums=%d want 1", n)
	}
}

func TestFormatBetContentForRule_dingwei(t *testing.T) {
	meta := dingweiMeta()
	got := FormatBetContentForRule(meta, "7")
	if got != "7,,,," {
		t.Fatalf("wire=%q want 7,,,,", got)
	}
	if n := CountBetNums(meta, got); n != 1 {
		t.Fatalf("betsNums=%d want 1", n)
	}
	if NeedsSoloForRule(meta, got) {
		t.Fatal("v6hs1 单注定位胆不应 solo")
	}
}

func TestSampleGroupContent_minSingleBet(t *testing.T) {
	meta := qian3FushiMeta()
	content := SampleGroupContent(meta)
	wire := FormatBetContentForRule(meta, content)
	if n := CountBetNums(meta, wire); n != 1 {
		t.Fatalf("sample betsNums=%d want 1 content=%q wire=%q", n, content, wire)
	}
}

func TestResolveBetsNums_fallbackDingwei(t *testing.T) {
	meta := RuleMeta{}
	wire := "39,,,,"
	if n := ResolveBetsNums(meta, wire, 4, 2, 1); n != 2 {
		t.Fatalf("got %d want 2", n)
	}
}

func TestResolveSolo_weishu(t *testing.T) {
	meta := ParseRuleMeta("ssc_std", "g001", "11", "和值尾数", "前三码", nil, "11")
	if NeedsSoloForRule(meta, "6") {
		t.Fatal("和值尾数不应 solo")
	}
}

func TestCountBetNums_baodan(t *testing.T) {
	meta := ParseRuleMeta("ssc_std", "g001", "9", "前三组选包胆", "前三码", nil, "9")
	wire := FormatBetContentForRule(meta, "3")
	if wire != "3" {
		t.Fatalf("wire=%q want 3", wire)
	}
	if n := CountBetNums(meta, wire); n != 54 {
		t.Fatalf("betsNums=%d want 54", n)
	}
	if NeedsSoloForRule(meta, wire) {
		t.Fatal("包胆不应 solo")
	}
}

func TestCountBetNums_zuxuanHezhi(t *testing.T) {
	meta := ParseRuleMeta("ssc_std", "g001", "8", "前三组选和值", "前三码", nil, "8")
	if n := CountBetNums(meta, "6"); n != 6 {
		t.Fatalf("sum6 betsNums=%d want 6", n)
	}
	if NeedsSoloForRule(meta, "6") {
		t.Fatal("组选和值不应 solo")
	}
	meta2 := ParseRuleMeta("ssc_std", "g004", "44", "组选和值", "前二", nil, "44")
	if n := CountBetNums(meta2, "6"); n != 3 {
		t.Fatalf("qian2 sum6 betsNums=%d want 3", n)
	}
	seg, _ := json.Marshal(map[string]string{"guajiGroup": "前中后三"})
	meta4 := ParseRuleMeta("ssc_std", "g007", "108", "组选和值", "前中后三", seg, "108")
	if n := CountBetNums(meta4, "6"); n != 18 {
		t.Fatalf("qianzhonghou3 sum6 betsNums=%d want 18", n)
	}
}

func TestFormatBetContentForRule_zuxuanFs(t *testing.T) {
	meta := ParseRuleMeta("ssc_std", "g004", "42", "组选复式", "前二", nil, "42")
	got := FormatBetContentForRule(meta, "1\n2")
	if got != "1,2" {
		t.Fatalf("wire=%q want 1,2", got)
	}
	if n := CountBetNums(meta, got); n != 1 {
		t.Fatalf("betsNums=%d want 1", n)
	}
}

func TestFormatBetContentForRule_zuxuanDs(t *testing.T) {
	meta := ParseRuleMeta("ssc_std", "g004", "43", "组选单式", "前二", nil, "43")
	got := FormatBetContentForRule(meta, "12")
	if got != "12" {
		t.Fatalf("wire=%q want 12", got)
	}
}

func TestCountBetNums_baodanQian2(t *testing.T) {
	meta := ParseRuleMeta("ssc_std", "g004", "45", "组选包胆", "前二", nil, "45")
	if n := CountBetNums(meta, "3"); n != 9 {
		t.Fatalf("qian2 baodan betsNums=%d want 9", n)
	}
}

func TestResolveSolo_ruleMeta(t *testing.T) {
	meta := dingweiMeta()
	if ResolveSolo(meta, "7,,,,", 1) {
		t.Fatal("v6hs1 定位胆单注不应 solo")
	}
	if ResolveSolo(meta, "39,,,,", 2) {
		t.Fatal("定位胆多注不应 solo")
	}
}

func TestCountZuxuanSumCombinations_values(t *testing.T) {
	if n := countZuxuanSumCombinations(6, 2); n != 3 {
		t.Fatalf("segLen2 sum6=%d want 3", n)
	}
	if n := countZuxuanSumCombinations(6, 3); n != 6 {
		t.Fatalf("segLen3 sum6=%d want 6", n)
	}
	if n := countZuxuanSumCombinations(6, 4); n != 24 {
		t.Fatalf("segLen4 sum6=%d want 24", n)
	}
}

func TestSegmentRange_qianzhonghou3(t *testing.T) {
	seg, _ := json.Marshal(map[string]string{"guajiGroup": "前中后三"})
	meta := ParseRuleMeta("ssc_std", "g007", "101", "直选复式", "前中后三", seg, "101")
	_, segLen := segmentRange(meta)
	if segLen != 3 {
		t.Fatalf("g007 segLen=%d want 3", segLen)
	}
	if segmentBetMultiplier(meta) != 3 {
		t.Fatalf("multiplier=%d want 3", segmentBetMultiplier(meta))
	}
}

func TestCountBetNums_qianzhonghou3Fushi(t *testing.T) {
	seg, _ := json.Marshal(map[string]string{"guajiGroup": "前中后三"})
	meta := ParseRuleMeta("ssc_std", "g007", "101", "直选复式", "前中后三", seg, "101")
	wire := FormatBetContentForRule(meta, "1\n1\n1")
	if wire != "1,1,1" {
		t.Fatalf("wire=%q want 1,1,1", wire)
	}
	if n := CountBetNums(meta, wire); n != 3 {
		t.Fatalf("betsNums=%d want 3", n)
	}
	if NeedsSoloForRule(meta, wire) {
		t.Fatal("前中后三不应 solo")
	}
}

func TestCountBetNums_qianhou3Fushi(t *testing.T) {
	seg, _ := json.Marshal(map[string]string{"guajiGroup": "前后三"})
	meta := ParseRuleMeta("ssc_std", "g012", "89", "直选复式", "前后三", seg, "89")
	wire := FormatBetContentForRule(meta, "1\n1\n1")
	if n := CountBetNums(meta, wire); n != 2 {
		t.Fatalf("betsNums=%d want 2", n)
	}
	if NeedsSoloForRule(meta, wire) {
		t.Fatal("前后三不应 solo")
	}
}

func TestCountBetNums_qianhou2Fushi(t *testing.T) {
	seg, _ := json.Marshal(map[string]string{"guajiGroup": "前后二"})
	meta := ParseRuleMeta("ssc_std", "g008", "119", "直选复式", "前后二", seg, "119")
	wire := FormatBetContentForRule(meta, "1\n1")
	if n := CountBetNums(meta, wire); n != 2 {
		t.Fatalf("betsNums=%d want 2", n)
	}
	if !NeedsSoloForRule(meta, wire) {
		t.Fatal("前后二最小注应 solo")
	}
}

func TestCountBetNums_sixingZu24(t *testing.T) {
	seg, _ := json.Marshal(map[string]string{"guajiGroup": "四星"})
	meta := ParseRuleMeta("ssc_std", "g013", "130", "组选24", "四星", seg, "130")
	if n := CountBetNums(meta, "1,2,3,4"); n != 1 {
		t.Fatalf("zu24 betsNums=%d want 1", n)
	}
	if NeedsSoloForRule(meta, "1,2,3,4") {
		t.Fatal("zu24 不应 solo")
	}
}

func TestCountBetNums_sixingZu6(t *testing.T) {
	seg, _ := json.Marshal(map[string]string{"guajiGroup": "四星"})
	meta := ParseRuleMeta("ssc_std", "g013", "132", "组选6", "四星", seg, "132")
	if mode := InferBetMode(meta); mode != "zu6" {
		t.Fatalf("mode=%q want zu6", mode)
	}
	if n := CountBetNums(meta, "1,2,3"); n != 3 {
		t.Fatalf("zu6 betsNums=%d want 3", n)
	}
	if n := CountBetNums(meta, "1,2,3,4"); n != 6 {
		t.Fatalf("zu6 n=4 betsNums=%d want 6", n)
	}
}

func TestCountBetNums_renxuanRen2(t *testing.T) {
	seg, _ := json.Marshal(map[string]string{"guajiGroup": "任选", "guajiTeam": "任选二"})
	meta := ParseRuleMeta("ssc_std", "g011", "74", "直选复式", "任选", seg, "74")
	if n := CountBetNums(meta, "1,2,3,4,5"); n != 10 {
		t.Fatalf("ren2 betsNums=%d want 10", n)
	}
	if NeedsSoloForRule(meta, "1,2,3,4,5") {
		t.Fatal("任二直选复式不应 solo")
	}
}

func TestNeedsSolo_budingweiTwoCode(t *testing.T) {
	meta := ParseRuleMeta("ssc_std", "g009", "114", "前三二码不定位", "不定位", nil, "114")
	if !NeedsSoloForRule(meta, "1,2") {
		t.Fatal("三星二码不定位应 solo")
	}
	meta1 := ParseRuleMeta("ssc_std", "g009", "113", "前三一码不定位", "不定位", nil, "113")
	if NeedsSoloForRule(meta1, "3") {
		t.Fatal("前三一码不定位不应 solo")
	}
	metaHou3 := ParseRuleMeta("ssc_std", "g009", "117", "后三一码不定位", "不定位", nil, "117")
	if !NeedsSoloForRule(metaHou3, "3") {
		t.Fatal("后三一码不定位应 solo")
	}
	metaQian4 := ParseRuleMeta("ssc_std", "g009", "147", "前四二码不定位", "不定位", nil, "147")
	if NeedsSoloForRule(metaQian4, "1,2") {
		t.Fatal("前四二码不定位不应 solo")
	}
	meta3 := ParseRuleMeta("ssc_std", "g009", "152", "五星三码不定位", "不定位", nil, "152")
	if NeedsSoloForRule(meta3, "1,2,3,4") {
		t.Fatal("五星三码不定位不应 solo")
	}
}

func TestCountBetNums_zu12Wire(t *testing.T) {
	meta := ParseRuleMeta("ssc_std", "g013", "131", "组选12", "四星", nil, "131")
	if n := CountBetNums(meta, "12,34"); n != 2 {
		t.Fatalf("zu12 betsNums=%d want 2", n)
	}
	metaQh4Seg, _ := json.Marshal(map[string]string{"guajiGroup": "前后四", "guajiTeam": "前后四组选", "guajiRuleId": "138"})
	metaQh4 := ParseRuleMeta("ssc_std", "g014", "138", "组选12", "前后四", metaQh4Seg, "138")
	if n := CountBetNums(metaQh4, "12,34"); n != 4 {
		t.Fatalf("前后四 zu12 betsNums=%d want 4", n)
	}
}

func TestCountBetNums_budingweiWuxing(t *testing.T) {
	meta := ParseRuleMeta("ssc_std", "g009", "151", "五星二码不定位", "不定位", nil, "151")
	if n := CountBetNums(meta, "1,2,3,4"); n != 6 {
		t.Fatalf("五星二码 betsNums=%d want 6", n)
	}
	content := SampleGroupContent(meta)
	if content != "1,2,3,4" {
		t.Fatalf("sample=%q want 1,2,3,4", content)
	}
}

func TestInferBetMode_longhuPair(t *testing.T) {
	meta := ParseRuleMeta("ssc_std", "g010", "54", "万千", "龙虎斗", nil, "54")
	if mode := InferBetMode(meta); mode != "longhu" {
		t.Fatalf("mode=%q want longhu", mode)
	}
}

func TestResolveSolo_highBets(t *testing.T) {
	seg, _ := json.Marshal(map[string]string{"guajiGroup": "前中后三"})
	meta := ParseRuleMeta("ssc_std", "g007", "103", "直选和值", "前中后三", seg, "103")
	if ResolveSolo(meta, "6", 84) {
		t.Fatal("84 注前中后三直选和值不应 solo")
	}
}

func TestPC28Hezhi_rule233(t *testing.T) {
	meta := ParseRuleMeta("pc28_std", "g001", "233", "和值", "2.0", nil, "233")
	wire := FormatBetContentForRule(meta, "1,2")
	if wire != "1,2" {
		t.Fatalf("wire=%q want 1,2", wire)
	}
	if n := CountBetNums(meta, wire); n != 2 {
		t.Fatalf("betsNums=%d want 2", n)
	}
	if NeedsSoloForRule(meta, wire) {
		t.Fatal("PC28 和值不应 solo")
	}
}

func TestFormatBetContentForRule_renxuanRen3Wire(t *testing.T) {
	seg, _ := json.Marshal(map[string]string{
		"guajiGroup": "任选", "guajiTeam": "任选三", "guajiFullName": "任三直选复式",
	})
	meta := ParseRuleMeta("fast_ssc_std", "g011", "80", "直选复式", "任选", seg, "80")
	got := FormatBetContentForRule(meta, "1,2,3")
	if got != "1,2,,,3" {
		t.Fatalf("wire=%q want 1,2,,,3", got)
	}
	if n := CountBetNums(meta, got); n != 1 {
		t.Fatalf("betsNums=%d want 1", n)
	}
}

func TestFormatBetContentForRule_fastHashPlays(t *testing.T) {
	for _, tc := range []struct {
		label, content, want string
		ruleID               string
	}{
		{"尾数单双", "单", "单", "267"},
		{"尾数大小", "大", "大", "270"},
		{"幸运庄闲", "庄", "庄", "268"},
	} {
		meta := ParseRuleMeta("fast_ssc_std", "g017", tc.ruleID, tc.label, "哈希玩法", nil, tc.ruleID)
		got := FormatBetContentForRule(meta, tc.content)
		if got != tc.want {
			t.Fatalf("%s wire=%q want %q mode=%s", tc.label, got, tc.want, InferBetMode(meta))
		}
		if NeedsSoloForRule(meta, got) {
			t.Fatalf("%s should not solo", tc.label)
		}
	}
}

func TestFormatBetContentForRule_renxuanRen2Wire(t *testing.T) {
	seg, _ := json.Marshal(map[string]string{"guajiGroup": "任选", "guajiTeam": "任选二"})
	meta := ParseRuleMeta("ssc_std", "g011", "74", "直选复式", "任选", seg, "74")
	got := FormatBetContentForRule(meta, "1,2,3,4,5")
	if got != "1,2,3,4,5" {
		t.Fatalf("wire=%q want 1,2,3,4,5", got)
	}
	if n := CountBetNums(meta, got); n != 10 {
		t.Fatalf("betsNums=%d want 10", n)
	}
}

func TestFormatBetContentForRule_syxwFushi(t *testing.T) {
	meta := ParseRuleMeta("syxw_std", "g001", "1", "前三直选复式", "前三", nil, "1")
	got := FormatBetContentForRule(meta, "1,2,3")
	if got != "01,02,03" {
		t.Fatalf("wire=%q want 01,02,03", got)
	}
	if n := CountBetNums(meta, got); n != 1 {
		t.Fatalf("betsNums=%d want 1", n)
	}
	if !NeedsSoloForRule(meta, got) {
		t.Fatal("SYXW 前三复式单注应 solo")
	}
}

func TestFormatBetContentForRule_syxwDingwei(t *testing.T) {
	meta := ParseRuleMeta("syxw_std", "dingwei", "1", "定位胆 · 第一位", "一星", nil, "1")
	got := FormatBetContentForRule(meta, "3")
	if got != "03,,,," {
		t.Fatalf("wire=%q want 03,,,,", got)
	}
	if NeedsSoloForRule(meta, got) {
		t.Fatal("SYXW 定位胆单注不应 solo")
	}
}

func TestFormatBetContentForRule_pk10Fushi(t *testing.T) {
	meta := ParseRuleMeta("pk10_std", "g001", "1", "冠亚直选复式", "冠亚", nil, "1")
	got := FormatBetContentForRule(meta, "1,2")
	if got != "01,02" {
		t.Fatalf("wire=%q want 01,02", got)
	}
	if !NeedsSoloForRule(meta, got) {
		t.Fatal("PK10 前二复式单注应 solo")
	}
}

func TestCountBetNums_k3Hezhi(t *testing.T) {
	meta := ParseRuleMeta("k3_std", "hezhi", "k3_hezhi", "快三和值", "和值", nil, "224")
	if n := CountBetNums(meta, "6"); n != 10 {
		t.Fatalf("k3 sum6 betsNums=%d want 10", n)
	}
}

func TestCountBetNums_k3ErtongFu(t *testing.T) {
	meta := ParseRuleMeta("k3_std", "tonghao", "ertong_fu", "二同号复选", "同号", nil, "226")
	if n := CountBetNums(meta, "1,2,3"); n != 3 {
		t.Fatalf("ertong_fu betsNums=%d want 3", n)
	}
}

func TestFormatBetContentForRule_renxuanZuxuanFs(t *testing.T) {
	seg, _ := json.Marshal(map[string]string{"guajiGroup": "任选", "guajiTeam": "任选二"})
	meta := ParseRuleMeta("ssc_std", "g011", "77", "组选复式", "任选", seg, "77")
	got := FormatBetContentForRule(meta, "千,个\n1,2")
	if got != "千个|1,2" {
		t.Fatalf("wire=%q want 千个|1,2", got)
	}
	if n := CountBetNums(meta, got); n != 1 {
		t.Fatalf("betsNums=%d want 1", n)
	}
	if NeedsSoloForRule(meta, got) {
		t.Fatal("任二组选复式不应 solo")
	}
}

func TestCountBetNums_qian2ZhixuanHezhi(t *testing.T) {
	meta := ParseRuleMeta("ssc_std", "g004", "40", "直选和值", "前二", nil, "40")
	got := FormatBetContentForRule(meta, "1,2")
	if got != "1,2" {
		t.Fatalf("wire=%q want 1,2", got)
	}
	if n := CountBetNums(meta, got); n != 5 {
		t.Fatalf("betsNums=%d want 5", n)
	}
	if !NeedsSoloForRule(meta, got) {
		t.Fatal("前二直选和值应 solo")
	}
}

func TestFormatBetContentForRule_renxuanHezhi(t *testing.T) {
	seg, _ := json.Marshal(map[string]string{"guajiGroup": "任选", "guajiTeam": "任选二"})
	meta := ParseRuleMeta("ssc_std", "g011", "76", "直选和值", "任选", seg, "76")
	got := FormatBetContentForRule(meta, "千,个\n1,2")
	if got != "千个|1,2" {
		t.Fatalf("wire=%q want 千个|1,2", got)
	}
	if n := CountBetNums(meta, got); n != 5 {
		t.Fatalf("betsNums=%d want 5", n)
	}
	if NeedsSoloForRule(meta, got) {
		t.Fatal("任二直选和值多注不应 solo")
	}
	wireMin := FormatBetContentForRule(meta, SampleGroupContent(meta))
	if gotMin := wireMin; gotMin != "千个|0" {
		t.Fatalf("sample hezhi wire=%q want 千个|0", gotMin)
	}
	if n := CountBetNums(meta, wireMin); n != 1 {
		t.Fatalf("sample hezhi betsNums=%d want 1 wire=%q", n, wireMin)
	}
	if !NeedsSoloForRule(meta, wireMin) {
		t.Fatal("任二直选和值单注应 solo")
	}
}

func TestFormatBetContentForRule_renxuanZuxuanHezhi(t *testing.T) {
	seg, _ := json.Marshal(map[string]string{"guajiGroup": "任选", "guajiTeam": "任选二"})
	meta := ParseRuleMeta("ssc_std", "g011", "79", "组选和值", "任选", seg, "79")
	got := FormatBetContentForRule(meta, "千,个\n1,2")
	if got != "千个|1,2" {
		t.Fatalf("wire=%q want 千个|1,2", got)
	}
	if n := CountBetNums(meta, got); n != 2 {
		t.Fatalf("betsNums=%d want 2", n)
	}
	wireMin := FormatBetContentForRule(meta, SampleGroupContent(meta))
	if wireMin != "千个|1" {
		t.Fatalf("sample zu hezhi wire=%q want 千个|1", wireMin)
	}
	if NeedsSoloForRule(meta, wireMin) {
		t.Fatal("任二组选和值单注不应 solo")
	}
}

func TestFormatBetContentForRule_renxuanDanshi(t *testing.T) {
	seg, _ := json.Marshal(map[string]string{"guajiGroup": "任选", "guajiTeam": "任选二"})
	meta := ParseRuleMeta("ssc_std", "g011", "75", "直选单式", "任选", seg, "75")
	got := FormatBetContentForRule(meta, "千,个\n12")
	if got != "千个|12" {
		t.Fatalf("wire=%q want 千个|12", got)
	}
	if n := CountBetNums(meta, got); n != 1 {
		t.Fatalf("betsNums=%d want 1", n)
	}
}

func TestFormatBetContentForRule_dxdsHou2(t *testing.T) {
	meta := ParseRuleMeta("ssc_std", "g016", "261", "后二大小单双", "后二", nil, "261")
	got := FormatBetContentForRule(meta, "大\n大")
	if got != "大,大" {
		t.Fatalf("wire=%q want 大,大", got)
	}
	if n := CountBetNums(meta, got); n != 1 {
		t.Fatalf("betsNums=%d want 1", n)
	}
	if NeedsSoloForRule(meta, got) {
		t.Fatal("后二大小单双不应 solo")
	}
}

func TestInferBetMode_wuxingTeshu(t *testing.T) {
	meta := ParseRuleMeta("ssc_std", "g015", "162", "一帆风顺", "五星", nil, "162")
	if mode := InferBetMode(meta); mode != "teshu" {
		t.Fatalf("mode=%q want teshu", mode)
	}
	if NeedsSoloForRule(meta, "6") {
		t.Fatal("一帆风顺不应 solo")
	}
}

func TestCountBetNums_syxwDingwei(t *testing.T) {
	meta := ParseRuleMeta("syxw_std", "dingwei", "1", "定位胆 · 第一位", "一星", nil, "1")
	got := FormatBetContentForRule(meta, "7")
	if got != "07,,,," {
		t.Fatalf("wire=%q want 07,,,,", got)
	}
	if n := CountBetNums(meta, got); n != 1 {
		t.Fatalf("betsNums=%d want 1", n)
	}
}

func TestFormatBetContentForRule_wuxingHzDs(t *testing.T) {
	meta := ParseRuleMeta("ssc_std", "g016", "263", "五星和值单双", "五星", nil, "263")
	if mode := InferBetMode(meta); mode != "danshuang" {
		t.Fatalf("mode=%q want danshuang", mode)
	}
	got := FormatBetContentForRule(meta, "单")
	if got != "单" {
		t.Fatalf("wire=%q want 单", got)
	}
}

func TestFormatBetContentForRule_syxwRenxuan(t *testing.T) {
	meta := ParseRuleMeta("syxw_std", "g005", "176", "任选一中一", "任选", nil, "176")
	got := FormatBetContentForRule(meta, "1")
	if got != "01" {
		t.Fatalf("wire=%q want 01", got)
	}
	if NeedsSoloForRule(meta, got) {
		t.Fatal("任选一单注不应 solo")
	}
	meta2 := ParseRuleMeta("syxw_std", "g005", "177", "任选二中二", "任选", nil, "177")
	got2 := FormatBetContentForRule(meta2, "1\n2")
	if got2 != "01,02" {
		t.Fatalf("wire=%q want 01,02", got2)
	}
	if NeedsSoloForRule(meta2, got2) {
		t.Fatal("任选二单注不应 solo")
	}
	meta4 := ParseRuleMeta("syxw_std", "g005", "179", "任选四中四", "任选", nil, "179")
	wire4 := FormatBetContentForRule(meta4, SampleGroupContent(meta4))
	if wire4 != "01,02,03,04" {
		t.Fatalf("wire=%q want 01,02,03,04", wire4)
	}
	if !NeedsSoloForRule(meta4, wire4) {
		t.Fatal("任选四单注应 solo")
	}
	metaDs := ParseRuleMeta("syxw_std", "g006", "185", "任选二中二", "任选", nil, "185")
	wireDs := FormatBetContentForRule(metaDs, SampleGroupContent(metaDs))
	if wireDs != "0102" {
		t.Fatalf("wire=%q want 0102", wireDs)
	}
	if NeedsSoloForRule(metaDs, wireDs) {
		t.Fatal("任选单式二单注不应 solo")
	}
	metaDs4 := ParseRuleMeta("syxw_std", "g006", "187", "任选四中四", "任选", nil, "187")
	wireDs4 := FormatBetContentForRule(metaDs4, SampleGroupContent(metaDs4))
	if wireDs4 != "01020304" {
		t.Fatalf("wire=%q want 01020304", wireDs4)
	}
	if !NeedsSoloForRule(metaDs4, wireDs4) {
		t.Fatal("任选单式四单注应 solo")
	}
}

func TestFormatBetContentForRule_syxwDanshiZuxuan(t *testing.T) {
	meta := ParseRuleMeta("syxw_std", "g001", "167", "直选单式", "前三", nil, "167")
	wire := FormatBetContentForRule(meta, SampleGroupContent(meta))
	if wire != "010203" {
		t.Fatalf("wire=%q want 010203", wire)
	}
	if !NeedsSoloForRule(meta, wire) {
		t.Fatal("前三直选单式单注应 solo")
	}
	meta2 := ParseRuleMeta("syxw_std", "g002", "172", "组选复式", "前二", nil, "172")
	wire2 := FormatBetContentForRule(meta2, SampleGroupContent(meta2))
	if wire2 != "01,02" {
		t.Fatalf("wire=%q want 01,02", wire2)
	}
	if !NeedsSoloForRule(meta2, wire2) {
		t.Fatal("前二组选复式单注应 solo")
	}
}

func TestFormatBetContentForRule_syxwBudingwei(t *testing.T) {
	meta := ParseRuleMeta("syxw_std", "g004", "175", "不定位", "不定位", nil, "175")
	wire := FormatBetContentForRule(meta, SampleGroupContent(meta))
	if wire != "01" {
		t.Fatalf("wire=%q want 01", wire)
	}
	if NeedsSoloForRule(meta, wire) {
		t.Fatal("11选5 不定位不应 solo")
	}
}

func TestSampleGroupContent_wuxingZu120(t *testing.T) {
	seg, _ := json.Marshal(map[string]string{"guajiGroup": "五星"})
	meta := ParseRuleMeta("ssc_std", "g015", "156", "组选120", "五星", seg, "156")
	content := SampleGroupContent(meta)
	if content != "0,1,2,3,4" {
		t.Fatalf("sample=%q want 0,1,2,3,4", content)
	}
}

func TestFormatBetContentForRule_k3ErtongDx(t *testing.T) {
	meta := ParseRuleMeta("k3_std", "g002", "225", "二同号单选", "同号", nil, "225")
	wire := FormatBetContentForRule(meta, SampleGroupContent(meta))
	if wire != "1,2" {
		t.Fatalf("wire=%q want 1,2", wire)
	}
	if n := CountBetNums(meta, wire); n != 1 {
		t.Fatalf("betsNums=%d want 1", n)
	}
	if !NeedsSoloForRule(meta, wire) {
		t.Fatal("二同号单选单注应 solo")
	}
}

func TestFormatBetContentForRule_k3Shoudong(t *testing.T) {
	meta := ParseRuleMeta("k3_std", "g005", "230", "手动输入", "标准选号", nil, "230")
	content := SampleGroupContent(meta)
	if content != "112" {
		t.Fatalf("sample=%q want 112", content)
	}
	wire := FormatBetContentForRule(meta, content)
	if wire != "112" {
		t.Fatalf("wire=%q want 112", wire)
	}
	if NeedsSoloForRule(meta, wire) {
		t.Fatal("K3 手动输入不应 solo")
	}
}

func TestFormatBetContentForRule_k3Santong(t *testing.T) {
	meta := ParseRuleMeta("k3_std", "g004", "228", "三同号", "同号", nil, "228")
	wire := FormatBetContentForRule(meta, SampleGroupContent(meta))
	if wire != "1" {
		t.Fatalf("wire=%q want 1", wire)
	}
	if n := CountBetNums(meta, wire); n != 1 {
		t.Fatalf("betsNums=%d want 1", n)
	}
	if !NeedsSoloForRule(meta, wire) {
		t.Fatal("三同号单选应 solo")
	}
	wireMulti := FormatBetContentForRule(meta, "1,2,3")
	if wireMulti != "1,2,3" {
		t.Fatalf("multi wire=%q want 1,2,3", wireMulti)
	}
	if n := CountBetNums(meta, wireMulti); n != 3 {
		t.Fatalf("multi betsNums=%d want 3", n)
	}
	if NeedsSoloForRule(meta, wireMulti) {
		t.Fatal("三同号复选不应 solo")
	}
	if got := MatrixSkipReason(meta); got != "" {
		t.Fatalf("228 不应 skip: %q", got)
	}
}

func TestMatrixSkipReason_k3Santong(t *testing.T) {
	meta230 := ParseRuleMeta("k3_std", "g005", "230", "手动输入", "标准选号", nil, "230")
	if got := MatrixSkipReason(meta230); got != "" {
		t.Fatalf("230 不应 skip: %q", got)
	}
}

func TestFormatBetContentForRule_pk10Hezhi(t *testing.T) {
	meta := ParseRuleMeta("pk10_std", "g010", "217", "冠亚和值", "冠亚", nil, "217")
	got := FormatBetContentForRule(meta, "3")
	if got != "03" {
		t.Fatalf("wire=%q want 03", got)
	}
	if NeedsSoloForRule(meta, got) {
		t.Fatal("PK10 和值不应 solo")
	}
}

func TestFormatBetContentForRule_pk10RankPlays(t *testing.T) {
	meta192 := ParseRuleMeta("pk10_std", "g003", "192", "前一直选复式", "前一", nil, "192")
	wire192 := FormatBetContentForRule(meta192, SampleGroupContent(meta192))
	if wire192 != "01" {
		t.Fatalf("192 wire=%q want 01", wire192)
	}
	if NeedsSoloForRule(meta192, wire192) {
		t.Fatal("前一复式单注不应 solo")
	}

	meta194 := ParseRuleMeta("pk10_std", "g004", "194", "前二直选单式", "前二", nil, "194")
	wire194 := FormatBetContentForRule(meta194, SampleGroupContent(meta194))
	if wire194 != "0102" {
		t.Fatalf("194 wire=%q want 0102", wire194)
	}
	if !NeedsSoloForRule(meta194, wire194) {
		t.Fatal("前二单式单注应 solo")
	}

	meta207 := ParseRuleMeta("pk10_std", "g008", "207", "冠军", "冠军", nil, "207")
	wire207 := FormatBetContentForRule(meta207, SampleGroupContent(meta207))
	if wire207 != "大" {
		t.Fatalf("207 wire=%q want 大", wire207)
	}
	if NeedsSoloForRule(meta207, wire207) {
		t.Fatal("PK10 冠军大小不应 solo")
	}

	meta212 := ParseRuleMeta("pk10_std", "g009", "212", "冠军", "冠军", nil, "212")
	wire212 := FormatBetContentForRule(meta212, SampleGroupContent(meta212))
	if wire212 != "单" {
		t.Fatalf("212 wire=%q want 单", wire212)
	}
}

func TestFormatBetContentForRule_lhcFushi(t *testing.T) {
	meta277 := ParseRuleMeta("lhc_std", "g003", "277", "复式", "二全中", nil, "277")
	content := SampleGroupContent(meta277)
	if content != "01,02" {
		t.Fatalf("sample=%q want 01,02", content)
	}
	wire := FormatBetContentForRule(meta277, content)
	if wire != "01,02" {
		t.Fatalf("wire=%q want 01,02", wire)
	}
	if n := CountBetNums(meta277, wire); n != 1 {
		t.Fatalf("betsNums=%d want 1", n)
	}

	meta295 := ParseRuleMeta("lhc_std", "g003", "295", "复式", "三全中", nil, "295")
	content295 := SampleGroupContent(meta295)
	if n := CountBetNums(meta295, FormatBetContentForRule(meta295, content295)); n != 1 {
		t.Fatalf("295 betsNums=%d want 1", n)
	}

	meta376 := ParseRuleMeta("lhc_std", "g015", "376", "复式", "三中二", nil, "376")
	content376 := SampleGroupContent(meta376)
	if content376 != "01,02,03" {
		t.Fatalf("sample=%q want 01,02,03", content376)
	}
	wire376 := FormatBetContentForRule(meta376, content376)
	if n := CountBetNums(meta376, wire376); n != 3 {
		t.Fatalf("betsNums=%d want 3", n)
	}

	meta346 := ParseRuleMeta("lhc_std", "g013", "346", "复式", "不中/选一", nil, "346")
	content346 := SampleGroupContent(meta346)
	if content346 != "01,02,03,04,05" {
		t.Fatalf("346 sample=%q", content346)
	}
	if n := CountBetNums(meta346, FormatBetContentForRule(meta346, content346)); n != 1 {
		t.Fatalf("346 betsNums=%d want 1", n)
	}

	meta299 := ParseRuleMeta("lhc_std", "g004", "299", "过关", "过关", nil, "299")
	if got := SampleGroupContent(meta299); got != "大,小" {
		t.Fatalf("299 sample=%q want 大,小", got)
	}
}

func TestMatrixSkipReason_lhcTemaWire(t *testing.T) {
	meta := ParseRuleMeta("lhc_std", "g001", "385", "特码A", "特码", nil, "385")
	if got := MatrixSkipReason(meta); got != "" {
		t.Fatalf("385 should not skip: %q", got)
	}
	wire := FormatBetContentForRule(meta, "07")
	if wire != "07||" {
		t.Fatalf("wire=%q want 07||", wire)
	}
	if n := CountBetNums(meta, wire); n != 1 {
		t.Fatalf("bets=%d want 1", n)
	}
	zx := ParseRuleMeta("lhc_std", "g005", "301", "总肖", "生肖", nil, "301")
	if FormatBetContentForRule(zx, "2") != "二肖" {
		t.Fatalf("zongxiao wire")
	}
	qm := ParseRuleMeta("lhc_std", "qima", "qima", "七码", "七码", nil, "313")
	if SampleGroupContent(qm) != "双1" {
		t.Fatalf("qima sample")
	}
	if FormatBetContentForRule(qm, "双1") != "双1" {
		t.Fatalf("qima wire")
	}
}

func TestFormatBetContentForRule_syxwZuxuanFs(t *testing.T) {
	meta := ParseRuleMeta("syxw_std", "g001", "168", "组选复式", "前三", nil, "168")
	content := SampleGroupContent(meta)
	if content != "1,2,3" {
		t.Fatalf("sample=%q want 1,2,3", content)
	}
	got := FormatBetContentForRule(meta, content)
	if got != "01,02,03" {
		t.Fatalf("wire=%q want 01,02,03", got)
	}
	if !NeedsSoloForRule(meta, got) {
		t.Fatal("SYXW 前三组选复式单注应 solo")
	}
}

func TestFormatBetContentForRule_pk10DxdsCombo(t *testing.T) {
	meta221 := ParseRuleMeta("pk10_std", "g010", "221", "冠亚大小单双", "冠亚", nil, "221")
	got221 := FormatBetContentForRule(meta221, "大")
	if got221 != "和大" {
		t.Fatalf("221 wire=%q want 和大", got221)
	}
	if n := CountBetNums(meta221, got221); n != 1 {
		t.Fatalf("221 betsNums=%d want 1", n)
	}
	if NeedsSoloForRule(meta221, got221) {
		t.Fatal("221 不应 solo")
	}
	meta222 := ParseRuleMeta("pk10_std", "g010", "222", "前三大小单双", "前三", nil, "222")
	if got := FormatBetContentForRule(meta222, "小"); got != "和小" {
		t.Fatalf("222 wire=%q want 和小", got)
	}
	meta223 := ParseRuleMeta("pk10_std", "g010", "223", "后三大小单双", "后三", nil, "223")
	if got := FormatBetContentForRule(meta223, "双"); got != "和双" {
		t.Fatalf("223 wire=%q want 和双", got)
	}
	if MatrixSkipReason(meta221) != "" {
		t.Fatalf("221 不应 skip: %q", MatrixSkipReason(meta221))
	}
}
