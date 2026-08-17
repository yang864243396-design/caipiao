package schemestate

import (
	"testing"

	"caipiao/backend/internal/playrules"
)

func TestEvaluateFormalRuleUsesInjectedFrozenEvaluator(t *testing.T) {
	previous := FormalRuleEvaluator
	t.Cleanup(func() { FormalRuleEvaluator = previous })

	called := false
	FormalRuleEvaluator = func(input FormalRuleEvaluationInput) (FormalRuleEvaluation, error) {
		called = true
		if input.LotteryCode != "tron_ffc_3s" || input.BetContent != "1\n2\n3" || len(input.Balls) != 5 {
			t.Fatalf("input = %+v, want frozen formal rule context", input)
		}
		return FormalRuleEvaluation{Hit: true, WinningUnits: 1}, nil
	}

	got, available, err := EvaluateFormalRule(FormalRuleEvaluationInput{
		LotteryCode: "tron_ffc_3s",
		BetContent:  "1\n2\n3",
		Balls:       []string{"1", "2", "3", "4", "5"},
		Snapshot:    playrules.Snapshot{EvaluatorKey: "ssc.direct"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !available || !called || !got.Hit || got.WinningUnits != 1 {
		t.Fatalf("result = %+v, available=%v, called=%v; want injected hit", got, available, called)
	}
}
