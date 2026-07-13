package schemes

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"caipiao/backend/internal/cloud/lookback"
	"caipiao/backend/internal/db"
	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/guajibet"
	"caipiao/backend/internal/guaji/periodsync"
	"caipiao/backend/internal/lottery"
	"caipiao/backend/internal/member"
	"caipiao/backend/internal/ws"
)

const defaultCountdownReset = 7

// Worker ticks running scheme instances: countdown → bet against lottery draw + scheme config.
type Worker struct {
	pool           *db.Pool
	q              *sqlcdb.Queries
	hub            *ws.Hub
	guajiBets      guajiBetPlacer
	periodSync     *periodsync.Syncer
	tickSec        int32
	countdownReset int32
	betSeq         atomic.Uint64
}

func NewWorker(pool *db.Pool, tickSec int, hub *ws.Hub, periodSync *periodsync.Syncer) *Worker {
	if pool == nil || tickSec <= 0 {
		return nil
	}
	return &Worker{
		pool:           pool,
		q:              sqlcdb.New(pool),
		hub:            hub,
		periodSync:     periodSync,
		tickSec:        int32(tickSec),
		countdownReset: defaultCountdownReset,
	}
}

func (w *Worker) Run(ctx context.Context) {
	if w == nil {
		return
	}
	ticker := time.NewTicker(time.Duration(w.tickSec) * time.Second)
	defer ticker.Stop()
	slog.Info("scheme worker started", "tickSec", w.tickSec)
	for {
		select {
		case <-ctx.Done():
			slog.Info("scheme worker stopped")
			return
		case <-ticker.C:
			w.tick(ctx)
		}
	}
}

func (w *Worker) tick(ctx context.Context) {
	instances, err := w.q.ListRunningSchemeInstances(ctx)
	if err != nil {
		slog.Warn("scheme worker list failed", "err", err)
	} else {
		planMultByMember := make(map[int64]float64, 8)
		for _, row := range instances {
			inst := sqlcdb.SchemeInstanceFromRunningRow(row)
			pm, ok := planMultByMember[inst.MemberID]
			if !ok {
				pm = w.memberPlanMultiplier(ctx, inst.MemberID)
				planMultByMember[inst.MemberID] = pm
			}
			w.tickInstance(ctx, inst, pm)
		}
	}
	w.tickMaintenanceResume(ctx)
}

