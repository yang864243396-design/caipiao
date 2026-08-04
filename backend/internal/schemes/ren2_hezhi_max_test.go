package schemes

import "testing"

func TestMaxBetUnits_ren2ZhixuanHezhiIs900(t *testing.T) {
	t.Parallel()
	raw := `{
		"playTemplate":"ssc_std",
		"playTypeId":"g011",
		"subPlayId":"76",
		"betMode":"hezhi",
		"playMethodLabel":"任二直选和值",
		"playTypeLabel":"任选",
		"guajiGroup":"任选"
	}`
	cfg := parseSchemeConfig("custom", []byte(raw), 0, 0)
	if !isRen2ZhixuanHezhiRule(cfg.Play) {
		t.Fatalf("want ren2 zhixuan hezhi, rule=%+v", cfg.Play)
	}
	if got := maxBetUnitsForPlay(cfg.Play); got != 900 {
		t.Fatalf("max=%d want 900", got)
	}
}

func TestMaxBetUnits_ren2ZuxuanHezhiNotRen2ZhixuanCap(t *testing.T) {
	t.Parallel()
	raw := `{
		"playTemplate":"ssc_std",
		"playTypeId":"g011",
		"subPlayId":"79",
		"betMode":"hezhi",
		"playMethodLabel":"任二组选和值",
		"playTypeLabel":"任选",
		"guajiGroup":"任选"
	}`
	cfg := parseSchemeConfig("custom", []byte(raw), 0, 0)
	if isRen2ZhixuanHezhiRule(cfg.Play) {
		t.Fatalf("组选和值不应走直选和值上限, rule=%+v", cfg.Play)
	}
}
