package schemes

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/lottery"
)

type repickAction int

const (
	repickOK repickAction = iota
	repickWait
	repickSkip
)

type repickResult struct {
	action   repickAction
	dec      pickDecision
	content  string
	playEval betEvaluation
}

// normalizeResolvedBetContent 将出号结果规范为可下单内容（含随机超限重抽）。
func normalizeResolvedBetContent(cfg parsedSchemeConfig, dec *pickDecision) string {
	if dec == nil {
		return ""
	}
	betContent := dec.Content
	// 随机出号：空内容直接重抽，勿回落 schemeGroups（满选可能超限，且违背「每期随机」）。
	if cfg.RunTypeID == RunTypeRandomDraw {
		betContent = joinPositionPoolGroupsIfNeeded(cfg, betContent)
		betContent = normalizeZhixuanDanshiContent(cfg.Play, betContent)
		if strings.TrimSpace(betContent) == "" || contentExceedsBetUnitsMax(cfg.Play, betContent) {
			betContent = randomDrawContentUnderMax(cfg)
			dec.Content = betContent
		}
		return betContent
	}
	if strings.TrimSpace(betContent) == "" {
		betContent = cfg.GroupContent
	}
	betContent = joinPositionPoolGroupsIfNeeded(cfg, betContent)
	betContent = normalizeZhixuanDanshiContent(cfg.Play, betContent)
	return betContent
}

// syncEvalBetUnitsWithWire 金额/上限以 wire 组合注数为准（和值/跨度 evaluate 可能是选项个数）。
func syncEvalBetUnitsWithWire(rule playRule, content string, eval *betEvaluation) {
	if eval == nil {
		return
	}
	if wire := countPlayWireBetUnits(rule, content); wire > 0 {
		eval.BetUnits = wire
	}
}

// resolveRandomDrawUnderMax 随机出号超限时再抽一次；仍超限或空则应跳过本期（不停方案）。
func resolveRandomDrawUnderMax(cfg parsedSchemeConfig, content string) (string, bool) {
	if strings.TrimSpace(content) != "" && !contentExceedsBetUnitsMax(cfg.Play, content) {
		return content, true
	}
	next := randomDrawContentUnderMax(cfg)
	if strings.TrimSpace(next) == "" || contentExceedsBetUnitsMax(cfg.Play, next) {
		return "", false
	}
	return next, true
}

// skipPeriodPick 本期策略跳过：推进第三方期号游标，不下注。
func (w *Worker) skipPeriodPick(ctx context.Context, inst sqlcdb.SchemeInstance, period, runType string) error {
	if w == nil || w.q == nil {
		return nil
	}
	slog.Debug("scheme worker bet skipped: pick strategy skip",
		"id", inst.ID, "period", period, "runType", runType)
	return w.skipPeriodPickWithQ(ctx, w.q, inst, period, runType)
}

func (w *Worker) skipPeriodPickWithQ(
	ctx context.Context,
	q *sqlcdb.Queries,
	inst sqlcdb.SchemeInstance,
	period, runType string,
) error {
	// 必须以「策略判定的本期」写游标。若此处改写成第三方当前开放期，
	// 临近翻页时会把 N+1 提前占掉，下一期开盘即 period_cursor_taken 失投。
	return w.skipPeriodPickWithCurrentPick(ctx, q, inst, period, runType, inst.CurrentPick)
}

// repickForFinalPeriod 在最终可投期锁定后重算出号（开某投某 / 跨期）。
func (w *Worker) repickForFinalPeriod(
	ctx context.Context,
	cfg parsedSchemeConfig,
	inst sqlcdb.SchemeInstance,
	draw *sqlcdb.LotteryDraw,
	finalPeriod, pickIssue string,
) (repickResult, error) {
	finalPeriod = strings.TrimSpace(finalPeriod)
	if draw == nil || finalPeriod == "" {
		return repickResult{action: repickOK}, nil
	}
	draw.IssueNo = finalPeriod

	needsPrev := cfg.RunTypeID == RunTypeAdvTriggerBet || cfg.RunTypeID == RunTypeHotColdWarm
	if needsPrev && !w.hotColdPreviousDrawReady(ctx, inst.LotteryCode, finalPeriod) {
		rem, ok := lottery.PeriodsCountdownSec(inst.LotteryCode, time.Now())
		// 开某投某：上期未到只 Wait，勿 Skip（Skip 会推进游标，期号翻页时易误占下一期）
		if cfg.RunTypeID == RunTypeAdvTriggerBet {
			slog.Debug("scheme worker bet deferred: waiting previous draw after period lock",
				"id", inst.ID, "period", finalPeriod, "pickIssue", pickIssue, "countdown", rem, "countdownOK", ok)
			return repickResult{action: repickWait}, nil
		}
		if !ok || rem >= hotColdPrevDrawWaitMinSec {
			slog.Debug("scheme worker bet deferred: waiting previous draw after period lock",
				"id", inst.ID, "period", finalPeriod, "pickIssue", pickIssue, "runType", cfg.RunTypeID, "countdown", rem)
			return repickResult{action: repickWait}, nil
		}
		// 冷热临近封盘：允许降级统计后重出号
		slog.Info("scheme worker hot/cold proceed without previous draw after period lock",
			"id", inst.ID, "period", finalPeriod, "countdown", rem)
	}

	dec := w.resolvePick(ctx, cfg, inst, *draw)
	if dec.Skip {
		slog.Info("scheme worker bet skipped after period lock",
			"id", inst.ID, "period", finalPeriod, "pickIssue", pickIssue, "runType", cfg.RunTypeID)
		return repickResult{action: repickSkip, dec: dec}, nil
	}
	content := normalizeResolvedBetContent(cfg, &dec)
	balls := sqlcdb.ParseDrawBalls(draw.Balls)
	playEval := evaluatePlayHit(cfg.Play, balls, content, cfg.Contrary, cfg.ContraryPlan, cfg.Play.PositionIdx)
	if finalPeriod != pickIssue {
		slog.Info("scheme worker repick after open period changed",
			"id", inst.ID, "pickIssue", pickIssue, "finalPeriod", finalPeriod,
			"runType", cfg.RunTypeID, "content", content)
	}
	return repickResult{action: repickOK, dec: dec, content: content, playEval: playEval}, nil
}
