package schemes

import "testing"

func TestMaxBetUnits_ren3ZhixuanHezhiIs9000(t *testing.T) {
	t.Parallel()
	raw := `{
		"playTemplate":"ssc_std",
		"playTypeId":"g011",
		"subPlayId":"82",
		"catalogSubId":"82",
		"betMode":"hezhi",
		"playMethodLabel":"任三直选和值",
		"playTypeLabel":"任选",
		"guajiGroup":"任选"
	}`
	cfg := parseSchemeConfig("custom", []byte(raw), 0, 0)
	if got := maxBetUnitsForPlay(cfg.Play); got != 9000 {
		t.Fatalf("max=%d want 9000 rule=%+v", got, cfg.Play)
	}
}