func (w *Worker) tickInstance(ctx context.Context, inst sqlcdb.SchemeInstance, planMult float64) {
	status, err := w.q.GetSchemeInstanceStatus(ctx, inst.ID)
	if err != nil || status != "running" {
		return
	}

	def, err := w.loadDefinitionForInstance(ctx, inst)
	if err != nil {
		if !isDefinitionNotFound(err) {
			slog.Warn("scheme worker load definition failed", "id", inst.ID, "err", err)
		}
		return
	}
	if reason, ok := w.checkAutoPause(ctx, inst, def); ok {
		w.pauseRunningInstance(ctx, inst, reason, "")
		return
	}
	if w.pauseRunningWithoutGuajiAuth(ctx, inst) {
		return
	}
	if w.gateScheduleBeforeBet(ctx, inst, def.Config) != schemeScheduleOK {
		return
	}
	now := time.Now()

	if w.periodSync != nil {
		if err := w.periodSync.EnsureFreshIfStale(ctx, inst.LotteryCode); err != nil {
			slog.Warn("scheme worker periods cache fallback failed", "id", inst.ID, "lottery", inst.LotteryCode, "err", err)
		}
	}

	if fresh, ferr := w.q.GetSchemeInstanceFull(ctx, inst.ID); ferr == nil {
		inst = fresh
	}

	w.syncRunningCountdown(ctx, inst)

	skipped, err := w.ensureStartPeriodSkipped(ctx, inst)
	if err != nil {
		slog.Warn("scheme worker skip start period failed", "id", inst.ID, "err", err)
	}
	if skipped {
		return
	}

	if inst.StatusReason == StatusReasonAwaitNextBet {
		activated, aerr := w.tryActivateAfterStartPeriod(ctx, inst, def.Config)
		if aerr != nil {
			slog.Warn("scheme worker activate after start period failed", "id", inst.ID, "err", aerr)
			return
		}
		if !activated {
			return
		}
		if fresh, ferr := w.q.GetSchemeInstanceFull(ctx, inst.ID); ferr == nil {
			inst = fresh
		}
	}

	if inst.StatusReason == StatusReasonCloudActive {
		if err := w.q.ClearStartSkipLastSettledCursor(ctx, inst.ID); err != nil {
			slog.Warn("scheme worker clear start skip cursor failed", "id", inst.ID, "err", err)
		} else if fresh, ferr := w.q.GetSchemeInstanceFull(ctx, inst.ID); ferr == nil {
			inst = fresh
		}
	}

	if !w.ensureBetWindowOpen(ctx, inst, now) {
		slog.Debug("scheme worker bet skipped: bet window closed", "id", inst.ID, "lottery", inst.LotteryCode, "simBet", inst.SimBet)
		return
	}

	if w.pauseRunningForSessionLimit(ctx, inst, def.Config) {
		return
	}
	if w.pauseAllRunningForCloudLimit(ctx, inst.MemberID) {
		return
	}

	if w.hasUnsettledGuajiBet(ctx, inst) {
		slog.Debug("scheme worker bet skipped: awaiting previous period settlement", "id", inst.ID)
		return
	}

	if rem, ok := lottery.PeriodsCountdownSec(inst.LotteryCode, now); ok {
		slog.Debug("scheme worker bet window", "id", inst.ID, "lottery", inst.LotteryCode, "countdown", rem, "simBet", inst.SimBet)
	}

	if err := w.placePeriodBet(ctx, inst, w.tickSec, planMult); err != nil {
		if errors.Is(err, errSchemeBetStopped) {
			return
		}
		if errors.Is(err, guajibet.ErrPeriodClosed) {
			slog.Info("scheme worker bet skipped: period closed", "id", inst.ID, "lottery", inst.LotteryCode)
			return
		}
		slog.Warn("scheme worker bet failed", "id", inst.ID, "err", err)
		w.pauseRunningInstance(ctx, inst, StatusReasonBetFailed, guajiBetFailedDetail(err))
	}
}

// ensureStartPeriodSkipped 开启后从平台缓存跳过最近一期。
func (w *Worker) ensureStartPeriodSkipped(ctx context.Context, inst sqlcdb.SchemeInstance) (bool, error) {
	if inst.StatusReason != StatusReasonAwaitNextBet || inst.StartSkipCloseAt.Valid {
		return false, nil
	}
	ok, err := ensureSchemeStartSkipSnapshot(ctx, w.q, w.periodSync, inst)
	if err != nil || !ok {
		return false, err
	}
	return true, nil
}

// tryActivateAfterStartPeriod 跳过的最近一期结束后切换为云端挂机，再允许首投。
func (w *Worker) tryActivateAfterStartPeriod(ctx context.Context, inst sqlcdb.SchemeInstance, cfgBytes []byte) (bool, error) {
	if inst.StatusReason != StatusReasonAwaitNextBet {
		return false, nil
	}
	if !inst.StartSkipCloseAt.Valid {
		if _, err := ensureSchemeStartSkipSnapshot(ctx, w.q, w.periodSync, inst); err != nil {
			slog.Warn("scheme worker ensure start skip snapshot failed", "id", inst.ID, "err", err)
		}
		if fresh, ferr := w.q.GetSchemeInstanceFull(ctx, inst.ID); ferr == nil {
			inst = fresh
		}
	}
	if !schemeStartPeriodEnded(inst, cfgBytes, time.Now()) {
		return false, nil
	}
	n, err := w.q.ActivateSchemeInstanceCloud(ctx, inst.ID)
	if err != nil {
		return false, err
	}
	if n == 0 {
		return false, nil
	}
	if w.periodSync != nil {
		if err := w.periodSync.ForceRefresh(ctx, inst.LotteryCode); err != nil {
			slog.Warn("scheme worker periods refresh after activate failed", "id", inst.ID, "lottery", inst.LotteryCode, "err", err)
		}
	}
	w.notifySchemeInstance(ctx, inst.MemberID, inst.ID, runModeFromSimBet(inst.SimBet), "running", StatusReasonCloudActive)
	slog.Info("scheme worker activated after start period ended",
		"id", inst.ID, "skippedPeriod", inst.LastSettledIssue.String, "simBet", inst.SimBet)
	return true, nil
}

