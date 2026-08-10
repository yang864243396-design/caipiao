package schemes

import (
	"strings"
	"testing"
)

func TestEvaluateZu6_sixingPoolC2(t *testing.T) {
	t.Parallel()
	raw := []byte(`{
		"subId":"132","typeId":"g013","betMode":"zu6","betUnit":"2",
		"runTypeId":"random_draw","subPlayId":"132","playTypeId":"g013",
		"randomDraw":{"counts":[2],"strategy":"every"},
		"schemeGroups":["3,8"],"playTemplate":"ssc_std"
	}`)
	cfg := parseSchemeConfig("custom", raw, 0, 0)
	if !isSixingZu6PlayRule(cfg.Play) {
		t.Fatalf("want sixing zu6, got %+v", cfg.Play)
	}
	// 2 码 → C(2,2)=1 注（勿套三星组六 C(n,3)=0）
	ev := evaluateZu6(cfg.Play, []string{"1", "2", "3", "4", "5"}, "3,8")
	if ev.BetUnits != 1 {
		t.Fatalf("betUnits=%d want 1", ev.BetUnits)
	}
	// AABB 且两码均在池中 → 中
	evHit := evaluateZu6(cfg.Play, []string{"3", "3", "8", "8", "0"}, "3,8")
	if !evHit.Hit || evHit.BetUnits != 1 {
		t.Fatalf("AABB hit want hit=true units=1, got hit=%v units=%d", evHit.Hit, evHit.BetUnits)
	}
	// 开出非两对 → 不中但仍有 1 注
	evMiss := evaluateZu6(cfg.Play, []string{"1", "2", "3", "4", "5"}, "3,8")
	if evMiss.Hit || evMiss.BetUnits != 1 {
		t.Fatalf("miss want hit=false units=1, got hit=%v units=%d", evMiss.Hit, evMiss.BetUnits)
	}
}

func TestRandomDraw_sixingZu6Counts2UnderMax(t *testing.T) {
	t.Parallel()
	raw := []byte(`{
		"subId":"132","typeId":"g013","betMode":"zu6","betUnit":"2",
		"runTypeId":"random_draw","subPlayId":"132","playTypeId":"g013",
		"randomDraw":{"counts":[2],"strategy":"every"},
		"schemeGroups":["3,8"],"playTemplate":"ssc_std"
	}`)
	cfg := parseSchemeConfig("custom", raw, 0, 0)
	okN := 0
	for i := 0; i < 30; i++ {
		got := randomDrawContentUnderMax(cfg)
		if strings.TrimSpace(got) == "" {
			t.Fatalf("empty random content at i=%d", i)
		}
		if !randomDrawContentAcceptable(cfg.Play, got) {
			t.Fatalf("unacceptable: %q", got)
		}
		ev := evaluatePlayHit(cfg.Play, []string{"1", "2", "3", "4", "5"}, got, false, "", 0)
		if ev.BetUnits <= 0 {
			t.Fatalf("evalUnits=0 content=%q (四星组选6 不可再按三星 C(n,3) 计 0 注)", got)
		}
		if contentExceedsBetUnitsMax(cfg.Play, got) {
			t.Fatalf("over max content=%q units=%d", got, ev.BetUnits)
		}
		okN++
	}
	if okN != 30 {
		t.Fatalf("ok=%d want 30", okN)
	}
}
