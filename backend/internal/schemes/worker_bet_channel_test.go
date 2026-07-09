package schemes

import (
	"testing"

	"caipiao/backend/internal/db/sqlcdb"
)

func TestUsesGuajiThirdParty(t *testing.T) {
	w := &Worker{}
	if w.usesGuajiThirdParty(sqlcdb.SchemeInstance{SimBet: true}) {
		t.Fatal("sim instance should not use third party")
	}
	if w.usesGuajiThirdParty(sqlcdb.SchemeInstance{SimBet: false}) {
		t.Fatal("formal instance without guaji placer should not use third party")
	}
}

func TestRunModeFromSimBet(t *testing.T) {
	if runModeFromSimBet(true) != "sim" {
		t.Fatal("simBet true -> sim")
	}
	if runModeFromSimBet(false) != "real" {
		t.Fatal("simBet false -> real")
	}
}

func TestSimBetFromClientRunMode(t *testing.T) {
	if !simBetFromClientRunMode("sim") {
		t.Fatal("sim -> true")
	}
	if simBetFromClientRunMode("prod") || simBetFromClientRunMode("real") {
		t.Fatal("prod/real -> false")
	}
}

func TestConfigSimBet(t *testing.T) {
	cfg := []byte(`{"simBet":true}`)
	if !configSimBet(cfg) {
		t.Fatal("simBet in config")
	}
	legacy := []byte(`{"runMode":"sim"}`)
	if !configSimBet(legacy) {
		t.Fatal("legacy runMode sim")
	}
}
