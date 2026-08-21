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
	errNoFreshFormalProviderTarget     = errors.New("no_fresh_provider_target")
	errUnsafeFormalProviderTarget      = errors.New("unsafe_provider_target_deadline")
	ErrContiguousTargetConfiguration   = errors.New("contiguous target configuration")
	ErrFormalPhaseOneInconsistentState = errors.New("formal phase one inconsistent state")
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

// FormalPhaseOneInconsistentStateError means the source decision uniqueness
// fence exists without the complete matching strategy/evaluation transaction.
type FormalPhaseOneInconsistentStateError struct {
	DecisionID int64
	Reason     string
}

func (e *FormalPhaseOneInconsistentStateError) Error() string {
	return fmt.Sprintf("%s: decision=%d reason=%s", ErrFormalPhaseOneInconsistentState, e.DecisionID, e.Reason)
}

func (e *FormalPhaseOneInconsistentStateError) Unwrap() error {
	return ErrFormalPhaseOneInconsistentState
}

type formalPhaseOneDecisionStore interface {
	InsertSchemePeriodDecision(context.Context, sqlcdb.InsertSchemePeriodDecisionParams) (int64, bool, error)
	GetFormalPhaseOneDecisionStateForUpdate(context.Context, string, string) (sqlcdb.FormalPhaseOneDecisionState, bool, error)
}

// reserveFormalPhaseOneDecision makes the database uniqueness constraint the
// authority for strategy advancement. A conflict never runs advance: it must
// prove that an earlier transaction completed every matching phase-one marker.
func reserveFormalPhaseOneDecision(
	ctx context.Context,
	store formalPhaseOneDecisionStore,
	params sqlcdb.InsertSchemePeriodDecisionParams,
	advance func() error,
) (decisionID int64, created bool, err error) {
	decisionID, created, err = store.InsertSchemePeriodDecision(ctx, params)
	if err != nil {
		return 0, false, err
	}
	if created {
		if err := advance(); err != nil {
			return 0, false, err
		}
		return decisionID, true, nil
	}
	existing, found, err := store.GetFormalPhaseOneDecisionStateForUpdate(ctx, params.SchemeID, params.SourcePeriodNo)
	if err != nil {
		return 0, false, err
	}
	if !found {
		return 0, false, formalPhaseOneInconsistent(decisionID, "conflicting decision is not visible")
	}
	if reason := validateExistingFormalPhaseOne(existing, decisionID, params); reason != "" {
		return 0, false, formalPhaseOneInconsistent(decisionID, reason)
	}
	return decisionID, false, nil
}

func formalPhaseOneInconsistent(decisionID int64, reason string) error {
	return &FormalPhaseOneInconsistentStateError{DecisionID: decisionID, Reason: reason}
}

