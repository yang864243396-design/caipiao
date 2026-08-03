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
	baozi := FormatBetContentForRule(meta, "7\n7\n7")
	if baozi != "7,7,7" {
		t.Fatalf("baozi wire=%q want 7,7,7", baozi)
	}
	if n := CountBetNums(meta, baozi); n != 0 {
		t.Fatalf("豹子 betsNums=%d want 0（对齐第三方网页）", n)
	}
	if n := ResolveBetsNums(meta, baozi, 2, 2, 1); n != 0 {
		t.Fatalf("ResolveBetsNums 豹子=%d want 0（禁止回退成 1）", n)
	}
	if !IsFushiBaoziZeroBet(meta, baozi) {
		t.Fatal("IsFushiBaoziZeroBet want true")
	}
	got := FormatBetContentForRule(meta, "1\n2\n3")
	if got != "1,2,3" {
		t.Fatalf("wire=%q want 1,2,3", got)
	}
	if !NeedsSoloForRule(meta, got) {
		t.Fatal("直选复式单注应 solo")
	}
	if n := CountBetNums(meta, got); n != 1 {
		t.Fatalf("betsNums=%d want 1", n)
	}
}

func TestFormatBetContentForRule_qian3DanshiKeepsComma(t *testing.T) {
	seg, _ := json.Marshal(map[string]string{
		"guajiGroup": "前三码", "guajiRuleId": "2",
	})
	meta := ParseRuleMeta("ssc_std", "g001", "2", "前三直选单式", "前三码", seg, "2")
	got := FormatBetContentForRule(meta, "012,345")
	if got != "012,345" {
		t.Fatalf("wire=%q want 012,345 (must keep comma)", got)
	}
	if n := CountBetNums(meta, got); n != 2 {
		t.Fatalf("betsNums=%d want 2", n)
	}
	chunked := FormatBetContentForRule(meta, "012345")
	if chunked != "012,345" {
		t.Fatalf("chunked wire=%q want 012,345", chunked)
	}
}

