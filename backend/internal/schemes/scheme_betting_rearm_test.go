package schemes

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/lottery"
)

type availableRearmBacklog struct{}

func (availableRearmBacklog) CheckSchemeBettingBacklog(context.Context, int32) error { return nil }

type pausedMissedRearmFixture struct {
	*resolverEntryFixture
	worker *Worker
}

type pausedMissedRearmState struct {
	status, statusReason, failureDetail                   string
	chainID, chainState, chainBlockReason                 string
	stateVersion, chainSeq                                int64
	roundIndex, pickIndex                                 int32
	currentPick, lastDirection, oldAwaitingDecisionStatus string
	outboxCount                                           int
}

func TestManualRearmEligibilityAllowsExplicitPausedMissedChain(t *testing.T) {
	if !isManualRearmInstanceEligible("paused", "event", "blocked_requires_rearm", "missed_contiguous_period") {
		t.Fatal("paused missed contiguous chain must be eligible for explicit manual rearm")
	}
}

func TestManualRearmEligibilityPreservesRunningBlockedChain(t *testing.T) {
	if !isManualRearmInstanceEligible("running", "event", "blocked_requires_rearm", "") {
		t.Fatal("existing running blocked-chain rearm must remain eligible")
	}
}

func TestManualRearmEligibilityAllowsExplicitOperatorBlockReasons(t *testing.T) {
	for _, reason := range []string{
		"provider_accepted_wrong_period",
		"provider_acceptance_unknown",
		"contiguous_target_configuration",
		"admin_cancelled_before_send",
	} {
		if !isManualRearmInstanceEligible("paused", "event", "blocked_requires_rearm", reason) {
			t.Fatalf("explicit paused block reason %q must be manually rearmable", reason)
		}
	}
}

func TestManualRearmEligibilityRejectsGenericPausedFailure(t *testing.T) {
	for _, candidate := range []struct {
		status, owner, chainState, blockReason string
	}{
		{status: "paused", owner: "event", chainState: "blocked_requires_rearm", blockReason: ""},
		{status: "paused", owner: "event", chainState: "blocked_requires_rearm", blockReason: "bet_failed"},
		{status: "paused", owner: "legacy", chainState: "blocked_requires_rearm", blockReason: "missed_contiguous_period"},
		{status: "paused", owner: "event", chainState: "active", blockReason: "missed_contiguous_period"},
	} {
		if isManualRearmInstanceEligible(candidate.status, candidate.owner, candidate.chainState, candidate.blockReason) {
			t.Fatalf("generic paused failure was eligible for manual rearm: %+v", candidate)
		}
	}
}

func TestRearmEventSchemeReactivatesPausedMissedChainAtomically(t *testing.T) {
	f := newPausedMissedRearmFixture(t)
	f.publishExactBoundary()
	before := f.snapshot("")

	if err := f.worker.RearmEventScheme(f.ctx, f.schemeID, "operator", "restart explicitly missed chain"); err != nil {
		t.Fatal(err)
	}
	after := f.snapshot("")
	if after.status != "running" || after.statusReason != "" || after.failureDetail != "" {
		t.Fatalf("rearmed status=%q reason=%q detail=%q, want running with failures cleared", after.status, after.statusReason, after.failureDetail)
	}
	if after.chainID == before.chainID || after.chainState != "active" || after.chainBlockReason != "" || after.chainSeq != 0 {
		t.Fatalf("rearmed chain id=%q state=%q reason=%q seq=%d", after.chainID, after.chainState, after.chainBlockReason, after.chainSeq)
	}
	if after.stateVersion != before.stateVersion+1 || after.roundIndex != 0 || after.pickIndex != 0 || after.currentPick != "" || after.lastDirection != "" {
		t.Fatalf("rearmed strategy version=%d round=%d pick=%d current=%q direction=%q", after.stateVersion, after.roundIndex, after.pickIndex, after.currentPick, after.lastDirection)
	}
	if after.outboxCount != before.outboxCount+1 {
		t.Fatalf("rearmed outbox count=%d, want %d", after.outboxCount, before.outboxCount+1)
	}
	var sourceVersion, outboxChainSeq int64
	if err := f.pool.QueryRow(f.ctx, `
SELECT source_state_version, chain_seq
FROM scheme_bet_outbox
WHERE scheme_id = $1 AND chain_id = $2`, f.schemeID, after.chainID).Scan(&sourceVersion, &outboxChainSeq); err != nil {
		t.Fatal(err)
	}
	if sourceVersion != after.stateVersion || outboxChainSeq != 1 {
		t.Fatalf("initial outbox sourceVersion=%d chainSeq=%d, want %d/1", sourceVersion, outboxChainSeq, after.stateVersion)
	}
}

