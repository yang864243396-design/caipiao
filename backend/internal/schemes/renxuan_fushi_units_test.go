package schemes

import (
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
