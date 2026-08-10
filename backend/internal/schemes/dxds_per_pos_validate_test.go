package schemes

import "testing"

func TestValidateGroupContent_hou2DxdsPerPosOnePick(t *testing.T) {
	rule := resolveSSCPlayRule("g016", "266", "dxds", "后二大小单双")
	if rule.SegmentLen != 2 {
		t.Fatalf("segLen=%d want 2", rule.SegmentLen)
	}
	if err := validateGroupContent(rule, "大\n小"); err != nil {
		t.Fatalf("合法内容应通过: %v", err)
	}
	if err := validateGroupContent(rule, "大,小\n单"); err == nil {
		t.Fatal("十位多选应拒绝")
	}
	if err := validateGroupContent(rule, "大\n"); err == nil {
		t.Fatal("缺位应拒绝")
	}
}

func TestEvaluateDxds_hou2OneUnit(t *testing.T) {
	rule := resolveSSCPlayRule("g016", "266", "dxds", "后二大小单双")
	// 开奖 …大(6+),小(3-) → 命中 大\n小
	ev := evaluateDxds(rule, []string{"1", "2", "4", "6", "3"}, "大\n小")
	if !ev.Hit || ev.BetUnits != 1 {
		t.Fatalf("hit=%v units=%d want hit=true units=1", ev.Hit, ev.BetUnits)
	}
	// 行内多选时按首个计 1 注
	ev2 := evaluateDxds(rule, []string{"1", "2", "4", "6", "3"}, "大,小\n小,单")
	if !ev2.Hit || ev2.BetUnits != 1 {
		t.Fatalf("multi hit=%v units=%d want hit=true units=1", ev2.Hit, ev2.BetUnits)
	}
}
