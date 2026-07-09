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
