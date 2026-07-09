package schemes

import (
	"encoding/json"
	"testing"
)

func TestEvaluateLHCTema(t *testing.T) {
	rule := resolveLHCPlayRule("tema", "tema_a", "tema")
	balls := []string{"3", "12", "25", "33", "41", "7", "49"}
	ev := evaluatePlayHit(rule, balls, "49,01", false, "", 0)
	if !ev.Hit {
		t.Fatalf("want tema hit, ev=%+v", ev)
	}
}

func TestEvaluateLHCErquanzhong(t *testing.T) {
	rule := resolveLHCPlayRule("erquanzhong", "fushi", "fushi")
	balls := []string{"3", "12", "25", "33", "41", "7", "49"}
	ev := evaluatePlayHit(rule, balls, "3,12,49", false, "", 0)
	if !ev.Hit {
		t.Fatalf("want erquanzhong hit, ev=%+v", ev)
	}
}

func TestEvaluateLHCBuzhong5(t *testing.T) {
	rule := resolveLHCPlayRule("buzhong_xuanyi", "5bz", "buzhong")
	balls := []string{"3", "12", "25", "33", "41", "7", "49"}
	ev := evaluatePlayHit(rule, balls, "1,2,4,5,6", false, "", 0)
	if !ev.Hit {
		t.Fatalf("want 5bz hit, ev=%+v", ev)
	}
}

func TestEvaluateLHCTexiao(t *testing.T) {
	rule := resolveLHCPlayRule("shengxiao", "texiao", "texiao")
	balls := []string{"3", "12", "25", "33", "41", "7", "49"}
	z := lhcZodiacOf(49)
	ev := evaluatePlayHit(rule, balls, z, false, "", 0)
	if !ev.Hit {
		t.Fatalf("want texiao hit for %s, ev=%+v", z, ev)
	}
}

func TestEvaluateLHCRenzhong(t *testing.T) {
	rule := resolveLHCPlayRule("renzhong", "2l_rz", "renzhong")
	balls := []string{"3", "12", "25", "33", "41", "7", "49"}
	ev := evaluatePlayHit(rule, balls, "49,01", false, "", 0)
	if !ev.Hit {
		t.Fatalf("want renzhong hit, ev=%+v", ev)
	}
}

func TestEvaluateLHCQima(t *testing.T) {
	rule := resolveLHCPlayRule("qima", "qima", "qima")
	// 3,12,25,33,41,7,49 → 单6 双1 大4 小3
	balls := []string{"3", "12", "25", "33", "41", "7", "49"}

	for _, tc := range []struct {
		content string
		hit     bool
		odds    float64
	}{
		{"双1", true, 19.497},
		{"单6", true, 19.497},
		{"大4", true, 3.239},
		{"小3", true, 3.239},
		{"双0", false, oddsLHCAttr},
		{"双2", false, oddsLHCAttr},
		{"双1,大9", true, 19.497}, // 大9 无效，双1 仍中
	} {
		ev := evaluatePlayHit(rule, balls, tc.content, false, "", 0)
		if ev.Hit != tc.hit {
			t.Fatalf("content=%q hit=%v want %v ev=%+v", tc.content, ev.Hit, tc.hit, ev)
		}
		if tc.hit && ev.Odds != tc.odds {
			t.Fatalf("content=%q odds=%v want %v", tc.content, ev.Odds, tc.odds)
		}
	}
}

func TestLHCQimaCategoryCount(t *testing.T) {
	nums := lhcAllNumbers([]string{"3", "12", "25", "33", "41", "7", "49"})
	if c := lhcQimaCategoryCount(nums, "双"); c != 1 {
		t.Fatalf("双=%d want 1", c)
	}
	if c := lhcQimaCategoryCount(nums, "单"); c != 6 {
		t.Fatalf("单=%d want 6", c)
	}
}

func TestParseLHCQimaPicks(t *testing.T) {
	picks := parseLHCQimaPicks("双1,单0,invalid")
	if len(picks) != 2 || picks[0].kind != "双" || picks[0].count != 1 {
		t.Fatalf("picks=%+v", picks)
	}
}

func TestEvaluateLHCZongxiao(t *testing.T) {
	rule := resolveLHCPlayRule("shengxiao", "zongxiao", "zongxiao")
	// 7 个号覆盖 7 个不同生肖 → 总肖=7
	balls := []string{"1", "2", "3", "4", "5", "6", "7"}
	ev := evaluatePlayHit(rule, balls, "七肖", false, "", 0)
	if !ev.Hit || ev.Odds != 5.335 {
		t.Fatalf("want 七肖 hit, ev=%+v", ev)
	}
	ev = evaluatePlayHit(rule, balls, "二肖", false, "", 0)
	if ev.Hit {
		t.Fatalf("want 二肖 miss, ev=%+v", ev)
	}
	ev = evaluatePlayHit(rule, balls, "0", false, "", 0)
	if ev.Hit {
		t.Fatalf("invalid 0 should not hit")
	}
}

func TestParseSchemeConfigLHC(t *testing.T) {
	raw, _ := json.Marshal(map[string]interface{}{
		"playTypeId":   "tema",
		"subPlayId":    "tema_a",
		"schemeGroups": []string{"01,13,49"},
	})
	cfg := parseSchemeConfig("custom", raw, 0, 0)
	if cfg.Play.PlayTemplate != "lhc_std" || cfg.Play.BetMode != "tema" {
		t.Fatalf("play=%+v", cfg.Play)
	}
	if cfg.GroupContent != "01,13,49" {
		t.Fatalf("content=%q", cfg.GroupContent)
	}
}

func TestInferLHCBetModeSubIDs(t *testing.T) {
	cases := []struct{ typeID, subID, want string }{
		{"buzhong_xuanyi", "5bz", "buzhong"},
		{"buzhong_xuanyi", "8x1", "xuanyi"},
		{"shengxiao", "2xiao_z", "xiao_z"},
		{"renzhong", "3l_rz", "renzhong"},
	}
	for _, c := range cases {
		if got := inferLHCBetMode(c.typeID, c.subID); got != c.want {
			t.Fatalf("%s/%s got %s want %s", c.typeID, c.subID, got, c.want)
		}
	}
}

func TestSynthLHCDrawBalls(t *testing.T) {
	a := synthLHCDrawBalls("tron_lhc_1m", "20260608001")
	b := synthLHCDrawBalls("tron_lhc_1m", "20260608001")
	if len(a) != 7 || len(b) != 7 {
		t.Fatalf("want 7 balls, got %d %d", len(a), len(b))
	}
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("not deterministic: %v vs %v", a, b)
		}
	}
}
