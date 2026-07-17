package schemes

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"caipiao/backend/internal/db/sqlcdb"
)

const (
	detailPreviewDrawLimit       = 120
	detailPreviewChartLimit      = 20
	detailPreviewHistoryLimit    = 100
	detailPreviewExecutionLimit  = 20
)

// PlanTrendChartPoint 计划走势折线图单点（round 为累计分值：中 +1、挂 -1）。
type PlanTrendChartPoint struct {
	Period string
	Round  int
	Win    bool
}

// PlanTrendHistoryRow 近期中挂汇总行。
type PlanTrendHistoryRow struct {
	Period string
	Win    bool
}

// BetRecordPreview 玩法详情投注记录 Tab 预览行。
type BetRecordPreview struct {
	Period     string
	PlayMethod string
	Multiplier string
	Round      string
	Amount     string
	ProfitLoss float64
	Status     string
}

// DetailPreviewResult 跟单大厅快照玩法详情推演结果。
type DetailPreviewResult struct {
	Executions         []BettingExecutionPreview
	PlanTrendGroupBets int
	PlanTrendHistory   []PlanTrendHistoryRow
	PlanTrendChart     []PlanTrendChartPoint
	BetRecords         []BetRecordPreview
}

type detailPeriodRow struct {
	issueNo     string
	periodShort string
	hit         bool
	roundIdx    int
	roundTotal  int
	betMult     float64
	amount      float64
	pnl         float64
	execution   BettingExecutionPreview
}

