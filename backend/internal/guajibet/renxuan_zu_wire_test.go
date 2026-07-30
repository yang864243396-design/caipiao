package guajibet

import (
	"encoding/json"
	"testing"
)

func renxuanMeta(t *testing.T, subID, label, team string) RuleMeta {
	t.Helper()
	seg, err := json.Marshal(map[string]string{
		"guajiGroup": "任选", "guajiTeam": team, "guajiFullName": team + label, "guajiRuleId": subID,
	})
	if err != nil {
		t.Fatalf("marshal seg: %v", err)
	}
	return ParseRuleMeta("ssc_std", "g011", subID, team+label, "任选", seg, subID)
}

// 任选四组选12/4 的投注内容是双区号池（12,34 = 二重号池,单号池），不是选号列表。
// 当成列表走号池补码会补出多余的号（12,34 → 12,34,3,4），
// 第三方回「投注数字不合规」（2026-07-28 真实下单 tron_ffc_1m 144 实测拒单）。
func TestRenxuanZu12WireKeepsTwoZonePool(t *testing.T) {
	meta := renxuanMeta(t, "144", "组选12", "任选四")

	sample := SampleGroupContent(meta)
	wire := FormatBetContentForRule(meta, sample)
	if wire != "万千百十|12,34" {
		t.Fatalf("wire = %q，期望 万千百十|12,34（双区号池原样带位名）", wire)
	}

	// 下注链路会对内容重复格式化一次，双区号池不能被再切一遍。
	if again := FormatBetContentForRule(meta, wire); again != wire {
		t.Fatalf("二次格式化 = %q，应与首次 %q 相同", again, wire)
	}

	if n := ResolveBetsNums(meta, wire, 0, 2, 1); n <= 0 {
		t.Fatalf("注数 = %d，双区号池应能算出注数", n)
	}
}

// 同组 组选24 是真的选号列表，须保持号池补码口径不变。
func TestRenxuanZu24WireStillPickList(t *testing.T) {
	meta := renxuanMeta(t, "143", "组选24", "任选四")
	wire := FormatBetContentForRule(meta, SampleGroupContent(meta))
	if wire != "万千百十|1,2,3,4" {
		t.Fatalf("wire = %q，期望 万千百十|1,2,3,4", wire)
	}
}
