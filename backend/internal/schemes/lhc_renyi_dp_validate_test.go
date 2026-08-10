package schemes

import (
	"strings"
	"testing"
)

func TestValidateSchemeBetContent_lhcRenyiDuipengMaxTen(t *testing.T) {
	cfg := []byte(`{"playTemplate":"lhc_std","typeId":"g003","subId":"284","betMode":"renyi_dp","playMethodLabel":"二全中任意对碰"}`)
	if vs := ValidateSchemeBetContent("custom", cfg, "01,02,03,04,05|06,07,08,09,10", 0); len(vs) != 0 {
		t.Fatalf("10 numbers should be valid: %+v", vs)
	}

	vs := ValidateSchemeBetContent("custom", cfg, "01,02,03,04,05,06|07,08,09,10,11", 0)
	for _, v := range vs {
		if strings.Contains(v.Detail, "合计最多选择 10 个号码") {
			return
		}
	}
	t.Fatalf("11 numbers should be rejected by the total cap: %+v", vs)
}

func TestValidateSchemeBetContent_lhcRenyiDuipengRejectsCrossZoneDuplicate(t *testing.T) {
	cfg := []byte(`{"playTemplate":"lhc_std","typeId":"g003","subId":"284","betMode":"renyi_dp","playMethodLabel":"二全中任意对碰"}`)
	vs := ValidateSchemeBetContent("custom", cfg, "01,02|02,03", 0)
	for _, v := range vs {
		if strings.Contains(v.Detail, "A区与B区号码不可重复") {
			return
		}
	}
	t.Fatalf("cross-zone duplicate should be rejected: %+v", vs)
}

func TestValidateSchemeBetContent_lhcRenyiDuipengRejectsNumbersOutsideOneToFortyNine(t *testing.T) {
	cfg := []byte(`{"playTemplate":"lhc_std","typeId":"g003","subId":"284","betMode":"renyi_dp","playMethodLabel":"二全中任意对碰"}`)
	vs := ValidateSchemeBetContent("custom", cfg, "01,50|02", 0)
	if len(vs) == 0 {
		t.Fatalf("out-of-range number should be rejected")
	}
}
