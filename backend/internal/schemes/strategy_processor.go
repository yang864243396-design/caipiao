package schemes

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync/atomic"

	"github.com/jackc/pgx/v5/pgtype"

	"caipiao/backend/internal/cloud/schemestate"
	"caipiao/backend/internal/db"
	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/playrules"
)

const strategyProcessorBatchSize = 50

type strategyCandidate struct{ Accepted, HasSnapshot, HasDraw bool }

func shouldProcessFormalStrategy(candidate strategyCandidate) bool {
	return candidate.Accepted && candidate.HasSnapshot && candidate.HasDraw
}

// StrategyProcessor advances real-scheme strategy state from persisted draws.
// Its query is bounded and idempotent; financial settlement is intentionally
// not part of this processor.
type StrategyProcessor struct {
	pool *db.Pool
	q    *sqlcdb.Queries
	busy atomic.Bool
}

func NewStrategyProcessor(pool *db.Pool) *StrategyProcessor {
	if pool == nil {
		return nil
	}
	return &StrategyProcessor{pool: pool, q: sqlcdb.New(pool)}
}

func (p *StrategyProcessor) Recover(ctx context.Context) error {
	if p == nil || p.q == nil || !p.busy.CompareAndSwap(false, true) {
		return nil
	}
	defer p.busy.Store(false)
	rows, err := p.q.ListPendingFormalStrategyRows(ctx, strategyProcessorBatchSize)
	if err != nil {
		return err
	}
	for _, row := range rows {
		if err := p.process(ctx, row); err != nil {
			slog.Warn("formal strategy process failed", "recordId", row.RecordID, "schemeId", row.SchemeID, "period", row.PeriodNo, "err", err)
		}
	}
	return nil
}

func (p *StrategyProcessor) NotifyDraw(ctx context.Context, _, _ string) {
	if p == nil || p.busy.Load() {
		return
	}
	go func() { _ = p.Recover(ctx) }()
}

func (p *StrategyProcessor) process(ctx context.Context, row sqlcdb.PendingFormalStrategyRow) error {
	if !shouldProcessFormalStrategy(strategyCandidate{Accepted: true, HasSnapshot: len(row.RuleSnapshot) > 0, HasDraw: len(row.Balls) > 0}) {
		return nil
	}
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	qtx := p.q.WithTx(tx)
	claimed, err := qtx.TryClaimSchemeStrategyEvaluation(ctx, sqlcdb.TryClaimSchemeStrategyEvaluationParams{InstanceID: row.SchemeID, LotteryCode: row.LotteryCode, PeriodNo: row.PeriodNo})
	if err != nil || !claimed {
		return err
	}
	evaluation, err := qtx.GetSchemeStrategyEvaluation(ctx, sqlcdb.GetSchemeStrategyEvaluationParams{InstanceID: row.SchemeID, PeriodNo: row.PeriodNo})
	if err != nil {
		return err
	}
	if n, err := qtx.MarkSchemeStrategyEvaluationProcessing(ctx, evaluation.ID); err != nil || n != 1 {
		if err != nil {
			return err
		}
		return fmt.Errorf("strategy evaluation %d not claimable", evaluation.ID)
	}
	inst, err := qtx.GetSchemeInstanceFull(ctx, row.SchemeID)
	if err != nil {
		return err
	}
	def, err := qtx.GetSchemeDefinitionByID(ctx, inst.DefinitionID)
	if err != nil {
		return err
	}
	var snapshot playrules.Snapshot
	if err := json.Unmarshal(row.RuleSnapshot, &snapshot); err != nil {
		return err
	}
	result, err := evaluateFormalStrategyRule(schemestate.FormalRuleEvaluationInput{Kind: inst.Kind, DefinitionConfig: def.Config, RoundIndex: int(inst.RoundIndex), LotteryCode: row.LotteryCode, BetContent: row.BetContent, Balls: row.Balls, Snapshot: snapshot})
	if err != nil {
		return err
	}
	if err := schemestate.ProcessStrategyAfterDraw(ctx, qtx, inst, row.PeriodNo, result.Hit, def.Config); err != nil {
		return err
	}
	diagnostics, err := json.Marshal(map[string]any{"source": "draw_ws_rule", "localHit": result.Hit, "winningUnits": result.WinningUnits})
	if err != nil {
		return err
	}
	if _, err := qtx.CompleteSchemeStrategyEvaluationWithStatus(ctx, sqlcdb.CompleteSchemeStrategyEvaluationWithStatusParams{CloudBetRecordID: pgtype.Int8{Int64: row.RecordID, Valid: true}, RuleVersion: row.RuleVersion, RuleSnapshotHash: row.RuleSnapshotHash, LocalHit: pgtype.Bool{Bool: result.Hit, Valid: true}, WinningUnits: pgtype.Int4{Int32: int32(result.WinningUnits), Valid: true}, Diagnostics: diagnostics, Status: "completed", ID: evaluation.ID}); err != nil {
		return err
	}
	if err := qtx.MarkCloudBetRecordStrategyEvaluated(ctx, row.RecordID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
