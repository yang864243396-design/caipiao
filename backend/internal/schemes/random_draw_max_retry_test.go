package schemes

import (
	"errors"
	"testing"
)

func TestNormalizeResolvedBetContent_randomDrawIgnoresGroupFallback(t *testing.T) {
	t.Parallel()
	// schemeGroups 满选 0–18=100 注；空出号不得回落满选，应重抽到 ≤90
	cfg := pickTestConfig(t, `{
		"runTypeId":"random_draw","betMode":"hezhi","playTypeId":"g004","subPlayId":"40",
		"playTemplate":"ssc_std","playMethod":"前二直选和值",
		"randomDraw":{"counts":[18],"strategy":"every"},
		"schemeGroups":["0,1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18"]
	}`)
	if max := maxBetUnitsForPlay(cfg.Play); max != 90 {
		t.Fatalf("max=%d want 90", max)
	}
	dec := pickDecision{Content: ""}
	got := normalizeResolvedBetContent(cfg, &dec)
	if got == "" {
		t.Fatal("expected redraw content")
	}
	if n := countPlayWireBetUnits(cfg.Play, got); n <= 0 || n > 90 {
		t.Fatalf("redraw wire=%d want 1..90 content=%q", n, got)
	}
	// 超限缓存号也应重抽
	dec2 := pickDecision{Content: "0,1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18"}
	got2 := normalizeResolvedBetContent(cfg, &dec2)
	if contentExceedsBetUnitsMax(cfg.Play, got2) {
		t.Fatalf("still over max: %q wire=%d", got2, countPlayWireBetUnits(cfg.Play, got2))
	}
}

func TestResolveRandomDrawUnderMax_skipsWhenImpossible(t *testing.T) {
	t.Parallel()
	cfg := pickTestConfig(t, `{
		"runTypeId":"random_draw","betMode":"hezhi","playTypeId":"g004","subPlayId":"40",
		"playTemplate":"ssc_std","randomDraw":{"counts":[18]}
	}`)
	okContent := "0,1,2,3,4,5,6,7,8,10,11,12,13,14,15,16,17,18" // 90
	got, ok := resolveRandomDrawUnderMax(cfg, okContent)
	if !ok || got != okContent {
		t.Fatalf("ok content should pass, got=%q ok=%v", got, ok)
	}
	got2, ok2 := resolveRandomDrawUnderMax(cfg, "")
	if !ok2 || contentExceedsBetUnitsMax(cfg.Play, got2) {
		t.Fatalf("empty should redraw under max, got=%q ok=%v", got2, ok2)
	}
}

func TestErrMaxBetUnitsExceeded_isSentinel(t *testing.T) {
	t.Parallel()
	err := errMaxBetUnitsExceeded(90)
	if !errors.Is(err, errBetUnitsExceeded) {
		t.Fatalf("errors.Is failed: %v", err)
	}
	if err.Error() != "投注注数超过最大投注注数:90" {
		t.Fatalf("msg=%q", err.Error())
	}
	if !isBetUnitsExceededError(err) {
		t.Fatal("isBetUnitsExceededError should match wrapped max error")
	}
	apiLike := errors.New("guaji api code=40000: 投注注数超过最大投注注数:90")
	if !isBetUnitsExceededError(apiLike) {
		t.Fatal("isBetUnitsExceededError should match third-party message")
	}
}

func TestEvaluateHezhi_usesWireUnits(t *testing.T) {
	t.Parallel()
	rule := resolveSSCPlayRule("g004", "40", "hezhi", "前二直选和值")
	// 和值 9 = 10 注；未开奖时也要按组合注数算金额
	ev := evaluateHezhi(rule, nil, "9")
	if ev.BetUnits != 10 {
		t.Fatalf("BetUnits=%d want 10 (wire), rule=%+v", ev.BetUnits, rule)
	}
	ev2 := evaluateHezhi(rule, []string{"1", "2", "3", "4", "5"}, "0,1,2,3,4,5,6,7,8,10,11,12,13,14,15,16,17,18")
	if ev2.BetUnits != 90 {
		t.Fatalf("BetUnits=%d want 90", ev2.BetUnits)
	}
}
