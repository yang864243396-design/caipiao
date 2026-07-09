package schemes

import "testing"

func TestIsBetUnitArtifact(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"1", true},
		{"2", true},
		{"0.01", true},
		{"0.001", true},
		{"dingwei", false},
		{"danshi", false},
		{"zu24", false},
		{"", false},
	}
	for _, tc := range cases {
		if got := isBetUnitArtifact(tc.in); got != tc.want {
			t.Fatalf("isBetUnitArtifact(%q)=%v want %v", tc.in, got, tc.want)
		}
	}
}

func TestNormalizeSchemeConfigBetFields(t *testing.T) {
	cfg := map[string]interface{}{"betMode": "1", "subId": "13"}
	normalizeSchemeConfigBetFields(cfg)
	if cfg["betUnit"] != "1" {
		t.Fatalf("betUnit=%v want 1", cfg["betUnit"])
	}
	if _, ok := cfg["betMode"]; ok {
		t.Fatalf("betMode should be removed, got %v", cfg["betMode"])
	}
}

func TestSchemeBetUnitFromConfig(t *testing.T) {
	if f := schemeBetUnitFromConfig(map[string]interface{}{"betUnit": "1"}); f != 1 {
		t.Fatalf("betUnit=1 got %v", f)
	}
	if f := schemeBetUnitFromConfig(map[string]interface{}{"betMode": "0.01"}); f != 0.01 {
		t.Fatalf("legacy betMode got %v", f)
	}
	if f := schemeBetUnitFromConfig(map[string]interface{}{"betMode": "dingwei"}); f != baseBetUnitYuan {
		t.Fatalf("play betMode should not affect unit, got %v", f)
	}
}

func TestPlayBetModeFromConfig(t *testing.T) {
	if m := playBetModeFromConfig(map[string]interface{}{"betMode": "1"}); m != "" {
		t.Fatalf("numeric betMode cleared, got %q", m)
	}
	if m := playBetModeFromConfig(map[string]interface{}{"betMode": "dingwei"}); m != "dingwei" {
		t.Fatalf("play betMode=%q", m)
	}
}
