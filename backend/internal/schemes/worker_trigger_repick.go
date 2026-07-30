package schemes

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

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
	if strings.TrimSpace(betContent) == "" {
		betContent = cfg.GroupContent
	}
	betContent = joinPositionPoolGroupsIfNeeded(cfg, betContent)
	betContent = normalizeZhixuanDanshiContent(cfg.Play, betContent)
	if cfg.RunTypeID == RunTypeRandomDraw {
		if strings.TrimSpace(betContent) == "" || contentExceedsBetUnitsMax(cfg.Play, betContent) {
			betContent = randomDrawContentUnderMax(cfg)
			dec.Content = betContent
		}
	}
	return betContent
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
	if q == nil {
		return nil
	}
	skipPeriod := strings.TrimSpace(period)
	if p, ok := thirdPartyOpenPeriod(inst.LotteryCode); ok {
		skipPeriod = p
	}
	if _, err := q.ApplySchemeInstanceBet(ctx, sqlcdb.ApplySchemeInstanceBetParams{
		ID:               inst.ID,
		CountdownSec:     w.periodCountdownForInst(inst, time.Now()),
		Turnover:         numericFromFloat(0),
		Pnl:              numericFromFloat(0),
		Multiplier:       inst.Multiplier,
		RoundIndex:       inst.RoundIndex,
		LastSettledIssue: pgtype.Text{String: skipPeriod, Valid: skipPeriod != ""},
		LookbackPnl:      numericFromFloat(0),
		PickIndex:        inst.PickIndex,
		CurrentPick:      inst.CurrentPick,
		LastDirection:    inst.LastDirection,
	}); err != nil {
		return err
	}
	_ = appendPickSkipAudit(ctx, q, inst, period)
	_ = runType
	return nil
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
		if !ok || rem >= hotColdPrevDrawWaitMinSec {
			slog.Debug("scheme worker bet deferred: waiting previous draw after period lock",
				"id", inst.ID, "period", finalPeriod, "pickIssue", pickIssue, "runType", cfg.RunTypeID, "countdown", rem)
			return repickResult{action: repickWait}, nil
		}
		if cfg.RunTypeID == RunTypeAdvTriggerBet {
			slog.Info("scheme worker trigger skip: previous draw missing after period lock",
				"id", inst.ID, "period", finalPeriod, "pickIssue", pickIssue, "countdown", rem)
			return repickResult{action: repickSkip}, nil
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
