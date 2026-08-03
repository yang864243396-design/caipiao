package schemes

import "testing"

func TestHezhiZuxuanWireUnitsUnder900(t *testing.T) {
	t.Parallel()
	// 方案只存 subPlayId=262、无 playMethodLabel 时也须识别组选和值，
	// 否则 1–26 按直选计 998 注误触「超过最大投注注数:900」。
	raw := []byte(`{
		"runTypeId":"fixed_rotate","playTemplate":"ssc_std",
		"playTypeId":"g002","subPlayId":"262","betMode":"hezhi",
		"schemeGroups":["1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18,19,20,21,22,23,24,25,26"]
	}`)
	cfg := parseSchemeConfig("custom", raw, 0, 0)
	if !cfg.Play.HezhiZuxuan {
		t.Fatalf("HezhiZuxuan=false, rule=%+v", cfg.Play)
	}
	content := cfg.Groups[0]
	units := countPlayWireBetUnits(cfg.Play, content)
	if units != 210 {
		t.Fatalf("wire units=%d want 210 (组选), HezhiZuxuan=%v", units, cfg.Play.HezhiZuxuan)
	}
	if max := maxBetUnitsForPlay(cfg.Play); max > 0 && units > max {
		t.Fatalf("units %d > max %d（不应误按直选 998）", units, max)
	}
}
