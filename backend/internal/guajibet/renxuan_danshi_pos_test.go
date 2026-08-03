package guajibet

import (
	"encoding/json"
	"testing"
)

func ren2DanshiMeta() RuleMeta {
	seg, _ := json.Marshal(map[string]string{"guajiGroup": "任选", "guajiTeam": "任选二"})
	return ParseRuleMeta("ssc_std", "g011", "75", "任二直选单式", "任选", seg, "75")
}

func TestFormatRenxuanDanshiKeepsMultiPositions(t *testing.T) {
	meta := ren2DanshiMeta()
	wire := FormatBetContentForRule(meta, "万,千,百\n12,34")
	if wire != "万千百|12,34" {
		t.Fatalf("wire=%q want 万千百|12,34", wire)
	}
	if n := CountBetNums(meta, wire); n != 6 {
		t.Fatalf("betsNums=%d want 6 (C(3,2)*2)", n)
	}
}

func TestCountRenxuanDanshiExactTwoPositions(t *testing.T) {
	meta := ren2DanshiMeta()
	wire := FormatBetContentForRule(meta, "千,个\n12")
	if wire != "千个|12" {
		t.Fatalf("wire=%q want 千个|12", wire)
	}
	if n := CountBetNums(meta, wire); n != 1 {
		t.Fatalf("betsNums=%d want 1", n)
	}
}

func TestCountRenxuanDanshiFivePositions(t *testing.T) {
	meta := ren2DanshiMeta()
	wire := FormatBetContentForRule(meta, "万,千,百,十,个\n12")
	if wire != "万千百十个|12" {
		t.Fatalf("wire=%q want 万千百十个|12", wire)
	}
	if n := CountBetNums(meta, wire); n != 10 {
		t.Fatalf("betsNums=%d want 10 (C(5,2))", n)
	}
}
