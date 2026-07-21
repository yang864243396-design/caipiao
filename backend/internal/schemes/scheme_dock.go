package schemes

import (
	"sort"
	"strings"

	"caipiao/backend/internal/db/sqlcdb"
)

// SchemeDockSummary 玩法详情底部投注区展示（由快照方案配置推演）。
type SchemeDockSummary struct {
	BetUnitYuan            float64
	BetMultiplier          float64
	PlanBetUnits           int
	ContraryBetUnits       int
	SchemePickDisplay      string
	EstimatedPrize         float64
	ContraryEstimatedPrize float64
}

// ComputeSchemeDockSummary 按方案配置与历史开奖推演投注区选中/总额/奖金。
func ComputeSchemeDockSummary(
	contentSeed, kind string,
	configJSON []byte,
	draws []sqlcdb.ListLotteryDrawsRow,
	lotteryCode string,
) SchemeDockSummary {
	contentSeed = strings.TrimSpace(contentSeed)
	if contentSeed == "" {
		contentSeed = "scheme-dock"
	}
	if kind == "" {
		kind = "custom"
	}
	configJSON = ensurePreviewConfigContent(configJSON, contentSeed)
	cfg := parseSchemeConfig(kind, configJSON, 0, 0)
	cfg.Play = attachOddsBase(cfg.Play, lotteryCode)
	if strings.TrimSpace(cfg.GroupContent) == "" && len(cfg.Groups) > 0 {
		cfg.GroupContent = cfg.Groups[0]
	}

	pick := resolveNextPlanPick(cfg, draws)
	if strings.TrimSpace(pick) == "" {
		pick = strings.TrimSpace(cfg.GroupContent)
	}
	planDisplay := formatPlanInverseDigits(pick, cfg.Play)
	planUnits := planPickBetUnits(cfg, pick)
	contraryUnits := contraryBetUnits(planDisplay, cfg.Play)
	if contraryUnits <= 0 {
		contraryUnits = planUnits
	}

	roundIdx := resolveRoundIndexAfterDraws(kind, configJSON, draws)
	if roundIdx < 0 || roundIdx >= len(cfg.Rounds) {
		roundIdx = 0
	}
	round := cfg.Rounds[roundIdx]
	baseCoef := previewBaseCoef(configJSON)
	betMult := effectiveBetMultiple(baseCoef, round)
	betUnit := cfg.BetUnitYuan
	if betUnit <= 0 {
		betUnit = baseBetUnitYuan
	}

	planEval := evaluatePlayHit(cfg.Play, nil, pick, false, "", cfg.Play.PositionIdx)
	contraryEval := evaluateContraryHit(cfg.Play, nil, planDisplay, cfg.Play.PositionIdx)

	return SchemeDockSummary{
		BetUnitYuan:            betUnit,
		BetMultiplier:          betMult,
		PlanBetUnits:           planUnits,
		ContraryBetUnits:       contraryUnits,
		SchemePickDisplay:      planDisplay,
		EstimatedPrize:         estimateMaxPrize(betUnit, betMult, planEval.Odds),
		ContraryEstimatedPrize: estimateMaxPrize(betUnit, betMult, contraryEval.Odds),
	}
}

func resolveRoundIndexAfterDraws(
	kind string,
	configJSON []byte,
	draws []sqlcdb.ListLotteryDrawsRow,
) int {
	ordered := append([]sqlcdb.ListLotteryDrawsRow(nil), draws...)
	if len(ordered) == 0 {
		return 0
	}
	sort.Slice(ordered, func(i, j int) bool {
		return ordered[i].IssueNo < ordered[j].IssueNo
	})
	if len(ordered) > detailPreviewDrawLimit {
		ordered = ordered[len(ordered)-detailPreviewDrawLimit:]
	}

	state := simPickState{}
	roundIdx := 0
	var prevBalls []string
	histDraws := make([][]string, 0, len(ordered))

	for _, draw := range ordered {
		cfgRound := parseSchemeConfig(kind, configJSON, roundIdx, 0)
		dec := resolvePickPreview(cfgRound, state, draw.IssueNo, prevBalls, histDraws)
		if dec.Skip {
			balls := sqlcdb.ParseDrawBalls(draw.Balls)
			if len(balls) > 0 {
				histDraws = append(histDraws, balls)
			}
			prevBalls = balls
			continue
		}
		content := strings.TrimSpace(dec.Content)
		if content == "" {
			content = cfgRound.GroupContent
		}
		if strings.TrimSpace(content) == "" {
			prevBalls = sqlcdb.ParseDrawBalls(draw.Balls)
			continue
		}
		balls := sqlcdb.ParseDrawBalls(draw.Balls)
		eval := evaluatePlayHit(cfgRound.Play, balls, content, cfgRound.Contrary, cfgRound.ContraryPlan, cfgRound.Play.PositionIdx)
		pickIdx, curPick, lastDir := advancePickState(cfgRound, previewInstState(state), dec, eval.Hit)
		state = simPickState{pickIndex: pickIdx, currentPick: curPick, lastDirection: lastDir}
		roundIdx = nextRoundIndex(cfgRound.Rounds, roundIdx, eval.Hit)
		if len(balls) > 0 {
			histDraws = append(histDraws, balls)
		}
		prevBalls = balls
	}
	return roundIdx
}

func estimateMaxPrize(betUnit, betMult, odds float64) float64 {
	if betUnit <= 0 {
		betUnit = baseBetUnitYuan
	}
	if betMult <= 0 {
		betMult = 1
	}
	if odds <= 0 {
		odds = oddsDingweiOdds(0)
	}
	return round2(betUnit * betMult * odds)
}
