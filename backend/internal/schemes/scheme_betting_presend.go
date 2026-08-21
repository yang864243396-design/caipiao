package schemes

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/providerperiodtarget"
	"caipiao/backend/internal/schemebetting"
)

var errNoFreshPreSendReplacementTarget = errors.New("no fresh provider target for pre-send replacement")

type permanentPreSendReplacementError struct{ err error }

func (err permanentPreSendReplacementError) Error() string { return err.err.Error() }
func (err permanentPreSendReplacementError) Unwrap() error { return err.err }

func (w *Worker) HandlePreSendFailure(ctx context.Context, outboxID int64) error {
	schemeID, err := w.reschedulePreSendFailure(ctx, outboxID)
	if err == nil {
		return nil
	}
	var permanent permanentPreSendReplacementError
	if !errors.As(err, &permanent) && w != nil && w.q != nil {
		return w.q.DeferPreSendFailureReschedule(ctx, outboxID, err.Error())
	}
	if schemeID != "" && w != nil && w.q != nil {
		reason := boundedPreSendFailureReason(err)
		if blockErr := w.q.BlockSchemeBettingChain(ctx, schemeID, reason, time.Now().UTC()); blockErr != nil {
			return fmt.Errorf("%v; block failed pre-send replacement: %w", err, blockErr)
		}
	}
	return err
}

func (w *Worker) reschedulePreSendFailure(ctx context.Context, outboxID int64) (string, error) {
	if w == nil || w.pool == nil || w.q == nil || w.strategyProcessor == nil {
		return "", errors.New("scheme pre-send replacement worker unavailable")
	}
	tx, err := w.pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)
	q := w.q.WithTx(tx)
	failed, found, err := q.GetPreSendFailureOutbox(ctx, outboxID)
	if err != nil || !found {
		return "", err
	}
	stateVersion, err := q.LockSchemeStateVersion(ctx, failed.SchemeID)
	if err != nil {
		return failed.SchemeID, err
	}
	inst, err := q.GetSchemeInstanceFull(ctx, failed.SchemeID)
	if err != nil {
		return failed.SchemeID, err
	}
	execution, err := q.GetSchemeBettingExecutionState(ctx, failed.SchemeID)
	if err != nil {
		return failed.SchemeID, err
	}
	if execution.Owner != "event" || execution.ChainState != string(schemebetting.ChainStateActive) || strings.TrimSpace(execution.ChainID) == "" {
		return failed.SchemeID, errors.New("scheme chain is not active for pre-send replacement")
	}
	shardNo := int32(schemebetting.ShardForScheme(failed.SchemeID, shadowOutboxShardCount))
	if w.bettingBacklog == nil {
		return failed.SchemeID, errors.New("capacity_strategy_backlog_probe_unavailable")
	}
	if err := w.bettingBacklog.CheckSchemeBettingBacklog(ctx, shardNo); err != nil {
		return failed.SchemeID, err
	}
	if err := q.CheckSchemeBettingCapacity(ctx, failed.LotteryCode, shardNo); err != nil {
		return failed.SchemeID, err
	}

	now := time.Now().UTC()
	target, providerSnapshotID, ok, err := w.preSendReplacementTarget(ctx, q, failed, now)
	if err != nil {
		return failed.SchemeID, err
	}
	if !ok {
		return failed.SchemeID, errNoFreshPreSendReplacementTarget
	}
	if providerSnapshotID <= 0 {
		return failed.SchemeID, errors.New("provider_snapshot_missing_for_pre_send_replacement")
	}
	sourcePeriod := fmt.Sprintf("pre-send:%d", outboxID)
	command, err := schemebetting.BuildShadowCommand(schemebetting.ShadowCommandInput{
		SchemeID: failed.SchemeID, LotteryCode: failed.LotteryCode, SourcePeriod: sourcePeriod,
		Target: target, ProviderSnapshotID: providerSnapshotID, StateVersion: stateVersion,
		Now: now, Budget: shadowDeadlineBudget(target), ShardCount: shadowOutboxShardCount,
	})
	if err != nil {
		if strings.Contains(err.Error(), "no safe dispatch window") {
			return failed.SchemeID, errNoFreshPreSendReplacementTarget
		}
		return failed.SchemeID, permanentPreSendReplacementError{err: err}
	}
	frozen, err := w.strategyProcessor.buildFormalFrozenRequest(ctx, q, inst, command.TargetPeriod, command.RequestID)
	if err != nil {
		return failed.SchemeID, permanentPreSendReplacementError{err: fmt.Errorf("freeze pre-send replacement: %w", err)}
	}
	diagnostics, err := json.Marshal(map[string]any{
		"mode": failed.Mode, "preSendReplacement": true, "failedOutboxId": outboxID,
		"failedPeriod": failed.FailedPeriod, "targetPeriod": command.TargetPeriod,
	})
	if err != nil {
		return failed.SchemeID, err
	}
	decisionID, created, err := q.InsertSchemePeriodDecision(ctx, sqlcdb.InsertSchemePeriodDecisionParams{
		SchemeID: failed.SchemeID, LotteryCode: failed.LotteryCode, SourcePeriodNo: sourcePeriod,
		StateVersionBefore: stateVersion, StateVersionAfter: stateVersion, Status: "completed", Diagnostics: diagnostics,
	})
	if err != nil {
		return failed.SchemeID, err
	}
	if created {
		if err := q.InsertFormalSchemeBetOutbox(ctx, sqlcdb.InsertFormalSchemeBetOutboxParams{
			DecisionID: decisionID, SchemeID: failed.SchemeID, MemberID: inst.MemberID, LotteryCode: failed.LotteryCode,
			SourcePeriodNo: sourcePeriod, TargetPeriodNo: command.TargetPeriod, Mode: failed.Mode,
			RequestID: command.RequestID, PayloadHash: command.PayloadHash, Payload: command.Payload,
			FrozenRequest: frozen, FrozenRequestHash: schemebetting.CanonicalJSONPayloadHash(frozen), ProviderSnapshotID: command.ProviderSnapshotID,
			CloseAt: command.CloseAt, SafeDeadlineAt: command.SafeDeadline, ShardNo: int32(command.ShardNo),
			SourceStateVersion: stateVersion, InitialBet: true, ChainID: execution.ChainID, ChainSeq: execution.ChainSeq + 1,
		}); err != nil {
			return failed.SchemeID, err
		}
	}
	replacementOutboxID, err := q.SchemeBetOutboxIDByDecision(ctx, decisionID)
	if err != nil {
		return failed.SchemeID, err
	}
	if err := q.MarkPreSendFailureRescheduled(ctx, outboxID, replacementOutboxID); err != nil {
		return failed.SchemeID, err
	}
	if err := tx.Commit(ctx); err != nil {
		return failed.SchemeID, err
	}
	return failed.SchemeID, nil
}

func (w *Worker) preSendReplacementTarget(
	ctx context.Context,
	q *sqlcdb.Queries,
	failed sqlcdb.PreSendFailureOutbox,
	now time.Time,
) (schemebetting.PeriodSnapshot, int64, bool, error) {
	return providerperiodtarget.Current(ctx, q, failed.LotteryCode, failed.FailedPeriod, now)
}

func boundedPreSendFailureReason(err error) string {
	const prefix = "pre_send_reschedule_failed: "
	const maxRunes = 500
	detail := prefix + strings.TrimSpace(err.Error())
	runes := []rune(detail)
	if len(runes) > maxRunes {
		detail = string(runes[:maxRunes])
	}
	return detail
}