func validateExistingFormalPhaseOne(
	existing sqlcdb.FormalPhaseOneDecisionState,
	decisionID int64,
	params sqlcdb.InsertSchemePeriodDecisionParams,
) string {
	executionMismatch := formalPhaseOneExecutionMismatch(existing, params)
	switch {
	case existing.DecisionID != decisionID:
		return "decision id mismatch"
	case existing.SchemeID != params.SchemeID || existing.LotteryCode != params.LotteryCode || existing.SourcePeriodNo != params.SourcePeriodNo:
		return "source identity mismatch"
	case !existing.SourceBetRecordID.Valid || existing.SourceBetRecordID.Int64 != params.SourceBetRecordID:
		return "source bet record mismatch"
	case !equalOptionalText(existing.DrawHash, pgtype.Text{String: params.DrawHash, Valid: params.DrawHash != ""}):
		return "draw hash mismatch"
	case existing.StateVersionBefore != params.StateVersionBefore || existing.StateVersionAfter != params.StateVersionAfter:
		return "decision state version mismatch"
	case existing.CurrentStateVersion != params.StateVersionAfter:
		return "instance state version is not phase-one result"
	case !equalOptionalInt4(existing.RuleVersion, params.RuleVersion) || !equalOptionalText(existing.RuleSnapshotHash, params.RuleSnapshotHash):
		return "decision rule snapshot mismatch"
	case !existing.LocalHit.Valid || existing.LocalHit.Bool != params.LocalHit:
		return "decision local result mismatch"
	case !existing.WinningUnits.Valid || existing.WinningUnits.Int32 != int32(params.WinningUnits):
		return "decision winning units mismatch"
	case !formalPhaseOneStatusMatches(params.Status, existing.Status):
		return "decision status is not a completed phase-one state"
	case executionMismatch != "":
		return executionMismatch
	case !equalOptionalTime(existing.TargetDeadlineAt, params.TargetDeadlineAt):
		return "target deadline mismatch"
	case !existing.EvaluationStatus.Valid || (existing.EvaluationStatus.String != "completed" && existing.EvaluationStatus.String != "mismatch"):
		return "strategy evaluation is not complete"
	case !existing.EvaluationCloudBetRecordID.Valid || existing.EvaluationCloudBetRecordID.Int64 != params.SourceBetRecordID:
		return "strategy evaluation source record mismatch"
	case !equalOptionalInt4(existing.EvaluationRuleVersion, params.RuleVersion) || !equalOptionalText(existing.EvaluationRuleSnapshotHash, params.RuleSnapshotHash):
		return "strategy evaluation rule snapshot mismatch"
	case !existing.EvaluationLocalHit.Valid || existing.EvaluationLocalHit.Bool != params.LocalHit:
		return "strategy evaluation local result mismatch"
	case !existing.EvaluationWinningUnits.Valid || existing.EvaluationWinningUnits.Int32 != int32(params.WinningUnits):
		return "strategy evaluation winning units mismatch"
	case !existing.EvaluationCompletedAt.Valid:
		return "strategy evaluation completion marker is missing"
	case !existing.CloudStrategyEvaluatedAt.Valid:
		return "cloud strategy marker is missing"
	default:
		return ""
	}
}

func formalPhaseOneExecutionMismatch(
	existing sqlcdb.FormalPhaseOneDecisionState,
	params sqlcdb.InsertSchemePeriodDecisionParams,
) string {
	switch existing.Status {
	case "awaiting_target", "completed":
		if existing.ExecutionChainState != "active" {
			return "nonterminal decision execution chain is not active"
		}
		if existing.ExecutionChainBlockReason.Valid {
			return "nonterminal decision execution chain has a block reason"
		}
	case "chain_broken":
		if existing.ExecutionChainState != "blocked_requires_rearm" {
			return "terminal decision execution chain is not blocked"
		}
		expectedReason := formalPhaseOneBlockReason(params.Diagnostics)
		if expectedReason == "" {
			return "terminal decision block reason evidence is missing"
		}
		if !existing.ExecutionChainBlockReason.Valid || existing.ExecutionChainBlockReason.String != expectedReason {
			return "terminal decision execution chain block reason mismatch"
		}
	case "missed_contiguous_period":
		if existing.ExecutionChainState != "blocked_requires_rearm" {
			return "missed decision execution chain is not blocked"
		}
		if !existing.ExecutionChainBlockReason.Valid || existing.ExecutionChainBlockReason.String != "missed_contiguous_period" {
			return "missed decision execution chain block reason mismatch"
		}
	}
	return ""
}

func formalPhaseOneBlockReason(diagnostics []byte) string {
	var payload struct {
		Reason string `json:"reason"`
	}
	if len(diagnostics) == 0 || json.Unmarshal(diagnostics, &payload) != nil {
		return ""
	}
	return strings.TrimSpace(payload.Reason)
}

func (p *StrategyProcessor) validateExistingFormalPhaseOneConflict(
	ctx context.Context,
	q *sqlcdb.Queries,
	recordID int64,
	schemeID, lotteryCode, periodNo string,
	expectedStateVersion *int64,
) error {
	if p.formalModeForLottery(lotteryCode) == "" {
		return nil
	}
	existing, found, err := q.GetFormalPhaseOneDecisionStateForUpdate(ctx, schemeID, periodNo)
	if err != nil || !found {
		return err
	}
	switch existing.Status {
	case "awaiting_target", "completed", "missed_contiguous_period", "chain_broken":
	default:
		return nil
	}
	stateVersionBefore := existing.StateVersionBefore
	if expectedStateVersion != nil {
		stateVersionBefore = *expectedStateVersion
	}
	drawHash := ""
	if existing.DrawHash.Valid {
		drawHash = existing.DrawHash.String
	}
	params := sqlcdb.InsertSchemePeriodDecisionParams{
		SchemeID: schemeID, LotteryCode: lotteryCode, SourcePeriodNo: periodNo,
		SourceBetRecordID: recordID, DrawHash: drawHash,
		StateVersionBefore: stateVersionBefore, StateVersionAfter: stateVersionBefore + 1,
		RuleVersion: existing.RuleVersion, RuleSnapshotHash: existing.RuleSnapshotHash,
		LocalHit: existing.LocalHit.Bool, WinningUnits: int(existing.WinningUnits.Int32),
		Status: existing.Status, Diagnostics: existing.DecisionDiagnostics, TargetDeadlineAt: existing.TargetDeadlineAt,
	}
	if reason := validateExistingFormalPhaseOne(existing, existing.DecisionID, params); reason != "" {
		return formalPhaseOneInconsistent(existing.DecisionID, reason)
	}
	return nil
}

