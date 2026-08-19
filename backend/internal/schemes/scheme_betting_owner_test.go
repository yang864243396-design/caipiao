package schemes

import "testing"

func TestLegacyFormalBetRequiresLegacyOwner(t *testing.T) {
	if !legacyOwnsFormalBet("legacy") {
		t.Fatal("legacy owner should be allowed")
	}
	for _, owner := range []string{"event", "", "LEGACY"} {
		if legacyOwnsFormalBet(owner) {
			t.Fatalf("owner %q must not be allowed", owner)
		}
	}
}
