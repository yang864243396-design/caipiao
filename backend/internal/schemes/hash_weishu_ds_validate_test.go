package schemes

import "testing"

func TestValidateGroupContent_hashWeishuDsOnePick(t *testing.T) {
	rule := resolveSSCPlayRule("g017", "387", "danshuang", "尾数单双")
	if !isHashWeishuDxdsRule(rule) {
		t.Fatal("387 应为哈希尾数单双")
	}
	if isWuxingSumDxdsRule(rule) {
		t.Fatal("尾数单双不应走五星和值判定")
	}
	if !isSingleTokenDxdsRule(rule) {
		t.Fatal("应为单 token 单双")
	}
	if err := validateGroupContent(rule, "单"); err != nil {
		t.Fatalf("合法单选应通过: %v", err)
	}
	if err := validateGroupContent(rule, "单,双"); err == nil {
		t.Fatal("多选应拒绝")
	}
	// 尾数按位：个位 5 → 单
	ev := evaluateDxds(rule, []string{"1", "2", "3", "4", "5"}, "单")
	if !ev.Hit || ev.BetUnits != 1 {
		t.Fatalf("hit=%v units=%d want hit units=1", ev.Hit, ev.BetUnits)
	}
	if max := randomDrawCountMax(rule); max != 1 {
		t.Fatalf("randomDrawCountMax=%d want 1", max)
	}
}

func TestValidateGroupContent_hashWeishuDxOnePick(t *testing.T) {
	rule := resolveSSCPlayRule("g017", "390", "daxiao", "尾数大小")
	if !isHashWeishuDxdsRule(rule) {
		t.Fatal("390 应为哈希尾数大小")
	}
	if err := validateGroupContent(rule, "大"); err != nil {
		t.Fatalf("合法单选应通过: %v", err)
	}
	if err := validateGroupContent(rule, "大,小"); err == nil {
		t.Fatal("多选应拒绝")
	}
}

func TestHashWeishuIdsNotConfusedWithSSC(t *testing.T) {
	// SSC 下 270 是前二大小单双，不是哈希尾数
	ssc := resolveSSCPlayRule("g016", "270", "dxds", "前二大小单双")
	if isHashWeishuDxdsRule(ssc) {
		t.Fatal("SSC 270 不应识别为哈希尾数")
	}
}
