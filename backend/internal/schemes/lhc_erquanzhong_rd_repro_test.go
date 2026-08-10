package schemes

import (
	"strings"
	"testing"
)

// def-1-1786106979838：随机二全中复式连续 10 期「超过注数上限」——
// 根因是 LHC SegmentLen 误为 7，随机出成 7 行后被直选复式校验打成无解。
func TestLHCErquanzhongRandomDraw_underMax(t *testing.T) {
	cfgJSON := `{
		"subId":"279","typeId":"g003","betMode":"fushi","runTypeId":"random_draw",
		"subPlayId":"279","playTypeId":"g003","playTemplate":"lhc_std",
		"randomDraw":{"counts":[4],"strategy":"every"},
		"schemeGroups":["05,18,22,34"]
	}`
	cfg := pickTestConfig(t, cfgJSON)
	if cfg.Play.SegmentLen != 1 {
		t.Fatalf("SegmentLen=%d want 1 (LHC 单区号池)", cfg.Play.SegmentLen)
	}
	if cfg.Play.PlayTemplate != "lhc_std" || cfg.Play.BetMode != "fushi" {
		t.Fatalf("play=%+v", cfg.Play)
	}

	for i := 0; i < 30; i++ {
		got := randomDrawContentUnderMax(cfg)
		if strings.TrimSpace(got) == "" {
			t.Fatalf("empty undermax at i=%d", i)
		}
		if strings.Contains(got, "\n") {
			t.Fatalf("unexpected multiline pick %q", got)
		}
		if !randomDrawContentAcceptable(cfg.Play, got) {
			t.Fatalf("unacceptable %q err=%v", got, validateGroupContent(cfg.Play, got))
		}
		if contentExceedsBetUnitsMax(cfg.Play, got) {
			t.Fatalf("over max units: %q units=%d", got, countPlayWireBetUnits(cfg.Play, got))
		}
		units := countPlayWireBetUnits(cfg.Play, got)
		// 4 码二全中复式 → C(4,2)=6
		if units != 6 {
			t.Fatalf("units=%d want 6 content=%q", units, got)
		}
	}
}

func TestResolveLHCPlayRule_keepsCatalogTypeID(t *testing.T) {
	rule := resolveLHCPlayRule("g003", "280", "tuotou")
	if rule.SegmentLen != 1 {
		t.Fatalf("SegmentLen=%d want 1", rule.SegmentLen)
	}
	// 查 rule_id 必须用目录 type g003，不能改成 erquanzhong
	if rule.PlayTypeID != "g003" {
		t.Fatalf("PlayTypeID=%q want g003", rule.PlayTypeID)
	}
	if rule.BetMode != "tuotou" || rule.CatalogSubID != "280" {
		t.Fatalf("betMode=%q catalogSub=%q", rule.BetMode, rule.CatalogSubID)
	}
	if got := lhcSettlePlayType(rule); got != "erquanzhong" {
		t.Fatalf("settleType=%q want erquanzhong", got)
	}
}
