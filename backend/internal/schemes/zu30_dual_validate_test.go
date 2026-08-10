package schemes

import (
	"encoding/json"
	"testing"
)

func TestValidateSchemeBetContent_wuxingZu30DualZone(t *testing.T) {
	raw, err := json.Marshal(map[string]any{
		"playTemplate":    "ssc_std",
		"playTypeId":      "g015",
		"subPlayId":       "158",
		"catalogSubId":    "158",
		"betMode":         "zu30",
		"playMethodLabel": "组选30",
		"segmentLen":      5,
		"runTypeId":       "fixed_rotate",
		"schemeGroups":    []string{"123,1"},
	})
	if err != nil {
		t.Fatal(err)
	}
	rule := parseSchemeConfig("custom", raw, 0, 0).Play
	if got := zuxuanPoolMinPick(rule); got != 0 {
		t.Fatalf("zuxuanPoolMinPick=%d want 0 (dual-zone)", got)
	}
	if vs := ValidateSchemeBetContent("custom", raw, "123,1", 0); len(vs) > 0 {
		t.Fatalf("123,1 should be valid zu30 dual, got %+v", vs)
	}
	if vs := ValidateSchemeBetContent("custom", raw, "123,45", 0); len(vs) > 0 {
		t.Fatalf("123,45 should be valid, got %+v", vs)
	}
	if vs := ValidateSchemeBetContent("custom", raw, "12,3", 0); !hasDetailContains(vs, "二重") {
		t.Fatalf("12,3 should fail format, got %+v", vs)
	}
	if n := countZu30DualZoneBetUnits("123,1"); n != 1 {
		t.Fatalf("123,1 units=%d want 1", n)
	}
	if n := countZu30DualZoneBetUnits("123,45"); n != 6 {
		t.Fatalf("123,45 units=%d want 6", n)
	}
	if n := countZu30DualZoneBetUnits("1234,56"); n != 12 {
		t.Fatalf("1234,56 units=%d want 12", n)
	}
}
