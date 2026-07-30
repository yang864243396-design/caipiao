package guajibet

import (
	"encoding/json"
	"testing"
)

// 2026-07-28 全量真实下单矩阵（tron_ffc_1m / game_id=19，171 条玩法）跑出的 solo 取值错误。
//
// 第三方的 40000「单挑参数错误」是双向的：solo 该给不给、不该给却给，报的是同一个错。
// 所以每条都用 bet-probe 正反各打一次真单定值，结论如下（括号内为接单成功的第三方注单号）：
//
//	rule=43  前二组选单式      solo=true  拒 / solo=false 过 (121493366)
//	rule=51  后二组选单式      solo=true  拒 / solo=false 过 (121493388)
//	rule=105 前中后三组合      solo=false 拒 / solo=true  过 (121493739)
//	rule=110 前中后三混合组选  solo=false 拒 / solo=true  过 (121493436)
//	rule=75  任二直选单式      solo=false 拒 / solo=true  过 (121493456)
//	rule=81  任三直选单式      solo=false 拒 / solo=true  过 (121493490)
//	rule=93  前后三组合        solo=false 拒 / solo=true  过 (121493516)
//	rule=98  前后三混合组选    solo=false 拒 / solo=true  过 (121493535)
//
// seg / wire / betsNums 均取自生产库 sub_plays 与 FormatBetContentForRule 的实际产物。

type soloCase struct {
	name     string
	typeID   string
	subID    string
	label    string
	group    string
	team     string
	wire     string
	betsNums int
	wantSolo bool
}

func (c soloCase) meta(t *testing.T) RuleMeta {
	t.Helper()
	seg, err := json.Marshal(map[string]string{
		"guajiGroup":    c.group,
		"guajiTeam":     c.team,
		"guajiFullName": c.label,
		"guajiRuleId":   c.subID,
	})
	if err != nil {
		t.Fatalf("marshal seg: %v", err)
	}
	return ParseRuleMeta("ssc_std", c.typeID, c.subID, c.label, c.group, seg, c.subID)
}

func TestResolveSolo_realBetMatrix20260728(t *testing.T) {
	cases := []soloCase{
		// 前二/后二组选单式：单注也不接受单挑（既有规则只在多注时关掉）
		{"前二组选单式", "g004", "43", "前二组选单式", "前二码", "前二组选", "12", 1, false},
		{"后二组选单式", "g005", "51", "后二组选单式", "后二码", "后二组选", "12", 1, false},

		// 前中后三 / 前后三 的组合与混合组选：既有规则误判为 false
		{"前中后三组合", "g007", "105", "前中后三组合", "前中后三", "前中后三直选", "1,2,3", 9, true},
		{"前中后三混合组选", "g007", "110", "前中后三混合组选", "前中后三", "前中后三组选", "123", 3, true},
		{"前后三组合", "g012", "93", "前后三组合", "前后三", "前后三直选", "1,2,3", 6, true},
		{"前后三混合组选", "g012", "98", "前后三混合组选", "前后三", "前后三组选", "123", 2, true},

		// 任选直选单式：单注须单挑，与同组直选复式一致（原实现多了 k>=4 条件）
		{"任二直选单式", "g011", "75", "任二直选单式", "任选", "任选二", "千个|12", 1, true},
		{"任三直选单式", "g011", "81", "任三直选单式", "任选", "任选三", "万千个|123", 1, true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			meta := c.meta(t)
			if got := ResolveSolo(meta, c.wire, c.betsNums); got != c.wantSolo {
				t.Errorf("ResolveSolo = %v, 实测须 %v（另一取值会被第三方以 40000 单挑参数错误拒单）", got, c.wantSolo)
			}
		})
	}
}

// TestResolveSolo_sweepPassersUnchanged 守住同批 162 条下单成功的玩法，防止修上面 8 条时改坏。
// 下列取值均为 2026-07-28 真实接单验证过的。
func TestResolveSolo_sweepPassersUnchanged(t *testing.T) {
	cases := []soloCase{
		{"前二直选单式", "g004", "39", "前二直选单式", "前二码", "前二直选", "12", 1, true},
		{"前三直选复式", "g001", "1", "前三直选复式", "前三码", "前三直选", "0,1,2", 1, true},
		{"前三直选和值", "g001", "3", "前三直选和值", "前三码", "前三直选", "13", 6, true},
		{"前二组选复式", "g004", "42", "前二组选复式", "前二码", "前二组选", "12", 1, false},
		{"后二组选复式", "g005", "50", "后二组选复式", "后二码", "后二组选", "12", 1, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			meta := c.meta(t)
			if got := ResolveSolo(meta, c.wire, c.betsNums); got != c.wantSolo {
				t.Errorf("ResolveSolo = %v, 2026-07-28 实测下单成功时为 %v", got, c.wantSolo)
			}
		})
	}
}
