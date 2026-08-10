package schemes

import "testing"

// 前后四直选组合：内容按序填 4 位（0–9），位与位逗号分隔，例如 1,2,3,4（rule136 实测可下单）。
func TestQianhou4ZhixuanZuhe_CommaWireUnitsAndHit(t *testing.T) {
	rule := playRule{
		PlayTypeID: "g014", BetMode: "zuhe",
		SegmentLen: 4, SegmentStart: 0,
	}
	content := "1,2,3,4"
	// 位积1 × 段长4 × 前后四区位2 = 8
	if units := countPlayWireBetUnits(rule, content); units != 8 {
		t.Fatalf("units=%d want 8 for %q", units, content)
	}
	ev := evaluateZuhe(rule, []string{"1", "2", "3", "4", "0"}, content) // 前四 1234
	if !ev.Hit || ev.BetUnits != 4 {
		t.Fatalf("前四命中: hit=%v units=%d want hit units=4", ev.Hit, ev.BetUnits)
	}
	ruleHou := rule
	ruleHou.SegmentStart = 1
	ev2 := evaluateZuhe(ruleHou, []string{"0", "1", "2", "3", "4"}, content) // 后四 1234
	if !ev2.Hit || ev2.BetUnits != 4 {
		t.Fatalf("后四命中: hit=%v units=%d want hit units=4", ev2.Hit, ev2.BetUnits)
	}
	ev3 := evaluateZuhe(rule, []string{"9", "9", "9", "9", "9"}, content)
	if ev3.Hit {
		t.Fatalf("不应命中全 9")
	}
}

func TestQianhou4ZhixuanZuhe_RejectsGluedAndWrongArity(t *testing.T) {
	rule := playRule{
		PlayTypeID: "g014", BetMode: "zuhe", SegmentLen: 4,
	}
	for _, bad := range []string{"1234", "1,2,3", "1,2,3,4,5"} {
		if units := countPlayWireBetUnits(rule, bad); units != 0 {
			t.Fatalf("%q: units=%d want 0", bad, units)
		}
	}
}

func TestQianhou4ZhixuanZuhe_MultiDigitPerPosition(t *testing.T) {
	rule := playRule{
		PlayTypeID: "g014", BetMode: "zuhe", SegmentLen: 4,
	}
	content := "12,3,4,5" // 位积 2×1×1×1=2 → ×4×2=16
	if units := countPlayWireBetUnits(rule, content); units != 16 {
		t.Fatalf("units=%d want 16", units)
	}
}
