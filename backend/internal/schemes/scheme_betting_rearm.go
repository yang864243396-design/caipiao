package schemes

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/schemebetting"
)

func (w *Worker) EnableEventScheme(ctx context.Context, schemeID, actor, reason string) error {
	return w.startEventSchemeChain(ctx, schemeID, actor, reason, "enable_event", true)
}

func (w *Worker) RearmEventScheme(ctx context.Context, schemeID, actor, reason string) error {
	return w.startEventSchemeChain(ctx, schemeID, actor, reason, "rearm", false)
}

func (w *Worker) startEventSchemeChain(ctx context.Context, schemeID, actor, reason, action string, allowLegacy bool) error {
	if w == nil || w.pool == nil || w.q == nil || w.strategyProcessor == nil {
		return errors.New("scheme event worker unavailable")
	}
	schemeID = strings.TrimSpace(schemeID)
	actor = strings.TrimSpace(actor)
	reason = strings.TrimSpace(reason)
	if schemeID == "" || actor == "" || len([]rune(reason)) < 4 {
		return errors.New("scheme, actor, and a reason of at least 4 characters are required")
	}
	if action != "enable_event" && action != "rearm" {
		return errors.New("unsupported scheme betting action")
	}
	tx, err := w.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	q := w.q.WithTx(tx)
	stateVersion, err := q.LockSchemeStateVersion(ctx, schemeID)
	if err != nil {
		return err
	}
	inst, err := q.GetSchemeInstanceFull(ctx, schemeID)
	if err != nil {
		return err
	}
	if inst.Status != "running" || inst.SimBet {
		return errors.New("only running formal schemes can use event betting")
	}
	mode := w.strategyProcessor.formalModeForLottery(inst.LotteryCode)
	if mode == "" {
		return errors.New("scheme lottery is not in formal event allowlist")
	}
	execution, err := q.GetSchemeBettingExecutionState(ctx, schemeID)
	if err != nil {
		return err
	}
	if allowLegacy {
		if execution.Owner != "legacy" {
			return errors.New("scheme is already event-owned")
		}
	} else if execution.Owner != "event" || execution.ChainState != string(schemebetting.ChainStateBlockedRequiresRearm) {
		return errors.New("only blocked event-owned schemes can be rearmed")
	}
	if action == "rearm" {
		if err := q.EnsureNoUnresolvedSchemeBet(ctx, schemeID); err != nil {
			return err
		}
	}
	shardNo := int32(schemebetting.ShardForScheme(schemeID, shadowOutboxShardCount))
	if w.bettingBacklog == nil {
		return errors.New("capacity_strategy_backlog_probe_unavailable")
	}
	if err := w.bettingBacklog.CheckSchemeBettingBacklog(ctx, shardNo); err != nil {
		return err
	}
	if err := q.CheckSchemeBettingCapacity(ctx, inst.LotteryCode, shardNo); err != nil {
		return err
	}
	now := time.Now().UTC()
	periodRows, err := q.ListOpenProviderPeriodSnapshots(ctx, inst.LotteryCode, "", now, now.Add(-6*time.Second), 8)
	if err != nil {
		return err
	}
	snapshots := make([]schemebetting.PeriodSnapshot, 0, len(periodRows))
	for _, item := range periodRows {
		openAt := time.Time{}
		if item.OpenAt.Valid {
			openAt = item.OpenAt.Time.UTC()
		}
		snapshots = append(snapshots, schemebetting.PeriodSnapshot{
			PeriodNo: item.PeriodNo, OpenAt: openAt, CloseAt: item.CloseAt.UTC(), ObservedAt: item.ObservedAt.UTC(),
		})
	}
	target, ok := schemebetting.SelectTargetPeriod(snapshots, "", now, 6*time.Second)
	if !ok {
		return errors.New("no_fresh_provider_target")
	}
	var providerSnapshotID int64
	for _, item := range periodRows {
		if item.PeriodNo == target.PeriodNo && item.CloseAt.UTC().Equal(target.CloseAt.UTC()) {
			providerSnapshotID = item.ID
			break
		}
	}
	chainID, err := newSchemeChainID()
	if err != nil {
		return err
	}
	sourcePeriod := "initial:" + chainID
	command, err := schemebetting.BuildShadowCommand(schemebetting.ShadowCommandInput{
		SchemeID: schemeID, LotteryCode: inst.LotteryCode, SourcePeriod: sourcePeriod,
		Target: target, ProviderSnapshotID: providerSnapshotID, StateVersion: stateVersion,
		Now: now, Budget: shadowDeadlineBudget(target), ShardCount: shadowOutboxShardCount,
	})
	if err != nil {
		return err
	}
	frozen, err := w.strategyProcessor.buildFormalFrozenRequest(ctx, q, inst, command.TargetPeriod, command.RequestID)
	if err != nil {
		return err
	}
	beforeState, _ := json.Marshal(execution)
	if err := q.ActivateSchemeBettingChain(ctx, schemeID, chainID, allowLegacy); err != nil {
		return err
	}
	diagnostics, _ := json.Marshal(map[string]any{
		"mode": mode, "initialBet": true, "actor": actor, "reason": reason, "chainId": chainID, "action": action,
	})
	decisionID, created, err := q.InsertSchemePeriodDecision(ctx, sqlcdb.InsertSchemePeriodDecisionParams{
		SchemeID: schemeID, LotteryCode: inst.LotteryCode, SourcePeriodNo: sourcePeriod,
		StateVersionBefore: stateVersion, StateVersionAfter: stateVersion,
		Status: "completed", Diagnostics: diagnostics,
	})
	if err != nil {
		return err
	}
	if !created {
		return errors.New("initial decision already exists")
	}
	if err := q.InsertFormalSchemeBetOutbox(ctx, sqlcdb.InsertFormalSchemeBetOutboxParams{
		DecisionID: decisionID, SchemeID: schemeID, MemberID: inst.MemberID, LotteryCode: inst.LotteryCode,
		SourcePeriodNo: sourcePeriod, TargetPeriodNo: command.TargetPeriod, Mode: mode,
		RequestID: command.RequestID, PayloadHash: command.PayloadHash, Payload: command.Payload,
		FrozenRequest: frozen, FrozenRequestHash: schemebetting.PayloadHash(frozen), ProviderSnapshotID: command.ProviderSnapshotID,
		CloseAt: command.CloseAt, SafeDeadlineAt: command.SafeDeadline, ShardNo: int32(command.ShardNo),
		SourceStateVersion: stateVersion, InitialBet: true, ChainID: chainID, ChainSeq: 1,
	}); err != nil {
		return err
	}
	outboxID, err := q.SchemeBetOutboxIDByDecision(ctx, decisionID)
	if err != nil {
		return err
	}
	afterState, _ := json.Marshal(map[string]any{
		"owner": "event", "chainState": "active", "chainId": chainID, "chainSeq": 0,
	})
	if err := q.InsertSchemeBettingAdminAction(ctx, sqlcdb.InsertSchemeBettingAdminActionParams{
		SchemeID: schemeID, OutboxID: outboxID, Action: action, Actor: actor,
		Reason: reason, BeforeState: beforeState, AfterState: afterState,
	}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func newSchemeChainID() (string, error) {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(value[:]), nil
}
