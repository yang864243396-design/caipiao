package schemes

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"caipiao/backend/internal/cloud/lookback"
	"caipiao/backend/internal/db"
	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/guaji"
	"caipiao/backend/internal/guaji/periodsync"
	"caipiao/backend/internal/guajibet"
	"caipiao/backend/internal/lottery"
	"caipiao/backend/internal/member"
	"caipiao/backend/internal/ws"
)

const defaultCountdownReset = 7

type periodRefreshRequester interface {
	RequestRefresh(lotteryCode string)
}

// Worker ticks running scheme instances: countdown → bet against lottery draw + scheme config.
type Worker struct {
	pool           *db.Pool
	q              *sqlcdb.Queries
	hub            *ws.Hub
	guajiBets      guajiBetPlacer
	periodSync     *periodsync.Syncer
	periodRefresh  periodRefreshRequester
	tickSec        int32
	concurrency    int32
	placeSem       chan struct{} // 真下单全站有界并发；nil 表示不额外限流
	countdownReset int32
	betSeq         atomic.Uint64
}

func NewWorker(pool *db.Pool, tickSec int, hub *ws.Hub, periodSync *periodsync.Syncer) *Worker {
	if pool == nil || tickSec <= 0 {
		return nil
	}
	w := &Worker{
		pool:           pool,
		q:              sqlcdb.New(pool),
		hub:            hub,
		periodSync:     periodSync,
		tickSec:        int32(tickSec),
		concurrency:    int32(defaultSchemeWorkerConcurrency),
		countdownReset: defaultCountdownReset,
	}
	w.SetPlaceConcurrency(defaultSchemeWorkerPlaceConcurrency)
	return w
}

func (w *Worker) SetPeriodRefreshRequester(requester periodRefreshRequester) {
	if w == nil {
		return
	}
	w.periodRefresh = requester
}

func (w *Worker) requestPeriodRefresh(lotteryCode string) {
	if w == nil || w.periodRefresh == nil {
		return
	}
	w.periodRefresh.RequestRefresh(lotteryCode)
}

func (w *Worker) Run(ctx context.Context) {
	if w == nil {
		return
	}
	ticker := time.NewTicker(time.Duration(w.tickSec) * time.Second)
	defer ticker.Stop()
	placeN := 0
	if w.placeSem != nil {
		placeN = cap(w.placeSem)
	}
	slog.Info("scheme worker started", "tickSec", w.tickSec, "concurrency", w.concurrency, "placeConcurrency", placeN)
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
	} else if len(instances) > 0 {
		w.prefetchPeriodSync(ctx, uniqueLotteryCodes(instances))
		instances = prioritizeOpenBetWindow(instances)
		planMultByMember := w.preloadPlanMultipliers(ctx, uniqueMemberIDs(instances))
		gate := newBetWindowGate(w)
		conc := int(w.concurrency)
		if conc <= 0 {
			conc = defaultSchemeWorkerConcurrency
		}
		planOf := func(memberID int64) float64 {
			if pm, ok := planMultByMember[memberID]; ok {
				return pm
			}
			return 1
		}
		if conc == 1 || len(instances) == 1 {
			for _, row := range instances {
				inst := sqlcdb.SchemeInstanceFromRunningRow(row)
				w.tickInstance(ctx, inst, planOf(inst.MemberID), gate)
			}
		} else {
			sem := make(chan struct{}, conc)
			var wg sync.WaitGroup
			for _, row := range instances {
				inst := sqlcdb.SchemeInstanceFromRunningRow(row)
				pm := planOf(inst.MemberID)
				wg.Add(1)
				sem <- struct{}{}
				go func(inst sqlcdb.SchemeInstance, planMult float64) {
					defer wg.Done()
					defer func() { <-sem }()
					defer func() {
						if r := recover(); r != nil {
							slog.Error("scheme worker tickInstance panic", "id", inst.ID, "panic", r)
						}
					}()
					w.tickInstance(ctx, inst, planMult, gate)
				}(inst, pm)
			}
			wg.Wait()
		}
	}
	w.tickSimSettlements(ctx)
	w.tickMaintenanceResume(ctx)
}

