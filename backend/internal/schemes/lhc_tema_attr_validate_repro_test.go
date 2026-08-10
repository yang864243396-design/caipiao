package schemes_test

import (
	"encoding/json"
	"testing"

	"caipiao/backend/internal/schemes"
)

func TestValidateSchemeConfig_lhcTemaAttrsOnly(t *testing.T) {
	cfg := map[string]any{
		"playTemplate": "lhc_std",
		"playTypeId":   "g001",
		"subPlayId":    "272",
		"typeId":       "g001",
		"subId":        "272",
		"betMode":      "tema",
		"runTypeId":    "fixed_rotate",
		"schemeGroups": []string{"|大,小,单|"},
	}
	raw, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if vs := schemes.ValidateSchemeConfig("custom", raw); len(vs) > 0 {
		t.Fatalf("attrs-only should save, got %+v", vs)
	}
	if vs := schemes.ValidateSchemeBetContent("custom", raw, "07,13|大|红波", 0); len(vs) > 0 {
		t.Fatalf("mixed content should pass, got %+v", vs)
	}
	if vs := schemes.ValidateSchemeBetContent("custom", raw, "|大,foo|", 0); len(vs) == 0 {
		t.Fatal("unknown attr should fail")
	}
}
