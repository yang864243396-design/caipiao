package schemes

import "testing"

func TestEvaluateLonghuWanQian(t *testing.T) {
	rule := resolveSSCPlayRule("longhu", "lh_wanqian_dou", "longhu")
	ev := evaluatePlayHit(rule, []string{"8", "3", "1", "2", "9"}, "龙", false, "", 0)
	if !ev.Hit {
		t.Fatalf("want hit for 龙 when wan>qian, ev=%+v", ev)
	}
}

func TestEvaluateHezhiQian3(t *testing.T) {
	rule := resolveSSCPlayRule("qian3", "qian3_zhixuan_hz", "hezhi")
	// 1+2+3=6
	ev := evaluatePlayHit(rule, []string{"1", "2", "3", "0", "0"}, "6", false, "", 0)
	if !ev.Hit {
		t.Fatalf("want hezhi hit, ev=%+v", ev)
	}
}

func TestEvaluateRenxuan2(t *testing.T) {
	rule := resolveSSCPlayRule("renxuan", "ren2_zhixuan_fs", "fushi")
	content := "1,3\n\n\n\n"
	ev := evaluatePlayHit(rule, []string{"1", "0", "0", "0", "0"}, content, false, "", 0)
	if !ev.Hit {
		t.Fatalf("want ren2 hit on wan, ev=%+v", ev)
	}
}

func TestEvaluateZu3Qian3(t *testing.T) {
	rule := resolveSSCPlayRule("qian3", "qian3_zu3", "zu3")
	// 1,1,2 组三形态
	ev := evaluatePlayHit(rule, []string{"1", "1", "2", "0", "0"}, "1,2,3", false, "", 0)
	if !ev.Hit {
		t.Fatalf("want zu3 hit, ev=%+v", ev)
	}
	ev2 := evaluatePlayHit(rule, []string{"1", "2", "3", "0", "0"}, "1,2,3", false, "", 0)
	if ev2.Hit {
		t.Fatalf("want zu6 miss on zu3 play, ev=%+v", ev2)
	}
}

func TestEvaluateBaodanQian2(t *testing.T) {
	rule := resolveSSCPlayRule("qian2", "qian2_zuxuan_bd", "baodan")
	ev := evaluatePlayHit(rule, []string{"3", "7", "0", "0", "0"}, "3", false, "", 0)
	if !ev.Hit {
		t.Fatalf("want baodan hit, ev=%+v", ev)
	}
}

func TestEvaluateWeishuQian3(t *testing.T) {
	rule := resolveSSCPlayRule("qian3", "qian3_hz_weishu", "weishu")
	// 1+2+3=6, tail 6
	ev := evaluatePlayHit(rule, []string{"1", "2", "3", "0", "0"}, "6", false, "", 0)
	if !ev.Hit {
		t.Fatalf("want weishu hit, ev=%+v", ev)
	}
}

func TestEvaluateZuheQian3(t *testing.T) {
	rule := resolveSSCPlayRule("qian3", "qian3_zuhe", "zuhe")
	ev := evaluatePlayHit(rule, []string{"1", "2", "5", "0", "0"}, "12", false, "", 0)
	if !ev.Hit {
		t.Fatalf("want zuhe hit, ev=%+v", ev)
	}
}

func TestEvaluateTeshuWuxingYifan(t *testing.T) {
	rule := resolveSSCPlayRule("wuxing", "wuxing_yifan", "teshu")
	ev := evaluatePlayHit(rule, []string{"8", "8", "8", "8", "8"}, "一帆风顺", false, "", 0)
	if !ev.Hit {
		t.Fatalf("want yifan hit, ev=%+v", ev)
	}
}

func TestEvaluateSixingZu24(t *testing.T) {
	rule := resolveSSCPlayRule("sixing", "sixing_zu24", "zu24")
	ev := evaluatePlayHit(rule, []string{"0", "1", "2", "3", "9"}, "0,1,2,3,4,5,9", false, "", 0)
	if !ev.Hit {
		t.Fatalf("want zu24 hit, ev=%+v", ev)
	}
}
