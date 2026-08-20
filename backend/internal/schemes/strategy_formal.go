package schemes

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"caipiao/backend/internal/cloud/schemestate"
	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/lottery"
	"caipiao/backend/internal/schemebetting"
)

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

	now := time.Now().UTC()
	periodRows, err := q.ListOpenProviderPeriodSnapshots(ctx, row.LotteryCode, row.PeriodNo, now, now.Add(-6*time.Second), 8)
	if err != nil {
		return true, shadowDecisionResult{}, err
	}
	if len(periodRows) > 0 && !periodRows[0].DatabaseNow.IsZero() {
		now = periodRows[0].DatabaseNow.UTC()
	}
	snapshots := make([]schemebetting.PeriodSnapshot, 0, len(periodRows))
	for _, item := range periodRows {
		openAt := time.Time{}
		if item.OpenAt.Valid {
			openAt = item.OpenAt.Time.UTC()
		}
		snapshots = append(snapshots, schemebetting.PeriodSnapshot{PeriodNo: item.PeriodNo, OpenAt: openAt, CloseAt: item.CloseAt.UTC(), ObservedAt: item.ObservedAt.UTC()})
	}
	target, ok := schemebetting.SelectTargetPeriod(snapshots, row.PeriodNo, now, 6*time.Second)
	if !ok {
		blocked, err := p.persistFormalBlocked(ctx, q, row, stateVersionBefore, result, "no_fresh_provider_target")
		return true, blocked, err
	}
	var providerSnapshotID int64
	for _, item := range periodRows {
		if item.PeriodNo == target.PeriodNo && item.CloseAt.UTC().Equal(target.CloseAt.UTC()) {
			providerSnapshotID = item.ID
			break
		}
	}
	command, err := schemebetting.BuildShadowCommand(schemebetting.ShadowCommandInput{
		SchemeID: row.SchemeID, LotteryCode: row.LotteryCode, SourcePeriod: row.PeriodNo,
		Target: target, ProviderSnapshotID: providerSnapshotID, StateVersion: stateVersionBefore + 1,
		RuleSnapshotHash: row.RuleSnapshotHash.String, LocalHit: result.Hit, Now: now,
		Budget: shadowDeadlineBudget(target), ShardCount: shadowOutboxShardCount,
	})
	if err != nil {
		blocked, blockErr := p.persistFormalBlocked(ctx, q, row, stateVersionBefore, result, "unsafe_deadline")
		return true, blocked, blockErr
	}

	if err := schemestate.ProcessStrategyAfterDraw(ctx, q, inst, row.PeriodNo, result.Hit, definitionConfig); err != nil {
		return true, shadowDecisionResult{}, err
	}
	advanced, err := q.GetSchemeInstanceFull(ctx, row.SchemeID)
	if err != nil {
		return true, shadowDecisionResult{}, err
	}
	frozen, err := p.buildFormalFrozenRequest(ctx, q, advanced, command.TargetPeriod, command.RequestID)
	if err != nil {
		return true, shadowDecisionResult{}, fmt.Errorf("freeze formal request: %w", err)
	}
	diagnostics, err := json.Marshal(map[string]any{
		"mode": mode, "targetPeriod": command.TargetPeriod, "requestId": command.RequestID,
		"safeDeadline": command.SafeDeadline.Format(time.RFC3339Nano), "chainId": execution.ChainID,
		"chainSeq": execution.ChainSeq + 1,
	})
	if err != nil {
		return true, shadowDecisionResult{}, err
	}
	decisionID, created, err := q.InsertSchemePeriodDecision(ctx, sqlcdb.InsertSchemePeriodDecisionParams{
		SchemeID: row.SchemeID, LotteryCode: row.LotteryCode, SourcePeriodNo: row.PeriodNo,
		SourceBetRecordID: row.RecordID, DrawHash: lottery.CanonicalDrawHash(row.LotteryCode, row.PeriodNo, row.Balls),
		StateVersionBefore: stateVersionBefore, StateVersionAfter: stateVersionBefore + 1,
		RuleVersion: row.RuleVersion, RuleSnapshotHash: row.RuleSnapshotHash,
		LocalHit: result.Hit, WinningUnits: result.WinningUnits, Status: "completed", Diagnostics: diagnostics,
	})
	if err != nil {
		return true, shadowDecisionResult{}, err
	}
	formal := shadowDecisionResult{Status: "completed", DecisionID: decisionID, TargetPeriod: command.TargetPeriod, RequestID: command.RequestID, SafeDeadline: command.SafeDeadline.Format(time.RFC3339Nano)}
	if !created {
		return true, formal, nil
	}
	if err := q.InsertFormalSchemeBetOutbox(ctx, sqlcdb.InsertFormalSchemeBetOutboxParams{
		DecisionID: decisionID, SchemeID: row.SchemeID, MemberID: inst.MemberID, LotteryCode: row.LotteryCode,
		SourcePeriodNo: row.PeriodNo, TargetPeriodNo: command.TargetPeriod, Mode: mode,
		RequestID: command.RequestID, PayloadHash: command.PayloadHash, Payload: command.Payload,
		FrozenRequest: frozen, FrozenRequestHash: schemebetting.CanonicalJSONPayloadHash(frozen), ProviderSnapshotID: command.ProviderSnapshotID,
		CloseAt: command.CloseAt, SafeDeadlineAt: command.SafeDeadline, ShardNo: int32(command.ShardNo),
		SourceStateVersion: stateVersionBefore + 1, InitialBet: false,
		ChainID: execution.ChainID, ChainSeq: execution.ChainSeq + 1,
	}); err != nil {
		return true, shadowDecisionResult{}, err
	}
	return true, formal, nil
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
