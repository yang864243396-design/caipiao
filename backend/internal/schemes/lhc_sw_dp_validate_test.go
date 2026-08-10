package schemes

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestValidateSchemeBetContent_lhcSwDuipeng(t *testing.T) {
	cfg := []byte(`{"playTemplate":"lhc_std","typeId":"g003","subId":"283","betMode":"sw_dp","playMethodLabel":"二全中生尾对碰","guajiGroup":"二全中"}`)
	for _, c := range []string{"马|0", "0|马", "马,0", "鼠|1"} {
		vs := ValidateSchemeBetContent("custom", cfg, c, 0)
		if len(vs) > 0 {
			t.Fatalf("content %q should be valid, got: %s", c, vs[0].Detail)
		}
	}
	if vs := ValidateSchemeBetContent("custom", cfg, "马", 0); len(vs) == 0 {
		t.Fatal("zodiac only should be invalid")
	}
	if vs := ValidateSchemeBetContent("custom", cfg, "0|1", 0); len(vs) == 0 {
		t.Fatal("two tails should be invalid")
	}
	if vs := ValidateSchemeBetContent("custom", cfg, "马|蛇", 0); len(vs) == 0 {
		t.Fatal("two zodiacs should be invalid")
	}
	vs := ValidateSchemeBetContent("custom", cfg, "马|0", 0)
	for _, v := range vs {
		if strings.Contains(v.Detail, "不在合法号池") {
			t.Fatalf("sw_dp must not use 1-49 pool check: %s", v.Detail)
		}
	}
}

func TestValidateSchemeConfig_lhcSwDuipeng(t *testing.T) {
	raw, err := json.Marshal(map[string]any{
		"playTemplate": "lhc_std",
		"playTypeId":   "g003",
		"subPlayId":    "283",
		"typeId":       "g003",
		"subId":        "283",
		"betMode":      "sw_dp",
		"runTypeId":    "fixed_rotate",
		"schemeGroups": []string{"马|0"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if vs := ValidateSchemeConfig("custom", raw); len(vs) > 0 {
		t.Fatalf("马|0 should save, got %+v", vs)
	}
}

func TestCountLHCSwDuipengBetUnits(t *testing.T) {
	if n := countLHCSwDuipengBetUnits("马|0"); n != 20 {
		t.Fatalf("马|0 units=%d want 20", n)
	}
	if n := countLHCSwDuipengBetUnits("鼠|0"); n != 16 {
		t.Fatalf("鼠|0 units=%d want 16", n)
	}
	if n := countLHCSwDuipengBetUnits("马|1"); n != 24 {
		t.Fatalf("马|1 units=%d want 24 (∩01)", n)
	}
	if n := countLHCSwDuipengBetUnits("狗|5"); n != 19 {
		t.Fatalf("狗|5 units=%d want 19 (∩45)", n)
	}
}
