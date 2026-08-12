package schemes

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestRen2ZhixuanFushiUnitsAndMax(t *testing.T) {
	raw := []byte(`{
		"runTypeId":"fixed_rotate","playTemplate":"ssc_std",
		"playTypeId":"g011","subPlayId":"74","betMode":"fushi",
		"playMethodLabel":"直选复式","playTypeLabel":"任选",
		"guajiGroup":"任选","schemeGroups":["x"]
	}`)
	cfg := parseSchemeConfig("custom", raw, 0, 0)
	if !isRenxuanPlayType(cfg.Play.PlayTypeID) {
		t.Fatalf("want renxuan, play=%+v", cfg.Play)
	}
	if got := maxBetUnitsForPlay(cfg.Play); got != 900 {
		t.Fatalf("任二直选复式 max=%d want 900", got)
	}

	// 10,10,9,9,9 → C(5,2) 组合积 = 883（勿乘成 72900）
	content := strings.Join([]string{
		"1,2,3,4,5,6,7,8,9,0",
		"1,2,3,4,5,6,7,8,9,0",
		"1,2,3,4,5,6,7,8,9",
		"1,2,3,4,5,6,7,8,9",
		"1,2,3,4,5,6,7,8,9",
	}, "\n")
	units := countRenxuanZhixuanFushiBetUnits(cfg.Play, content)
	if units != 883 {
		t.Fatalf("units=%d want 883", units)
	}
	vs := ValidateSchemeBetContent("custom", raw, content, 0)
	for _, v := range vs {
		if strings.Contains(v.Detail, "超过上限") {
			t.Fatalf("883 应低于上限 900, got %+v", vs)
		}
	}

	// 粘连 wire 也应计 883
	wire := "1234567890,1234567890,123456789,123456789,123456789"
	if u := countRenxuanZhixuanFushiBetUnits(cfg.Play, wire); u != 883 {
		t.Fatalf("wire units=%d want 883", u)
	}
}

func TestRen3ZhixuanFushiMaxIs9000(t *testing.T) {
	raw := []byte(`{
		"runTypeId":"fixed_rotate","playTemplate":"ssc_std",
		"playTypeId":"g011","subPlayId":"80","betMode":"fushi",
		"playMethodLabel":"任三直选复式","playTypeLabel":"任选",
		"guajiGroup":"任选","schemeGroups":["x"]
	}`)
	cfg := parseSchemeConfig("custom", raw, 0, 0)
	if got := maxBetUnitsForPlay(cfg.Play); got != 9000 {
		t.Fatalf("任三直选复式 max=%d want 9000（前三900×C(5,3)）", got)
	}

	full := strings.Join([]string{
		"0,1,2,3,4,5,6,7,8,9",
		"0,1,2,3,4,5,6,7,8,9",
		"0,1,2,3,4,5,6,7,8,9",
		"0,1,2,3,4,5,6,7,8,9",
		"0,1,2,3,4,5,6,7,8,9",
	}, "\n")
	units := countRenxuanZhixuanFushiBetUnits(cfg.Play, full)
	if units != 10000 {
		t.Fatalf("units=%d want 10000 (C(5,3)×1000)", units)
	}
	vs := ValidateSchemeBetContent("custom", raw, full, 0)
	found := false
	for _, v := range vs {
		if strings.Contains(v.Detail, "9000") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("10000 应报超过 9000, got %+v", vs)
	}
}

func TestNormalizeBetPayload_Ren3ZhixuanFushiAllowsThreeSelectedPositions(t *testing.T) {
	input := BetPayload{
		PlayTemplate: "ssc_std",
		TypeID:       "g011",
		SubID:        "80",
		BetMode:      "fushi",
		PlayMethod:   "任选 任三直选复式",
		GroupContent: "0,1\n1,2\n\n\n4,5",
	}
	raw, err := NormalizeBetPayload(input)
	if err != nil {
		t.Fatalf("任三直选复式允许任选三个位置，空的未选位置不应拒绝: %v", err)
	}
	var got BetPayload
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if got.GroupContent != input.GroupContent {
		t.Fatalf("content=%q want %q", got.GroupContent, input.GroupContent)
	}
}
