package schemes

import (
	"encoding/json"
	"testing"
)

func TestValidateSchemeBetContent_wuxingZu60DualZone(t *testing.T) {
	raw, err := json.Marshal(map[string]any{
		"playTemplate":    "ssc_std",
		"playTypeId":      "g015",
		"subPlayId":       "157",
		"catalogSubId":    "157",
		"betMode":         "zu60",
		"playMethodLabel": "组选60",
		"segmentLen":      5,
		"runTypeId":       "fixed_rotate",
		"schemeGroups":    []string{"1,234"},
	})
	if err != nil {
		t.Fatal(err)
	}
	rule := parseSchemeConfig("custom", raw, 0, 0).Play
	if got := zuxuanPoolMinPick(rule); got != 0 {
		t.Fatalf("zuxuanPoolMinPick=%d want 0 (dual-zone)", got)
	}
	if vs := ValidateSchemeBetContent("custom", raw, "1,234", 0); len(vs) > 0 {
		t.Fatalf("1,234 should be valid zu60 dual, got %+v", vs)
	}
	if vs := ValidateSchemeBetContent("custom", raw, "12,345", 0); len(vs) > 0 {
		t.Fatalf("12,345 should be valid, got %+v", vs)
	}
	if vs := ValidateSchemeBetContent("custom", raw, "12,3", 0); !hasDetailContains(vs, "二重") {
		t.Fatalf("12,3 should fail format, got %+v", vs)
	}
	if n := countZu60DualZoneBetUnits("1,234"); n != 1 {
		t.Fatalf("1,234 units=%d want 1", n)
	}
	if n := countZu60DualZoneBetUnits("12,345"); n != 2 {
		t.Fatalf("12,345 units=%d want 2", n)
	}
	if n := countZu60DualZoneBetUnits("12,3456"); n != 8 {
		t.Fatalf("12,3456 units=%d want 8", n)
	}
}

func TestValidateSchemeBetContent_wuxingZu120StillNeeds5(t *testing.T) {
	raw, err := json.Marshal(map[string]any{
		"playTemplate":    "ssc_std",
		"playTypeId":      "g015",
		"subPlayId":       "156",
		"catalogSubId":    "156",
		"betMode":         "zu120",
		"playMethodLabel": "组选120",
		"segmentLen":      5,
	})
	if err != nil {
		t.Fatal(err)
	}
	rule := parseSchemeConfig("custom", raw, 0, 0).Play
	if got := zuxuanPoolMinPick(rule); got != 5 {
		t.Fatalf("zu120 minPick=%d want 5", got)
	}
	if vs := ValidateSchemeBetContent("custom", raw, "1,2,3,4", 0); !hasDetail(vs, "号码池至少选择 5 个号码") &&
		!hasDetailContains(vs, "组选120至少选择 5") {
		t.Fatalf("zu120 with 4 digits should fail min pick, got %+v", vs)
	}
}
