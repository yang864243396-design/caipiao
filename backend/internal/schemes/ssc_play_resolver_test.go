package schemes

import (
	"encoding/json"
	"testing"
)

func TestResolveSSCPlayRuleDingweiWan(t *testing.T) {
	rule := resolveSSCPlayRule("dingwei", "dingwei_wan", "dingwei")
	if rule.PositionIdx != 0 || rule.SegmentLen != 1 || rule.SubPlayID != "dingwei" {
		t.Fatalf("rule=%+v", rule)
	}
}

func TestResolveSSCPlayRuleQian3Fushi(t *testing.T) {
	rule := resolveSSCPlayRule("qian3", "qian3_zhixuan_fs", "fushi")
	if rule.SegmentStart != 0 || rule.SegmentLen != 3 || rule.SubPlayID != "zhixuan_fs" {
		t.Fatalf("rule=%+v", rule)
	}
}

func TestResolveSSCPlayRuleG001Fushi(t *testing.T) {
	rule := resolveSSCPlayRule("g001", "1", "fushi")
	if rule.SegmentStart != 0 || rule.SegmentLen != 3 || rule.SubPlayID != "zhixuan_fs" {
		t.Fatalf("g001 rule=%+v want segment 0,3 zhixuan_fs", rule)
	}
	eval := evaluatePlayHit(rule, nil, "0,1,3\n0\n0", false, "", 0)
	if eval.BetUnits != 3 {
		t.Fatalf("BetUnits=%d want 3 (3×1×1)", eval.BetUnits)
	}
	amount := calcBetAmount(eval.BetUnits, 1, 1)
	if amount != 3 {
		t.Fatalf("amount=%v want 3 (=注数×倍数×单位)", amount)
	}
}

func TestNormalizeBetPayloadSSCNewFields(t *testing.T) {
	raw, err := NormalizeBetPayload(BetPayload{
		PlayTemplate: "ssc_std",
		TypeID:       "dingwei",
		SubID:        "dingwei_wan",
		PlayMethod:   "一星定位胆 · 万位",
		GroupContent: "1,3,7",
	})
	if err != nil {
		t.Fatal(err)
	}
	var p BetPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		t.Fatal(err)
	}
	if p.TypeID != "dingwei" || p.SubID != "dingwei_wan" {
		t.Fatalf("payload=%+v", p)
	}
}
