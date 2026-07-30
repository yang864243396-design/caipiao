package rulessync

import (
	"strings"
	"testing"
)

// 上游或解码损坏出来的名称（含 U+FFFD）必须让 BuildPlan 报错，
// 而不是静默入库——玩法名是下注模式与号池的判定依据，损坏一个字就会选错号。
// 真实案例：lhc_std g003/299「三全中复式」的「式」被存成两个 U+FFFD，
// 该玩法在正式盘长期无法下单（见 migrations/00137）。
func TestBuildPlanRejectsReplacementChar(t *testing.T) {
	okTpl := func() RulesTemplate {
		return RulesTemplate{
			Name: "六合彩",
			Groups: []RulesGroup{{
				Name: "连码",
				Team: []RulesTeam{{
					Name: "三全中",
					Rule: []RulesRule{
						{ID: "299", Name: "复式", FullName: "三全中复式", Active: true},
					},
				}},
			}},
		}
	}

	if _, err := BuildPlan("lhc_std", "5", okTpl()); err != nil {
		t.Fatalf("干净数据应能建计划: %v", err)
	}

	// 「式」损坏成替换字符，正是生产库里出现过的形态。
	const broken = "三全中复\uFFFD\uFFFD"

	cases := []struct {
		name   string
		mutate func(*RulesTemplate)
		want   string
	}{
		{"子玩法名损坏", func(tpl *RulesTemplate) {
			tpl.Groups[0].Team[0].Rule[0].FullName = broken
		}, "子玩法"},
		{"玩法类型名损坏", func(tpl *RulesTemplate) {
			tpl.Groups[0].Name = "连\uFFFD"
		}, "玩法类型"},
		{"模板名损坏", func(tpl *RulesTemplate) {
			tpl.Name = "六合\uFFFD"
		}, "名称含替换字符"},
		{"队名损坏（进 segment_rule）", func(tpl *RulesTemplate) {
			tpl.Groups[0].Team[0].Name = "三全\uFFFD"
		}, "子玩法"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tpl := okTpl()
			tc.mutate(&tpl)
			_, err := BuildPlan("lhc_std", "5", tpl)
			if err == nil {
				t.Fatal("含替换字符的名称应被拒绝，否则会静默写坏玩法名")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("报错应指明损坏位置，含 %q，实际: %v", tc.want, err)
			}
		})
	}
}
