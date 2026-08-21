package schemes

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"caipiao/backend/internal/cloud/schemestate"
	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/lottery"
)

var (
	errNoFreshFormalProviderTarget   = errors.New("no_fresh_provider_target")
	errUnsafeFormalProviderTarget    = errors.New("unsafe_provider_target_deadline")
	ErrContiguousTargetConfiguration = errors.New("contiguous target configuration")
)

// ContiguousTargetConfigurationError identifies a chain configuration that
// cannot safely derive the immediately following target deadline.
type ContiguousTargetConfigurationError struct {
	IntervalSec int
}

func (e *ContiguousTargetConfigurationError) Error() string {
	return fmt.Sprintf("%s: draw interval must be positive (got %d)", ErrContiguousTargetConfiguration, e.IntervalSec)
}

func (e *ContiguousTargetConfigurationError) Unwrap() error {
	return ErrContiguousTargetConfiguration
}

func contiguousTargetDeadline(drawnAt time.Time, intervalSec int, safety time.Duration) (time.Time, error) {
	if intervalSec <= 0 {
		return time.Time{}, &ContiguousTargetConfigurationError{IntervalSec: intervalSec}
	}
	return drawnAt.UTC().Add(time.Duration(intervalSec) * time.Second).Add(-safety), nil
}

func formalProviderTargetError(available bool, buildErr error) error {
	if !available {
		return errNoFreshFormalProviderTarget
	}
	if buildErr != nil {
		return fmt.Errorf("%w: %v", errUnsafeFormalProviderTarget, buildErr)
	}
	return nil
}

func (p *StrategyProcessor) SetBettingMode(mode string, lotteries []string) {
	if p == nil {
		return
	}
	p.bettingMode = strings.ToLower(strings.TrimSpace(mode))
	p.bettingLotteries = make(map[string]struct{}, len(lotteries))
	for _, code := range lotteries {
		if code = strings.TrimSpace(code); code != "" {
			p.bettingLotteries[code] = struct{}{}
		}
	}
}

func (w *Worker) SetSchemeBettingMode(mode string, lotteries []string) {
	if w != nil && w.strategyProcessor != nil {
		w.strategyProcessor.SetBettingMode(mode, lotteries)
	}
}

func (p *StrategyProcessor) formalModeForLottery(lotteryCode string) string {
	if p == nil || (p.bettingMode != "gray" && p.bettingMode != "production") {
		return ""
	}
	if _, ok := p.bettingLotteries[strings.TrimSpace(lotteryCode)]; !ok {
		return ""
	}
	return p.bettingMode
}

func (p *StrategyProcessor) tryProcessFormalCandidate(
	ctx context.Context,
	q *sqlcdb.Queries,
	row sqlcdb.PendingFormalStrategyRow,
	inst sqlcdb.SchemeInstance,
	definitionConfig []byte,
	stateVersionBefore int64,
	result schemestate.FormalRuleEvaluation,
) (bool, shadowDecisionResult, error) {
	mode := p.formalModeForLottery(row.LotteryCode)
	if mode == "" {
		return false, shadowDecisionResult{}, nil
	}
	execution, err := q.GetSchemeBettingExecutionState(ctx, row.SchemeID)
	if err != nil {
		return true, shadowDecisionResult{}, err
	}
	if execution.Owner != "event" {
		return false, shadowDecisionResult{}, nil
	}
	if execution.ChainState != "active" || strings.TrimSpace(execution.ChainID) == "" {
		blocked, err := p.persistFormalBlocked(ctx, q, row, stateVersionBefore, result, "chain_not_active")
		return true, blocked, err
	}

	deadline, err := contiguousTargetDeadline(
		row.DrawnAt,
		lottery.DrawIntervalSecForLottery(ctx, q, row.LotteryCode),
		guajiPlaceCloseSafety,
	)
	if err != nil {
		if errors.Is(err, ErrContiguousTargetConfiguration) {
			blocked, blockErr := p.persistFormalConfigurationFailure(ctx, q, row, inst, definitionConfig, stateVersionBefore, result)
			return true, blocked, blockErr
		}
		return true, shadowDecisionResult{}, err
	}
	awaiting, err := p.persistFormalAwaitingTarget(ctx, q, row, inst, definitionConfig, stateVersionBefore, result, deadline)
	return true, awaiting, err
}

