package schemes

import (
	"testing"
)

func TestEvaluateZuxuanFushi_qian2InsufficientPool(t *testing.T) {
	cfg := pickTestConfig(t, `{"runTypeId":"random_draw","playTemplate":"ssc_std","playTypeId":"g004","subPlayId":"42","betMode":"zuxuan_fs","randomDraw":{"counts":[2]}}`)
	ev := evaluateZuxuanFushi(cfg.Play, []string{"1", "2", "3", "4", "5"}, "5")
	if ev.BetUnits != 0 {
		t.Fatalf("单码 BetUnits=%d want 0", ev.BetUnits)
	}
	ev2 := evaluateZuxuanFushi(cfg.Play, []string{"1", "2", "3", "4", "5"}, "1,2")
	if ev2.BetUnits != 1 {
		t.Fatalf("两码 BetUnits=%d want 1 (C(2,2))", ev2.BetUnits)
	}
	ev3 := evaluateZuxuanFushi(cfg.Play, []string{"1", "2", "3", "4", "5"}, "6,8,0")
	if ev3.BetUnits != 3 {
		t.Fatalf("三码 BetUnits=%d want 3", ev3.BetUnits)
	}
	if n := countPlayWireBetUnits(cfg.Play, "1,2"); n != 1 {
		t.Fatalf("wire units=%d want 1", n)
	}
}
