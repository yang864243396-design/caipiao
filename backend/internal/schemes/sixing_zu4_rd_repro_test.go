package schemes

import (
	"strings"
	"testing"
)

func TestEvaluateZu4_dualZoneUnitsAndHit(t *testing.T) {
	t.Parallel()
	raw := []byte(`{
		"subId":"133","typeId":"g013","betMode":"zu4","betUnit":"2",
		"runTypeId":"random_draw","subPlayId":"133","playTypeId":"g013",
		"randomDraw":{"counts":[1,2],"strategy":"every"},
		"schemeGroups":["8,6"],"playTemplate":"ssc_std"
	}`)
	cfg := parseSchemeConfig("custom", raw, 0, 0)
	if !isZu4PlayRule(cfg.Play) {
		t.Fatalf("want zu4, got %+v", cfg.Play)
	}
	// 8,6 → 1 注；勿落到直选复式计 0 注
	ev := evaluatePlayHit(cfg.Play, []string{"1", "2", "3", "4", "5"}, "8,6", false, "", 0)
	if ev.BetUnits != 1 {
		t.Fatalf("betUnits=%d want 1", ev.BetUnits)
	}
	// AAAB：8886 且三重落 8、单号落 6 → 中
	evHit := evaluatePlayHit(cfg.Play, []string{"8", "8", "8", "6", "0"}, "8,6", false, "", 0)
	if !evHit.Hit || evHit.BetUnits != 1 {
		t.Fatalf("AAAB hit want hit=true units=1, got hit=%v units=%d", evHit.Hit, evHit.BetUnits)
	}
	// 形态对但区反（三重在单号区）→ 不中
	evMissZone := evaluatePlayHit(cfg.Play, []string{"6", "6", "6", "8", "0"}, "8,6", false, "", 0)
	if evMissZone.Hit || evMissZone.BetUnits != 1 {
		t.Fatalf("zone miss want hit=false units=1, got hit=%v units=%d", evMissZone.Hit, evMissZone.BetUnits)
	}
	// 非 AAAB → 不中但仍有注
	evMiss := evaluatePlayHit(cfg.Play, []string{"1", "2", "3", "4", "5"}, "8,6", false, "", 0)
	if evMiss.Hit || evMiss.BetUnits != 1 {
		t.Fatalf("miss want hit=false units=1, got hit=%v units=%d", evMiss.Hit, evMiss.BetUnits)
	}
	// 12,34 → 4 注
	ev4 := evaluatePlayHit(cfg.Play, []string{"1", "2", "3", "4", "5"}, "12,34", false, "", 0)
	if ev4.BetUnits != 4 {
		t.Fatalf("12,34 units=%d want 4", ev4.BetUnits)
	}
}

func TestRandomDraw_sixingZu4CountsUnderMax(t *testing.T) {
	t.Parallel()
	raw := []byte(`{
		"subId":"133","typeId":"g013","betMode":"zu4","betUnit":"2",
		"runTypeId":"random_draw","subPlayId":"133","playTypeId":"g013",
		"randomDraw":{"counts":[1,2],"strategy":"every"},
		"schemeGroups":["8,6"],"playTemplate":"ssc_std"
	}`)
	cfg := parseSchemeConfig("custom", raw, 0, 0)
	for i := 0; i < 30; i++ {
		got := randomDrawContentUnderMax(cfg)
		if strings.TrimSpace(got) == "" {
			t.Fatalf("empty random content at i=%d", i)
		}
		if !randomDrawContentAcceptable(cfg.Play, got) {
			t.Fatalf("unacceptable: %q", got)
		}
		wire := countPlayWireBetUnits(cfg.Play, got)
		ev := evaluatePlayHit(cfg.Play, []string{"1", "2", "3", "4", "5"}, got, false, "", 0)
		if ev.BetUnits <= 0 {
			t.Fatalf("evalUnits=0 content=%q wire=%d（组选4 须注册 evaluateZu4 双区计注）", got, wire)
		}
		if ev.BetUnits != wire {
			t.Fatalf("eval=%d wire=%d content=%q", ev.BetUnits, wire, got)
		}
		if contentExceedsBetUnitsMax(cfg.Play, got) {
			t.Fatalf("over max content=%q units=%d", got, ev.BetUnits)
		}
	}
}

func TestEvaluateZu12_dualZoneUnitsAndHit(t *testing.T) {
	t.Parallel()
	raw := []byte(`{
		"subId":"131","typeId":"g013","betMode":"zu12","betUnit":"2",
		"subPlayId":"131","playTypeId":"g013","playTemplate":"ssc_std"
	}`)
	cfg := parseSchemeConfig("custom", raw, 0, 0)
	// 8,67 → C(2,2)=1 注（勿按扁选 3 码计）
	ev := evaluatePlayHit(cfg.Play, []string{"1", "2", "3", "4", "5"}, "8,67", false, "", 0)
	if ev.BetUnits != 1 {
		t.Fatalf("betUnits=%d want 1", ev.BetUnits)
	}
	// AABC：8867 → 中
	evHit := evaluatePlayHit(cfg.Play, []string{"8", "8", "6", "7", "0"}, "8,67", false, "", 0)
	if !evHit.Hit || evHit.BetUnits != 1 {
		t.Fatalf("AABC hit want hit=true units=1, got hit=%v units=%d", evHit.Hit, evHit.BetUnits)
	}
	// 二重不在二重区 → 不中
	evMiss := evaluatePlayHit(cfg.Play, []string{"6", "6", "8", "7", "0"}, "8,67", false, "", 0)
	if evMiss.Hit || evMiss.BetUnits != 1 {
		t.Fatalf("zone miss want hit=false units=1, got hit=%v units=%d", evMiss.Hit, evMiss.BetUnits)
	}
}
