package accountsvc

import (
	"encoding/json"
	"testing"

	"caipiao/backend/internal/cloud/schemestate"
	"caipiao/backend/internal/playrules"
)

func TestResolveFormalStrategyVerdictUsesFrozenLocalResultAndDetectsMismatch(t *testing.T) {
	previous := schemestate.FormalRuleEvaluator
	t.Cleanup(func() { schemestate.FormalRuleEvaluator = previous })
	schemestate.FormalRuleEvaluator = func(input schemestate.FormalRuleEvaluationInput) (schemestate.FormalRuleEvaluation, error) {
		return schemestate.FormalRuleEvaluation{Hit: false, WinningUnits: 4}, nil
	}

	snapshot, err := json.Marshal(playrules.Snapshot{EvaluatorKey: "ssc.direct"})
	if err != nil {
		t.Fatal(err)
	}
	got := resolveFormalStrategyVerdict(formalStrategyInput{
		ProviderHit: true,
		LotteryCode: "tron_ffc_3s",
		BetContent:  "1\n2\n3",
		Balls:       []string{"1", "2", "4", "4", "5"},
		Snapshot:    snapshot,
	})
	if !got.UsedLocal || got.Hit || got.WinningUnits != 4 || !got.Mismatch {
		t.Fatalf("verdict = %+v, want local miss with provider mismatch", got)
	}
}

func TestResolveFormalStrategyVerdictKeepsProviderResultWithoutFrozenRule(t *testing.T) {
	got := resolveFormalStrategyVerdict(formalStrategyInput{ProviderHit: true})
	if got.UsedLocal || !got.Hit || got.Mismatch {
		t.Fatalf("verdict = %+v, want provider fallback", got)
	}
}

func TestReconciledStrategyStatusChangesOnlyOnHitDifference(t *testing.T) {
	if got, ok := reconciledStrategyStatus("completed", true, true, false); !ok || got != "mismatch" {
		t.Fatalf("different hits: got (%q,%t), want (mismatch,true)", got, ok)
	}
	if got, ok := reconciledStrategyStatus("completed", true, true, true); !ok || got != "completed" {
		t.Fatalf("equal hits: got (%q,%t), want (completed,true)", got, ok)
	}
	if got, ok := reconciledStrategyStatus("skipped", true, true, false); ok || got != "skipped" {
		t.Fatalf("skipped evaluation: got (%q,%t), want (skipped,false)", got, ok)
	}
}
