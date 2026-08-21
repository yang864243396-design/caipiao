package schemes

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
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
	pool             *db.Pool
	q                *sqlcdb.Queries
	busy             atomic.Bool
	bettingMode      string
	bettingLotteries map[string]struct{}
	ruleRegistry     *playrules.RegistryStore

	lifecycleMu     sync.Mutex
	closing         bool
	recoverWait     sync.WaitGroup
	recoverFn       func(context.Context) error
	recoverScopedFn func(context.Context, string, string) error
}

func NewStrategyProcessor(pool *db.Pool) *StrategyProcessor {
	if pool == nil {
		return nil
	}
	return &StrategyProcessor{pool: pool, q: sqlcdb.New(pool)}
}

func (p *StrategyProcessor) Recover(ctx context.Context) error {
	if p == nil || (p.q == nil && p.recoverFn == nil) || !p.busy.CompareAndSwap(false, true) {
		return nil
	}
	defer p.busy.Store(false)
	if p.recoverFn != nil {
		return p.recoverFn(ctx)
	}
	return p.recoverPending(ctx)
}

func (p *StrategyProcessor) recoverPending(ctx context.Context) error {
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

func (p *StrategyProcessor) recoverPendingScope(ctx context.Context, lotteryCode, periodNo string) error {
	if p.recoverScopedFn != nil {
		return p.recoverScopedFn(ctx, lotteryCode, periodNo)
	}
	if p.q == nil {
		if p.recoverFn != nil {
			return p.recoverFn(ctx)
		}
		return nil
	}
	rows, err := p.q.ListPendingFormalStrategyRowsForDraw(ctx, lotteryCode, periodNo, strategyProcessorBatchSize)
	if err != nil {
		return err
	}
	for _, row := range rows {
		if err := p.process(ctx, row); err != nil {
			slog.Warn("scoped formal strategy process failed", "recordId", row.RecordID, "schemeId", row.SchemeID, "lottery", lotteryCode, "period", periodNo, "err", err)
		}
	}
	return nil
}

func (p *StrategyProcessor) NotifyDraw(ctx context.Context, lotteryCode, periodNo string) {
	if p == nil {
		return
	}
	lotteryCode = strings.TrimSpace(lotteryCode)
	periodNo = strings.TrimSpace(periodNo)
	if lotteryCode == "" || periodNo == "" {
		return
	}
	p.lifecycleMu.Lock()
	if p.closing {
		p.lifecycleMu.Unlock()
		return
	}
	p.recoverWait.Add(1)
	p.lifecycleMu.Unlock()
	go func() {
		defer p.recoverWait.Done()
		_ = p.recoverPendingScope(ctx, lotteryCode, periodNo)
	}()
}

func (p *StrategyProcessor) Close() {
	if p == nil {
		return
	}
	p.lifecycleMu.Lock()
	p.closing = true
	p.lifecycleMu.Unlock()
	p.recoverWait.Wait()
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
	if fence, ok := strategyLeaseFenceFromContext(ctx); ok {
		if err := qtx.AssertSchemeBettingShardLease(ctx, "strategy", fence.ShardNo, fence.Owner, fence.Epoch); err != nil {
			return err
		}
	}
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
	stateVersionBefore, err := qtx.LockSchemeStateVersion(ctx, row.SchemeID)
	if err != nil {
		return err
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
	formal, shadow, err := p.tryProcessFormalCandidate(ctx, qtx, row, inst, def.Config, stateVersionBefore, result)
	if err != nil {
		return err
	}
	if !formal {
		if err := schemestate.ProcessStrategyAfterDraw(ctx, qtx, inst, row.PeriodNo, result.Hit, def.Config); err != nil {
			return err
		}
		shadow, err = persistShadowDecision(ctx, qtx, row, stateVersionBefore, result.Hit, result.WinningUnits)
	}
	if err != nil {
		return err
	}
	diagnostics, err := json.Marshal(map[string]any{
		"source": "draw_ws_rule", "localHit": result.Hit, "winningUnits": result.WinningUnits,
		"providerHit": row.ProviderHit, "mismatch": row.ProviderHit.Valid && row.ProviderHit.Bool != result.Hit,
		"shadow": shadow,
	})
	if err != nil {
		return err
	}
	evaluationStatus := "completed"
	if row.ProviderHit.Valid && row.ProviderHit.Bool != result.Hit {
		evaluationStatus = "mismatch"
	}
	if _, err := qtx.CompleteSchemeStrategyEvaluationWithStatus(ctx, sqlcdb.CompleteSchemeStrategyEvaluationWithStatusParams{CloudBetRecordID: pgtype.Int8{Int64: row.RecordID, Valid: true}, RuleVersion: row.RuleVersion, RuleSnapshotHash: row.RuleSnapshotHash, LocalHit: pgtype.Bool{Bool: result.Hit, Valid: true}, WinningUnits: pgtype.Int4{Int32: int32(result.WinningUnits), Valid: true}, Diagnostics: diagnostics, Status: evaluationStatus, ID: evaluation.ID}); err != nil {
		return err
	}
	if err := qtx.MarkCloudBetRecordStrategyEvaluated(ctx, row.RecordID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
