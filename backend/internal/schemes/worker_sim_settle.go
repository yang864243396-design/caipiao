package schemes

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"caipiao/backend/internal/cloud/schemestate"
	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/member"
	"caipiao/backend/internal/playrules"
	"caipiao/backend/internal/schemeevents"
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

// simSettlement 模拟盘一注的结算结论。
type simSettlement struct {
	Status string
	Hit    bool
	Pnl    float64
	Payout float64
}

// decideSimSettlement 由方案定义与开奖球号推出结算结论，不碰 DB。
//
// 这是全系统唯一明确写下 status = hit|miss 的地方，从事务里拆出来是为了能被表驱动测试
// 直接覆盖——否则要跑通它得先造出成员、方案定义、实例、注单一整张图。
// ok=false 表示开奖号缺失，不应结算。
func decideSimSettlement(
	kind string, config []byte, lotteryCode, betContent string,
	roundIndex int, balls []string, amount float64,
) (simSettlement, bool) {
	settlement, ok, _ := decideSimSettlementWithSnapshot(kind, config, lotteryCode, betContent, roundIndex, balls, amount, nil)
	return settlement, ok
}

func decideSimSettlementWithSnapshot(
	kind string, config []byte, lotteryCode, betContent string,
	roundIndex int, balls []string, amount float64, snapshot *playrules.Snapshot,
) (simSettlement, bool, error) {
	if len(balls) == 0 {
		return simSettlement{}, false, nil
	}
	groupIndex := 0
	if roundIndex > 0 {
		groupIndex = roundIndex
	}
	cfg := parseSchemeConfig(kind, config, roundIndex, groupIndex)
	cfg.Play = attachOddsBase(cfg.Play, lotteryCode)

	if strings.TrimSpace(betContent) == "" {
		betContent = cfg.GroupContent
	}
	// cloud_bet_records.bet_content 已是当期实际投注内容（含反投展开），按原玩法验奖即可。
	playEval := betEvaluation{}
	if snapshot != nil {
		frozen, err := evaluateFrozenRule(*snapshot, cfg.Play, balls, betContent, false, "")
		if err != nil {
			return simSettlement{}, false, err
		}
		playEval = frozen
	} else {
		playEval = evaluatePlayHit(cfg.Play, balls, betContent, false, "", cfg.Play.PositionIdx)
	}

	amount = member.RoundMoney(amount)
	pnl := calcPnLWithOdds(amount, playEval.Hit, playEval.Odds)
	out := simSettlement{Status: "miss", Hit: playEval.Hit, Pnl: pnl}
	if playEval.Hit {
		out.Status = "hit"
		out.Payout = member.RoundMoney(amount + pnl)
	}
	return out, true, nil
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

	var snapshot *playrules.Snapshot
	if len(row.RuleSnapshot) > 0 && string(row.RuleSnapshot) != "null" {
		var frozen playrules.Snapshot
		if err := json.Unmarshal(row.RuleSnapshot, &frozen); err != nil {
			return fmt.Errorf("decode frozen play rule: %w", err)
		}
		if frozen.EvaluatorKey != "" {
			snapshot = &frozen
		}
	}
	settle, ok, err := decideSimSettlementWithSnapshot(
		inst.Kind, def.Config, row.LotteryCode, row.BetContent,
		int(inst.RoundIndex), balls, row.Amount, snapshot,
	)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}
	pnl := settle.Pnl

	tx, err := w.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	qtx := w.q.WithTx(tx)

	n, err := qtx.UpdateCloudBetRecordFromSettlementByID(
		ctx, row.ID, settle.Status, numericFromFloat(pnl), numericFromFloat(settle.Payout),
	)
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
			ctx, qtx, fresh, row.PeriodNo, pnl, settle.Hit, def.Config, numericFromFloat,
		); err != nil {
			return err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	w.afterCommittedSimSettlement(ctx, inst, def.Config)

	slog.Info("scheme worker sim bet settled",
		"instanceId", row.SchemeID, "period", row.PeriodNo,
		"status", settle.Status, "pnl", pnl, "amount", row.Amount)
	return nil
}

func (w *Worker) afterCommittedSimSettlement(ctx context.Context, inst sqlcdb.SchemeInstance, defConfig []byte) {
	w.withCommittedSchemeMarker(ctx, RealtimeInstanceRef{MemberID: inst.MemberID, InstanceID: inst.ID},
		func(operationCtx context.Context, marker schemeevents.Marker) {
			if w.q == nil {
				return
			}
			fresh, err := w.q.GetSchemeInstanceFull(operationCtx, inst.ID)
			if err != nil || fresh.Status != "running" {
				return
			}
			w.pauseRunningForSessionLimit(operationCtx, fresh, defConfig)
			w.pauseAllRunningForCloudLimitWithMarker(operationCtx, fresh.MemberID, marker)
			if status, err := w.q.GetSchemeInstanceStatus(operationCtx, fresh.ID); err == nil && status == "running" {
				w.notifySchemeInstance(operationCtx, fresh.MemberID, fresh.ID, runModeFromSimBet(fresh.SimBet), "running", StatusReasonCloudActive)
			}
		})
}
