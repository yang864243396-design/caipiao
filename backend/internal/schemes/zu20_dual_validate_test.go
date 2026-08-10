package schemes

import (
	"encoding/json"
	"testing"
)

func TestValidateSchemeBetContent_wuxingZu20DualZoneMin2(t *testing.T) {
	raw, err := json.Marshal(map[string]any{
		"playTemplate":    "ssc_std",
		"playTypeId":      "g015",
		"subPlayId":       "159",
		"catalogSubId":    "159",
		"betMode":         "zu20",
		"playMethodLabel": "组选20",
		"segmentLen":      5,
		"runTypeId":       "fixed_rotate",
		"schemeGroups":    []string{"12,34"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if vs := ValidateSchemeBetContent("custom", raw, "12,34", 0); len(vs) > 0 {
		t.Fatalf("12,34 should be valid, got %+v", vs)
	}
	if vs := ValidateSchemeBetContent("custom", raw, "123,345", 0); len(vs) > 0 {
		t.Fatalf("123,345 should be valid, got %+v", vs)
	}
	if vs := ValidateSchemeBetContent("custom", raw, "123,456", 0); len(vs) > 0 {
		t.Fatalf("123,456 should be valid, got %+v", vs)
	}
	if vs := ValidateSchemeBetContent("custom", raw, "1,2", 0); !hasDetailContains(vs, "各 2") {
		t.Fatalf("1,2 should fail min2, got %+v", vs)
	}
	if vs := ValidateSchemeBetContent("custom", raw, "12,345", 0); !hasDetailContains(vs, "各 2") {
		t.Fatalf("12,345 should fail unequal counts, got %+v", vs)
	}
	if n := countZu20DualZoneBetUnits("12,34"); n != 2 {
		t.Fatalf("12,34 units=%d want 2", n)
	}
	if n := countZu20DualZoneBetUnits("123,345"); n != 7 {
		t.Fatalf("123,345 units=%d want 7", n)
	}
}
