package schemes

import "testing"

func TestStrategyCandidateRequiresAcceptedFrozenRuleAndDraw(t *testing.T) {
	if shouldProcessFormalStrategy(strategyCandidate{Accepted: true, HasSnapshot: true, HasDraw: true}) != true {
		t.Fatal("accepted frozen rule with draw must be eligible")
	}
	for _, candidate := range []strategyCandidate{
		{HasSnapshot: true, HasDraw: true},
		{Accepted: true, HasDraw: true},
		{Accepted: true, HasSnapshot: true},
	} {
		if shouldProcessFormalStrategy(candidate) {
			t.Fatalf("candidate %+v must not be eligible", candidate)
		}
	}
}