// persistFormalAwaitingTarget is phase one of a formal contiguous chain. It
// advances only local strategy state and commits the target wait; resolving a
// period, producing a frozen request, and creating an Outbox are phase two.
func (p *StrategyProcessor) persistFormalAwaitingTarget(
	ctx context.Context,
	q *sqlcdb.Queries,
	row sqlcdb.PendingFormalStrategyRow,
	inst sqlcdb.SchemeInstance,
	definitionConfig []byte,
	stateVersionBefore int64,
	result schemestate.FormalRuleEvaluation,
	deadline time.Time,
) (shadowDecisionResult, error) {
	if err := schemestate.ProcessStrategyAfterDraw(ctx, q, inst, row.PeriodNo, result.Hit, definitionConfig); err != nil {
		return shadowDecisionResult{}, err
	}
	diagnostics, err := json.Marshal(map[string]any{
		"mode":           p.formalModeForLottery(row.LotteryCode),
		"targetDeadline": deadline.UTC().Format(time.RFC3339Nano),
	})
	if err != nil {
		return shadowDecisionResult{}, err
	}
	decisionID, created, err := q.InsertSchemePeriodDecision(ctx, sqlcdb.InsertSchemePeriodDecisionParams{
		SchemeID: row.SchemeID, LotteryCode: row.LotteryCode, SourcePeriodNo: row.PeriodNo,
		SourceBetRecordID: row.RecordID, DrawHash: lottery.CanonicalDrawHash(row.LotteryCode, row.PeriodNo, row.Balls),
		StateVersionBefore: stateVersionBefore, StateVersionAfter: stateVersionBefore + 1,
		RuleVersion: row.RuleVersion, RuleSnapshotHash: row.RuleSnapshotHash,
		LocalHit: result.Hit, WinningUnits: result.WinningUnits, Status: "awaiting_target", Diagnostics: diagnostics,
		TargetDeadlineAt: pgtype.Timestamptz{Time: deadline.UTC(), Valid: true},
	})
	if err != nil {
		return shadowDecisionResult{}, err
	}
	return shadowDecisionResult{Status: "awaiting_target", DecisionID: decisionID, Created: created}, nil
}

func (p *StrategyProcessor) persistFormalConfigurationFailure(
	ctx context.Context,
	q *sqlcdb.Queries,
	row sqlcdb.PendingFormalStrategyRow,
	inst sqlcdb.SchemeInstance,
	definitionConfig []byte,
	stateVersionBefore int64,
	result schemestate.FormalRuleEvaluation,
) (shadowDecisionResult, error) {
	if err := schemestate.ProcessStrategyAfterDraw(ctx, q, inst, row.PeriodNo, result.Hit, definitionConfig); err != nil {
		return shadowDecisionResult{}, err
	}
	return p.persistFormalBlocked(ctx, q, row, stateVersionBefore+1, result, "contiguous_target_configuration")
}

func (p *StrategyProcessor) persistFormalBlocked(
	ctx context.Context,
	q *sqlcdb.Queries,
	row sqlcdb.PendingFormalStrategyRow,
	stateVersion int64,
	result schemestate.FormalRuleEvaluation,
	reason string,
) (shadowDecisionResult, error) {
	diagnostics, err := json.Marshal(map[string]any{"mode": p.bettingMode, "reason": reason})
	if err != nil {
		return shadowDecisionResult{}, err
	}
	decisionID, _, err := q.InsertSchemePeriodDecision(ctx, sqlcdb.InsertSchemePeriodDecisionParams{
		SchemeID: row.SchemeID, LotteryCode: row.LotteryCode, SourcePeriodNo: row.PeriodNo,
		SourceBetRecordID: row.RecordID, DrawHash: lottery.CanonicalDrawHash(row.LotteryCode, row.PeriodNo, row.Balls),
		StateVersionBefore: stateVersion, StateVersionAfter: stateVersion,
		RuleVersion: row.RuleVersion, RuleSnapshotHash: row.RuleSnapshotHash,
		LocalHit: result.Hit, WinningUnits: result.WinningUnits, Status: "chain_broken", Diagnostics: diagnostics,
	})
	if err != nil {
		return shadowDecisionResult{}, err
	}
	if err := q.BlockSchemeBettingChain(ctx, row.SchemeID, reason, time.Now().UTC()); err != nil {
		return shadowDecisionResult{}, err
	}
	return shadowDecisionResult{Status: "chain_broken", Reason: reason, DecisionID: decisionID}, nil
}
