package schemes

import (
	"strings"
	"testing"
)

// def-1-1785720432375：定码任二直选和值，满选 0–18=100 注，勿套前二上限 90。
func TestDef1785720432375_ren2HezhiMax100(t *testing.T) {
	t.Parallel()
	raw := `{
		"subId":"76","typeId":"g011","betMode":"hezhi","runTypeId":"fixed_rotate",
		"subPlayId":"76","playTypeId":"g011","schemeName":"定码任二直选和值",
		"playTemplate":"ssc_std",
		"schemeGroups":["千,个\n12","千,个\n0,1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18"]
	}`
	cfg := parseSchemeConfig("custom", []byte(raw), 0, 1)
	content := "千,个\n0,1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18"
	if got := strings.TrimSpace(cfg.GroupContent); got != "" {
		content = got
	}
	if !isRen2ZhixuanHezhiRule(cfg.Play) {
		t.Fatalf("want ren2 zhixuan hezhi, rule=%+v", cfg.Play)
	}
	if max := maxBetUnitsForPlay(cfg.Play); max != 100 {
		t.Fatalf("max=%d want 100; rule=%+v", max, cfg.Play)
	}
	units := countPlayWireBetUnits(cfg.Play, content)
	if units != 100 {
		t.Fatalf("units=%d want 100 content=%q", units, content)
	}
}