func TestRearmEventSchemeNoFreshTargetPreservesPausedMissedState(t *testing.T) {
	f := newPausedMissedRearmFixture(t)
	f.publishStaleBoundary()
	oldWait := f.seedOldAwaitingDecision()
	before := f.snapshot(oldWait)

	err := f.worker.RearmEventScheme(f.ctx, f.schemeID, "operator", "restart explicitly missed chain")
	if err == nil || !strings.Contains(err.Error(), "no_fresh_provider_target") {
		t.Fatalf("RearmEventScheme() error=%v, want no_fresh_provider_target", err)
	}
	after := f.snapshot(oldWait)
	if after != before {
		t.Fatalf("no-target rearm mutated paused state:\n before=%+v\n after=%+v", before, after)
	}
}

func TestRearmEventSchemeFrozenBuildFailureRollsBackPausedMissedState(t *testing.T) {
	f := newPausedMissedRearmFixture(t)
	f.publishExactBoundary()
	f.worker.SetRuleRegistry(nil)
	oldWait := f.seedOldAwaitingDecision()
	before := f.snapshot(oldWait)

	err := f.worker.RearmEventScheme(f.ctx, f.schemeID, "operator", "restart explicitly missed chain")
	if err == nil || !strings.Contains(err.Error(), "published rule snapshot unavailable") {
		t.Fatalf("RearmEventScheme() error=%v, want injected frozen-request failure", err)
	}
	after := f.snapshot(oldWait)
	if after != before {
		t.Fatalf("frozen-build failure mutated paused state:\n before=%+v\n after=%+v", before, after)
	}
}

func newPausedMissedRearmFixture(t *testing.T) *pausedMissedRearmFixture {
	t.Helper()
	base := newResolverEntryFixture(t)
	q := sqlcdb.New(base.pool)
	decisionID := base.seed(base.databaseNow().Add(-time.Second))
	missed, err := q.MissAwaitingContiguousTarget(base.ctx, sqlcdb.MissAwaitingContiguousTargetParams{
		DecisionID: decisionID, FailureReason: "missed_contiguous_period", Diagnostics: []byte(`{"source":"rearm-test"}`),
	})
	if err != nil || !missed {
		t.Fatalf("seed paused missed chain: missed=%v err=%v", missed, err)
	}
	var capacityReady bool
	if err := base.pool.QueryRow(base.ctx, `
SELECT EXISTS (SELECT 1 FROM scheme_betting_capacity_limits WHERE lottery_code = $1 AND enabled)`, base.lotteryCode).Scan(&capacityReady); err != nil || !capacityReady {
		t.Skipf("scheme betting capacity unavailable: ready=%v err=%v", capacityReady, err)
	}
	owner := fmt.Sprintf("manual-rearm-test-%d", time.Now().UnixNano())
	for _, kind := range []string{"strategy", "dispatcher"} {
		epoch, acquired, err := q.AcquireSchemeBettingShardLease(base.ctx, kind, base.shardNo, owner, time.Minute)
		if err != nil {
			t.Fatal(err)
		}
		if !acquired {
			t.Skipf("%s shard %d lease is held by another process", kind, base.shardNo)
		}
		t.Cleanup(func() {
			_, _ = base.pool.Exec(context.Background(), `
DELETE FROM scheme_betting_shard_leases
WHERE lease_kind = $1 AND shard_no = $2 AND lease_owner = $3 AND lease_epoch = $4`, kind, base.shardNo, owner, epoch)
		})
	}
	worker := NewWorker(base.pool, 1, nil, nil)
	worker.SetSchemeBettingMode("gray", []string{base.lotteryCode})
	worker.SetRuleRegistry(base.processor.ruleRegistry)
	worker.SetSchemeBettingBacklogProbe(availableRearmBacklog{})
	return &pausedMissedRearmFixture{resolverEntryFixture: base, worker: worker}
}

func (f *pausedMissedRearmFixture) publishStaleBoundary() {
	f.t.Helper()
	if !lottery.UpdatePeriodState(f.lotteryCode, f.source, f.target, time.Now().UTC().Add(-time.Minute), 15) {
		f.t.Fatalf("publish stale websocket boundary %s -> %s", f.source, f.target)
	}
}

func (f *pausedMissedRearmFixture) seedOldAwaitingDecision() string {
	f.t.Helper()
	source := f.source + "-old-wait"
	_, created, err := sqlcdb.New(f.pool).InsertSchemePeriodDecision(f.ctx, sqlcdb.InsertSchemePeriodDecisionParams{
		SchemeID: f.schemeID, LotteryCode: f.lotteryCode, SourcePeriodNo: source,
		StateVersionBefore: 7, StateVersionAfter: 8, Status: "awaiting_target",
	})
	if err != nil || !created {
		f.t.Fatalf("seed old awaiting decision: created=%v err=%v", created, err)
	}
	return source
}

