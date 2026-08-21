package schemes

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/lottery"
	"caipiao/backend/internal/periodissue"
	"caipiao/backend/internal/providerperiodtarget"
	"caipiao/backend/internal/schemebetting"
)

type contiguousTargetResolution uint8

const (
	contiguousTargetWaiting contiguousTargetResolution = iota
	contiguousTargetResolved
	contiguousTargetMissed
)

// classifyContiguousTargetBoundary keeps a wait recoverable until a websocket
// boundary proves that its source has been passed. A matching boundary may
// resolve only its literal immediate successor.
func classifyContiguousTargetBoundary(sourcePeriod, wsCurrent, wsNext string) contiguousTargetResolution {
	sourcePeriod = strings.TrimSpace(sourcePeriod)
	wsCurrent = strings.TrimSpace(wsCurrent)
	wsNext = strings.TrimSpace(wsNext)
	if sourcePeriod == "" || wsCurrent == "" {
		return contiguousTargetWaiting
	}
	if wsCurrent == sourcePeriod {
		if isImmediateContiguousSuccessor(sourcePeriod, wsNext) {
			return contiguousTargetResolved
		}
		return contiguousTargetMissed
	}
	if periodissue.Advances(sourcePeriod, wsCurrent) {
		return contiguousTargetMissed
	}
	return contiguousTargetWaiting
}

func isImmediateContiguousSuccessor(sourcePeriod, targetPeriod string) bool {
	sourcePrefix, sourceSequence, sourceOK := splitTrailingDecimal(sourcePeriod)
	targetPrefix, targetSequence, targetOK := splitTrailingDecimal(targetPeriod)
	if !sourceOK || !targetOK || sourcePrefix != targetPrefix {
		return false
	}
	return incrementDecimal(sourceSequence) == targetSequence
}

func splitTrailingDecimal(period string) (prefix, sequence string, ok bool) {
	period = strings.TrimSpace(period)
	start := len(period)
	for start > 0 && period[start-1] >= '0' && period[start-1] <= '9' {
		start--
	}
	if start == len(period) {
		return "", "", false
	}
	return period[:start], period[start:], true
}

func incrementDecimal(sequence string) string {
	if sequence == "" {
		return ""
	}
	value := []byte(sequence)
	for index := len(value) - 1; index >= 0; index-- {
		if value[index] < '9' {
			value[index]++
			return string(value)
		}
		value[index] = '0'
	}
	return "1" + string(value)
}

