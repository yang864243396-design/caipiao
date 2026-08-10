package schemes

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestValidateSchemeBetContent_lhcSxDuipeng(t *testing.T) {
	cfg := []byte(`{"playTemplate":"lhc_std","typeId":"g003","subId":"281","betMode":"sx_dp","playMethodLabel":"二全中生肖对碰","guajiGroup":"二全中"}`)
	for _, c := range []string{"马|蛇", "马,蛇", "蛇|龙"} {
		vs := ValidateSchemeBetContent("custom", cfg, c, 0)
		if len(vs) > 0 {
			t.Fatalf("content %q should be valid, got: %s", c, vs[0].Detail)
		}
	}
	if vs := ValidateSchemeBetContent("custom", cfg, "马", 0); len(vs) == 0 {
		t.Fatal("single zodiac should be invalid")
	}
	if vs := ValidateSchemeBetContent("custom", cfg, "foo|bar", 0); len(vs) == 0 {
		t.Fatal("unknown tokens should be invalid")
	}
	vs := ValidateSchemeBetContent("custom", cfg, "马|蛇", 0)
	for _, v := range vs {
		if strings.Contains(v.Detail, "不在合法号池") {
			t.Fatalf("sx_dp must not use 1-49 pool check: %s", v.Detail)
		}
	}
}

func TestValidateSchemeConfig_lhcSxDuipeng(t *testing.T) {
	raw, err := json.Marshal(map[string]any{
		"playTemplate": "lhc_std",
		"playTypeId":   "g003",
		"subPlayId":    "281",
		"typeId":       "g003",
		"subId":        "281",
		"betMode":      "sx_dp",
		"runTypeId":    "fixed_rotate",
		"schemeGroups": []string{"马|蛇"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if vs := ValidateSchemeConfig("custom", raw); len(vs) > 0 {
		t.Fatalf("马|蛇 should save, got %+v", vs)
	}
	raw2, _ := json.Marshal(map[string]any{
		"playTemplate": "lhc_std",
		"typeId":       "g003",
		"subId":        "281",
		"betMode":      "sx_dp",
		"runTypeId":    "fixed_rotate",
		"schemeGroups": []string{"蛇|龙"},
	})
	if vs := ValidateSchemeConfig("custom", raw2); len(vs) > 0 {
		t.Fatalf("蛇|龙 should save, got %+v", vs)
	}
}

func TestCountLHCSxDuipengBetUnits(t *testing.T) {
	if n := countLHCSxDuipengBetUnits("马|蛇"); n != 20 {
		t.Fatalf("马|蛇 units=%d want 20", n)
	}
	if n := countLHCSxDuipengBetUnits("蛇,龙"); n != 16 {
		t.Fatalf("蛇,龙 units=%d want 16", n)
	}
}