func (w *Worker) ensureBetWindowOpen(ctx context.Context, inst sqlcdb.SchemeInstance, now time.Time) bool {
	if _, ok := lottery.StrictOpenIssueForGuajiBet(inst.LotteryCode); ok {
		return true
	}
	if w == nil || w.periodSync == nil {
		return false
	}
	if err := w.periodSync.ForceRefresh(ctx, inst.LotteryCode); err != nil {
		slog.Debug("scheme worker force refresh before bet failed", "id", inst.ID, "lottery", inst.LotteryCode, "err", err)
	}
	_, ok := lottery.StrictOpenIssueForGuajiBet(inst.LotteryCode)
	return ok
}

func (w *Worker) placePeriodBet(ctx context.Context, inst sqlcdb.SchemeInstance, delta int32, planMult float64) error {
	if inst.StatusReason == StatusReasonAwaitNextBet {
		return nil
	}
	if requiresGuajiRealBet(inst) && !w.guajiRealEnabled() {
		slog.Debug("scheme worker bet skipped: guaji required for real betting", "id", inst.ID)
		return nil
	}

	def, err := w.q.GetSchemeDefinitionByID(ctx, inst.DefinitionID)
	if err != nil {
		return fmt.Errorf("definition: %w", err)
	}

	if fresh, ferr := w.q.GetSchemeInstanceFull(ctx, inst.ID); ferr == nil {
		inst = fresh
	}

	switch w.gateScheduleBeforeBet(ctx, inst, def.Config) {
	case schemeSchedulePastEnd:
		return errSchemeBetStopped
	case schemeScheduleBeforeStart:
		return nil
	}
	if w.pauseRunningForSessionLimit(ctx, inst, def.Config) {
		return nil
	}
	if w.pauseAllRunningForCloudLimit(ctx, inst.MemberID) {
		return errSchemeBetStopped
	}
	if w.hasUnsettledGuajiBet(ctx, inst) {
		return nil
	}

	dedup, derr := w.evaluateGuajiBetDedup(ctx, w.q, inst)
	if derr != nil {
		return derr
	}
	if dedup.Skip {
		slog.Info("scheme worker bet skipped: period dedup",
			"id", inst.ID, "reason", dedup.Reason, "currentOpen", dedup.CurrentOpen, "lastBet", dedup.LastBet, "simBet", inst.SimBet)
		if dedup.CurrentOpen != "" {
			w.syncPeriodBetCursor(ctx, w.q, inst, dedup.CurrentOpen)
		}
		return nil
	}

	draw, ok, err := drawForOpenIssue(ctx, w.q, inst.LotteryCode, dedup.CurrentOpen)
	if err != nil {
		return fmt.Errorf("draw: %w", err)
	}
	if !ok {
		draw = sqlcdb.LotteryDraw{
			LotteryCode: inst.LotteryCode,
			IssueNo:     dedup.CurrentOpen,
			PeriodShort: issuePeriodShort(dedup.CurrentOpen),
		}
	}

	groupIndex := 0
	if inst.RoundIndex > 0 {
		groupIndex = int(inst.RoundIndex)
	}
	cfg := parseSchemeConfig(inst.Kind, def.Config, int(inst.RoundIndex), groupIndex)

	roundIdx := int(inst.RoundIndex)
	if roundIdx < 0 || roundIdx >= len(cfg.Rounds) {
		roundIdx = 0
	}
	round := cfg.Rounds[roundIdx]
	baseCoef := combinedBaseCoef(inst.Multiplier, planMult)
	betMult := effectiveBetMultiple(baseCoef, round)

	// 出号体系：按运行类型决定本期下注内容（与倍投体系独立，v8 §0）
	dec := w.resolvePick(ctx, cfg, inst, draw)
	if dec.Skip {
		slog.Debug("scheme worker bet skipped: pick strategy skip", "id", inst.ID, "period", draw.IssueNo, "runType", cfg.RunTypeID)
		skipPeriod := draw.IssueNo
		if p, ok := thirdPartyOpenPeriod(inst.LotteryCode); ok {
			skipPeriod = p
		}
		// 本期跳过：仅推进第三方期号游标，不下注
		if _, err := w.q.ApplySchemeInstanceBet(ctx, sqlcdb.ApplySchemeInstanceBetParams{
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
		_ = appendPickSkipAudit(ctx, w.q, inst, draw.IssueNo)
		return nil
	}
	betContent := dec.Content
	if strings.TrimSpace(betContent) == "" {
		betContent = cfg.GroupContent
	}

	balls := sqlcdb.ParseDrawBalls(draw.Balls)
	playEval := evaluatePlayHit(cfg.Play, balls, betContent, cfg.Contrary, cfg.ContraryPlan, cfg.Play.PositionIdx)
	amount := calcBetAmount(playEval.BetUnits, betMult, cfg.BetUnitYuan)
	pnl := calcPnLWithOdds(amount, playEval.Hit, playEval.Odds)

	status := "miss"
	if playEval.Hit {
		status = "hit"
	}

	nextPickIndex, nextCurrentPick, nextLastDirection := advancePickState(cfg, inst, dec, playEval.Hit)
	nextRound := nextRoundIndex(cfg.Rounds, roundIdx, playEval.Hit)

	settings := w.loadLookbackSettings(ctx, inst.MemberID)
	var overallRT lookback.Runtime
	if lookback.AppliesTo(settings, inst.SimBet) && settings.Judgment == lookback.JudgmentOverall {
		overallRT = w.loadLookbackRuntime(ctx, inst.MemberID, inst.SimBet)
	}
	lbEval := evaluateLookback(settings, inst.SimBet, numericToFloat(inst.LookbackPnl), overallRT, draw.IssueNo, pnl, playEval.Hit)
	resetIndividual := lbEval.ResetIndividual
	resetOverall := lbEval.ResetOverall
	overallRT = lbEval.OverallRT
	trackOverall := lbEval.TrackOverall

	if resetIndividual || resetOverall {
		nextRound = 0
	}

	if !inst.SimBet {
		cat, cerr := w.q.GetLotteryCatalogByCode(ctx, inst.LotteryCode)
		if cerr != nil {
			return fmt.Errorf("lottery catalog: %w", cerr)
		}
		if _, _, err := resolveOutboundPlayCode(ctx, w.q, cfg, textVal(cat.PlayTemplate)); err != nil {
			w.pauseRunningInstance(ctx, inst, StatusReasonBetFailed, guajiBetFailedDetail(err))
			return errSchemeBetStopped
		}
	}

	recordNo := fmt.Sprintf("CB%d%04d", time.Now().UTC().UnixNano(), w.betSeq.Add(1)%10000)

	recordStatus := status
	recordPnl := pnl
	guajiReal := w.usesGuajiThirdParty(inst)
	if guajiReal {
		recordStatus = "pending"
		recordPnl = 0
	}

	if !guajiReal {
		reserved, err := w.reserveCloudBetPeriod(ctx, inst, draw, cfg, recordNo, recordStatus, amount, recordPnl, betContent, betMult, roundIdx, len(cfg.Rounds))
		if err != nil {
			return err
		}
		if !reserved {
			return nil
		}
	}

	committed := false
	guajiAccepted := false
	var betMeta schemeGuajiBetMeta
	defer func() {
		if committed || guajiReal {
			if committed || !guajiAccepted {
				return
			}
			w.finalizeCloudBetAfterGuaji(ctx, inst, cfg, recordNo, amount, betMult, roundIdx, betContent, betMeta)
			return
		}
		if derr := w.q.DeleteCloudBetRecordForInstancePeriod(ctx, inst.ID, draw.IssueNo); derr != nil {
			slog.Debug("scheme worker cleanup reserve failed", "id", inst.ID, "period", draw.IssueNo, "err", derr)
		}
	}()

	tx, err := w.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	qtx := w.q.WithTx(tx)

	if _, err := qtx.LockSchemeInstanceForBet(ctx, inst.ID); err != nil {
		return err
	}
	running, err := qtx.GetSchemeInstanceStatus(ctx, inst.ID)
	if err != nil {
		return err
	}
	if running != "running" {
		slog.Debug("scheme worker bet aborted: instance no longer running", "id", inst.ID, "period", draw.IssueNo)
		return nil
	}

	if !guajiReal {
		dedup, herr := w.evaluateGuajiBetDedup(ctx, qtx, inst)
		if herr != nil {
			return herr
		}
		if dedup.Skip {
			slog.Debug("scheme worker bet skipped: period dedup in tx",
				"id", inst.ID, "reason", dedup.Reason, "currentOpen", dedup.CurrentOpen, "lastBet", dedup.LastBet, "simBet", inst.SimBet)
			if dedup.CurrentOpen != "" {
				w.syncPeriodBetCursor(ctx, qtx, inst, dedup.CurrentOpen)
			}
			return nil
		}
		draw.IssueNo = dedup.CurrentOpen
	}

	var guajiTargetPeriodNo string
	if guajiReal {
		dedup, herr := w.evaluateGuajiBetDedup(ctx, qtx, inst)
		if herr != nil {
			return herr
		}
		if dedup.Skip {
			slog.Debug("scheme worker bet skipped: guaji period dedup in tx",
				"id", inst.ID, "reason", dedup.Reason, "currentOpen", dedup.CurrentOpen, "lastBet", dedup.LastBet)
			if dedup.CurrentOpen != "" {
				w.syncPeriodBetCursor(ctx, qtx, inst, dedup.CurrentOpen)
			}
			return nil
		}
		guajiTargetPeriodNo = dedup.CurrentOpen
		draw.IssueNo = guajiTargetPeriodNo
		claimed, cerr := qtx.TryClaimCloudBetPeriod(ctx, sqlcdb.ReserveCloudBetPeriodParams{
			RecordNo:       recordNo,
			MemberID:       inst.MemberID,
			SimBet:         inst.SimBet,
			SchemeID:       inst.ID,
			SchemeName:     inst.SchemeName,
			PeriodNo:       guajiTargetPeriodNo,
			PlayType:       cfg.PlayTypeLabel,
			Multiplier:     strconv.Itoa(betMultipleAsInt(betMult)),
			RoundLabel:     strconv.Itoa(roundIdx + 1),
			Amount:         numericFromFloat(amount),
			Pnl:            numericFromFloat(0),
			Status:         "pending",
			BetContent:     betContent,
			GuajiAccountID: activeGuajiAccountIDForInst(ctx, qtx, inst),
		})
		if cerr != nil {
			return cerr
		}
		if !claimed {
			w.syncPeriodBetCursor(ctx, qtx, inst, guajiTargetPeriodNo)
			slog.Debug("scheme worker bet skipped: period claim conflict", "id", inst.ID, "period", guajiTargetPeriodNo)
			return nil
		}
		betMeta, err = w.placeGuajiSchemeBet(ctx, qtx, inst, cfg, draw, betContent, amount, playEval.BetUnits, betMult)
		if err != nil {
			stopErr := w.stopAfterThirdPartyBetFailed(ctx, qtx, inst, amount, err)
			if errors.Is(stopErr, guajibet.ErrPeriodClosed) {
				return guajibet.ErrPeriodClosed
			}
			if errors.Is(stopErr, errSchemeBetStopped) {
				if cerr := tx.Commit(ctx); cerr != nil {
					return cerr
				}
				committed = true
				reason := betFailureReason(err)
				w.notifySchemeInstance(ctx, inst.MemberID, inst.ID, runModeFromSimBet(inst.SimBet), "pending", reason)
				slog.Warn("scheme worker stopped: third party bet failed",
					"instanceId", inst.ID, "memberId", inst.MemberID, "period", draw.IssueNo, "reason", reason, "err", err)
				return errSchemeBetStopped
			}
			return stopErr
		}
		if betMeta.Amount > 0 {
			amount = betMeta.Amount
		}
		guajiAccepted = true
	}

	acceptedPeriod := strings.TrimSpace(draw.IssueNo)
	if guajiReal {
		acceptedPeriod = strings.TrimSpace(betMeta.Periods)
		if acceptedPeriod == "" {
			return fmt.Errorf("%w: upstream did not return periods", guajibet.ErrPlaceRejected)
		}
		if acceptedPeriod != guajiTargetPeriodNo {
			if err := qtx.MoveCloudBetRecordPeriod(ctx, inst.ID, guajiTargetPeriodNo, acceptedPeriod); err != nil {
				return err
			}
		}
		if err := qtx.UpdateCloudBetRecordGuajiMeta(ctx, inst.ID, acceptedPeriod,
			pgtype.Text{String: betMeta.ThirdPartyBetID, Valid: betMeta.ThirdPartyBetID != ""},
			pgtype.Text{String: betMeta.OrderNo, Valid: betMeta.OrderNo != ""},
			guajiPeriodsPgtext(betMeta.Periods),
			numericFromFloat(0), "pending",
			numericFromFloat(amount),
		); err != nil {
			return err
		}
	}

	applyPnl := pnl
	applyLookbackPnl := pnl
	applyRoundIndex := int32(nextRound)
	applyPickIndex := nextPickIndex
	applyCurrentPick := nextCurrentPick
	applyLastDirection := nextLastDirection
	if guajiReal {
		applyPnl = 0
		applyLookbackPnl = 0
		resetIndividual = false
		resetOverall = false
		// 正式盘：轮次/出号游标待派奖后按实际中/未中推进，下单时仅累加流水与期号游标。
		applyRoundIndex = inst.RoundIndex
		applyPickIndex = inst.PickIndex
		applyCurrentPick = inst.CurrentPick
		applyLastDirection = inst.LastDirection
	}

	if _, err := qtx.ApplySchemeInstanceBet(ctx, sqlcdb.ApplySchemeInstanceBetParams{
		ID:               inst.ID,
		CountdownSec:     w.periodCountdownForInst(inst, time.Now()),
		Turnover:         numericFromFloat(amount),
		Pnl:              numericFromFloat(applyPnl),
		Multiplier:       inst.Multiplier,
		RoundIndex:       applyRoundIndex,
		LastSettledIssue: pgtype.Text{String: acceptedPeriod, Valid: acceptedPeriod != ""},
		LookbackPnl:      numericFromFloat(applyLookbackPnl),
		PickIndex:        applyPickIndex,
		CurrentPick:      applyCurrentPick,
		LastDirection:    applyLastDirection,
	}); err != nil {
		return err
	}

	if trackOverall && !guajiReal {
		if err := w.saveLookbackRuntime(ctx, qtx, inst.MemberID, inst.SimBet, overallRT, resetOverall); err != nil {
			return err
		}
	}

	if (resetIndividual || resetOverall) && !guajiReal {
		if err := w.applyLookbackResets(ctx, qtx, inst, acceptedPeriod, resetIndividual, resetOverall); err != nil {
			return err
		}
		if resetIndividual && !resetOverall {
			slog.Info("lookback reset individual", "instanceId", inst.ID, "memberId", inst.MemberID)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}
	committed = true
	if !guajiReal {
		if fresh, ferr := w.q.GetSchemeInstanceFull(ctx, inst.ID); ferr == nil {
			w.pauseRunningForSessionLimit(ctx, fresh, def.Config)
			w.pauseAllRunningForCloudLimit(ctx, inst.MemberID)
		}
	}
	if st, serr := w.q.GetSchemeInstanceStatus(ctx, inst.ID); serr == nil && st == "running" {
		w.notifySchemeInstance(ctx, inst.MemberID, inst.ID, runModeFromSimBet(inst.SimBet), "running", StatusReasonCloudActive)
	}
	slog.Info("scheme worker bet placed", "instanceId", inst.ID, "memberId", inst.MemberID, "period", acceptedPeriod, "guajiPeriod", betMeta.Periods, "amount", amount, "simBet", inst.SimBet, "thirdParty", w.usesGuajiThirdParty(inst))
	return nil
}

func (w *Worker) reserveCloudBetPeriod(
	ctx context.Context,
	inst sqlcdb.SchemeInstance,
	draw sqlcdb.LotteryDraw,
	cfg parsedSchemeConfig,
	recordNo, recordStatus string,
	amount, recordPnl float64,
	betContent string,
	mult float64,
	roundIdx, roundCount int,
) (bool, error) {
	tx, err := w.pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx)

	qtx := w.q.WithTx(tx)
	if _, err := qtx.LockSchemeInstanceForBet(ctx, inst.ID); err != nil {
		return false, err
	}
	status, err := qtx.GetSchemeInstanceStatus(ctx, inst.ID)
	if err != nil {
		return false, err
	}
	if status != "running" {
		return false, nil
	}

	dedup, err := w.evaluateGuajiBetDedup(ctx, qtx, inst)
	if err != nil {
		return false, err
	}
	if dedup.Skip {
		if dedup.CurrentOpen != "" {
			w.syncPeriodBetCursor(ctx, qtx, inst, dedup.CurrentOpen)
		}
		if err := tx.Commit(ctx); err != nil {
			return false, err
		}
		slog.Debug("scheme worker bet skipped: period dedup on reserve",
			"id", inst.ID, "reason", dedup.Reason, "currentOpen", dedup.CurrentOpen, "lastBet", dedup.LastBet, "simBet", inst.SimBet)
		return false, nil
	}
	periodNo := dedup.CurrentOpen

	ok, err := qtx.ReserveCloudBetPeriod(ctx, sqlcdb.ReserveCloudBetPeriodParams{
		RecordNo:       recordNo,
		MemberID:       inst.MemberID,
		SimBet:         inst.SimBet,
		SchemeID:       inst.ID,
		SchemeName:     inst.SchemeName,
		PeriodNo:       periodNo,
		PlayType:       cfg.PlayTypeLabel,
		Multiplier:     strconv.Itoa(betMultipleAsInt(mult)),
		RoundLabel:     strconv.Itoa(roundIdx + 1),
		Amount:         numericFromFloat(amount),
		Pnl:            numericFromFloat(recordPnl),
		Status:         recordStatus,
		BetContent:     betContent,
		GuajiAccountID: activeGuajiAccountIDForInst(ctx, qtx, inst),
	})
	if err != nil {
		return false, err
	}
	if !ok {
		if err := tx.Commit(ctx); err != nil {
			return false, err
		}
		slog.Debug("scheme worker bet skipped: period reserve conflict", "id", inst.ID, "period", periodNo)
		return false, nil
	}
	if err := tx.Commit(ctx); err != nil {
		return false, err
	}
	return true, nil
}

func activeGuajiAccountIDForInst(ctx context.Context, q *sqlcdb.Queries, inst sqlcdb.SchemeInstance) pgtype.Int8 {
	if inst.SimBet || q == nil {
		return pgtype.Int8{}
	}
	id, err := member.LookupActiveGuajiAccountID(ctx, q, inst.MemberID)
	if err != nil || !id.Valid {
		return pgtype.Int8{}
	}
	return id
}