// ResolveAwaitingTarget is phase two of a formal contiguous chain. Its entire
// state transition, target snapshot, frozen command, outbox, and chain
// sequence update are committed together. It never consults a REST period
// source and never evaluates or settles strategy state.
func (p *StrategyProcessor) ResolveAwaitingTarget(ctx context.Context, decisionID int64) error {
	if p == nil || p.pool == nil || p.q == nil || decisionID <= 0 {
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
	row, found, err := qtx.GetAwaitingContiguousTargetForUpdate(ctx, decisionID)
	if err != nil || !found {
		if err != nil {
			return err
		}
		return tx.Commit(ctx)
	}
	if fence, ok := strategyLeaseFenceFromContext(ctx); ok && row.ShardNo != fence.ShardNo {
		return fmt.Errorf("contiguous target decision %d belongs to shard %d, not lease shard %d", row.DecisionID, row.ShardNo, fence.ShardNo)
	}
	var databaseNowTime time.Time
	if err := tx.QueryRow(ctx, `SELECT clock_timestamp()`).Scan(&databaseNowTime); err != nil {
		return err
	}
	databaseNowTime = databaseNowTime.UTC()
	if !databaseNowTime.Before(row.TargetDeadlineAt.UTC()) {
		if err := p.missResolvedContiguousTarget(ctx, qtx, row.DecisionID, "target_deadline_elapsed", false); err != nil {
			return err
		}
		return tx.Commit(ctx)
	}
	if !lottery.RequiresFreshShortPeriodWSBetTarget(row.LotteryCode) {
		// Awaiting-target rows are only produced for formal short lotteries in
		// production. Never weaken that contract with a REST fallback if a
		// malformed or historical row slips through.
		return tx.Commit(ctx)
	}
	state, _ := lottery.PeriodStateFor(row.LotteryCode)
	switch classifyContiguousTargetBoundary(row.SourcePeriodNo, state.CurrentIssue, state.NextIssue) {
	case contiguousTargetMissed:
		if err := p.missResolvedContiguousTarget(ctx, qtx, row.DecisionID, "missed_contiguous_period", true); err != nil {
			return err
		}
		return tx.Commit(ctx)
	case contiguousTargetWaiting:
		return tx.Commit(ctx)
	}
	target, snapshotID, ready, err := providerperiodtarget.CurrentUncached(ctx, qtx, row.LotteryCode, row.SourcePeriodNo, databaseNowTime)
	if err != nil {
		return err
	}
	if !ready {
		state, _ = lottery.PeriodStateFor(row.LotteryCode)
		if classifyContiguousTargetBoundary(row.SourcePeriodNo, state.CurrentIssue, state.NextIssue) == contiguousTargetMissed {
			if err := p.missResolvedContiguousTarget(ctx, qtx, row.DecisionID, "missed_contiguous_period", true); err != nil {
				return err
			}
		}
		return tx.Commit(ctx)
	}
	if !isImmediateContiguousSuccessor(row.SourcePeriodNo, target.PeriodNo) {
		if err := p.missResolvedContiguousTarget(ctx, qtx, row.DecisionID, "missed_contiguous_period", true); err != nil {
			return err
		}
		return tx.Commit(ctx)
	}
	if snapshotID <= 0 {
		return fmt.Errorf("contiguous target %q has no provider snapshot", target.PeriodNo)
	}
	inst, err := qtx.GetSchemeInstanceFull(ctx, row.SchemeID)
	if err != nil {
		return err
	}
	command, err := schemebetting.BuildShadowCommand(schemebetting.ShadowCommandInput{
		SchemeID: row.SchemeID, LotteryCode: row.LotteryCode, SourcePeriod: row.SourcePeriodNo,
		Target: target, ProviderSnapshotID: snapshotID, StateVersion: row.StateVersionAfter,
		RuleSnapshotHash: row.RuleSnapshotHash, LocalHit: row.LocalHit, Now: databaseNowTime,
		Budget: shadowDeadlineBudget(target), ShardCount: shadowOutboxShardCount,
	})
	if err != nil {
		return err
	}
	frozen, err := p.buildFormalFrozenRequest(ctx, qtx, inst, command.TargetPeriod, command.RequestID)
	if err != nil {
		return err
	}
	diagnostics, err := json.Marshal(map[string]any{
		"source": "draw_ws_boundary", "targetPeriod": command.TargetPeriod,
		"providerSnapshotId": snapshotID, "requestId": command.RequestID,
	})
	if err != nil {
		return err
	}
	if p.beforeCompleteAwaitingTargetFn != nil {
		if err := p.beforeCompleteAwaitingTargetFn(ctx, row.DecisionID); err != nil {
			return err
		}
	}
	completed, err := qtx.CompleteAwaitingContiguousTarget(ctx, sqlcdb.CompleteAwaitingContiguousTargetParams{
		DecisionID: row.DecisionID, TargetPeriodNo: command.TargetPeriod, Diagnostics: diagnostics,
	})
	if err != nil {
		return err
	}
	if !completed {
		if err := p.missResolvedContiguousTarget(ctx, qtx, row.DecisionID, "target_deadline_elapsed", false); err != nil {
			return err
		}
		return tx.Commit(ctx)
	}
	if _, err := qtx.SchemeBetOutboxIDByDecision(ctx, row.DecisionID); err == nil {
		return fmt.Errorf("awaiting decision %d already has an outbox", row.DecisionID)
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	if err := qtx.InsertFormalSchemeBetOutbox(ctx, sqlcdb.InsertFormalSchemeBetOutboxParams{
		DecisionID: row.DecisionID, SchemeID: row.SchemeID, MemberID: row.MemberID, LotteryCode: row.LotteryCode,
		SourcePeriodNo: row.SourcePeriodNo, TargetPeriodNo: command.TargetPeriod, Mode: row.Mode,
		RequestID: command.RequestID, PayloadHash: command.PayloadHash, Payload: command.Payload,
		FrozenRequest: frozen, FrozenRequestHash: schemebetting.CanonicalJSONPayloadHash(frozen),
		ProviderSnapshotID: snapshotID, CloseAt: command.CloseAt, SafeDeadlineAt: command.SafeDeadline,
		ShardNo: row.ShardNo, SourceStateVersion: row.StateVersionAfter, InitialBet: false,
		ChainID: row.ChainID, ChainSeq: row.ChainSeq + 1,
	}); err != nil {
		return err
	}
	advanced, err := qtx.AdvanceContiguousTargetChainSequence(ctx, row.SchemeID, row.ChainID, row.ChainSeq)
	if err != nil {
		return err
	}
	if !advanced {
		return fmt.Errorf("contiguous target chain sequence fence lost for %q", row.SchemeID)
	}
	return tx.Commit(ctx)
}

func (p *StrategyProcessor) missResolvedContiguousTarget(
	ctx context.Context,
	q *sqlcdb.Queries,
	decisionID int64,
	reason string,
	gap bool,
) error {
	diagnostics, err := json.Marshal(map[string]string{"source": "contiguous_target_resolver", "reason": reason})
	if err != nil {
		return err
	}
	args := sqlcdb.MissAwaitingContiguousTargetParams{DecisionID: decisionID, FailureReason: reason, Diagnostics: diagnostics}
	if gap {
		_, err = q.MissAwaitingContiguousTargetGap(ctx, args)
	} else {
		_, err = p.missAwaitingTarget(ctx, q, args)
	}
	return err
}