func formalPhaseOneStatusMatches(expected, actual string) bool {
	if expected == "awaiting_target" {
		return actual == "awaiting_target" || actual == "completed" || actual == "missed_contiguous_period"
	}
	return actual == expected
}

func equalOptionalText(a, b pgtype.Text) bool {
	return a.Valid == b.Valid && (!a.Valid || a.String == b.String)
}

func equalOptionalInt4(a, b pgtype.Int4) bool {
	return a.Valid == b.Valid && (!a.Valid || a.Int32 == b.Int32)
}

func equalOptionalTime(a, b pgtype.Timestamptz) bool {
	return a.Valid == b.Valid && (!a.Valid || a.Time.Equal(b.Time))
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
	diagnostics, err := json.Marshal(map[string]any{
		"mode":           p.formalModeForLottery(row.LotteryCode),
		"targetDeadline": deadline.UTC().Format(time.RFC3339Nano),
	})
	if err != nil {
		return shadowDecisionResult{}, err
	}
	params := sqlcdb.InsertSchemePeriodDecisionParams{
		SchemeID: row.SchemeID, LotteryCode: row.LotteryCode, SourcePeriodNo: row.PeriodNo,
		SourceBetRecordID: row.RecordID, DrawHash: lottery.CanonicalDrawHash(row.LotteryCode, row.PeriodNo, row.Balls),
		StateVersionBefore: stateVersionBefore, StateVersionAfter: stateVersionBefore + 1,
		RuleVersion: row.RuleVersion, RuleSnapshotHash: row.RuleSnapshotHash,
		LocalHit: result.Hit, WinningUnits: result.WinningUnits, Status: "awaiting_target", Diagnostics: diagnostics,
		TargetDeadlineAt: pgtype.Timestamptz{Time: deadline.UTC(), Valid: true},
	}
	decisionID, created, err := reserveFormalPhaseOneDecision(ctx, q, params, func() error {
		return schemestate.ProcessStrategyAfterDraw(ctx, q, inst, row.PeriodNo, result.Hit, definitionConfig)
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
	diagnostics, err := json.Marshal(map[string]any{"mode": p.bettingMode, "reason": "contiguous_target_configuration"})
	if err != nil {
		return shadowDecisionResult{}, err
	}
	params := sqlcdb.InsertSchemePeriodDecisionParams{
		SchemeID: row.SchemeID, LotteryCode: row.LotteryCode, SourcePeriodNo: row.PeriodNo,
		SourceBetRecordID: row.RecordID, DrawHash: lottery.CanonicalDrawHash(row.LotteryCode, row.PeriodNo, row.Balls),
		StateVersionBefore: stateVersionBefore, StateVersionAfter: stateVersionBefore + 1,
		RuleVersion: row.RuleVersion, RuleSnapshotHash: row.RuleSnapshotHash,
		LocalHit: result.Hit, WinningUnits: result.WinningUnits, Status: "chain_broken", Diagnostics: diagnostics,
	}
	decisionID, created, err := reserveFormalPhaseOneDecision(ctx, q, params, func() error {
		if err := schemestate.ProcessStrategyAfterDraw(ctx, q, inst, row.PeriodNo, result.Hit, definitionConfig); err != nil {
			return err
		}
		return q.BlockSchemeBettingChain(ctx, row.SchemeID, "contiguous_target_configuration", time.Now().UTC())
	})
	if err != nil {
		return shadowDecisionResult{}, err
	}
	return shadowDecisionResult{Status: "chain_broken", Reason: "contiguous_target_configuration", DecisionID: decisionID, Created: created}, nil
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
