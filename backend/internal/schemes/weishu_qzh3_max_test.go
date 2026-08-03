package schemes

import "testing"

// def-1-1785644233119：前中后三和值尾数 1–9 = 27 注，上限须为 27（勿拦成 9）。
func TestWeishuQzh3MaxUnits_def1785644233119(t *testing.T) {
	cfg := pickTestConfig(t, `{
		"runTypeId":"fixed_rotate","playTemplate":"ssc_std",
		"playTypeId":"g007","subPlayId":"111","typeId":"g007","betMode":"weishu",
		"schemeGroups":["1","1,2,3,4,5,6,7,8,9"]
	}`)
	content := "1,2,3,4,5,6,7,8,9"
	z := playZoneMultiplier(cfg.Play)
	max := maxBetUnitsForPlay(cfg.Play)
	n := countPlayWireBetUnits(cfg.Play, content)
	t.Logf("PlayTypeID=%q zone=%d max=%d units=%d", cfg.Play.PlayTypeID, z, max, n)
	if z != 3 {
		t.Fatalf("zone=%d want 3 (PlayTypeID=%q)", z, cfg.Play.PlayTypeID)
	}
	if max != 27 {
		t.Fatalf("max=%d want 27", max)
	}
	if n != 27 {
		t.Fatalf("units=%d want 27", n)
	}
	if contentExceedsBetUnitsMax(cfg.Play, content) {
		t.Fatalf("should not exceed: units=%d max=%d", n, max)
	}
}
