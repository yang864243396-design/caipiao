package schemes

import "testing"

func TestCountPlayWireBetUnits_fushiQian3(t *testing.T) {
	t.Parallel()
	rule := pickTestConfig(t, `{
		"playTemplate":"ssc_std","playTypeId":"qian3","subPlayId":"1","betMode":"fushi"
	}`).Play
	full := "0,1,2,3,4,5,6,7,8,9\n0,1,2,3,4,5,6,7,8,9\n0,1,2,3,4,5,6,7,8,9"
	if got := countPlayWireBetUnits(rule, full); got != 1000 {
		t.Fatalf("full fushi units=%d want 1000", got)
	}
	if !contentExceedsBetUnitsMax(rule, full) {
		t.Fatal("1000 should exceed qian3 fushi max 900")
	}
	ok := "0,1,2,3,4,5,6,7,8\n0,1,2,3,4,5,6,7,8,9\n0,1,2,3,4,5,6,7,8,9"
	if got := countPlayWireBetUnits(rule, ok); got != 900 {
		t.Fatalf("9×10×10 units=%d want 900", got)
	}
	if contentExceedsBetUnitsMax(rule, ok) {
		t.Fatal("900 should not exceed max")
	}
}

// 四星直选组合：上限=复式上限×段长（9000×4=36000），勿写死三星 2700。
func TestZuheMaxBetUnits_sixing(t *testing.T) {
	t.Parallel()
	rule := resolveSSCPlayRule("sixing", "sixing_zuhe", "zuhe", "直选组合")
	if rule.SegmentLen != 4 {
		t.Fatalf("SegmentLen=%d want 4", rule.SegmentLen)
	}
	if got := maxBetUnitsForPlay(rule); got != 36000 {
		t.Fatalf("max=%d want 36000 (9000×4)", got)
	}
	// 10×10×10×9 ×4 = 36000，贴边不超限
	ok := "0,1,2,3,4,5,6,7,8,9\n0,1,2,3,4,5,6,7,8,9\n0,1,2,3,4,5,6,7,8,9\n0,1,2,3,4,5,6,7,8"
	if got := countPlayWireBetUnits(rule, ok); got != 36000 {
		t.Fatalf("wire units=%d want 36000", got)
	}
	if contentExceedsBetUnitsMax(rule, ok) {
		t.Fatal("36000 should not exceed sixing zuhe max")
	}
	// 三星组合仍为 2700
	q3 := resolveSSCPlayRule("qian3", "qian3_zuhe", "zuhe", "前三组合")
	if got := maxBetUnitsForPlay(q3); got != 2700 {
		t.Fatalf("qian3 zuhe max=%d want 2700", got)
	}
}