func (w *Worker) tickInstance(ctx context.Context, inst sqlcdb.SchemeInstance, planMult float64, gate *betWindowGate) {
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

	if !w.ensureBetWindowOpen(ctx, inst, now, gate) {
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
	w.requestPeriodRefresh(inst.LotteryCode)
	w.notifySchemeInstance(ctx, inst.MemberID, inst.ID, runModeFromSimBet(inst.SimBet), "running", StatusReasonCloudActive)
	slog.Info("scheme worker activated after start period ended",
		"id", inst.ID, "skippedPeriod", inst.LastSettledIssue.String, "simBet", inst.SimBet)
	return true, nil
}

func (w *Worker) ensureBetWindowOpen(ctx context.Context, inst sqlcdb.SchemeInstance, now time.Time, gate *betWindowGate) bool {
	_ = now
	if gate != nil {
		return gate.ensureOpen(ctx, inst.LotteryCode)
	}
	if _, ok := lottery.StrictOpenIssueForGuajiBet(inst.LotteryCode); ok {
		return true
	}
	w.requestPeriodRefresh(inst.LotteryCode)
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
		// 仅本方案已在该第三方期下过注时对齐游标。
		// period_record_exists / period_cursor_taken 再 sync
		// 会把未出手的期号提前写进 last_settled，开某投某等依赖上期的方案会整段失投。
		if dedup.Reason == "same_third_party_period" && dedup.CurrentOpen != "" {
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
	cfg.Play = attachOddsBase(cfg.Play, inst.LotteryCode)

	roundIdx := int(inst.RoundIndex)
	if roundIdx < 0 || roundIdx >= len(cfg.Rounds) {
		roundIdx = 0
	}
	round := cfg.Rounds[roundIdx]
	baseCoef := combinedBaseCoef(inst.Multiplier, planMult)
	betMult := effectiveBetMultiple(baseCoef, round)

	// 冷热 / 开某投某：都依赖「相邻上期」开奖。上期未入库时勿用更早开奖硬投。
	pickIssue := strings.TrimSpace(draw.IssueNo)
	if cfg.RunTypeID == RunTypeHotColdWarm || cfg.RunTypeID == RunTypeAdvTriggerBet {
		needsPrev := true
		if cfg.RunTypeID == RunTypeHotColdWarm {
			curPick := strings.TrimSpace(inst.CurrentPick)
			needsPrev = curPick == "" || hotColdPickNeedsRebuild(cfg, curPick)
		}
		if needsPrev && !w.hotColdPreviousDrawReady(ctx, inst.LotteryCode, draw.IssueNo) {
			rem, ok := lottery.PeriodsCountdownSec(inst.LotteryCode, time.Now())
			// 开某投某：上期未入库则只等待，绝不推进 last_settled 游标。
			// 旧逻辑 rem<8 即 skipPeriodPick，且期号翻页时可能把「下一期」游标提前占掉，
			// 导致 1 分彩（history 延迟约一期）整段永远 period_cursor_taken。
			if cfg.RunTypeID == RunTypeAdvTriggerBet {
				slog.Debug("scheme worker bet deferred: waiting previous draw",
					"id", inst.ID, "period", draw.IssueNo, "countdown", rem, "countdownOK", ok)
				return nil
			}
			if !ok || rem >= hotColdPrevDrawWaitMinSec {
				slog.Debug("scheme worker bet deferred: waiting previous draw",
					"id", inst.ID, "period", draw.IssueNo, "runType", cfg.RunTypeID, "countdown", rem)
				return nil
			}
			// 冷热临近封盘：降级用更早开奖统计继续出号
			slog.Info("scheme worker hot/cold proceed without previous draw",
				"id", inst.ID, "period", draw.IssueNo, "countdown", rem)
		}
	}

	// 出号体系：按运行类型决定本期下注内容（与倍投体系独立，v8 §0）
	dec := w.resolvePick(ctx, cfg, inst, draw)
	if dec.Skip {
		slog.Info("scheme worker bet skipped: pick strategy skip",
			"id", inst.ID, "period", draw.IssueNo, "runType", cfg.RunTypeID)
		return w.skipPeriodPick(ctx, inst, draw.IssueNo, cfg.RunTypeID)
	}
	betContent := normalizeResolvedBetContent(cfg, &dec)
	// 混合组选 / 组三组六号池不足：本期跳过。随机出号无解走连续计数（满 10 期停方案）。
	if strings.TrimSpace(betContent) == "" {
		if cfg.RunTypeID == RunTypeRandomDraw {
			return w.skipRandomDrawUnsolvable(ctx, inst, draw.IssueNo)
		}
		if shouldSkipZeroBetUnits(cfg.Play) {
			slog.Info("scheme worker skip: empty pick content",
				"id", inst.ID, "period", draw.IssueNo, "betMode", cfg.Play.BetMode, "runType", cfg.RunTypeID)
			return w.skipPeriodPick(ctx, inst, draw.IssueNo, cfg.RunTypeID)
		}
	}

	balls := sqlcdb.ParseDrawBalls(draw.Balls)
	playEval := evaluatePlayHit(cfg.Play, balls, betContent, cfg.Contrary, cfg.ContraryPlan, cfg.Play.PositionIdx)
	// 先按 wire 组合注数校准（组选20 等双区若 evaluate 未覆盖，勿在校准前误报 0 注停方案）
	syncEvalBetUnitsWithWire(cfg.Play, betContent, &playEval)
	if playEval.BetUnits <= 0 {
		if cfg.RunTypeID == RunTypeRandomDraw {
			return w.skipRandomDrawUnsolvable(ctx, inst, draw.IssueNo)
		}
		if shouldSkipZeroBetUnits(cfg.Play) {
			slog.Info("scheme worker skip: zero bet units",
				"id", inst.ID, "period", draw.IssueNo, "content", betContent, "betMode", cfg.Play.BetMode)
			return w.skipPeriodPick(ctx, inst, draw.IssueNo, cfg.RunTypeID)
		}
		w.pauseRunningInstance(ctx, inst, StatusReasonBetFailed, guajibet.ErrZeroBets.Error())
		return errSchemeBetStopped
	}
	// 超该玩法第三方单组上限：随机出号再重抽；仍超限则计入连续无解。其它模式暂停。
	if max := maxBetUnitsForPlay(cfg.Play); max > 0 && playEval.BetUnits > max {
		if cfg.RunTypeID == RunTypeRandomDraw {
			next, ok := resolveRandomDrawUnderMax(cfg, "")
			if !ok {
				return w.skipRandomDrawUnsolvable(ctx, inst, draw.IssueNo)
			}
			betContent = next
			dec.Content = next
			playEval = evaluatePlayHit(cfg.Play, balls, betContent, cfg.Contrary, cfg.ContraryPlan, cfg.Play.PositionIdx)
			syncEvalBetUnitsWithWire(cfg.Play, betContent, &playEval)
			if playEval.BetUnits <= 0 || playEval.BetUnits > max {
				return w.skipRandomDrawUnsolvable(ctx, inst, draw.IssueNo)
			}
		} else {
			detail := errMaxBetUnitsExceeded(max).Error()
			slog.Warn("scheme worker pause: bet units over max",
				"id", inst.ID, "runType", cfg.RunTypeID, "units", playEval.BetUnits, "max", max, "content", betContent)
			w.pauseRunningInstance(ctx, inst, StatusReasonBetFailed, detail)
			return errSchemeBetStopped
		}
	}
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
		if _, subPlay, err := resolveOutboundPlayCode(ctx, w.q, cfg, textVal(cat.PlayTemplate)); err != nil {
			w.pauseRunningInstance(ctx, inst, StatusReasonBetFailed, guajiBetFailedDetail(err))
			return errSchemeBetStopped
		} else if label := strings.TrimSpace(subPlay.Label); label != "" {
			cfg.SubPlayLabel = label
		}
	}

	recordNo := fmt.Sprintf("CB%d%04d", time.Now().UTC().UnixNano(), w.betSeq.Add(1)%10000)
	playTypeLabel := cloudPlayTypeLabel(cfg.PlayTypeLabel, cfg.SubPlayLabel)
	if playTypeLabel == "" {
		playTypeLabel = cfg.PlayTypeLabel
	}
	betUnits := playEval.BetUnits

	recordStatus := status
	recordPnl := pnl
	guajiReal := w.usesGuajiThirdParty(inst)
	// 模拟盘与正式盘一致：当期先占位 pending，等真实开奖入库后再验奖；
	// 禁止对未开奖的开放期即时判负（空球号几乎总 miss，表现为「一直挂」）。
	deferSettle := guajiReal || inst.SimBet
	if deferSettle {
		recordStatus = "pending"
		recordPnl = 0
	}

	if !guajiReal {
		reserved, err := w.reserveCloudBetPeriod(ctx, inst, draw, cfg, recordNo, recordStatus, amount, recordPnl, betContent, betMult, roundIdx, betUnits, playTypeLabel)
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
	var guajiTargetPeriodNo string
	defer func() {
		if committed || guajiReal {
			if committed || !guajiAccepted {
				return
			}
			w.finalizeCloudBetAfterGuaji(ctx, inst, cfg, recordNo, amount, betMult, roundIdx, betContent, betMeta, playEval.BetUnits, guajiTargetPeriodNo)
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
	rollbackTx := true
	defer func() {
		if rollbackTx {
			_ = tx.Rollback(ctx)
		}
	}()

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

	// 模拟盘：期号已在 reserveCloudBetPeriod 占位；此处不可再跑 evaluateGuajiBetDedup，
	// 否则 CloudBetPeriodHandled 会把刚插入的 cloud_bet_records 判成 period_record_exists，
	// 返回后 defer 删除占位，表现为永远无模拟投注记录/流水。

	if guajiReal {
		dedup, herr := w.evaluateGuajiBetDedup(ctx, qtx, inst)
		if herr != nil {
			return herr
		}
		if dedup.Skip {
			slog.Debug("scheme worker bet skipped: guaji period dedup in tx",
				"id", inst.ID, "reason", dedup.Reason, "currentOpen", dedup.CurrentOpen, "lastBet", dedup.LastBet)
			if dedup.Reason == "same_third_party_period" && dedup.CurrentOpen != "" {
				w.syncPeriodBetCursor(ctx, qtx, inst, dedup.CurrentOpen)
			}
			return nil
		}
		guajiTargetPeriodNo = dedup.CurrentOpen
		draw.IssueNo = guajiTargetPeriodNo
		// 出号期与最终可投期不一致时（periods 缓存跨期）必须按最终期重算。
		// 开某投某尤其危险：按上期(N-1)映射出的号若落到 N+1，等价于错用上上期。
		if cfg.RunTypeID == RunTypeAdvTriggerBet || guajiTargetPeriodNo != pickIssue {
			repick, rerr := w.repickForFinalPeriod(ctx, cfg, inst, &draw, guajiTargetPeriodNo, pickIssue)
			if rerr != nil {
				return rerr
			}
			switch repick.action {
			case repickWait:
				return nil // 事务回滚，下 tick 再等上期
			case repickSkip:
				w.syncPeriodBetCursor(ctx, qtx, inst, guajiTargetPeriodNo)
				if err := w.skipPeriodPickWithQ(ctx, qtx, inst, guajiTargetPeriodNo, cfg.RunTypeID); err != nil {
					return err
				}
				if err := tx.Commit(ctx); err != nil {
					return err
				}
				rollbackTx = false
				committed = true
				return nil
			case repickOK:
				dec = repick.dec
				betContent = repick.content
				playEval = repick.playEval
				syncEvalBetUnitsWithWire(cfg.Play, betContent, &playEval)
				if playEval.BetUnits <= 0 {
					if shouldSkipZeroBetUnits(cfg.Play) {
						w.syncPeriodBetCursor(ctx, qtx, inst, guajiTargetPeriodNo)
						if err := w.skipPeriodPickWithQ(ctx, qtx, inst, guajiTargetPeriodNo, cfg.RunTypeID); err != nil {
							return err
						}
						if err := tx.Commit(ctx); err != nil {
							return err
						}
						rollbackTx = false
						committed = true
						return nil
					}
					w.pauseRunningInstance(ctx, inst, StatusReasonBetFailed, guajibet.ErrZeroBets.Error())
					return errSchemeBetStopped
				}
				if max := maxBetUnitsForPlay(cfg.Play); max > 0 && playEval.BetUnits > max {
					if cfg.RunTypeID == RunTypeRandomDraw {
						next, ok := resolveRandomDrawUnderMax(cfg, "")
						if !ok {
							w.syncPeriodBetCursor(ctx, qtx, inst, guajiTargetPeriodNo)
							if err := w.skipRandomDrawUnsolvableWithQ(ctx, qtx, inst, guajiTargetPeriodNo); err != nil {
								return err
							}
							if err := tx.Commit(ctx); err != nil {
								return err
							}
							rollbackTx = false
							committed = true
							return nil
						}
						betContent = next
						dec.Content = next
						playEval = evaluatePlayHit(cfg.Play, sqlcdb.ParseDrawBalls(draw.Balls), betContent, cfg.Contrary, cfg.ContraryPlan, cfg.Play.PositionIdx)
						syncEvalBetUnitsWithWire(cfg.Play, betContent, &playEval)
						if playEval.BetUnits <= 0 || playEval.BetUnits > max {
							w.syncPeriodBetCursor(ctx, qtx, inst, guajiTargetPeriodNo)
							if err := w.skipRandomDrawUnsolvableWithQ(ctx, qtx, inst, guajiTargetPeriodNo); err != nil {
								return err
							}
							if err := tx.Commit(ctx); err != nil {
								return err
							}
							rollbackTx = false
							committed = true
							return nil
						}
					} else {
						w.pauseRunningInstance(ctx, inst, StatusReasonBetFailed, errMaxBetUnitsExceeded(max).Error())
						return errSchemeBetStopped
					}
				}
				amount = calcBetAmount(playEval.BetUnits, betMult, cfg.BetUnitYuan)
				pnl = calcPnLWithOdds(amount, playEval.Hit, playEval.Odds)
				betUnits = playEval.BetUnits
				status = "miss"
				if playEval.Hit {
					status = "hit"
				}
				nextPickIndex, nextCurrentPick, nextLastDirection = advancePickState(cfg, inst, dec, playEval.Hit)
				nextRound = nextRoundIndex(cfg.Rounds, roundIdx, playEval.Hit)
				if resetIndividual || resetOverall {
					nextRound = 0
				}
			}
		}
		claimed, cerr := qtx.TryClaimCloudBetPeriod(ctx, sqlcdb.ReserveCloudBetPeriodParams{
			RecordNo:       recordNo,
			MemberID:       inst.MemberID,
			SimBet:         inst.SimBet,
			SchemeID:       inst.ID,
			SchemeName:     inst.SchemeName,
			PeriodNo:       guajiTargetPeriodNo,
			PlayType:       playTypeLabel,
			Multiplier:     strconv.Itoa(betMultipleAsInt(betMult)),
			RoundLabel:     betRoundLabel(cfg, roundIdx, int(inst.PickIndex)),
			Amount:         numericFromFloat(amount),
			Pnl:            numericFromFloat(0),
			Status:         "pending",
			BetContent:     betContent,
			GuajiAccountID: activeGuajiAccountIDForInst(ctx, qtx, inst),
			Currency:       cfg.Currency,
			LotteryCode:    inst.LotteryCode,
			LotteryLabel:   inst.LotteryLabel,
			DefinitionID:   inst.DefinitionID,
			BetUnits:       betUnits,
		})
		if cerr != nil {
			return cerr
		}
		if !claimed {
			w.syncPeriodBetCursor(ctx, qtx, inst, guajiTargetPeriodNo)
			slog.Debug("scheme worker bet skipped: period claim conflict", "id", inst.ID, "period", guajiTargetPeriodNo)
			return nil
		}
		// 占位必须先提交，再调第三方。若占位与 PlaceRealBet 同事务，
		// 接单成功后本地 InsertBetOrder/Commit 失败会回滚占位，下 tick 再次 Place → 同期限连打。
		if err := tx.Commit(ctx); err != nil {
			return err
		}
		rollbackTx = false
		committed = true

		betMeta, err = w.placeGuajiSchemeBet(ctx, w.q, inst, cfg, draw, betContent, amount, playEval.BetUnits, betMult)
		if err != nil {
			// 仅「未发第三方请求」的预检失败可删占位重试；其余一律保留占位+推进游标，
			// 防止上游已接单而本地释放后同期限连打。
			if errors.Is(err, errGuajiPlacePreflight) {
				if derr := w.q.DeleteCloudBetRecordForInstancePeriod(ctx, inst.ID, guajiTargetPeriodNo); derr != nil {
					slog.Debug("scheme worker release claim after preflight fail", "id", inst.ID, "period", guajiTargetPeriodNo, "err", derr)
				}
			} else {
				w.syncPeriodBetCursor(ctx, w.q, inst, guajiTargetPeriodNo)
				slog.Error("scheme worker keep claim after place fail",
					"instanceId", inst.ID, "period", guajiTargetPeriodNo, "err", err)
			}
			if errors.Is(err, guajibet.ErrPeriodClosed) {
				return guajibet.ErrPeriodClosed
			}
			if guaji.IsRetryableTransportError(err) {
				// 占位已保留：下 tick 会被 period_record_exists 挡住，不会重投；
				// 若实为未接单，待期号翻页后自然越过该占位。
				slog.Warn("scheme worker bet deferred: transient upstream (claim kept)",
					"instanceId", inst.ID, "memberId", inst.MemberID, "period", draw.IssueNo, "err", err)
				return nil
			}
			// 随机出号超限（本端或第三方文案）：计入连续无解；满 10 期停方案。
			if cfg.RunTypeID == RunTypeRandomDraw && isBetUnitsExceededError(err) {
				return w.skipRandomDrawUnsolvable(ctx, inst, guajiTargetPeriodNo)
			}
			placeErr := err
			stopErr := w.stopAfterThirdPartyBetFailed(ctx, w.q, inst, amount, placeErr)
			if errors.Is(stopErr, guajibet.ErrPeriodClosed) {
				return guajibet.ErrPeriodClosed
			}
			if errors.Is(stopErr, errSchemeBetStopped) {
				reason := betFailureReason(placeErr)
				w.notifySchemeInstance(ctx, inst.MemberID, inst.ID, runModeFromSimBet(inst.SimBet), "pending", reason)
				slog.Warn("scheme worker stopped: third party bet failed",
					"instanceId", inst.ID, "memberId", inst.MemberID, "period", draw.IssueNo, "reason", reason, "err", placeErr)
				return errSchemeBetStopped
			}
			if stopErr != nil {
				slog.Warn("scheme worker stop after bet failed also errored",
					"instanceId", inst.ID, "placeErr", placeErr, "stopErr", stopErr)
				w.pauseRunningInstance(ctx, inst, StatusReasonBetFailed, guajiBetFailedDetail(placeErr))
				return errSchemeBetStopped
			}
			return placeErr
		}
		if betMeta.Amount > 0 {
			amount = betMeta.Amount
		}
		if betMeta.BetsNums > 0 {
			betUnits = betMeta.BetsNums
		}
		if label := strings.TrimSpace(betMeta.PlayType); label != "" {
			playTypeLabel = label
		}
		if gc := strings.TrimSpace(betMeta.GroupContent); gc != "" {
			betContent = gc
		}
		guajiAccepted = true
		// 接单后立刻 upsert 第三方注单号，缩小「已扣款但 cloud 无 tid」窗口。
		w.finalizeCloudBetAfterGuaji(ctx, inst, cfg, recordNo, amount, betMult, roundIdx, betContent, betMeta, playEval.BetUnits, guajiTargetPeriodNo)

		// 新事务写流水/游标。失败时 defer 再 finalize 一次（幂等 upsert）。
		tx, err = w.pool.Begin(ctx)
		if err != nil {
			return err
		}
		rollbackTx = true
		qtx = w.q.WithTx(tx)
		committed = false
		if _, err := qtx.LockSchemeInstanceForBet(ctx, inst.ID); err != nil {
			return err
		}
	}

	acceptedPeriod := strings.TrimSpace(draw.IssueNo)
	cursorPeriod := acceptedPeriod
	periodMismatch := false
	if guajiReal {
		acceptedPeriod = strings.TrimSpace(betMeta.Periods)
		if acceptedPeriod == "" {
			return fmt.Errorf("%w: upstream did not return periods", guajibet.ErrPlaceRejected)
		}
		metaPeriod := acceptedPeriod
		if isAcceptedPeriodMismatch(guajiTargetPeriodNo, acceptedPeriod) {
			periodMismatch = true
			renamed, merr := qtx.MoveCloudBetRecordPeriod(ctx, inst.ID, guajiTargetPeriodNo, acceptedPeriod)
			if merr != nil {
				return merr
			}
			if !renamed {
				// 目标期已有单：占位仍在本地开放期，元数据写回占位行。
				metaPeriod = guajiTargetPeriodNo
				slog.Error("scheme worker place period mismatch; keep claim on local open",
					"instanceId", inst.ID, "claimed", guajiTargetPeriodNo, "accepted", acceptedPeriod,
					"thirdPartyBetId", betMeta.ThirdPartyBetID)
			}
		}
		metaTID := strings.TrimSpace(betMeta.ThirdPartyBetID)
		metaOrder := strings.TrimSpace(betMeta.OrderNo)
		metaAmount := amount
		// 同一第三方单号已挂在本方案其它期：多为回查命中旧单，禁止再写到新占位造成「一单两期」。
		if metaTID != "" {
			if prevPeriod, ok, perr := qtx.SchemePeriodForThirdPartyBetID(ctx, inst.ID, metaTID); perr != nil {
				return perr
			} else if ok && prevPeriod != metaPeriod {
				slog.Error("scheme worker skip duplicate third-party bet id on claim",
					"instanceId", inst.ID, "tid", metaTID, "prevPeriod", prevPeriod, "claimPeriod", metaPeriod,
					"accepted", acceptedPeriod)
				metaTID, metaOrder, metaAmount = clearDuplicateThirdPartyBetReference(metaTID, metaOrder, metaAmount, true)
				metaPeriod = guajiTargetPeriodNo
			}
		}
		updated, err := qtx.FinalizeClaimedCloudBetRecordGuajiMeta(ctx, inst.ID, metaPeriod,
			pgtype.Text{String: metaTID, Valid: metaTID != ""},
			pgtype.Text{String: metaOrder, Valid: metaOrder != ""},
			guajiPeriodsPgtext(betMeta.Periods),
			numericFromFloat(0), "pending",
			numericFromFloat(metaAmount),
			betUnits,
			playTypeLabel,
			betContent,
		)
		if err != nil {
			return err
		}
		if !updated {
			return fmt.Errorf("accepted third-party bet claim missing: instance=%s period=%s third_party_bet_id=%s", inst.ID, metaPeriod, metaTID)
		}
		if metaAmount > 0 {
			amount = metaAmount
		} else {
			amount = 0
		}
		// 游标必须推进「本地开放期」：接单期错位时若只写 accepted，会再次对同一开放期下单。
		cursorPeriod = strings.TrimSpace(guajiTargetPeriodNo)
		if cursorPeriod == "" {
			cursorPeriod = acceptedPeriod
		}
	}

	if deferSettle {
		// 待开奖：只加流水/期号游标，绝不写 round/pick/current_pick/direction。
		// 否则会与派奖事务竞态，把已推进的局数/冷热锁号盖回旧值（连投同局、中后丢锁）。
		if err := qtx.ApplySchemeInstanceBetPlacePending(
			ctx,
			inst.ID,
			w.periodCountdownForInst(inst, time.Now()),
			numericFromFloat(amount),
			pgtype.Text{String: cursorPeriod, Valid: cursorPeriod != ""},
		); err != nil {
			return err
		}
	} else if _, err := qtx.ApplySchemeInstanceBet(ctx, sqlcdb.ApplySchemeInstanceBetParams{
		ID:               inst.ID,
		CountdownSec:     w.periodCountdownForInst(inst, time.Now()),
		Turnover:         numericFromFloat(amount),
		Pnl:              numericFromFloat(pnl),
		Multiplier:       inst.Multiplier,
		RoundIndex:       int32(nextRound),
		LastSettledIssue: pgtype.Text{String: acceptedPeriod, Valid: acceptedPeriod != ""},
		LookbackPnl:      numericFromFloat(pnl),
		PickIndex:        nextPickIndex,
		CurrentPick:      nextCurrentPick,
		LastDirection:    nextLastDirection,
	}); err != nil {
		return err
	}

	if trackOverall && !deferSettle {
		if err := w.saveLookbackRuntime(ctx, qtx, inst.MemberID, inst.SimBet, overallRT, resetOverall); err != nil {
			return err
		}
	}

	if (resetIndividual || resetOverall) && !deferSettle {
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
	rollbackTx = false
	committed = true
	if periodMismatch {
		detail := fmt.Sprintf("third-party accepted a different period: target=%s accepted=%s", guajiTargetPeriodNo, acceptedPeriod)
		w.pauseRunningInstance(ctx, inst, StatusReasonBetFailed, detail)
		return errSchemeBetStopped
	}
	// 即时结算路径（历史兼容）：下单即有盈亏后检查止盈止损。
	// 模拟盘改走开奖后结算，止盈止损在 settleSimCloudBet 中检查。
	if !deferSettle {
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
	roundIdx, betUnits int,
	playTypeLabel string,
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
		if dedup.Reason == "same_third_party_period" && dedup.CurrentOpen != "" {
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

	if playTypeLabel == "" {
		playTypeLabel = cloudPlayTypeLabel(cfg.PlayTypeLabel, cfg.SubPlayLabel)
	}
	if playTypeLabel == "" {
		playTypeLabel = cfg.PlayTypeLabel
	}
	ok, err := qtx.ReserveCloudBetPeriod(ctx, sqlcdb.ReserveCloudBetPeriodParams{
		RecordNo:       recordNo,
		MemberID:       inst.MemberID,
		SimBet:         inst.SimBet,
		SchemeID:       inst.ID,
		SchemeName:     inst.SchemeName,
		PeriodNo:       periodNo,
		PlayType:       playTypeLabel,
		Multiplier:     strconv.Itoa(betMultipleAsInt(mult)),
		RoundLabel:     betRoundLabel(cfg, roundIdx, int(inst.PickIndex)),
		Amount:         numericFromFloat(amount),
		Pnl:            numericFromFloat(recordPnl),
		Status:         recordStatus,
		BetContent:     betContent,
		GuajiAccountID: activeGuajiAccountIDForInst(ctx, qtx, inst),
		Currency:       cfg.Currency,
		LotteryCode:    inst.LotteryCode,
		LotteryLabel:   inst.LotteryLabel,
		DefinitionID:   inst.DefinitionID,
		BetUnits:       betUnits,
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
