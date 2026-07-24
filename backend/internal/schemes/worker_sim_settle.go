package schemes

import (
	"context"
	"log/slog"
	"strings"

	"caipiao/backend/internal/cloud/schemestate"
	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/member"
)

const simSettlementBatchSize = 50

// tickSimSettlements 用真实 lottery_draws 球号结算模拟盘 pending 注单。
func (w *Worker) tickSimSettlements(ctx context.Context) {
	if w == nil || w.q == nil {
		return
	}
	rows, err := w.q.ListPendingSimCloudBetsReady(ctx, simSettlementBatchSize)
	if err != nil {
		slog.Warn("scheme worker list pending sim bets failed", "err", err)
		return
	}
	for _, row := range rows {
		if err := w.settleSimCloudBet(ctx, row); err != nil {
			slog.Warn("scheme worker sim settle failed",
				"recordId", row.ID, "schemeId", row.SchemeID, "period", row.PeriodNo, "err", err)
		}
	}
}

func (w *Worker) settleSimCloudBet(ctx context.Context, row sqlcdb.PendingSimCloudBetRow) error {
	balls := sqlcdb.ParseDrawBalls(row.Balls)
	if len(balls) == 0 {
		return nil
	}

	inst, err := w.q.GetSchemeInstanceFull(ctx, row.SchemeID)
	if err != nil {
		return err
	}
	def, err := w.q.GetSchemeDefinitionByID(ctx, inst.DefinitionID)
	if err != nil {
		return err
	}

	groupIndex := 0
	if inst.RoundIndex > 0 {
		groupIndex = int(inst.RoundIndex)
	}
	cfg := parseSchemeConfig(inst.Kind, def.Config, int(inst.RoundIndex), groupIndex)
	cfg.Play = attachOddsBase(cfg.Play, row.LotteryCode)

	betContent := row.BetContent
	if strings.TrimSpace(betContent) == "" {
		betContent = cfg.GroupContent
	}
	// cloud_bet_records.bet_content 已是当期实际投注内容（含反投展开），按原玩法验奖即可。
	playEval := evaluatePlayHit(cfg.Play, balls, betContent, false, "", cfg.Play.PositionIdx)
	amount := member.RoundMoney(row.Amount)
	pnl := calcPnLWithOdds(amount, playEval.Hit, playEval.Odds)
	status := "miss"
	if playEval.Hit {
		status = "hit"
	}

	tx, err := w.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	qtx := w.q.WithTx(tx)

	n, err := qtx.UpdateCloudBetRecordFromSettlementByID(ctx, row.ID, status, numericFromFloat(pnl))
	if err != nil {
		return err
	}
	if n == 0 {
		return nil // 已被其它 tick 结算
	}
	if err := qtx.ApplySchemeStatsFromCloudBetSettlementByID(ctx, row.ID, numericFromFloat(pnl)); err != nil {
		return err
	}

	fresh, ferr := qtx.GetSchemeInstanceFull(ctx, row.SchemeID)
	if ferr == nil {
		if err := schemestate.ProcessAfterSettlement(
			ctx, qtx, fresh, row.PeriodNo, pnl, playEval.Hit, def.Config, numericFromFloat,
		); err != nil {
			return err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	if fresh, ferr := w.q.GetSchemeInstanceFull(ctx, row.SchemeID); ferr == nil && fresh.Status == "running" {
		w.pauseRunningForSessionLimit(ctx, fresh, def.Config)
		w.pauseAllRunningForCloudLimit(ctx, fresh.MemberID)
		if st, serr := w.q.GetSchemeInstanceStatus(ctx, fresh.ID); serr == nil && st == "running" {
			w.notifySchemeInstance(ctx, fresh.MemberID, fresh.ID, runModeFromSimBet(fresh.SimBet), "running", StatusReasonCloudActive)
		}
	}

	slog.Info("scheme worker sim bet settled",
		"instanceId", row.SchemeID, "period", row.PeriodNo, "status", status, "pnl", pnl, "amount", amount)
	return nil
}
