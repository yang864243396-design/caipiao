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
