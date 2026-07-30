package guajibet

import (
	"encoding/json"
	"testing"
)

// FormatBetContentForRule 必须幂等：下注链路会格式化两次
// （schemes/worker_guaji.go 先 format 一次传给 accountsvc，
//   accountsvc/place_bet.go 收到后又 format 一次做兜底）。
//
// 两位数号码彩种（十一选五 01-11、PK10 01-10）的单式原先不幂等，
// 第二次格式化会按单字符取号再补零，把号码改成别的号：
//
//	"010203"           -> "000100"
//	"0102030405060708" -> "0001000200030004"
//
// 2026-07-28 真实下单矩阵里十一选五 12/26、极速 PK10 7/32 条因此被第三方
// 以 40000「投注数字不合规」拒单——即这些彩种的单式玩法在生产上根本下不了单。

func twoDigitMeta(t *testing.T, template, typeID, subID, label, group, team string) RuleMeta {
	t.Helper()
	seg, err := json.Marshal(map[string]string{
		"guajiGroup": group, "guajiTeam": team, "guajiFullName": label, "guajiRuleId": subID,
	})
	if err != nil {
		t.Fatalf("marshal seg: %v", err)
	}
	return ParseRuleMeta(template, typeID, subID, label, group, seg, subID)
}

func TestFormatBetContentForRule_idempotentTwoDigitTemplates(t *testing.T) {
	cases := []struct {
		name     string
		template string
		typeID   string
		subID    string
		label    string
		group    string
		team     string
		raw      string
		wantWire string
	}{
		{"十一选五 前三直选单式", "syxw_std", "g001", "167", "前三直选单式", "前三直选", "前三直选", "123", "010203"},
		{"十一选五 前二直选单式", "syxw_std", "g002", "171", "前二直选单式", "前二直选", "前二直选", "12", "0102"},
		{"极速PK10 前二直选单式", "pk10_std", "g004", "194", "前二直选单式", "前二直选", "前二直选", "12", "0102"},
		{"极速PK10 前五直选单式", "pk10_std", "g007", "200", "前五直选单式", "前五直选", "前五直选", "12345", "0102030405"},
		// 和值大小单双走 formatPK10DxdsComboWire，二次格式化会叠前缀："和大" → "和和大"
		{"极速PK10 前三大小单双", "pk10_std", "g010", "222", "前三大小单双", "前三", "前三", "大", "和大"},
		{"极速PK10 后三大小单双", "pk10_std", "g010", "223", "后三大小单双", "后三", "后三", "大", "和大"},
		// 标签含「和值」，InferBetMode 会推成 hezhi，须仍走 和大/和小/和单/和双
		{"极速PK10 冠亚和值大小单双", "pk10_std", "g010", "221", "冠亚和值大小单双", "冠亚", "冠亚", "大", "和大"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			meta := twoDigitMeta(t, c.template, c.typeID, c.subID, c.label, c.group, c.team)

			once := FormatBetContentForRule(meta, c.raw)
			if once != c.wantWire {
				t.Fatalf("首次格式化 = %q, want %q", once, c.wantWire)
			}
			if twice := FormatBetContentForRule(meta, once); twice != once {
				t.Errorf("二次格式化把号码改了: %q -> %q（下注链路会格式化两次，第三方将以「投注数字不合规」拒单）", once, twice)
			}
		})
	}
}

// TestFormatBetContentForRule_idempotentSSC 单字符号码彩种本就幂等，防止修两位数时改坏。
func TestFormatBetContentForRule_idempotentSSC(t *testing.T) {
	cases := []struct {
		name          string
		typeID, subID string
		label, group  string
		raw           string
	}{
		{"前三直选单式", "g001", "2", "前三直选单式", "前三码", "0,1,2"},
		{"前三直选复式", "g001", "1", "前三直选复式", "前三码", "0,1,2"},
		{"后三直选单式", "g003", "27", "后三直选单式", "后三码", "1,2,3"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			meta := twoDigitMeta(t, "ssc_std", c.typeID, c.subID, c.label, c.group, c.group)
			once := FormatBetContentForRule(meta, c.raw)
			if twice := FormatBetContentForRule(meta, once); twice != once {
				t.Errorf("二次格式化不幂等: %q -> %q", once, twice)
			}
		})
	}
}