func TestFormatBetContentForRule_qian3Zuhe(t *testing.T) {
	seg, _ := json.Marshal(map[string]string{
		"guajiGroup": "前三码", "guajiRuleId": "5",
	})
	meta := ParseRuleMeta("ssc_std", "g001", "5", "前三组合", "前三码", seg, "5")
	got := FormatBetContentForRule(meta, "0,1,3\n0\n0")
	if got != "013,0,0" {
		t.Fatalf("wire=%q want 013,0,0", got)
	}
	if n := CountBetNums(meta, got); n != 9 {
		t.Fatalf("betsNums=%d want 9 (3×1×1×3)", n)
	}
	two := FormatBetContentForRule(meta, "0,1\n2\n3")
	if two != "01,2,3" {
		t.Fatalf("wire=%q want 01,2,3", two)
	}
	if n := CountBetNums(meta, two); n != 6 {
		t.Fatalf("betsNums=%d want 6 (2×1×1×3)", n)
	}
	flat := FormatBetContentForRule(meta, "1,2,3")
	if flat != "1,2,3" {
		t.Fatalf("flat wire=%q want 1,2,3", flat)
	}
	if n := CountBetNums(meta, flat); n != 3 {
		t.Fatalf("flat betsNums=%d want 3", n)
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

func TestFormatBetContentForRule_dingweiKeepsEmptyLeadingSlots(t *testing.T) {
	meta := dingweiMeta()
	// 百位选 1、2：方案内容 ",,12,," / 换行位格式均不得压成 "12,,,,"
	for _, in := range []string{",,12,,", "\n\n1,2\n\n"} {
		got := FormatBetContentForRule(meta, in)
		if got != ",,12,," {
			t.Fatalf("in=%q wire=%q want ,,12,,", in, got)
		}
		if n := CountBetNums(meta, got); n != 2 {
			t.Fatalf("in=%q betsNums=%d want 2", in, n)
		}
	}
}

func TestInferBetMode_qian3NotPollutedByDingweiTypeLabel(t *testing.T) {
	meta := qian3FushiMeta()
	meta.TypeLabel = "定位胆" // resolvePlayTypeLabel 旧默认值
	if got := InferBetMode(meta); got != "fushi" {
		t.Fatalf("mode=%q want fushi (TypeLabel 定位胆不得覆盖前三直选复式)", got)
	}
	wire := FormatBetContentForRule(meta, "0,1,3\n0\n0")
	if wire != "013,0,0" {
		t.Fatalf("wire=%q want 013,0,0", wire)
	}
}

func TestNeedsSoloForRule_qian3FushiPaddedWireNotDingwei(t *testing.T) {
	meta := qian3FushiMeta()
	padded := "013,0,0,,"
	if !IsSSCDingweiBetContent(padded) {
		t.Fatal("precondition: padded wire looks like dingwei")
	}
	if !NeedsSoloForRule(meta, padded) {
		t.Fatal("前三直选复式误垫五位时仍应 solo")
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
	// base=6>1：须 solo=false（实测 solo=true→单挑参数错误）
	if NeedsSoloForRule(meta, "6") {
		t.Fatal("前三组选和值 sum=6 须 solo=false")
	}
	if !NeedsSoloForRule(meta, "1") {
		t.Fatal("前三组选和值 sum=1 须 solo=true")
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

// 复现 def-1-1785490406841 / 1785488745547 等中三组选和值：
// 总注≤3 须 solo=true；≥4 须 solo=false（2026-07-31 实测 rule262/108）。
// Label 空时仍须按组选计注（1 注）。
func TestResolveSolo_zhong3ZuxuanHezhiSoloByBets(t *testing.T) {
	seg, _ := json.Marshal(map[string]string{
		"guajiGroup":    "中三码",
		"guajiTeam":     "中三组选",
		"guajiFullName": "中三组选和值",
		"guajiRuleId":   "262",
	})
	meta := ParseRuleMeta("ssc_std", "g002", "262", "", "中三码", seg, "262")
	if mode := InferBetMode(meta); mode != "hezhi" {
		t.Fatalf("mode=%q want hezhi", mode)
	}
	if n := CountBetNums(meta, "1"); n != 1 {
		t.Fatalf("组选和值 sum=1 bets=%d want 1（空 Label 时勿按直选计成 3）", n)
	}
	if !NeedsSoloForRule(meta, "1") || !ResolveSolo(meta, "1", 1) {
		t.Fatal("中三组选和值 bets=1 须 solo=true")
	}
	metaOK := ParseRuleMeta("ssc_std", "g002", "262", "中三组选和值", "中三码", seg, "262")
	if !ResolveSolo(metaOK, "1", 1) {
		t.Fatal("中三组选和值 bets=1 须 solo=true（solo=false→单挑参数错误）")
	}
	// content=25 → 2 注；solo=false 拒单、solo=true 接单（随机出号曾踩坑）
	if n := CountBetNums(metaOK, "25"); n != 2 {
		t.Fatalf("sum=25 bets=%d want 2", n)
	}
	if !NeedsSoloForRule(metaOK, "25") || !ResolveSolo(metaOK, "25", 2) {
		t.Fatal("中三组选和值 bets=2 须 solo=true（solo=false→单挑参数错误）")
	}
	for _, sum := range []string{"2", "3", "24"} {
		if n := CountBetNums(metaOK, sum); n != 2 {
			t.Fatalf("sum=%s bets=%d want 2", sum, n)
		}
		if !ResolveSolo(metaOK, sum, 2) {
			t.Fatalf("中三组选和值 sum=%s bets=2 须 solo=true", sum)
		}
	}
	// content=4 → 4 注；solo=true 拒单、solo=false 接单
	if n := CountBetNums(metaOK, "4"); n != 4 {
		t.Fatalf("sum=4 bets=%d want 4", n)
	}
	if NeedsSoloForRule(metaOK, "4") || ResolveSolo(metaOK, "4", 4) {
		t.Fatal("中三组选和值 bets=4 须 solo=false")
	}
	// content=12 → 14 注；solo=true 拒单、solo=false 接单
	if n := CountBetNums(metaOK, "12"); n != 14 {
		t.Fatalf("sum=12 bets=%d want 14", n)
	}
	if NeedsSoloForRule(metaOK, "12") || ResolveSolo(metaOK, "12", 14) {
		t.Fatal("中三组选和值 bets≥4 须 solo=false（solo=true→单挑参数错误）")
	}
	if NeedsSoloForRule(metaOK, "21,22") || ResolveSolo(metaOK, "21,22", 11) {
		t.Fatal("中三组选和值多和值须 solo=false")
	}
	// 前中后三：按总注（含×3）判；content=1 总注3 solo=true；content=25 总注6 solo=false
	seg3, _ := json.Marshal(map[string]string{
		"guajiGroup": "前中后三", "guajiTeam": "前中后三组选",
		"guajiFullName": "前中后三组选和值", "guajiRuleId": "108",
	})
	meta3 := ParseRuleMeta("ssc_std", "g007", "108", "前中后三组选和值", "前中后三", seg3, "108")
	if n := CountBetNums(meta3, "1"); n != 3 {
		t.Fatalf("前中后三 sum=1 bets=%d want 3", n)
	}
	if !NeedsSoloForRule(meta3, "1") || !ResolveSolo(meta3, "1", 3) {
		t.Fatal("前中后三组选和值 总注3 须 solo=true")
	}
	if n := CountBetNums(meta3, "25"); n != 6 {
		t.Fatalf("前中后三 sum=25 bets=%d want 6", n)
	}
	if NeedsSoloForRule(meta3, "25") || ResolveSolo(meta3, "25", 6) {
		t.Fatal("前中后三组选和值 总注6 须 solo=false（勿按倍乘前 base=2 判 solo）")
	}
	if n := CountBetNums(meta3, "6"); n != 18 {
		t.Fatalf("前中后三 sum=6 bets=%d want 18", n)
	}
	if NeedsSoloForRule(meta3, "6") || ResolveSolo(meta3, "6", 18) {
		t.Fatal("前中后三组选和值 总注18 须 solo=false")
	}
}

func TestFormatBetContentForRule_hezhiStripLeadingZeros(t *testing.T) {
	seg, _ := json.Marshal(map[string]string{"guajiGroup": "前中后三"})
	meta := ParseRuleMeta("ssc_std", "g007", "108", "前中后三组选和值", "前中后三", seg, "108")
	got := FormatBetContentForRule(meta, "01,02,03,04,05,06,07,08,09,10")
	want := "1,2,3,4,5,6,7,8,9,10"
	if got != want {
		t.Fatalf("wire=%q want %q（第三方拒补零和值）", got, want)
	}
	if n := CountBetNums(meta, got); n <= 0 {
		t.Fatalf("betsNums=%d", n)
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

// 前二组选复式：按位误产号会带重号（3,5\\n5）；粘连 12 也不能整段上送。
func TestFormatBetContentForRule_qian2ZuxuanFsDedupeAndExplode(t *testing.T) {
	seg, err := json.Marshal(map[string]string{
		"guajiGroup": "前二码", "guajiTeam": "前二组选", "guajiFullName": "前二组选复式", "guajiRuleId": "42",
	})
	if err != nil {
		t.Fatal(err)
	}
	meta := ParseRuleMeta("ssc_std", "g004", "42", "前二组选复式", "前二码", seg, "42")
	if got := FormatBetContentForRule(meta, "3,5\n5"); got != "3,5" {
		t.Fatalf("dup wire=%q want 3,5", got)
	}
	if got := FormatBetContentForRule(meta, "12"); got != "1,2" {
		t.Fatalf("glued wire=%q want 1,2", got)
	}
	if n := CountBetNums(meta, "12"); n != 1 {
		t.Fatalf("CountBetNums(12)=%d want 1", n)
	}
	if n := CountBetNums(meta, "5,5"); n != 0 {
		// 去重后只剩 1 码，C(1,2)=0
		t.Fatalf("CountBetNums(5,5)=%d want 0", n)
	}
}

func TestFormatBetContentForRule_zuxuanDs(t *testing.T) {
	meta := ParseRuleMeta("ssc_std", "g004", "43", "组选单式", "前二", nil, "43")
	got := FormatBetContentForRule(meta, "12")
	if got != "12" {
		t.Fatalf("wire=%q want 12", got)
	}
}

func TestCountBetNums_qian2ZuxuanDanshiExcludeDuizi(t *testing.T) {
	meta := ParseRuleMeta("ssc_std", "g004", "43", "前二组选单式", "前二码", nil, "43")
	raw := "11,12,13,14,15,16,17,22,24,25"
	// 第三方：排除对子 11/22 → 8 注；wire 也应去掉对子
	wantWire := "12,13,14,15,16,17,24,25"
	if got := FormatBetContentForRule(meta, raw); got != wantWire {
		t.Fatalf("wire=%q want %q", got, wantWire)
	}
	if n := CountBetNums(meta, raw); n != 8 {
		t.Fatalf("CountBetNums raw=%d want 8", n)
	}
	if n := CountBetNums(meta, wantWire); n != 8 {
		t.Fatalf("CountBetNums wire=%d want 8", n)
	}
	// 组选形态去重：12 与 21 计 1；对子不计
	if n := CountBetNums(meta, "12,21,11"); n != 1 {
		t.Fatalf("CountBetNums form-dedup=%d want 1", n)
	}
	// 冷热/误按位单码号池：展成整注，勿计 0
	for _, pool := range []string{"5,6", "5\n6", "1,2,3"} {
		n := CountBetNums(meta, pool)
		wire := FormatBetContentForRule(meta, pool)
		if n <= 0 || wire == "" {
			t.Fatalf("digit-pool %q count=%d wire=%q", pool, n, wire)
		}
	}
	if got := FormatBetContentForRule(meta, "5,6"); got != "56" {
		t.Fatalf("pool 5,6 wire=%q want 56", got)
	}
	if n := CountBetNums(meta, "1,2,3"); n != 3 {
		t.Fatalf("pool 1,2,3 count=%d want 3 (12,13,23)", n)
	}
}

func TestCountBetNums_baodanQian2(t *testing.T) {
	meta := ParseRuleMeta("ssc_std", "g004", "45", "组选包胆", "前二", nil, "45")
	if n := CountBetNums(meta, "3"); n != 9 {
		t.Fatalf("qian2 baodan betsNums=%d want 9", n)
	}
	// 多胆须压成单胆再计注（冷热多档曾出 1,3,5,6,7 → 第三方「投注数字不合规」）
	if got := FormatBetContentForRule(meta, "1,3,5,6,7"); got != "1" {
		t.Fatalf("baodan wire=%q want 1", got)
	}
	if n := CountBetNums(meta, FormatBetContentForRule(meta, "1,3,5,6,7")); n != 9 {
		t.Fatalf("baodan after format betsNums=%d want 9", n)
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
	if n := CountBetNums(meta, FormatBetContentForRule(meta, "7\n7\n7")); n != 0 {
		t.Fatalf("豹子 betsNums=%d want 0", n)
	}
	wire := FormatBetContentForRule(meta, "1\n2\n3")
	if wire != "1,2,3" {
		t.Fatalf("wire=%q want 1,2,3", wire)
	}
	if n := CountBetNums(meta, wire); n != 3 {
		t.Fatalf("betsNums=%d want 3", n)
	}
	if !NeedsSoloForRule(meta, wire) {
		t.Fatal("前中后三直选复式最小注应 solo（实测 solo=false→单挑参数错误）")
	}
}

func TestCountBetNums_qianhou3Fushi(t *testing.T) {
	seg, _ := json.Marshal(map[string]string{"guajiGroup": "前后三"})
	meta := ParseRuleMeta("ssc_std", "g012", "89", "直选复式", "前后三", seg, "89")
	wire := FormatBetContentForRule(meta, "1\n2\n3")
	if n := CountBetNums(meta, wire); n != 2 {
		t.Fatalf("betsNums=%d want 2", n)
	}
	if !NeedsSoloForRule(meta, wire) {
		t.Fatal("前后三直选复式最小注应 solo（实测 solo=false→单挑参数错误）")
	}
}

func TestCountBetNums_qianhou2Fushi(t *testing.T) {
	seg, _ := json.Marshal(map[string]string{"guajiGroup": "前后二"})
	meta := ParseRuleMeta("ssc_std", "g008", "119", "直选复式", "前后二", seg, "119")
	if n := CountBetNums(meta, FormatBetContentForRule(meta, "1\n1")); n != 0 {
		t.Fatalf("对子 betsNums=%d want 0", n)
	}
	wire := FormatBetContentForRule(meta, "1\n2")
	if n := CountBetNums(meta, wire); n != 2 {
		t.Fatalf("betsNums=%d want 2", n)
	}
	if NeedsSoloForRule(meta, wire) {
		t.Fatal("前后二应 solo=false（实测 solo=true→单挑参数错误）")
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
		t.Fatal("任二直选复式多注不应 solo")
	}
	// 单注（两位各 1 码 → C 位组合后 1 注）须 solo=true
	if !NeedsSoloForRule(meta, "1,,,,2") && !NeedsSoloForRule(meta, "0,1") {
		// wire 形态因 Format 而异；以 CountBetNums==1 的样本为准
		wire := FormatBetContentForRule(meta, "1\n\n\n\n2")
		if CountBetNums(meta, wire) == 1 && !NeedsSoloForRule(meta, wire) {
			t.Fatalf("任二直选复式单注应 solo, wire=%q", wire)
		}
	}
}

func TestNeedsSolo_budingweiTwoCode(t *testing.T) {
	// 实测一码/二码不定位 solo=true →「单挑参数错误」，一律 solo=false
	meta := ParseRuleMeta("ssc_std", "g009", "114", "前三二码不定位", "不定位", nil, "114")
	if NeedsSoloForRule(meta, "1,2") {
		t.Fatal("三星二码不定位不应 solo")
	}
	meta1 := ParseRuleMeta("ssc_std", "g009", "113", "前三一码不定位", "不定位", nil, "113")
	if NeedsSoloForRule(meta1, "3") {
		t.Fatal("前三一码不定位不应 solo")
	}
	metaHou3 := ParseRuleMeta("ssc_std", "g009", "117", "后三一码不定位", "不定位", nil, "117")
	if NeedsSoloForRule(metaHou3, "3") {
		t.Fatal("后三一码不定位不应 solo")
	}
	if ResolveSolo(meta, "1,2", 1) {
		t.Fatal("二码不定位 ResolveSolo 应为 false")
	}
	metaQian4 := ParseRuleMeta("ssc_std", "g009", "147", "前四二码不定位", "不定位", nil, "147")
	if NeedsSoloForRule(metaQian4, "1,2") {
		t.Fatal("前四二码不定位不应 solo")
	}
	metaWuxing3 := ParseRuleMeta("ssc_std", "g009", "152", "五星三码不定位", "不定位", nil, "152")
	if NeedsSoloForRule(metaWuxing3, "1,2,3,4") {
		t.Fatal("五星三码不定位不应 solo")
	}
}

func TestResolveSolo_qianzhonghou3AndQianhou2(t *testing.T) {
	seg3, _ := json.Marshal(map[string]string{"guajiGroup": "前中后三"})
	meta3 := ParseRuleMeta("ssc_std", "g007", "102", "直选单式", "前中后三", seg3, "102")
	if !ResolveSolo(meta3, "012,345", 6) {
		t.Fatal("前中后三直选单式多注应 solo=true")
	}
	seg2, _ := json.Marshal(map[string]string{"guajiGroup": "前后二"})
	meta2 := ParseRuleMeta("ssc_std", "g008", "119", "直选复式", "前后二", seg2, "119")
	if ResolveSolo(meta2, "013,0", 6) {
		t.Fatal("前后二直选复式应 solo=false")
	}
}

func TestCountBudingwei_yimaMaxTwo(t *testing.T) {
	meta := ParseRuleMeta("ssc_std", "g009", "113", "前三一码不定位", "不定位", nil, "113")
	if n := CountBetNums(meta, "0,2,4"); n != 2 {
		t.Fatalf("一码三选应截断计 2 注, got %d", n)
	}
	if got := FormatBetContentForRule(meta, "0,2,4"); got != "0,2" {
		t.Fatalf("wire=%q want 0,2", got)
	}
	// 粘连三码会被第三方拒「投注数字不可超过两位数」；须拆开再截断
	if got := FormatBetContentForRule(meta, "123"); got != "1,2" {
		t.Fatalf("glued wire=%q want 1,2", got)
	}
	if n := CountBetNums(meta, FormatBetContentForRule(meta, "123")); n != 2 {
		t.Fatalf("glued bets=%d want 2", n)
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

// 前中后三直选和值 0–9：单区三星组合 220 × 3 区 = 660（对齐第三方；非四星 715×3=2145）。
func TestCountBetNums_qianZhongHou3ZhixuanHezhi0to9(t *testing.T) {
	seg, _ := json.Marshal(map[string]string{
		"guajiGroup":    "前中后三",
		"guajiTeam":     "前中后三直选",
		"guajiFullName": "前中后三直选和值",
		"guajiRuleId":   "103",
	})
	meta := ParseRuleMeta("ssc_std", "g007", "103", "直选和值", "前中后三", seg, "103")
	if n := CountBetNums(meta, "0,1,2,3,4,5,6,7,8,9"); n != 660 {
		t.Fatalf("betsNums=%d want 660", n)
	}
}

// 前中后三和值尾数 1–9：单区 9 × 3 区 = 27（对齐第三方；勿只计选项个数 9）。
func TestCountBetNums_qianZhongHou3Weishu1to9(t *testing.T) {
	seg, _ := json.Marshal(map[string]string{
		"guajiGroup":    "前中后三",
		"guajiTeam":     "前中后三其他",
		"guajiFullName": "前中后三和值尾数",
		"guajiRuleId":   "111",
	})
	meta := ParseRuleMeta("ssc_std", "g007", "111", "和值尾数", "前中后三", seg, "111")
	if n := CountBetNums(meta, "1,2,3,4,5,6,7,8,9"); n != 27 {
		t.Fatalf("betsNums=%d want 27", n)
	}
}

// def-1-1785643738664：前中后三混合组选按总注数判 solo（含×3）。
// ju1=123→3 注 solo=true；ju2=4 形→12 注须 solo=false（旧逻辑无条件 true → 单挑参数错误）。
func TestResolveSolo_qianZhongHou3HunheByBets(t *testing.T) {
	seg, _ := json.Marshal(map[string]string{
		"guajiGroup":    "前中后三",
		"guajiTeam":     "前中后三组选",
		"guajiFullName": "前中后三混合组选",
		"guajiRuleId":   "110",
	})
	meta := ParseRuleMeta("ssc_std", "g007", "110", "混合组选", "前中后三", seg, "110")
	cases := []struct {
		wire string
		want int
		solo bool
	}{
		{"123", 3, true},
		{"123,432", 6, false},
		{"123,432,654,786", 12, false},
	}
	for _, c := range cases {
		n := CountBetNums(meta, c.wire)
		if n != c.want {
			t.Fatalf("wire=%q bets=%d want %d", c.wire, n, c.want)
		}
		if got := ResolveSolo(meta, c.wire, n); got != c.solo {
			t.Fatalf("wire=%q bets=%d ResolveSolo=%v want %v", c.wire, n, got, c.solo)
		}
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
	// 两位短输入须落到默认千/个，不能 flat「0,1」（第三方报格式错误）
	got2 := FormatBetContentForRule(meta, "0\n1")
	if got2 != ",0,,,1" {
		t.Fatalf("short wire=%q want ,0,,,1", got2)
	}
	if n := CountBetNums(meta, got2); n != 1 {
		t.Fatalf("short betsNums=%d want 1", n)
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
	// NeedsSolo 对单注样例仍为 true；多注须 ResolveSolo=false（否则第三方报单挑参数错误）
	if !NeedsSoloForRule(meta, got) {
		t.Fatal("前二直选和值 NeedsSolo 单注语义应 true")
	}
	if ResolveSolo(meta, got, 5) {
		t.Fatal("前二直选和值 5 注不可 solo")
	}
	if !ResolveSolo(meta, "0", 1) {
		t.Fatal("前二直选和值 1 注应 solo")
	}
}

func TestResolveSolo_sscQian2ZuxuanFs(t *testing.T) {
	meta := ParseRuleMeta("ssc_std", "g004", "42", "组选复式", "前二码", nil, "42")
	wire := FormatBetContentForRule(meta, "1\n2")
	if wire != "1,2" {
		t.Fatalf("wire=%q want 1,2", wire)
	}
	if n := CountBetNums(meta, wire); n != 1 {
		t.Fatalf("bets=%d want 1", n)
	}
	if ResolveSolo(meta, wire, 1) {
		t.Fatal("SSC 前二组选复式应 solo=false（实测 solo=true→单挑参数错误）")
	}
	metaHou := ParseRuleMeta("ssc_std", "g005", "52", "组选复式", "后二", nil, "52")
	if ResolveSolo(metaHou, "0,1", 1) {
		t.Fatal("SSC 后二组选复式应 solo=false")
	}
}

// def-1-1785564083367：按位残留去重后只剩 1 码时须 0 注，禁止 ResolveBetsNums 回落成 1。
func TestResolveBetsNums_qian2ZuxuanFsInsufficientPool(t *testing.T) {
	meta := ParseRuleMeta("ssc_std", "g004", "42", "组选复式", "前二", nil, "42")
	for _, raw := range []string{"5", "5,5", "5\n5"} {
		wire := FormatBetContentForRule(meta, raw)
		if n := CountBetNums(meta, wire); n != 0 {
			t.Fatalf("content=%q wire=%q Count=%d want 0", raw, wire, n)
		}
		if n := ResolveBetsNums(meta, wire, 2, 2, 1); n != 0 {
			t.Fatalf("content=%q wire=%q ResolveBetsNums=%d want 0（勿回落 1→投注数字不合规）", raw, wire, n)
		}
	}
	if n := ResolveBetsNums(meta, "1,2", 2, 2, 1); n != 1 {
		t.Fatalf("合法 2 码 ResolveBetsNums=%d want 1", n)
	}
	if n := ResolveBetsNums(meta, "6,8,0", 6, 2, 1); n != 3 {
		t.Fatalf("3 码 ResolveBetsNums=%d want 3", n)
	}
}

// def-1-1785567172297：前二组选和值任意注数 solo=true → 单挑参数错误。
func TestResolveSolo_sscQian2ZuxuanHezhi(t *testing.T) {
	meta := ParseRuleMeta("ssc_std", "g004", "44", "组选和值", "前二", nil, "44")
	if ResolveSolo(meta, "1", 1) {
		t.Fatal("前二组选和值单注须 solo=false（实测 solo=true→单挑参数错误）")
	}
	if ResolveSolo(meta, "6", 3) {
		t.Fatal("前二组选和值 3 注须 solo=false")
	}
	if NeedsSoloForRule(meta, "1") {
		t.Fatal("前二组选和值 NeedsSolo 亦须 false")
	}
	// 仅数字 id、无「组选」文案时仍须识别
	metaID := ParseRuleMeta("ssc_std", "g004", "44", "和值", "前二码", nil, "44")
	if !isZuxuanHezhiMeta(metaID) {
		t.Fatal("rule 44 应按组选和值识别")
	}
	if ResolveSolo(metaID, "1", 1) {
		t.Fatal("rule 44 无组选文案时单注仍须 solo=false")
	}
	metaHou := ParseRuleMeta("ssc_std", "g005", "52", "组选和值", "后二", nil, "52")
	if ResolveSolo(metaHou, "1", 1) {
		t.Fatal("后二组选和值单注须 solo=false")
	}
	// 直选和值不受影响
	metaZX := ParseRuleMeta("ssc_std", "g004", "40", "直选和值", "前二", nil, "40")
	if !ResolveSolo(metaZX, "0", 1) {
		t.Fatal("前二直选和值 1 注仍应 solo=true")
	}
}

func TestGuajiGroupRequiresSoloTrue_qianhou4Fushi(t *testing.T) {
	// 正确 ruleId：复式 134（旧测试误用 130）
	meta := ParseRuleMeta("ssc_std", "g014", "134", "直选复式", "", nil, "134")
	if guajiGroupRequiresSoloFalse(meta) {
		t.Fatal("g014 前后四复式不应走 solo=false 白名单")
	}
	if !guajiGroupRequiresSoloTrue(meta) {
		t.Fatal("g014 前后四复式应 solo=true")
	}
	if !ResolveSolo(meta, "0\n1\n2\n3", 2) {
		t.Fatal("前后四直选复式 ResolveSolo 应为 true（bets=段积×2）")
	}
	metaZuhe := ParseRuleMeta("ssc_std", "g014", "136", "直选组合", "", nil, "136")
	if guajiGroupRequiresSoloTrue(metaZuhe) {
		t.Fatal("前后四直选组合不应强制 solo=true")
	}
}

func TestResolveSolo_wuxingFushiHotColdPool(t *testing.T) {
	// inst-1-1784774566098：冷热五星复式 32 注；实测 solo=false → 单挑参数错误，须 solo=true（绕过 guajiSoloMaxBets）
	seg, _ := json.Marshal(map[string]string{
		"guajiGroup": "五星", "guajiTeam": "五星直选", "guajiFullName": "五星直选复式", "guajiRuleId": "153",
	})
	meta := ParseRuleMeta("ssc_std", "g015", "153", "五星直选复式", "五星", seg, "153")
	raw := "2,5\n2,6\n3,7\n1,6\n4,8"
	wire := FormatBetContentForRule(meta, raw)
	if wire != "25,26,37,16,48" {
		t.Fatalf("wire=%q want 25,26,37,16,48", wire)
	}
	if n := CountBetNums(meta, raw); n != 32 {
		t.Fatalf("raw bets=%d want 32", n)
	}
	if n := CountBetNums(meta, wire); n != 32 {
		t.Fatalf("wire bets=%d want 32", n)
	}
	if !guajiGroupRequiresSoloTrue(meta) {
		t.Fatal("五星直选复式应强制 solo=true")
	}
	if !ResolveSolo(meta, wire, 32) {
		t.Fatal("五星复式 32 注须 solo=true（实测 solo=false→单挑参数错误）")
	}
	if !ResolveSolo(meta, "2,2,3,1,4", 1) {
		t.Fatal("五星复式 1 注应 solo")
	}
	// format 幂等
	if again := FormatBetContentForRule(meta, wire); again != wire {
		t.Fatalf("re-format wire=%q again=%q", wire, again)
	}
	// 组选/组合勿误伤
	metaZu := ParseRuleMeta("ssc_std", "g015", "157", "组选60", "五星", seg, "157")
	if guajiGroupRequiresSoloTrue(metaZu) {
		t.Fatal("五星组选60 不应强制 solo=true")
	}
}

// TestResolveSolo_zhong3FushiOverSoloMax inst-1-1785133877645：
// 中三直选复式 27 注 content=012,234,345；实测 solo=true →「单挑参数错误」，须 solo=false。
func TestResolveSolo_zhong3FushiOverSoloMax(t *testing.T) {
	seg, _ := json.Marshal(map[string]string{
		"guajiGroup": "中三码", "guajiTeam": "中三直选", "guajiFullName": "中三直选复式", "guajiRuleId": "14",
	})
	meta := ParseRuleMeta("ssc_std", "g002", "14", "中三直选复式", "中三码", seg, "14")
	raw := "0,1,2\n2,3,4\n3,4,5"
	wire := FormatBetContentForRule(meta, raw)
	if wire != "012,234,345" {
		t.Fatalf("wire=%q want 012,234,345", wire)
	}
	if n := CountBetNums(meta, wire); n != 27 {
		t.Fatalf("bets=%d want 27", n)
	}
	if ResolveSolo(meta, wire, 27) {
		t.Fatal("中三复式 27 注须 solo=false（实测 solo=true→单挑参数错误）")
	}
	// 18 恰为上限：betsNums > 18 才关 solo；等于 18 仍可 solo
	if !ResolveSolo(meta, "012,012,01", 18) {
		t.Fatal("中三复式 18 注应仍可 solo=true")
	}
	if !ResolveSolo(meta, "0,1,2", 1) {
		t.Fatal("中三复式 1 注应 solo=true")
	}
	if ResolveSolo(meta, "01234,01,01", 20) {
		t.Fatal("中三复式 20 注须 solo=false")
	}
}

func TestResolveSolo_qian2FushiMulti(t *testing.T) {
	meta := ParseRuleMeta("ssc_std", "g004", "38", "前二直选复式", "前二码", nil, "38")
	wire := FormatBetContentForRule(meta, "0,1,3\n0")
	n := CountBetNums(meta, wire)
	if n != 3 {
		t.Fatalf("bets=%d want 3 wire=%q", n, wire)
	}
	if ResolveSolo(meta, wire, n) {
		t.Fatalf("前二复式 %d 注不可 solo wire=%q", n, wire)
	}
	metaDS := ParseRuleMeta("ssc_std", "g004", "39", "前二直选单式", "前二码", nil, "39")
	wireDS := FormatBetContentForRule(metaDS, "12,13,14,15,12")
	if wireDS != "12,13,14,15" {
		t.Fatalf("danshi wire=%q", wireDS)
	}
	if n := CountBetNums(metaDS, wireDS); n != 4 {
		t.Fatalf("danshi bets=%d want 4", n)
	}
	if ResolveSolo(metaDS, wireDS, 4) {
		t.Fatal("前二单式 4 注不可 solo")
	}
	if !ResolveSolo(metaDS, "12", 1) {
		t.Fatal("前二单式 1 注应 solo")
	}
}

func TestFormatBetContentForRule_danshiPositionPool(t *testing.T) {
	// 冷热出号按位号池 → 直选单式笛卡尔积
	meta := ParseRuleMeta("ssc_std", "g001", "2", "前三直选单式", "前三码", nil, "2")
	wire := FormatBetContentForRule(meta, "4,5\n3,5\n2,5")
	want := "432,435,452,455,532,535,552,555"
	if wire != want {
		t.Fatalf("wire=%q want %q", wire, want)
	}
	if n := CountBetNums(meta, wire); n != 8 {
		t.Fatalf("bets=%d want 8", n)
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

func TestSegmentRange_dxdsHou2ByRuleID266(t *testing.T) {
	// 与线上一致：group=大小单双、subId=266，须解析为十/个
	seg, _ := json.Marshal(map[string]string{
		"guajiGroup": "大小单双", "guajiTeam": "后大小单双", "guajiFullName": "后二大小单双",
	})
	meta := ParseRuleMeta("ssc_std", "g016", "266", "后二大小单双", "大小单双", seg, "266")
	start, length := segmentRange(meta)
	if start != 3 || length != 2 {
		t.Fatalf("seg=%d+%d want 3+2", start, length)
	}
	// 仅数字 id、无中文全名时也能定位
	bare := ParseRuleMeta("ssc_std", "g016", "266", "", "大小单双", nil, "266")
	start, length = segmentRange(bare)
	if start != 3 || length != 2 {
		t.Fatalf("bare seg=%d+%d want 3+2", start, length)
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

func TestCountBetNums_zhixuanDanshiDedup(t *testing.T) {
	meta := ParseRuleMeta("ssc_std", "g004", "39", "前二直选单式", "前二码", nil, "39")
	wire := FormatBetContentForRule(meta, "12,13,14,12")
	if wire != "12,13,14" {
		t.Fatalf("wire=%q want 12,13,14", wire)
	}
	if n := CountBetNums(meta, wire); n != 3 {
		t.Fatalf("CountBetNums=%d want 3", n)
	}
	if n := CountBetNums(meta, "12,13,14,12"); n != 3 {
		t.Fatalf("CountBetNums raw=%d want 3", n)
	}
	// 用户复现：12,13,14,15,12 → 4 注（末尾重复 12）
	if n := CountBetNums(meta, "12,13,14,15,12"); n != 4 {
		t.Fatalf("CountBetNums 15dup=%d want 4", n)
	}
	if got := FormatBetContentForRule(meta, "12,13,14,15,12"); got != "12,13,14,15" {
		t.Fatalf("wire 15dup=%q want 12,13,14,15", got)
	}
	// 全重复
	if n := CountBetNums(meta, "12,12,12"); n != 1 {
		t.Fatalf("CountBetNums all-dup=%d want 1", n)
	}
}

func TestCountBetNums_qian3Hunhe(t *testing.T) {
	seg, _ := json.Marshal(map[string]string{
		"guajiGroup":    "前三码",
		"guajiTeam":     "前三组选",
		"guajiFullName": "前三混合组选",
		"guajiRuleId":   "10",
	})
	meta := ParseRuleMeta("ssc_std", "g001", "10", "前三混合组选", "前三码", seg, "10")
	if mode := InferBetMode(meta); mode != "hunhe" {
		t.Fatalf("mode=%q want hunhe", mode)
	}
	// 第三方：排除豹子，组选形态去重 → 123/321 计 1，232 计 1，542 计 1
	content := "123,321,232,222,333,444,542"
	if n := CountBetNums(meta, content); n != 3 {
		t.Fatalf("betsNums=%d want 3 for %q", n, content)
	}
	if n := countSSCHunheBetNums(content, 3); n != 3 {
		t.Fatalf("countSSCHunheBetNums=%d want 3", n)
	}
	if n := countSSCHunheBetNums("222,333", 3); n != 0 {
		t.Fatalf("all baozi should be 0, got %d", n)
	}
	if n := countSSCHunheBetNums("123", 3); n != 1 {
		t.Fatalf("single zu6=%d want 1", n)
	}
	// wire 必须滤掉豹子，否则第三方「投注数字不合规」
	if got := FormatBetContentForRule(meta, "111，，123，"); got != "123" {
		t.Fatalf("hunhe wire filter baozi=%q want 123", got)
	}
	if got := FormatBetContentForRule(meta, "111,123,321,222"); got != "123" {
		t.Fatalf("hunhe wire dedupe=%q want 123", got)
	}
	if n := CountBetNums(meta, FormatBetContentForRule(meta, "111，123")); n != 1 {
		t.Fatalf("filtered hunhe bets=%d want 1", n)
	}
}

func TestCountBetNums_qian3Teshu(t *testing.T) {
	seg, _ := json.Marshal(map[string]string{
		"guajiGroup":    "前三码",
		"guajiTeam":     "前三其他",
		"guajiFullName": "前三特殊号",
		"guajiRuleId":   "12",
	})
	meta := ParseRuleMeta("ssc_std", "g001", "12", "前三特殊号", "前三码", seg, "12")
	if mode := InferBetMode(meta); mode != "teshu" {
		t.Fatalf("mode=%q want teshu", mode)
	}
	content := "豹子,对子,顺子"
	if n := CountBetNums(meta, content); n != 3 {
		t.Fatalf("betsNums=%d want 3 for %q", n, content)
	}
	if n := CountBetNums(meta, "豹子"); n != 1 {
		t.Fatalf("single=%d want 1", n)
	}
}
