package schemes

import (
	"sort"
	"strconv"
	"strings"

	"caipiao/backend/internal/db/sqlcdb"
)

// PlanInverseDisplay 玩法详情「计划反集」Tab 展示数据。
type PlanInverseDisplay struct {
	Digits   string
	BetCount int
}

// ComputePlanInverseDisplay 由方案配置与历史开奖推演当前计划号码及反集注数。
func ComputePlanInverseDisplay(
	contentSeed, kind string,
	configJSON []byte,
	draws []sqlcdb.ListLotteryDrawsRow,
) PlanInverseDisplay {
	contentSeed = strings.TrimSpace(contentSeed)
	if contentSeed == "" {
		contentSeed = "plan-inverse"
	}
	if kind == "" {
		kind = "custom"
	}
	configJSON = ensurePreviewConfigContent(configJSON, contentSeed)
	cfg := parseSchemeConfig(kind, configJSON, 0, 0)
	if strings.TrimSpace(cfg.GroupContent) == "" && len(cfg.Groups) > 0 {
		cfg.GroupContent = cfg.Groups[0]
	}

	pick := resolveNextPlanPick(cfg, draws)
	if strings.TrimSpace(pick) == "" {
		pick = strings.TrimSpace(cfg.GroupContent)
	}
	planFormatted := formatPlanInverseDigits(pick, cfg.Play)
	digits := formatContraryDisplay(pick, cfg.Play)
	if digits == "" {
		return PlanInverseDisplay{}
	}
	units := contraryBetUnits(planFormatted, cfg.Play)
	if units <= 0 {
		units = 1
	}
	return PlanInverseDisplay{Digits: digits, BetCount: units}
}

func resolveNextPlanPick(cfg parsedSchemeConfig, draws []sqlcdb.ListLotteryDrawsRow) string {
	ordered := append([]sqlcdb.ListLotteryDrawsRow(nil), draws...)
	if len(ordered) == 0 {
		dec := resolvePickPreview(cfg, simPickState{}, "", nil)
		if dec.Skip {
			return strings.TrimSpace(cfg.GroupContent)
		}
		if c := strings.TrimSpace(dec.Content); c != "" {
			return c
		}
		return strings.TrimSpace(cfg.GroupContent)
	}
	sort.Slice(ordered, func(i, j int) bool {
		return ordered[i].IssueNo < ordered[j].IssueNo
	})

	state := simPickState{}
	var prevBalls []string
	lastIssue := ordered[len(ordered)-1].IssueNo

	for _, draw := range ordered {
		dec := resolvePickPreview(cfg, state, draw.IssueNo, prevBalls)
		if dec.Skip {
			prevBalls = sqlcdb.ParseDrawBalls(draw.Balls)
			continue
		}
		content := strings.TrimSpace(dec.Content)
		if content == "" {
			content = cfg.GroupContent
		}
		if strings.TrimSpace(content) == "" {
			prevBalls = sqlcdb.ParseDrawBalls(draw.Balls)
			continue
		}
		balls := sqlcdb.ParseDrawBalls(draw.Balls)
		eval := evaluatePlayHit(cfg.Play, balls, content, cfg.Contrary, cfg.ContraryPlan, cfg.Play.PositionIdx)
		pickIdx, curPick, lastDir := advancePickState(cfg, previewInstState(state), dec, eval.Hit)
		state = simPickState{pickIndex: pickIdx, currentPick: curPick, lastDirection: lastDir}
		prevBalls = balls
		lastIssue = draw.IssueNo
	}

	nextIssue := bumpPreviewIssue(lastIssue)
	dec := resolvePickPreview(cfg, state, nextIssue, prevBalls)
	if dec.Skip {
		return strings.TrimSpace(cfg.GroupContent)
	}
	if c := strings.TrimSpace(dec.Content); c != "" {
		return c
	}
	return strings.TrimSpace(cfg.GroupContent)
}

// formatPlanInverseDigits 将计划选号格式化为 planInverseNumbers 展示串。
func formatPlanInverseDigits(pick string, rule playRule) string {
	pick = strings.TrimSpace(pick)
	if pick == "" {
		return ""
	}
	if rule.PlayTemplate == "lhc_std" || isLHCTypeID(rule.PlayTypeID) {
		return formatLHCPickLine(pick)
	}
	lines := splitGroupLines(pick)
	if len(lines) <= 1 {
		return formatSSCPlanLine(pick)
	}
	posCount := playPositionCount(rule)
	if posCount <= 1 {
		posCount = len(lines)
	}
	parts := make([]string, 0, posCount)
	for i := 0; i < posCount; i++ {
		line := ""
		if i < len(lines) {
			line = lines[i]
		}
		parts = append(parts, formatSSCPlanLine(line))
	}
	return strings.Join(parts, "\n")
}

func formatSSCPlanLine(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	tokens := parseDigitTokens(raw)
	if len(tokens) == 0 {
		return raw
	}
	return strings.Join(tokens, ",")
}

func formatLHCPickLine(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	tokens := parseNumberTokens(raw, 0)
	if len(tokens) == 0 {
		return raw
	}
	out := make([]string, 0, len(tokens))
	for _, t := range tokens {
		n, err := strconv.Atoi(t)
		if err != nil || n < 1 || n > 49 {
			continue
		}
		out = append(out, strconv.Itoa(n))
	}
	if len(out) == 0 {
		return raw
	}
	return strings.Join(out, ",")
}

// planPickBetUnits 计算方案选号本身的注数（非反集注数）。
func planPickBetUnits(cfg parsedSchemeConfig, pick string) int {
	pick = strings.TrimSpace(pick)
	if pick == "" {
		return 1
	}
	eval := evaluatePlayHit(cfg.Play, nil, pick, false, "", cfg.Play.PositionIdx)
	if eval.BetUnits > 0 {
		return eval.BetUnits
	}
	return 1
}

func contraryBetUnits(planInverse string, rule playRule) int {
	planInverse = strings.TrimSpace(planInverse)
	if planInverse == "" {
		return 0
	}
	eval := evaluateContraryHit(rule, nil, planInverse, rule.PositionIdx)
	return eval.BetUnits
}

// formatContraryDisplay 将计划选号对应的反集投注内容格式化为展示串。
func formatContraryDisplay(pick string, rule playRule) string {
	pick = strings.TrimSpace(pick)
	if pick == "" {
		return ""
	}
	if rule.PlayTemplate == "lhc_std" || isLHCTypeID(rule.PlayTypeID) {
		return formatPlanInverseDigits(pick, rule)
	}
	if rule.SegmentLen <= 1 {
		planLine := formatSSCPlanLine(pick)
		picks := parseContraryPicks(planLine, rule.PositionIdx)
		if len(picks) == 0 {
			return ""
		}
		return strings.Join(picks, ",")
	}
	lines := splitGroupLines(pick)
	posCount := playPositionCount(rule)
	if posCount <= 1 {
		posCount = len(lines)
	}
	if posCount <= 0 {
		posCount = rule.SegmentLen
	}
	parts := make([]string, 0, posCount)
	for i := 0; i < posCount; i++ {
		line := ""
		if i < len(lines) {
			line = lines[i]
		}
		forbidden := parseDigitTokens(line)
		inverse := allDigitsExcept(forbidden)
		parts = append(parts, strings.Join(inverse, ","))
	}
	return strings.Join(parts, "\n")
}
