package guajibet

import (
	"encoding/json"
	"testing"
)

// 与 lhc_minpick_test.go 的 lhcMeta 不同：这些用例要靠 guajiTeam 区分同组子玩法。
func lhcTeamMeta(t *testing.T, typeID, subID, label, group, team string) RuleMeta {
	t.Helper()
	seg, err := json.Marshal(map[string]string{
		"guajiGroup": group, "guajiTeam": team, "guajiFullName": label, "guajiRuleId": subID,
	})
	if err != nil {
		t.Fatalf("marshal seg: %v", err)
	}
	return ParseRuleMeta("lhc_std", typeID, subID, label, group, seg, subID)
}

// 六合彩玩法名是「组名+子玩法名」，组名本身可能含别的子玩法关键词：
// 「五行家野家野」里「五行」先命中、「一肖尾数一肖」里「尾数」先命中。
// 直接按整串匹配会取到前缀里的词，选出该玩法不接受的号码，第三方回
// 「投注内容不合规」。2026-07-28 真实下单矩阵 tron_lhc_1m 实测拒单。
func TestInferLHCBetModeUsesSubPlayName(t *testing.T) {
	cases := []struct {
		name               string
		typeID, subID      string
		label, group, team string
		wantMode           string
	}{
		// 同组两条只有玩法名尾巴不同，组名「五行家野」同时含两个关键词。
		{"五行家野-五行", "g007", "310", "五行家野五行", "五行家野", "五行家野", "wuxing"},
		{"五行家野-家野", "g007", "311", "五行家野家野", "五行家野", "五行家野", "jiaye"},
		// 组名「一肖尾数」含「尾数」，三条曾全部被判成同一模式。
		{"一肖尾数-一肖", "g010", "316", "一肖尾数一肖", "一肖尾数", "一肖", "xiao"},
		{"一肖尾数-一肖不中", "g010", "317", "一肖尾数一肖不中", "一肖尾数", "一肖不中", "xiao_bz"},
		{"一肖尾数-尾数", "g010", "318", "一肖尾数尾数", "一肖尾数", "尾数", "weishu"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			meta := lhcTeamMeta(t, tc.typeID, tc.subID, tc.label, tc.group, tc.team)
			if got := inferLHCBetMode(meta); got != tc.wantMode {
				t.Fatalf("模式 = %q，期望 %q（子玩法名 %q）",
					got, tc.wantMode, lhcSubPlayName(meta))
			}
		})
	}
}

// 玩法名不以组名开头时必须退回整串匹配，否则「波色红波」这类会丢模式。
func TestInferLHCBetModeFallsBackToFullLabel(t *testing.T) {
	cases := []struct {
		name               string
		typeID, subID      string
		label, group, team string
		wantMode           string
	}{
		{"拖头（玩法名不含组名）", "g003", "300", "三全中拖头", "连码", "三全中", "tuotou"},
		{"复式（组名非前缀）", "g003", "299", "三全中复式", "连码", "三全中", "fushi"},
		// 子玩法名「红波」单独匹配不出模式，必须靠整串里的「波色」。
		{"波色（子玩法名匹配不到）", "g005", "305", "波色红波", "波色", "红波", "bose"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			meta := lhcTeamMeta(t, tc.typeID, tc.subID, tc.label, tc.group, tc.team)
			if got := inferLHCBetMode(meta); got != tc.wantMode {
				t.Fatalf("模式 = %q，期望 %q", got, tc.wantMode)
			}
		})
	}
}

// 三全中复式第三方要求「只能投注3~10个数字」，取样必须给够 3 个。
// 该玩法名曾在库里损坏成「三全中复\uFFFD\uFFFD」，模式推断返回空、
// 取样退化成单个号码而长期拒单（见 migrations/00137）。
func TestLHCFushiSamplesEnoughNumbers(t *testing.T) {
	meta := lhcTeamMeta(t, "g003", "299", "三全中复式", "连码", "三全中")

	if got := lhcMinPickCount(meta, inferLHCBetMode(meta)); got != 3 {
		t.Fatalf("最少选号数 = %d，三全中应为 3", got)
	}

	sample := SampleGroupContent(meta)
	wire := FormatBetContentForRule(meta, sample)
	if n := len(splitCommaParts(wire)); n < 3 || n > 10 {
		t.Fatalf("取样号码数 = %d（wire=%q），第三方只接受 3~10 个", n, wire)
	}
}
