package schemes

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestValidateSchemeBetContent_lhcWsDuipeng(t *testing.T) {
	cfg := []byte(`{"playTemplate":"lhc_std","typeId":"g003","subId":"282","betMode":"ws_dp","playMethodLabel":"二全中尾数对碰","guajiGroup":"二全中"}`)
	for _, c := range []string{"0|1", "0,1", "1|2"} {
		vs := ValidateSchemeBetContent("custom", cfg, c, 0)
		if len(vs) > 0 {
			t.Fatalf("content %q should be valid, got: %s", c, vs[0].Detail)
		}
	}
	if vs := ValidateSchemeBetContent("custom", cfg, "0", 0); len(vs) == 0 {
		t.Fatal("single tail should be invalid")
	}
	if vs := ValidateSchemeBetContent("custom", cfg, "a|b", 0); len(vs) == 0 {
		t.Fatal("unknown tokens should be invalid")
	}
	vs := ValidateSchemeBetContent("custom", cfg, "0|1", 0)
	for _, v := range vs {
		if strings.Contains(v.Detail, "不在合法号池") {
			t.Fatalf("ws_dp must not use 1-49 pool check: %s", v.Detail)
		}
	}
}

func TestValidateSchemeConfig_lhcWsDuipeng(t *testing.T) {
	raw, err := json.Marshal(map[string]any{
		"playTemplate": "lhc_std",
		"playTypeId":   "g003",
		"subPlayId":    "282",
		"typeId":       "g003",
		"subId":        "282",
		"betMode":      "ws_dp",
		"runTypeId":    "fixed_rotate",
		"schemeGroups": []string{"0|1"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if vs := ValidateSchemeConfig("custom", raw); len(vs) > 0 {
		t.Fatalf("0|1 should save, got %+v", vs)
	}
}

func TestCountLHCWsDuipengBetUnits(t *testing.T) {
	if n := countLHCWsDuipengBetUnits("0|1"); n != 20 {
		t.Fatalf("0|1 units=%d want 20", n)
	}
	if n := countLHCWsDuipengBetUnits("1,2"); n != 25 {
		t.Fatalf("1,2 units=%d want 25", n)
	}
}
