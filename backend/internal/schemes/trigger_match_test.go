package schemes

import "testing"

func TestSupportsAdvTriggerBetPC28(t *testing.T) {
	cases := []struct {
		typeID, subID, group, label string
		want                        bool
	}{
		{"pc28_20", "hezhi", "", "", true},
		{"pc28_28", "dxds", "", "", true},
		{"pc28_20", "longhubao", "", "", true},
		{"pc28_20", "teshu", "", "", false},
		{"dingwei", "dingwei_ge", "", "", true},
		{"longhu", "lh_1v10", "", "", true},
		{"hou3", "hou3_zhixuan_fs", "", "", false},
		{"g006", "13", "一星", "定位胆", true},
		{"g010", "233", "2.0模式", "和值", true},
		{"g010", "999", "2.0模式", "特殊号", false},
	}
	for _, c := range cases {
		if got := SupportsAdvTriggerBet(c.typeID, c.subID, c.group, c.label); got != c.want {
			t.Errorf("SupportsAdvTriggerBet(%q,%q,%q,%q)=%v want %v", c.typeID, c.subID, c.group, c.label, got, c.want)
		}
	}
}

func TestTriggerOpenMatchesPC28(t *testing.T) {
	rule := playRule{PlayTemplate: "pc28_std", BetMode: "hezhi"}
	balls := []string{"3", "5", "7"} // sum=15
	if !triggerOpenMatches(rule, balls, "15") {
		t.Fatal("hezhi open 15 should match")
	}
	if triggerOpenMatches(rule, balls, "14") {
		t.Fatal("hezhi open 14 should not match")
	}

	rule.BetMode = "dxds"
	if !triggerOpenMatches(rule, balls, "大") || !triggerOpenMatches(rule, balls, "单") {
		t.Fatal("dxds 大/单 should match sum=15")
	}
	if triggerOpenMatches(rule, balls, "小") || triggerOpenMatches(rule, balls, "双") {
		t.Fatal("dxds 小/双 should not match sum=15")
	}

	rule.BetMode = "longhubao"
	balls = []string{"9", "2", "1"}
	if !triggerOpenMatches(rule, balls, "龙") {
		t.Fatal("9 vs 1 should match 龙")
	}
	if triggerOpenMatches(rule, balls, "虎") {
		t.Fatal("9 vs 1 should not match 虎")
	}
}

func TestIsLonghuPlayExcludesLonghubao(t *testing.T) {
	if isLonghuPlay(playRule{BetMode: "longhubao"}) {
		t.Fatal("longhubao must not be treated as longhu")
	}
	if !isLonghuPlay(playRule{BetMode: "longhu"}) {
		t.Fatal("longhu bet mode should match")
	}
}
