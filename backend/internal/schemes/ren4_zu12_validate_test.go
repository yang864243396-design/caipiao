package schemes

import (
	"strings"
	"testing"
)

func TestValidateSchemeBetContent_ren4Zu12DualZone(t *testing.T) {
	t.Parallel()
	raw := []byte(`{
		"playTemplate":"ssc_std","playTypeId":"g011","subPlayId":"144","catalogSubId":"144",
		"betMode":"zu12","playMethodLabel":"任四组选12","renPositionCount":4,"segmentLen":1
	}`)
	if vs := ValidateSchemeBetContent("custom", raw, "万,千,百,十\n12,34", 0); len(vs) > 0 {
		t.Fatalf("dual zone should pass: %+v", vs)
	}
	if vs := ValidateSchemeBetContent("custom", raw, "万,千,百,十\n1,2", 0); !hasDetailContains(vs, "二重号码") {
		t.Fatalf("singles <2 should fail: %+v", vs)
	}
	if vs := ValidateSchemeBetContent("custom", raw, "1,2,3,4", 0); !hasDetailContains(vs, "二重号码") {
		t.Fatalf("flat pool should fail dual-zone validate: %+v", vs)
	}
	if vs := ValidateSchemeBetContent("custom", raw, "1,12", 0); !hasDetailContains(vs, "二重") {
		t.Fatalf("overlap 1,12 should fail: %+v", vs)
	}
	// 23,123：跨区重叠保留，计 2 注，可投
	if vs := ValidateSchemeBetContent("custom", raw, "万,千,百,十\n23,123", 0); len(vs) > 0 {
		t.Fatalf("23,123 should pass: %+v", vs)
	}
	if vs := ValidateSchemeBetContent("custom", raw, "万,千,百,十\n2,123", 0); len(vs) > 0 {
		t.Fatalf("2,123 should pass: %+v", vs)
	}
	if n := countZu12DualZoneBetUnits("12,34"); n != 2 {
		t.Fatalf("12,34 units=%d want 2", n)
	}
	if n := countZu12DualZoneBetUnits("1,234"); n != 3 {
		t.Fatalf("1,234 units=%d want 3", n)
	}
	if n := countZu12DualZoneBetUnits("1,12"); n != 0 {
		t.Fatalf("1,12 units=%d want 0", n)
	}
	if n := countZu12DualZoneBetUnits("2,123"); n != 1 {
		t.Fatalf("2,123 units=%d want 1", n)
	}
	if n := countZu12DualZoneBetUnits("23,123"); n != 2 {
		t.Fatalf("23,123 units=%d want 2", n)
	}
	if n := countZu12DualZoneBetUnits("12,3234"); n != 4 {
		t.Fatalf("12,3234 units=%d want 4", n)
	}
}

func hasDetailContains(vs []Violation, want string) bool {
	for _, v := range vs {
		if strings.Contains(v.Detail, want) {
			return true
		}
	}
	return false
}
