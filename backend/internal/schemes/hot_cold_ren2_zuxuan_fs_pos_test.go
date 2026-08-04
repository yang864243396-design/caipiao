package schemes

import "testing"

// 任二组选复式冷热：号码池频次须随投注选位变化（万千 vs 千个）。
func TestHotColdWarmTiersOverall_ren2ZuxuanFsFollowsPositionIdxs(t *testing.T) {
	cfg := pickTestConfig(t, `{
		"runTypeId":"hot_cold_warm",
		"playTemplate":"ssc_std",
		"playTypeId":"g011",
		"subPlayId":"77",
		"betMode":"zuxuan_fs",
		"playMethodLabel":"任二组选复式",
		"playTypeLabel":"任选",
		"guajiGroup":"任选"
	}`)
	rule := cfg.Play
	if !isRenxuanNeedsPositionRule(rule) {
		t.Fatalf("want renxuan needs position, rule=%+v", rule)
	}
	if !isHotColdDigitOverall(rule) {
		t.Fatalf("want digit overall, rule=%+v", rule)
	}
	pool := []string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9"}
	// 万位偏 9；个位偏 1；千位偏 3 → 万千与千个热序不同
	draws := [][]string{
		{"9", "3", "5", "5", "1"},
		{"9", "3", "5", "5", "1"},
		{"9", "3", "5", "5", "1"},
		{"9", "3", "5", "5", "1"},
		{"2", "3", "5", "5", "1"},
		{"2", "3", "5", "5", "1"},
	}
	wanQianHot, wanQianCold := hotColdWarmTiersOverallForPositions(draws, rule, pool, []int{0, 1})
	qianGeHot, qianGeCold := hotColdWarmTiersOverallForPositions(draws, rule, pool, []int{1, 4})
	if len(wanQianHot) == 0 || len(qianGeHot) == 0 {
		t.Fatalf("empty hot: wanqian=%v qiange=%v", wanQianHot, qianGeHot)
	}
	fullWQ := append(append([]string{}, wanQianHot...), wanQianCold...)
	fullQG := append(append([]string{}, qianGeHot...), qianGeCold...)
	if equalStringSlices(fullWQ, fullQG) {
		t.Fatalf("不同投注选位频次序应不同: wanqian=%v qiange=%v", fullWQ, fullQG)
	}
	// 万千：9 出现 4 次应进热区；千个：1 出现 6 次应进热区且常高于 9
	if !containsStr(wanQianHot, "9") {
		t.Fatalf("万千热区应含 9, hot=%v", wanQianHot)
	}
	if !containsStr(qianGeHot, "1") {
		t.Fatalf("千个热区应含 1, hot=%v", qianGeHot)
	}
}

func containsStr(ss []string, want string) bool {
	for _, s := range ss {
		if s == want {
			return true
		}
	}
	return false
}

func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
