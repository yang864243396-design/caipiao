package schemes

import "testing"

func TestZhixuanFushiMaxBetUnits_qian3Is900(t *testing.T) {
	rule := resolveSSCPlayRule("g001", "1", "fushi", "前三直选复式")
	if rule.SegmentLen != 3 {
		t.Fatalf("segmentLen=%d want 3", rule.SegmentLen)
	}
	got := zhixuanFushiMaxBetUnits(rule)
	if got != 900 {
		t.Fatalf("max=%d want 900", got)
	}
}

func TestZhixuanFushiMaxBetUnits_qian2Is90(t *testing.T) {
	rule := resolveSSCPlayRule("g004", "qian2_zhixuan_fs", "fushi", "前二直选复式")
	got := zhixuanFushiMaxBetUnits(rule)
	if got != 90 {
		t.Fatalf("max=%d want 90", got)
	}
}

func TestValidateGroupContent_qian3FushiOver900(t *testing.T) {
	rule := resolveSSCPlayRule("g001", "1", "fushi", "前三直选复式")
	// 10×10×10 = 1000 > 900
	full := "0,1,2,3,4,5,6,7,8,9\n0,1,2,3,4,5,6,7,8,9\n0,1,2,3,4,5,6,7,8,9"
	if err := validateGroupContent(rule, full); err == nil {
		t.Fatal("want max bet error")
	} else if got := err.Error(); got != "投注注数超过最大投注注数:900" {
		t.Fatalf("err=%q", got)
	}
	// 10×10×9 = 900 刚好过
	ok := "0,1,2,3,4,5,6,7,8,9\n0,1,2,3,4,5,6,7,8,9\n0,1,2,3,4,5,6,7,8"
	if err := validateGroupContent(rule, ok); err != nil {
		t.Fatalf("900 units should pass: %v", err)
	}
}
