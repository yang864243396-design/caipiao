package guajibet

import (
	"encoding/json"
	"testing"
)

// 直选跨度带 solo=true 会被第三方以 40000「单挑参数错误」拒单。
//
// 2026-07-28 实测（tron_ffc_1m / game_id=19，账号 v6ceshi01）：
//
//	rule_id=4  前三直选跨度  content=0 bets=10 solo=true  → 40000 单挑参数错误
//	rule_id=4  前三直选跨度  content=0 bets=10 solo=false → OK id=121485611
//	rule_id=29 后三直选跨度  content=0 bets=10 solo=true  → 40000 单挑参数错误
//	rule_id=29 后三直选跨度  content=0 bets=10 solo=false → OK id=121486409
//
// 跨度取的是一个属性值（最大位减最小位），一个值天然对应多个号码组合，
// 不存在「单挑某一注」的语义，所以第三方不接受单挑标记。
// 修复前 NeedsSoloForRule 里没有 kuadu 分支，一路落到末尾的 return true。
//
// 同批实测里「直选和值」带 solo=true 是能过的（rule_id=3 OK），
// 所以不能顺手把和值一起关掉——这张表本来就不对称。

func kuaduMeta(t *testing.T, typeID, subID, label, group, ruleID string) RuleMeta {
	t.Helper()
	seg, err := json.Marshal(map[string]string{
		"guajiGroup": group, "guajiFullName": label, "guajiRuleId": ruleID,
	})
	if err != nil {
		t.Fatalf("marshal seg: %v", err)
	}
	return ParseRuleMeta("ssc_std", typeID, subID, label, group, seg, ruleID)
}

// TestResolveSolo_zhixuanKuadu 直选跨度一律 solo=false。
func TestResolveSolo_zhixuanKuadu(t *testing.T) {
	for _, tc := range []struct {
		name   string
		typeID string
		subID  string
		label  string
		group  string
		ruleID string
	}{
		{"前三直选跨度", "g001", "4", "前三直选跨度", "前三码", "4"},
		{"中三直选跨度", "g002", "17", "中三直选跨度", "中三码", "17"},
		{"后三直选跨度", "g003", "29", "后三直选跨度", "后三码", "29"},
		{"前二直选跨度", "g004", "41", "前二直选跨度", "前二码", "41"},
		{"后二直选跨度", "g005", "49", "后二直选跨度", "后二码", "49"},
		{"前中后三直选跨度", "g007", "104", "前中后三直选跨度", "前中后三", "104"},
		{"前后三直选跨度", "g012", "92", "前后三直选跨度", "前后三", "92"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			meta := kuaduMeta(t, tc.typeID, tc.subID, tc.label, tc.group, tc.ruleID)
			if mode := InferBetMode(meta); mode != "kuadu" {
				t.Fatalf("betMode = %q，本用例前提是跨度", mode)
			}
			// 跨度 0 即各位同码，前三/后三下对应 10 注
			if ResolveSolo(meta, "0", 10) {
				t.Error("跨度带 solo=true 会被第三方判「单挑参数错误」直接拒单")
			}
			// 注数与内容都不该改变结论
			if ResolveSolo(meta, "4", 1) {
				t.Error("单注跨度同样不接受 solo=true")
			}
			if ResolveSolo(meta, "0,1,2", 30) {
				t.Error("多值跨度同样不接受 solo=true")
			}
		})
	}
}

// TestResolveSolo_hezhiStillSolo 和值不受跨度改动影响。
// 同一批实测中 rule_id=3 前三直选和值带 solo=true 下单成功，
// 若把和值也一起关掉就会反向踩坑。
func TestResolveSolo_hezhiKeepsExistingBehavior(t *testing.T) {
	meta := kuaduMeta(t, "g001", "3", "前三直选和值", "前三码", "3")
	if mode := InferBetMode(meta); mode != "hezhi" {
		t.Fatalf("betMode = %q，本用例前提是和值", mode)
	}
	if !ResolveSolo(meta, "13", 6) {
		t.Error("前三直选和值应保持 solo=true（2026-07-28 实测 rule_id=3 solo=true 下单成功）")
	}
}
