package guajibet

import (
	"encoding/json"
	"testing"
)

// 六合彩全不中 / 多选中一 / 特平中的最少选号数必须等于玩法名里的数字。
//
// 2026-07-28 真实下单矩阵（tron_lhc_1m，115 条）中第三方回传的约束：
//
//	rule 362 全不中12不中 → 最少投注12个号码, 最多只能投注14个号码
//	rule 364 全不中15不中 → 最少投注15个号码, 最多只能投注15个号码
//	rule 376 十选中一     → 最少投注10个号码, 最多只能投注12个号码
//	rule 379 特平中二粒   → 最少投注2个号码,  最多只能投注10个号码
//	rule 381 特平中三粒   → 最少投注3个号码
//	rule 383 特平中四粒   → 最少投注4个号码
//	rule 385 特平中五粒   → 最少投注5个号码
//	rule 299 三全中复式   → 只能投注3~10个数字
//
// 八条显式约束全部与玩法名一致，故以玩法名为准。
// 原实现按 rule_id 硬编码且整表错位一位（348「全不中5不中」→ 6、350「6不中」→ 7 …），
// 除了被拒单，还会让 CountBetNums 的注数与下注金额偏大。

func lhcMeta(t *testing.T, typeID, subID, label, group string) RuleMeta {
	t.Helper()
	seg, err := json.Marshal(map[string]string{
		"guajiGroup": group, "guajiFullName": label, "guajiRuleId": subID,
	})
	if err != nil {
		t.Fatalf("marshal seg: %v", err)
	}
	return ParseRuleMeta("lhc_std", typeID, subID, label, group, seg, subID)
}

func TestLhcBuzhongMinPick_matchesPlayLabel(t *testing.T) {
	cases := []struct {
		typeID string
		subID  string
		label  string
		group  string
		want   int
	}{
		// g013 全不中：玩法名用阿拉伯数字
		{"g013", "348", "全不中5不中复式", "全不中", 5},
		{"g013", "349", "全不中5不中拖头", "全不中", 5},
		{"g013", "350", "全不中6不中复式", "全不中", 6},
		{"g013", "352", "全不中7不中复式", "全不中", 7},
		{"g013", "356", "全不中9不中复式", "全不中", 9},
		{"g013", "358", "全不中10不中复式", "全不中", 10},
		{"g013", "360", "全不中11不中复式", "全不中", 11},
		{"g013", "362", "全不中12不中复式", "全不中", 12},
		{"g013", "363", "全不中12不中拖头", "全不中", 12},
		{"g013", "364", "全不中15不中复式", "全不中", 15},
		{"g013", "365", "全不中15不中拖头", "全不中", 15},
		// g014 多选中一：玩法名用中文数字
		{"g014", "366", "多选中一五选中一复式", "多选中一", 5},
		{"g014", "368", "多选中一六选中一复式", "多选中一", 6},
		{"g014", "370", "多选中一七选中一复式", "多选中一", 7},
		{"g014", "372", "多选中一八选中一复式", "多选中一", 8},
		{"g014", "374", "多选中一九选中一复式", "多选中一", 9},
		{"g014", "376", "多选中一十选中一复式", "多选中一", 10},
		{"g014", "377", "多选中一十选中一拖头", "多选中一", 10},
	}

	for _, c := range cases {
		t.Run(c.label, func(t *testing.T) {
			meta := lhcMeta(t, c.typeID, c.subID, c.label, c.group)
			if got := lhcBuzhongMinPick(meta); got != c.want {
				t.Errorf("lhcBuzhongMinPick = %d, 玩法名与第三方约束均为 %d", got, c.want)
			}
		})
	}
}

// TestLhcMinPickCount_teping 特平中N粒任中：sub_id 为数字时须回退玩法名，不可默认 1 个。
func TestLhcMinPickCount_tepingRenzhong(t *testing.T) {
	for _, c := range []struct {
		subID string
		label string
		want  int
	}{
		{"379", "特平中二粒任中", 2},
		{"381", "特平中三粒任中", 3},
		{"383", "特平中四粒任中", 4},
		{"385", "特平中五粒任中", 5},
	} {
		t.Run(c.label, func(t *testing.T) {
			meta := lhcMeta(t, "g015", c.subID, c.label, "特平中")
			mode := inferLHCBetMode(meta)
			if got := lhcMinPickCount(meta, mode); got != c.want {
				t.Errorf("lhcMinPickCount(mode=%s) = %d, 第三方要求最少 %d 个号码", mode, got, c.want)
			}
		})
	}
}

// TestLhcArabicPickCount 「全不中」前缀里也含「不中」，只能取到带数字那处。
func TestLhcArabicPickCount(t *testing.T) {
	for _, c := range []struct {
		label string
		want  int
	}{
		{"全不中5不中复式", 5},
		{"全不中12不中拖头", 12},
		{"全不中15不中复式", 15},
		{"10选中一", 10},
		{"特平中2粒任中", 2},
		{"多选中一五选中一复式", 0}, // 中文数字，交给 lhcPickCountFromLabel
		{"全不中6不中拖头", 6},
		{"全不中", 0},
		{"", 0},
	} {
		if got := lhcArabicPickCount(c.label); got != c.want {
			t.Errorf("lhcArabicPickCount(%q) = %d, want %d", c.label, got, c.want)
		}
	}
}
