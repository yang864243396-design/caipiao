package schemes

import "testing"

func TestFormatLHCPlayLabel(t *testing.T) {
	if got := formatLHCPlayLabel("tema", "tema_a"); got != "特码特码A" {
		t.Fatalf("got %q", got)
	}
	if got := formatLHCPlayLabel("erquanzhong", "fushi"); got != "二全中复式" {
		t.Fatalf("got %q", got)
	}
}

func TestResolvePlayTypeLabelLHC(t *testing.T) {
	cfg := map[string]interface{}{
		"playTemplate": "lhc_std",
		"playTypeId":   "tema",
		"subPlayId":    "tema_a",
	}
	if got := resolvePlayTypeLabel(cfg); got != "特码特码A" {
		t.Fatalf("got %q", got)
	}
}