func (f *pausedMissedRearmFixture) snapshot(oldWait string) pausedMissedRearmState {
	f.t.Helper()
	var state pausedMissedRearmState
	if err := f.pool.QueryRow(f.ctx, `
SELECT status, COALESCE(status_reason, ''), COALESCE(bet_failed_detail, ''),
       COALESCE(chain_id, ''), strict_chain_state, COALESCE(chain_block_reason, ''),
       state_version, chain_seq, round_index, pick_index, current_pick, last_direction,
       (SELECT COUNT(*)::int FROM scheme_bet_outbox WHERE scheme_id = scheme_instances.id)
FROM scheme_instances WHERE id = $1`, f.schemeID).Scan(
		&state.status, &state.statusReason, &state.failureDetail,
		&state.chainID, &state.chainState, &state.chainBlockReason,
		&state.stateVersion, &state.chainSeq, &state.roundIndex, &state.pickIndex,
		&state.currentPick, &state.lastDirection, &state.outboxCount,
	); err != nil {
		f.t.Fatal(err)
	}
	if oldWait != "" {
		if err := f.pool.QueryRow(f.ctx, `
SELECT status FROM scheme_period_decisions WHERE scheme_id = $1 AND source_period_no = $2`, f.schemeID, oldWait).Scan(&state.oldAwaitingDecisionStatus); err != nil {
			f.t.Fatal(err)
		}
	}
	return state
}

func TestManualRestartCreatesNewChainAndResetsRound(t *testing.T) {
	f := newFormalStrategyEntryFixture(t, "15s", fakedPastDrawTime())
	q := sqlcdb.New(f.pool)
	oldChain := "manual-rearm-old-chain"
	if _, err := f.pool.Exec(f.ctx, `
UPDATE scheme_instances
SET state_version = 7,
    round_index = 2,
    pick_index = 3,
    current_pick = '1,2,3',
    last_direction = 'neg',
    lookback_round_reset_pending = TRUE,
    strict_chain_state = 'blocked_requires_rearm',
    chain_block_reason = 'missed_contiguous_period',
    chain_id = $2,
    chain_seq = 4
WHERE id = $1`, f.schemeID, oldChain); err != nil {
		t.Fatal(err)
	}

	version, err := q.ResetSchemeStrategyForNewChain(f.ctx, f.schemeID, 7)
	if err != nil {
		t.Fatal(err)
	}
	if version != 8 {
		t.Fatalf("state version=%d, want 8", version)
	}
	if err := q.ActivateSchemeBettingChain(f.ctx, f.schemeID, "manual-rearm-new-chain", false); err != nil {
		t.Fatal(err)
	}

	var stateVersion int64
	var roundIndex, pickIndex int32
	var currentPick, lastDirection, chainID string
	var resetPending bool
	var chainBlockReason *string
	if err := f.pool.QueryRow(f.ctx, `
SELECT state_version, round_index, pick_index, current_pick, last_direction,
       lookback_round_reset_pending, chain_id, chain_block_reason
FROM scheme_instances WHERE id = $1`, f.schemeID).Scan(
		&stateVersion, &roundIndex, &pickIndex, &currentPick, &lastDirection,
		&resetPending, &chainID, &chainBlockReason,
	); err != nil {
		t.Fatal(err)
	}
	if stateVersion != 8 || roundIndex != 0 || pickIndex != 0 || currentPick != "" || lastDirection != "" || resetPending {
		t.Fatalf("manual restart state was not reset: version=%d round=%d pick=%d current=%q direction=%q reset=%v", stateVersion, roundIndex, pickIndex, currentPick, lastDirection, resetPending)
	}
	if chainID == oldChain || chainID != "manual-rearm-new-chain" || chainBlockReason != nil {
		t.Fatalf("manual restart chain was not replaced and unblocked: chain=%q reason=%v", chainID, chainBlockReason)
	}

	_, found, err := q.PendingFormalStrategyRowForSchemeDraw(f.ctx, f.recordID, f.schemeID, f.lotteryCode, f.periodNo, 7)
	if err != nil {
		t.Fatal(err)
	}
	if found {
		t.Fatal("stale strategy-ready event matched reset state")
	}
}

func TestManualRestartBreaksAwaitingTargetWithoutCreatingOrder(t *testing.T) {
	f := newFormalStrategyEntryFixture(t, "15s", fakedPastDrawTime())
	f.preseedIncompleteDecision()
	q := sqlcdb.New(f.pool)

	if err := q.BreakAwaitingTargetRowsForNewChain(f.ctx, f.schemeID); err != nil {
		t.Fatal(err)
	}

	var status, reason string
	if err := f.pool.QueryRow(f.ctx, `
SELECT status, COALESCE(failure_reason, '')
FROM scheme_period_decisions
WHERE scheme_id = $1 AND source_period_no = $2`, f.schemeID, f.periodNo).Scan(&status, &reason); err != nil {
		t.Fatal(err)
	}
	if status != "chain_broken" || reason != "manual_rearm_replaced_chain" {
		t.Fatalf("awaiting target was not broken: status=%q reason=%q", status, reason)
	}
	var outboxCount int
	if err := f.pool.QueryRow(f.ctx, `SELECT COUNT(*)::int FROM scheme_bet_outbox WHERE scheme_id = $1`, f.schemeID).Scan(&outboxCount); err != nil {
		t.Fatal(err)
	}
	if outboxCount != 0 {
		t.Fatalf("breaking old awaiting target created %d outbox rows", outboxCount)
	}
}

func fakedPastDrawTime() time.Time { return time.Now().UTC() }
