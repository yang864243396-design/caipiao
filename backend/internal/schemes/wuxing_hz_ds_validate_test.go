package schemes

import "testing"

func TestValidateGroupContent_wuxingHzDsOnePick(t *testing.T) {
	rule := resolveSSCPlayRule("g016", "268", "danshuang", "五星和值单双")
	if !isWuxingSumDxdsRule(rule) {
		t.Fatal("268 应为五星和值单双")
	}
	if err := validateGroupContent(rule, "单"); err != nil {
		t.Fatalf("合法单选应通过: %v", err)
	}
	if err := validateGroupContent(rule, "单,双"); err == nil {
		t.Fatal("多选应拒绝")
	}
	ev := evaluateDxds(rule, []string{"1", "2", "3", "4", "5"}, "单") // sum=15 小且单
	if !ev.Hit || ev.BetUnits != 1 {
		t.Fatalf("hit=%v units=%d want hit units=1", ev.Hit, ev.BetUnits)
	}
}
