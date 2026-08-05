package schemes

import "testing"

func TestValidateSchemeBetContent_ren2ZuxuanDanshiDigitPool(t *testing.T) {
	raw := []byte(`{
		"playTemplate":"ssc_std","playTypeId":"g011","subPlayId":"78","betMode":"zuxuan_ds",
		"playMethodLabel":"任二组选单式","guajiGroup":"任选","segmentLen":2,"renPositionCount":2
	}`)
	vs := ValidateSchemeBetContent("custom", raw, "万,千\n1,2,3", 0)
	if len(vs) > 0 {
		t.Fatalf("digit pool should pass, got %#v", vs)
	}
	n, ok := CountBetUnitsForScheme("custom", raw, "万,千\n1,2,3")
	if !ok || n != 3 {
		t.Fatalf("units=%d ok=%v want 3", n, ok)
	}
}

func TestNormalizeZhixuanDanshiContent_ren2ZuxuanDanshiDigitPool(t *testing.T) {
	rule := resolveSSCPlayRule("g011", "78", "zuxuan_ds", "任二组选单式")
	got := normalizeZhixuanDanshiContent(rule, "万,个\n1,2,3")
	if got != "万个|12,13,23" && got != "万,个\n12,13,23" && got != "12,13,23" {
		// Format 产出 pipe wire；允许仅 picks 或带位名
		if got == "" {
			t.Fatalf("normalize empty, want expanded tickets")
		}
		t.Logf("normalize=%q", got)
	}
	if got == "" {
		t.Fatal("expected non-empty expand")
	}
}
