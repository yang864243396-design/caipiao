package schemes

import "testing"

func TestCountBetUnits_ren3HunhePattern(t *testing.T) {
	t.Parallel()
	raw := `{
		"playTemplate":"ssc_std",
		"playTypeId":"g011",
		"subPlayId":"87",
		"catalogSubId":"87",
		"betMode":"hunhe",
		"playMethodLabel":"任三混合组选",
		"playTypeLabel":"任选",
		"guajiGroup":"任选"
	}`
	cfg := parseSchemeConfig("custom", []byte(raw), 0, 0)
	rule := cfg.Play
	if !isHunhePlayRule(rule) {
		t.Fatalf("expected hunhe rule, got %+v", rule)
	}
	if got := countRenxuanNeedsPositionBetUnits(rule, "万,千,个\n012"); got != 1 {
		t.Fatalf("012 units=%d want 1", got)
	}
	if got := countRenxuanNeedsPositionBetUnits(rule, "万,千,个\n123,321"); got != 1 {
		t.Fatalf("123,321 form-dedupe units=%d want 1", got)
	}
	if got := countRenxuanNeedsPositionBetUnits(rule, "万,千,个\n111"); got != 0 {
		t.Fatalf("111 (baozi) units=%d want 0", got)
	}
	if got := countRenxuanNeedsPositionBetUnits(rule, "万,千,百,个\n012,345"); got != 8 {
		t.Fatalf("C(4,3)*2 units=%d want 8", got)
	}
}

func TestValidateGroupContent_ren3HunheRejectsBaozi(t *testing.T) {
	t.Parallel()
	raw := `{
		"playTemplate":"ssc_std",
		"playTypeId":"g011",
		"subPlayId":"87",
		"catalogSubId":"任三混合组选",
		"betMode":"hunhe",
		"playMethodLabel":"任三混合组选",
		"guajiGroup":"任选"
	}`
	cfg := parseSchemeConfig("custom", []byte(raw), 0, 0)
	if err := validateGroupContent(cfg.Play, "万,千,个\n111"); err == nil {
		t.Fatal("expected error for baozi on hunhe")
	}
	if err := validateGroupContent(cfg.Play, "万,千,个\n012"); err != nil {
		t.Fatalf("012 should pass: %v", err)
	}
}