// RunDetailPreview 按方案配置与历史开奖推演玩法详情各 Tab 展示数据。
func RunDetailPreview(
	schemeName, contentSeed, kind string,
	configJSON []byte,
	playMethod string,
	draws []sqlcdb.ListLotteryDrawsRow,
	lotteryCode string,
) DetailPreviewResult {
	schemeName = strings.TrimSpace(schemeName)
	if schemeName == "" {
		schemeName = "方案"
	}
	if strings.TrimSpace(contentSeed) == "" {
		contentSeed = schemeName
	}
	if kind == "" {
		kind = "custom"
	}
	playMethod = strings.TrimSpace(playMethod)
	configJSON = ensurePreviewConfigContent(configJSON, contentSeed)
	oddsBase := oddsBaseForLottery(lotteryCode)
	cfg := parseSchemeConfig(kind, configJSON, 0, 0)
	cfg.Play.OddsBase = oddsBase
	if strings.TrimSpace(cfg.GroupContent) == "" && len(cfg.Groups) > 0 {
		cfg.GroupContent = cfg.Groups[0]
	}

	ordered := append([]sqlcdb.ListLotteryDrawsRow(nil), draws...)
	if len(ordered) == 0 {
		return DetailPreviewResult{PlanTrendGroupBets: previewGroupBetUnits(cfg, configJSON, contentSeed, kind, nil)}
	}
	sort.Slice(ordered, func(i, j int) bool {
		return ordered[i].IssueNo < ordered[j].IssueNo
	})
	if len(ordered) > detailPreviewDrawLimit {
		ordered = ordered[len(ordered)-detailPreviewDrawLimit:]
	}

	baseCoef := previewBaseCoef(configJSON)
	state := simPickState{}
	roundIdx := 0
	var prevBalls []string
	periodRows := make([]detailPeriodRow, 0, len(ordered))

	for _, draw := range ordered {
		cfgRound := parseSchemeConfig(kind, configJSON, roundIdx, 0)
		cfgRound.Play.OddsBase = oddsBase
		dec := resolvePickPreview(cfgRound, state, draw.IssueNo, prevBalls)
		if dec.Skip {
			prevBalls = sqlcdb.ParseDrawBalls(draw.Balls)
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

		if roundIdx < 0 || roundIdx >= len(cfgRound.Rounds) {
			roundIdx = 0
		}
		round := cfgRound.Rounds[roundIdx]
		betMult := effectiveBetMultiple(baseCoef, round)
		amount := calcBetAmount(eval.BetUnits, betMult, cfgRound.BetUnitYuan)
		pnl := calcPnLWithOdds(amount, eval.Hit, eval.Odds)

		periodShort := thirdPartyPeriodShort(draw.IssueNo)
		nextShort := thirdPartyPeriodShort(bumpPreviewIssue(draw.IssueNo))
		timeLabel := periodShort
		if nextShort != "" && nextShort != "—" {
			timeLabel = periodShort + "-" + nextShort
		}

		periodRows = append(periodRows, detailPeriodRow{
			issueNo:     draw.IssueNo,
			periodShort: periodShort,
			hit:         eval.Hit,
			roundIdx:    roundIdx,
			roundTotal:  len(cfgRound.Rounds),
			betMult:     betMult,
			amount:      amount,
			pnl:         pnl,
			execution: BettingExecutionPreview{
				Time:    timeLabel,
				Scheme:  schemeName,
				Numbers: formatPreviewNumbers(content),
				Period:  periodShort,
				Draw:    strings.Join(balls, " "),
				Win:     eval.Hit,
			},
		})

		pickIdx, curPick, lastDir := advancePickState(cfgRound, previewInstState(state), dec, eval.Hit)
		state = simPickState{pickIndex: pickIdx, currentPick: curPick, lastDirection: lastDir}
		roundIdx = nextRoundIndex(cfgRound.Rounds, roundIdx, eval.Hit)
		prevBalls = balls
	}

	execSource := tailDetailPeriodRows(periodRows, detailPreviewExecutionLimit)
	executions := make([]BettingExecutionPreview, len(execSource))
	for i, row := range execSource {
		executions[i] = row.execution
	}
	for i, j := 0, len(executions)-1; i < j; i, j = i+1, j-1 {
		executions[i], executions[j] = executions[j], executions[i]
	}

	if playMethod == "" {
		playMethod = cfg.PlayTypeLabel
	}
	recordPlayMethod := resolveBetRecordPlayMethod(cfg, playMethod)

	return DetailPreviewResult{
		Executions:         executions,
		PlanTrendGroupBets: previewGroupBetUnits(cfg, configJSON, contentSeed, kind, ordered),
		PlanTrendHistory:   buildPlanTrendPeriodHistory(periodRows, detailPreviewHistoryLimit),
		PlanTrendChart:     buildPlanTrendChart(tailDetailPeriodRows(periodRows, detailPreviewChartLimit)),
		BetRecords:         buildBetRecordPreviews(execSource, recordPlayMethod),
	}
}

func resolveBetRecordPlayMethod(cfg parsedSchemeConfig, queryPlayMethod string) string {
	label := strings.TrimSpace(cfg.PlayTypeLabel)
	query := strings.TrimSpace(queryPlayMethod)
	if label != "" && !isBarePlayToken(label) {
		return label
	}
	if query != "" && !isBarePlayToken(query) {
		return query
	}
	if label != "" {
		return label
	}
	return query
}

func previewGroupBetUnits(
	cfg parsedSchemeConfig,
	configJSON []byte,
	contentSeed, kind string,
	draws []sqlcdb.ListLotteryDrawsRow,
) int {
	configJSON = ensurePreviewConfigContent(configJSON, contentSeed)
	cfg = parseSchemeConfig(kind, configJSON, 0, 0)
	if strings.TrimSpace(cfg.GroupContent) == "" && len(cfg.Groups) > 0 {
		cfg.GroupContent = cfg.Groups[0]
	}
	pick := resolveNextPlanPick(cfg, draws)
	if strings.TrimSpace(pick) == "" {
		pick = strings.TrimSpace(cfg.GroupContent)
	}
	return planPickBetUnits(cfg, pick)
}

func previewBaseCoef(configJSON []byte) float64 {
	if len(configJSON) == 0 {
		return 1
	}
	var cfg map[string]interface{}
	if err := json.Unmarshal(configJSON, &cfg); err != nil {
		return 1
	}
	raw, ok := cfg["betMultiplier"]
	if !ok {
		return 1
	}
	m, ok := raw.(map[string]interface{})
	if !ok {
		return 1
	}
	kind, _ := m["kind"].(string)
	switch strings.TrimSpace(kind) {
	case "2":
		if v, ok := m["simpleMult"].(float64); ok && v > 0 {
			return v
		}
	case "3":
		return 1
	default:
		if v, ok := m["value"].(float64); ok && v > 0 {
			return v
		}
	}
	return 1
}

func buildPlanTrendChart(rows []detailPeriodRow) []PlanTrendChartPoint {
	score := 0
	out := make([]PlanTrendChartPoint, 0, len(rows))
	for _, row := range rows {
		if row.hit {
			score++
		} else {
			score--
		}
		out = append(out, PlanTrendChartPoint{
			Period: row.periodShort,
			Round:  score,
			Win:    row.hit,
		})
	}
	return out
}

func tailDetailPeriodRows(rows []detailPeriodRow, limit int) []detailPeriodRow {
	if limit <= 0 || len(rows) <= limit {
		return rows
	}
	return rows[len(rows)-limit:]
}

func buildPlanTrendPeriodHistory(rows []detailPeriodRow, limit int) []PlanTrendHistoryRow {
	if len(rows) == 0 || limit <= 0 {
		return nil
	}
	slice := tailDetailPeriodRows(rows, limit)
	if len(slice) == 0 {
		return nil
	}

	type streak struct {
		start string
		end   string
		win   bool
	}
	streaks := make([]streak, 0, len(slice))
	cur := streak{start: slice[0].periodShort, end: slice[0].periodShort, win: slice[0].hit}
	for i := 1; i < len(slice); i++ {
		row := slice[i]
		if row.hit == cur.win {
			cur.end = row.periodShort
			continue
		}
		streaks = append(streaks, cur)
		cur = streak{start: row.periodShort, end: row.periodShort, win: row.hit}
	}
	streaks = append(streaks, cur)

	out := make([]PlanTrendHistoryRow, 0, len(streaks))
	for i := len(streaks) - 1; i >= 0; i-- {
		s := streaks[i]
		label := s.end
		if s.start != s.end {
			label = s.start + " - " + s.end
		}
		out = append(out, PlanTrendHistoryRow{Period: label, Win: s.win})
	}
	return out
}

func buildPlanTrendHistory(rows []detailPeriodRow) []PlanTrendHistoryRow {
	if len(rows) == 0 {
		return nil
	}
	groups := make([]PlanTrendHistoryRow, 0)
	cycleStart := rows[0].periodShort

	for i, row := range rows {
		if !row.hit {
			continue
		}
		end := row.periodShort
		label := end
		if cycleStart != end {
			label = cycleStart + " - " + end
		}
		groups = append(groups, PlanTrendHistoryRow{Period: label, Win: true})
		if i+1 < len(rows) {
			cycleStart = rows[i+1].periodShort
		}
	}

	lastHit := -1
	for i, row := range rows {
		if row.hit {
			lastHit = i
		}
	}
	missFrom := lastHit + 1
	if missFrom < len(rows) {
		start := rows[missFrom].periodShort
		end := rows[len(rows)-1].periodShort
		label := end
		if start != end {
			label = start + " - " + end
		}
		groups = append(groups, PlanTrendHistoryRow{Period: label, Win: false})
	}

	for i, j := 0, len(groups)-1; i < j; i, j = i+1, j-1 {
		groups[i], groups[j] = groups[j], groups[i]
	}
	return groups
}

func buildBetRecordPreviews(rows []detailPeriodRow, playMethod string) []BetRecordPreview {
	if len(rows) == 0 {
		return nil
	}
	out := make([]BetRecordPreview, 0, len(rows))
	for _, row := range rows {
		out = append(out, BetRecordPreview{
			Period:     ThirdPartyPeriodDisplay(row.issueNo),
			PlayMethod: playMethod,
			Multiplier: formatPreviewMultiplier(row.betMult),
			Round:      roundLabel(row.roundIdx, row.roundTotal),
			Amount:     formatPreviewAmount(row.amount),
			ProfitLoss: row.pnl,
			Status:     "已结算",
		})
	}
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out
}

func formatPreviewMultiplier(v float64) string {
	if v <= 0 {
		return "1"
	}
	if v == float64(int(v)) {
		return strconv.Itoa(int(v))
	}
	return fmt.Sprintf("%.1f", v)
}

func formatPreviewAmount(v float64) string {
	return fmt.Sprintf("%.2f", round2(v))
}
