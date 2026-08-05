package schemes

import "testing"

func TestCountBetUnits_ren3Zu3DanshiPattern(t *testing.T) {
	t.Parallel()
	raw := `{
		"playTemplate":"ssc_std",
		"playTypeId":"g011",
		"subPlayId":"84",
		"catalogSubId":"84",
		"betMode":"danshi",
		"playMethodLabel":"任三组三单式",
		"playTypeLabel":"任选",
		"guajiGroup":"任选"
	}`
	cfg := parseSchemeConfig("custom", []byte(raw), 0, 0)
	rule := cfg.Play
	if !isZu3DanshiPlayRule(rule) {
		t.Fatalf("expected zu3 danshi rule, got %+v", rule)
	}
	if got := countRenxuanNeedsPositionBetUnits(rule, "万,千,个\n112"); got != 1 {
		t.Fatalf("112 units=%d want 1", got)
	}
	if got := countRenxuanNeedsPositionBetUnits(rule, "万,千,个\n112,121"); got != 1 {
		t.Fatalf("112,121 form-dedupe units=%d want 1", got)
	}
	if got := countRenxuanNeedsPositionBetUnits(rule, "万,千,个\n012"); got != 0 {
		t.Fatalf("012 (zu6) units=%d want 0", got)
	}
	if got := countRenxuanNeedsPositionBetUnits(rule, "万,千,个\n111"); got != 0 {
		t.Fatalf("111 (baozi) units=%d want 0", got)
	}
	if got := countRenxuanNeedsPositionBetUnits(rule, "万,千,百,个\n112,223"); got != 8 {
		t.Fatalf("C(4,3)*2 units=%d want 8", got)
	}
}

func TestValidateGroupContent_ren3Zu3DanshiRejectsZu6(t *testing.T) {
	t.Parallel()
	raw := `{
		"playTemplate":"ssc_std",
		"playTypeId":"g011",
		"subPlayId":"84",
		"catalogSubId":"任三组三单式",
		"betMode":"danshi",
		"playMethodLabel":"任三组三单式",
		"guajiGroup":"任选"
	}`
	cfg := parseSchemeConfig("custom", []byte(raw), 0, 0)
	err := validateGroupContent(cfg.Play, "万,千,个\n012")
	if err == nil {
		t.Fatal("expected error for 012 on zu3 danshi")
	}
}
