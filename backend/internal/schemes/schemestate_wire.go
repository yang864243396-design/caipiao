package schemes

import "caipiao/backend/internal/cloud/schemestate"

// 注入正式盘派奖后的出号游标推进逻辑，打破
// schemes -> periodsync -> accountsvc -> schemestate 依赖链导致的 import cycle。
func init() {
	schemestate.FormalPickAdvancer = AdvancePickAfterFormalSettlement
	schemestate.FormalRuleEvaluator = evaluateFormalStrategyRule
}

func evaluateFormalStrategyRule(input schemestate.FormalRuleEvaluationInput) (schemestate.FormalRuleEvaluation, error) {
	groupIndex := 0
	if input.RoundIndex > 0 {
		groupIndex = input.RoundIndex
	}
	cfg := parseSchemeConfig(input.Kind, input.DefinitionConfig, input.RoundIndex, groupIndex)
	cfg.Play = attachOddsBase(cfg.Play, input.LotteryCode)
	evaluation, err := evaluateFrozenRule(input.Snapshot, cfg.Play, input.Balls, input.BetContent, false, "")
	if err != nil {
		return schemestate.FormalRuleEvaluation{}, err
	}
	return schemestate.FormalRuleEvaluation{Hit: evaluation.Hit, WinningUnits: evaluation.BetUnits}, nil
}
