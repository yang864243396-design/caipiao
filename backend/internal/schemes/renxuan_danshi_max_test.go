package schemes

import (
	"fmt"
	"strings"
	"testing"
)

func TestRen2ZhixuanDanshiMaxIs900Not90(t *testing.T) {
	raw := []byte(`{
		"runTypeId":"fixed_number","playTemplate":"ssc_std",
		"playTypeId":"g011","subPlayId":"75","betMode":"danshi",
		"playMethodLabel":"任二直选单式","playTypeLabel":"任选",
		"guajiGroup":"任选","schemeGroups":["x"]
	}`)
	cfg := parseSchemeConfig("custom", raw, 0, 0)
	if got := maxBetUnitsForPlay(cfg.Play); got != 900 {
		t.Fatalf("任二直选单式 max=%d want 900", got)
	}

	var parts []string
	for a := 0; a <= 9; a++ {
		for b := 0; b <= 9; b++ {
			parts = append(parts, fmt.Sprintf("%d%d", a, b))
		}
	}
	// 五位 × 00–99 = C(5,2)*100 = 1000 注 > 900
	content := "万,千,百,十,个\n" + strings.Join(parts, ",")
	units := countRenxuanZhixuanDanshiBetUnits(cfg.Play, content)
	if units != 1000 {
		t.Fatalf("units=%d want 1000", units)
	}
	vs := ValidateSchemeBetContent("custom", raw, content, 0)
	found := false
	for _, v := range vs {
		if v.Code == ViolationUnitsOverLimit {
			found = true
			if v.Detail != "投注注数超过最大投注注数:900" {
				t.Fatalf("Detail=%q want 投注注数超过最大投注注数:900", v.Detail)
			}
		}
	}
	if !found {
		t.Fatalf("want units over limit, got %+v", vs)
	}
	if err := validateGroupContent(cfg.Play, content); err == nil {
		t.Fatal("want max bet error")
	} else if got := err.Error(); got != "投注注数超过最大投注注数:900" {
		t.Fatalf("err=%q", got)
	}
}
