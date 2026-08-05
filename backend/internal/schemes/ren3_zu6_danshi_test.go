package schemes

import "testing"

func TestCountBetUnits_ren3Zu6DanshiPattern(t *testing.T) {
	t.Parallel()
	raw := `{
		"playTemplate":"ssc_std",
		"playTypeId":"g011",
		"subPlayId":"86",
		"catalogSubId":"86",
		"betMode":"danshi",
		"playMethodLabel":"任三组六单式",
		"playTypeLabel":"任选",
		"guajiGroup":"任选"
	}`
	cfg := parseSchemeConfig("custom", []byte(raw), 0, 0)
	rule := cfg.Play
	if !isZu6DanshiPlayRule(rule) {
		t.Fatalf("expected zu6 danshi rule, got %+v", rule)
	}
	if got := countRenxuanNeedsPositionBetUnits(rule, "万,千,个\n012"); got != 1 {
		t.Fatalf("012 units=%d want 1", got)
	}
	if got := countRenxuanNeedsPositionBetUnits(rule, "万,千,个\n012,210"); got != 1 {
		t.Fatalf("012,210 form-dedupe units=%d want 1", got)
	}
	if got := countRenxuanNeedsPositionBetUnits(rule, "万,千,个\n112"); got != 0 {
		t.Fatalf("112 (zu3) units=%d want 0", got)
	}
	if got := countRenxuanNeedsPositionBetUnits(rule, "万,千,个\n111"); got != 0 {
		t.Fatalf("111 (baozi) units=%d want 0", got)
	}
	if got := countRenxuanNeedsPositionBetUnits(rule, "万,千,百,个\n012,345"); got != 8 {
		t.Fatalf("C(4,3)*2 units=%d want 8", got)
	}
}

func TestValidateGroupContent_ren3Zu6DanshiRejectsZu3(t *testing.T) {
	t.Parallel()
	raw := `{
		"playTemplate":"ssc_std",
		"playTypeId":"g011",
		"subPlayId":"86",
		"catalogSubId":"任三组六单式",
		"betMode":"danshi",
		"playMethodLabel":"任三组六单式",
		"guajiGroup":"任选"
	}`
	cfg := parseSchemeConfig("custom", []byte(raw), 0, 0)
	err := validateGroupContent(cfg.Play, "万,千,个\n112")
	if err == nil {
		t.Fatal("expected error for 112 on zu6 danshi")
	}
	if err := validateGroupContent(cfg.Play, "万,千,个\n012"); err != nil {
		t.Fatalf("012 should pass: %v", err)
	}
}
