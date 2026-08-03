package schemes

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestValidateSchemeBetContent_expandsZhong3DanshiPositionPool(t *testing.T) {
	t.Parallel()
	raw := []byte(`{
		"runTypeId":"adv_trigger_bet",
		"playTemplate":"ssc_std",
		"playTypeId":"g002",
		"subPlayId":"2",
		"betMode":"danshi",
		"schemeGroups":["1,2\n3,4\n5,6"]
	}`)
	// 按位号池 2×2×2=8 注，不应再报「6 个单式组合不合法」
	vs := ValidateSchemeBetContent("custom", raw, "1,2\n3,4\n5,6", 0)
	for _, v := range vs {
		if strings.Contains(v.Detail, "单式组合不合法") || v.Code == ViolationZeroUnits {
			t.Fatalf("unexpected violation after expand: %+v", vs)
		}
	}
}

func TestValidateSchemeConfig_hotColdWarmIgnoresFlatSchemeGroupsPlaceholder(t *testing.T) {
	t.Parallel()
	// 历史/误拼：schemeGroups 被压成一行 15 个单码；真正配置在 hotColdWarm.pool
	raw := []byte(`{
		"runTypeId":"hot_cold_warm",
		"playTemplate":"ssc_std",
		"playTypeId":"g002",
		"subPlayId":"2",
		"betMode":"danshi",
		"schemeGroups":["0,1,2,3,4,5,6,7,8,9,0,1,2,3,4"],
		"hotColdWarm":{
			"totalPeriods":20,
			"strategy":"every",
			"pickTypes":["hot"],
			"pool":["0,1,2,3,4","5,6,7,8,9","0,1,2,3,4"]
		}
	}`)
	vs := ValidateSchemeConfig("custom", raw)
	for _, v := range vs {
		if strings.Contains(v.Detail, "单式组合不合法") || v.Code == ViolationZeroUnits {
			t.Fatalf("hot_cold_warm must not validate schemeGroups as danshi tickets: %+v", vs)
		}
	}
	if len(vs) != 0 {
		t.Fatalf("expected no violations, got %+v", vs)
	}
}

func TestNormalizeZhixuanDanshiContent_reshapesFlatDigitPools(t *testing.T) {
	t.Parallel()
	rule := resolveSSCPlayRule("g002", "2", "danshi", "中三直选单式")
	got := normalizeZhixuanDanshiContent(rule, "0,1,2,3,4,5,6,7,8,9,0,1,2,3,4")
	parts := strings.Split(got, ",")
	if len(parts) != 125 {
		t.Fatalf("want 5×5×5=125 tickets after reshape+expand, got %d content=%q", len(parts), got)
	}
}

func TestValidateSchemeConfig_triggerBetRequiresPosAndNeg(t *testing.T) {
	t.Parallel()
	// 中三组选和值：启用行缺反投时须拦下（前端同口径）
	cfg := map[string]any{
		"runTypeId":       "adv_trigger_bet",
		"playTemplate":    "ssc_std",
		"playTypeId":      "g002",
		"subPlayId":       "262",
		"betMode":         "hezhi",
		"playMethodLabel": "中三组选和值",
		"schemeGroups":    []string{"1"},
		"triggerBet": map[string]any{
			"mode": "always_pos",
			"rows": []map[string]any{
				{"enabled": true, "open": "1", "pos": "1", "neg": ""},
				{"enabled": true, "open": "2", "pos": "2", "neg": "3"},
				{"enabled": false, "open": "3", "pos": "", "neg": ""},
			},
		},
	}
	raw, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	vs := ValidateSchemeConfig("custom", raw)
	found := false
	for _, v := range vs {
		if v.Code == ViolationEmptyContent && strings.Contains(v.Detail, "开出 1") && strings.Contains(v.Detail, "反投") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("want empty neg violation for open=1, got %+v", vs)
	}
}

func TestValidateSchemeConfig_triggerBetValidatesRowsNotSampleGroup(t *testing.T) {
	t.Parallel()
	type row struct {
		Enabled bool   `json:"enabled"`
		Open    string `json:"open"`
		Pos     string `json:"pos"`
		Neg     string `json:"neg"`
	}
	cfg := map[string]any{
		"runTypeId":     "adv_trigger_bet",
		"playTemplate":  "ssc_std",
		"playTypeId":    "g002",
		"subPlayId":     "2",
		"betMode":       "danshi",
		"schemeGroups":  []string{"garbage"},
		"triggerBet": map[string]any{
			"mode": "always_pos",
			"rows": []row{{
				Enabled: true,
				Open:    "0",
				Pos:     "1\n2\n3",
				Neg:     "1\n2\n3",
			}},
		},
	}
	raw, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	vs := ValidateSchemeConfig("custom", raw)
	for _, v := range vs {
		if strings.Contains(v.Detail, "garbage") || strings.Contains(v.Detail, "单式组合不合法") {
			t.Fatalf("should validate trigger rows (expanded), not schemeGroups sample: %+v", vs)
		}
	}
	if len(vs) != 0 {
		t.Fatalf("expected no violations, got %+v", vs)
	}
}
