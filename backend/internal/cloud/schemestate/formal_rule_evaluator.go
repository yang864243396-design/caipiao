package schemestate

import "caipiao/backend/internal/playrules"

// FormalRuleEvaluationInput is the immutable context used to re-evaluate a
// real bet from its frozen published rule. It deliberately carries no money:
// real financial settlement remains third-party authoritative.
type FormalRuleEvaluationInput struct {
	Kind             string
	DefinitionConfig []byte
	RoundIndex       int
	LotteryCode      string
	BetContent       string
	Balls            []string
	Snapshot         playrules.Snapshot
}

// FormalRuleEvaluation is the local strategy conclusion kept alongside the
// third-party financial settlement.
type FormalRuleEvaluation struct {
	Hit          bool
	WinningUnits int
}

// FormalRuleEvaluator is injected by schemes to avoid the import cycle
// schemes -> periodsync -> accountsvc -> schemestate.
var FormalRuleEvaluator func(FormalRuleEvaluationInput) (FormalRuleEvaluation, error)

// EvaluateFormalRule invokes the injected frozen-rule evaluator. available is
// false for binaries that do not link the schemes package, so callers can
// retain legacy provider-result behavior without failing financial settlement.
func EvaluateFormalRule(input FormalRuleEvaluationInput) (result FormalRuleEvaluation, available bool, err error) {
	if FormalRuleEvaluator == nil {
		return FormalRuleEvaluation{}, false, nil
	}
	result, err = FormalRuleEvaluator(input)
	return result, true, err
}
