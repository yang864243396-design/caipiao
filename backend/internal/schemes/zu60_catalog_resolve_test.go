package schemes

import (
	"encoding/json"
	"testing"
)

// 模拟前端落库：有 playMethodLabel、数字 subId，betMode 可能有/无
func TestZu60CatalogResolve_infersDual(t *testing.T) {
	cases := []map[string]any{
		{
			"playTemplate": "ssc_std", "playTypeId": "g015", "subPlayId": "157", "catalogSubId": "157",
			"betMode": "zu60", "playMethodLabel": "组选60", "segmentLen": 1,
		},
		{
			"playTemplate": "ssc_std", "playTypeId": "g015", "subPlayId": "157", "catalogSubId": "157",
			"playMethodLabel": "组选60", "segmentLen": 5,
		},
		{
			"playTemplate": "fast_ssc_std", "playTypeId": "g015", "subPlayId": "157",
			"playMethodLabel": "组选60", "guajiFullName": "五星组选60",
		},
	}
	for i, cfg := range cases {
		raw, err := json.Marshal(cfg)
		if err != nil {
			t.Fatal(err)
		}
		rule := parseSchemeConfig("custom", raw, 0, 0).Play
		t.Logf("case%d BetMode=%q Sub=%q Cat=%q Seg=%d isZu60=%v minPick=%d",
			i, rule.BetMode, rule.SubPlayID, rule.CatalogSubID, rule.SegmentLen, isZu60PlayRule(rule), zuxuanPoolMinPick(rule))
		if !isZu60PlayRule(rule) {
			t.Fatalf("case%d should recognize zu60 dual, rule=%+v", i, rule)
		}
		if got := zuxuanPoolMinPick(rule); got != 0 {
			t.Fatalf("case%d minPick=%d want 0", i, got)
		}
		vs := ValidateSchemeBetContent("custom", raw, "1,234", 0)
		if len(vs) > 0 {
			t.Fatalf("case%d dual zu60 should accept 1,234, got %+v", i, vs)
		}
	}
}

func TestValidateSchemeConfig_fixedRotateZu60(t *testing.T) {
	raw, err := json.Marshal(map[string]any{
		"playTemplate":    "ssc_std",
		"playTypeId":      "g015",
		"subPlayId":       "157",
		"catalogSubId":    "157",
		"playMethodLabel": "组选60",
		"runTypeId":       "fixed_rotate",
		"schemeGroups":    []string{"1,234"},
	})
	if err != nil {
		t.Fatal(err)
	}
	vs := ValidateSchemeConfig("custom", raw)
	if len(vs) > 0 {
		t.Fatalf("fixed_rotate 1,234 should pass, got %+v", vs)
	}
}
