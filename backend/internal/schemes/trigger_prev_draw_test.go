package schemes

import "testing"

func TestPickPreviousIssueNo(t *testing.T) {
	t.Parallel()
	// 模拟「当期=308、列表缺 307」时，应取最大的 <308，而不是任意第一条。
	candidates := []string{"1014104300306", "1014104300305", "1014104300308", "1014104300310"}
	current := "1014104300308"
	best := ""
	for _, iss := range candidates {
		if iss == current || iss >= current {
			continue
		}
		if best == "" || iss > best {
			best = iss
		}
	}
	if best != "1014104300306" {
		t.Fatalf("best=%q want 1014104300306", best)
	}
	// 有 307 时应取 307
	candidates = append(candidates, "1014104300307")
	best = ""
	for _, iss := range candidates {
		if iss == current || iss >= current {
			continue
		}
		if best == "" || iss > best {
			best = iss
		}
	}
	if best != "1014104300307" {
		t.Fatalf("best=%q want 1014104300307", best)
	}
}

func TestResolveSSCPlayRuleG006DingweiPosition(t *testing.T) {
	t.Parallel()
	rule := resolveSSCPlayRule("g006", "13", "dingwei", "一星定位胆")
	if rule.PositionIdx != 0 {
		t.Fatalf("PositionIdx=%d want 0 (万位默认)", rule.PositionIdx)
	}
	if rule.BetMode != "dingwei" {
		t.Fatalf("BetMode=%q want dingwei", rule.BetMode)
	}
	if playPositionCount(rule) != 5 {
		t.Fatalf("g006/13 five-position panel: playPositionCount=%d want 5 (SegmentPos=%v)", playPositionCount(rule), rule.SegmentPos)
	}
	rule = resolveSSCPlayRule("g006", "sub_ge", "dingwei", "个位")
	if rule.PositionIdx != 4 {
		t.Fatalf("PositionIdx=%d want 4", rule.PositionIdx)
	}
	if playPositionCount(rule) != 1 {
		t.Fatalf("locked 个位: playPositionCount=%d want 1", playPositionCount(rule))
	}
}
