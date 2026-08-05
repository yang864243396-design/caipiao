package guajibet

import (
	"encoding/json"
	"testing"
)

func ren2ZuxuanDanshiMeta() RuleMeta {
	seg, _ := json.Marshal(map[string]string{"guajiGroup": "任选", "guajiTeam": "任选二"})
	return ParseRuleMeta("ssc_std", "g011", "78", "任二组选单式", "任选", seg, "78")
}

func TestFormatRenxuanZuxuanDanshiDigitPool(t *testing.T) {
	meta := ren2ZuxuanDanshiMeta()
	wire := FormatBetContentForRule(meta, "万,千\n1,2,3")
	if wire != "万千|12,13,23" {
		t.Fatalf("wire=%q want 万千|12,13,23", wire)
	}
	if n := CountBetNums(meta, wire); n != 3 {
		t.Fatalf("betsNums=%d want 3", n)
	}
}

func TestFormatRenxuanZuxuanDanshiKeepsTickets(t *testing.T) {
	meta := ren2ZuxuanDanshiMeta()
	wire := FormatBetContentForRule(meta, "千,个\n12,21,34")
	if wire != "千个|12,34" {
		t.Fatalf("wire=%q want 千个|12,34", wire)
	}
}

func TestCountRenxuanZuxuanDanshiDigitPoolRaw(t *testing.T) {
	meta := ren2ZuxuanDanshiMeta()
	// 未 Format 的号池+多选位：C(3,2)×C(3,2)=9
	if n := CountBetNums(meta, "万,千,个\n1,2,3"); n != 9 {
		t.Fatalf("betsNums=%d want 9", n)
	}
}
