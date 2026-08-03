package schemes

import (
	"fmt"
	"strings"
	"testing"
)

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

func TestHezhiMaxBetUnits_qian2Is90(t *testing.T) {
	rule := resolveSSCPlayRule("g004", "41", "hezhi", "前二直选和值")
	if rule.SegmentLen != 2 {
		t.Fatalf("segmentLen=%d want 2", rule.SegmentLen)
	}
	got := maxBetUnitsForPlay(rule)
	if got != 90 {
		t.Fatalf("max=%d want 90", got)
	}
	// 0–18 满选 = 100 注 > 90
	full := "0,1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18"
	if err := validateGroupContent(rule, full); err == nil {
		t.Fatal("want max bet error for full 0-18")
	} else if got := err.Error(); got != "投注注数超过最大投注注数:90" {
		t.Fatalf("err=%q", got)
	}
	// 单和值应过
	if err := validateGroupContent(rule, "9"); err != nil {
		t.Fatalf("single sum should pass: %v", err)
	}
}

func TestZhixuanDanshiMaxBetUnits_qian2Is90(t *testing.T) {
	rule := resolveSSCPlayRule("g004", "39", "danshi", "前二直选单式")
	if rule.SegmentLen != 2 {
		t.Fatalf("segmentLen=%d want 2", rule.SegmentLen)
	}
	got := maxBetUnitsForPlay(rule)
	if got != 90 {
		t.Fatalf("max=%d want 90", got)
	}
	// 00–99 共 100 注 > 90 应拒
	var parts []string
	for a := 0; a <= 9; a++ {
		for b := 0; b <= 9; b++ {
			parts = append(parts, fmt.Sprintf("%d%d", a, b))
		}
	}
	if err := validateGroupContent(rule, strings.Join(parts, ",")); err == nil {
		t.Fatal("want max bet error for 100 units")
	} else if got := err.Error(); got != "投注注数超过最大投注注数:90" {
		t.Fatalf("err=%q", got)
	}
	// 前 90 注刚好过
	if err := validateGroupContent(rule, strings.Join(parts[:90], ",")); err != nil {
		t.Fatalf("90 units should pass: %v", err)
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
