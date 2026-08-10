package schemes

import (
	"encoding/json"
	"testing"
)

func TestSetSchemeConfigMultiplierInitializesEmptyConfig(t *testing.T) {
	raw, err := setSchemeConfigMultiplier(nil, 6)
	if err != nil {
		t.Fatalf("setSchemeConfigMultiplier: %v", err)
	}
	var cfg map[string]interface{}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if got := cfg["multCoeff"]; got != "6" {
		t.Fatalf("multCoeff=%v, want 6", got)
	}
}
