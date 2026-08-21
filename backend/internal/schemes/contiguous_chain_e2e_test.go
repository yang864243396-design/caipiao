package schemes

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/guaji"
	"caipiao/backend/internal/guaji/drawsync"
	"caipiao/backend/internal/guajibet"
	"caipiao/backend/internal/lottery"
	"caipiao/backend/internal/playrules"
	"caipiao/backend/internal/schemebettingdispatch"
	"caipiao/backend/internal/schemeeventbus"
)

// A regression in any phase of the formal chain must make this test fail:
// source settlement must advance strategy exactly once, resolution must persist
// exactly one N+1 command, and duplicate deliveries must never submit twice.
func TestFormalShortPeriodContiguousLifecycleUsesProductionTransactions(t *testing.T) {
	f := newProductionContiguousChainE2EFixture(t)

	f.deliverBoundaryBeforePhaseOne()
	f.deliverStrategyReady(3)
	f.deliverBoundaryReady(3)
	f.dispatchBetReady(3)

	f.assertOneStrategyAdvance()
	f.assertOneDecisionForSource()
	f.assertOneOutboxWithStableRequestID()
	f.assertFrozenStrategyState(12, 2, "8")
	f.assertOneProviderSubmission()
}

// Removing the durable recovery of a boundary that arrived before phase one
// must fail this test. The first production draw ingestion expands no rows;
// phase one then commits a real waiting decision while its immediate resolver
// is transiently blocked. A redelivered authoritative boundary must publish
// two target-ready deliveries while that decision is still active, and exactly
// one resolver may create the frozen command/outbox.
func TestFormalShortPeriodBoundaryBeforePhaseOneRecoversActiveTargetReadyRedelivery(t *testing.T) {
	f := newProductionContiguousChainE2EFixture(t)
	f.deliverBoundaryBeforePhaseOne()
	f.processPhaseOneWithProviderSnapshotBlocked()
	f.assertAwaitingWithoutOutbox()
	f.redeliverBoundaryToTwoActiveTargetReadyHandlers()

	f.assertOneStrategyAdvance()
	f.assertOneDecisionForSource()
	f.assertOneOutboxWithStableRequestID()
}

// productionContiguousChainE2EFixture keeps PostgreSQL and every production
// state transition real. Only the external Guaji transport is replaced.
// newResolverEntryFixture owns schema gating and cleanup.
type productionContiguousChainE2EFixture struct {
	*resolverEntryFixture
	q             *sqlcdb.Queries
	worker        *Worker
	drawWorker    *drawsync.Worker
	boundaryPipe  *task9BoundaryPipeline
	dispatcher    *schemebettingdispatch.Runtime
	provider      *task9Provider
	recordID      int64
	originalChain string
	dispatchOwner string
}

func newProductionContiguousChainE2EFixture(t *testing.T) *productionContiguousChainE2EFixture {
	t.Helper()
	base := newResolverEntryFixture(t)
	f := &productionContiguousChainE2EFixture{resolverEntryFixture: base, q: sqlcdb.New(base.pool)}
	t.Cleanup(f.cleanupProductionRows)
	if err := base.pool.QueryRow(base.ctx, `SELECT chain_id FROM scheme_instances WHERE id=$1`, base.schemeID).Scan(&f.originalChain); err != nil {
		t.Fatal(err)
	}
	definitionConfig := []byte(`{
  "runTypeId":"fixed_rotate","playTemplate":"fast_ssc_std","playTypeId":"g006","subPlayId":"13",
  "typeId":"g006","subId":"13","betMode":"dingwei","schemeGroups":["1","8"],
  "rounds":[{"mult":2,"afterHit":1,"afterMiss":1},{"mult":3,"afterHit":1,"afterMiss":1}]
}`)
	if _, err := base.pool.Exec(base.ctx, `UPDATE scheme_definitions SET config=$2::jsonb WHERE id=$1`, base.definition, definitionConfig); err != nil {
		t.Fatal(err)
	}
	if _, err := base.pool.Exec(base.ctx, `
UPDATE scheme_instances
SET state_version=7, round_index=0, pick_index=0, current_pick='', last_direction='',
    multiplier=4, chain_seq=4, strict_chain_state='active', chain_block_reason=NULL,
    status='running', status_reason='', bet_failed_detail=''
WHERE id=$1`, base.schemeID); err != nil {
		t.Fatal(err)
	}
	spec, err := json.Marshal(playrules.EvaluationSpec{
		Mode: "dingwei", NumberMin: 0, NumberMax: 9, SegmentLen: 1,
		BetMode: "dingwei", CatalogSubID: "13",
	})
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := json.Marshal(playrules.Snapshot{
		Locator:     playrules.Locator{TemplateCode: "fast_ssc_std", TypeID: "g006", SubID: "13"},
		RuleVersion: 1, EvaluatorVersion: 1, EvaluatorKey: "ssc.direct", EvaluationSpec: spec,
	})
	if err != nil {
		t.Fatal(err)
	}
	// Existing source settlement is durable before the websocket boundary. The
	// later drawsync.Ingest call deliberately sees this REST-like duplicate and
	// must still publish the authoritative PeriodBoundary.
	drawnAt := base.databaseNow().Add(30 * time.Second)
	if _, err := base.pool.Exec(base.ctx, `
INSERT INTO lottery_draws (lottery_code, issue_no, period_short, balls, sum_value, drawn_at)
VALUES ($1,$2,'task9','["9","2","3","4","5"]'::jsonb,23,$3)`, base.lotteryCode, base.source, drawnAt); err != nil {
		t.Fatal(err)
	}
	stamp := time.Now().UnixNano()
	if err := base.pool.QueryRow(base.ctx, `
INSERT INTO cloud_bet_records
    (record_no, member_id, sim_bet, scheme_id, scheme_name, lottery_code, period_no, play_type,
     multiplier, round_label, amount, status, bet_content, third_party_bet_id,
     rule_snapshot, rule_version, rule_snapshot_hash)
VALUES ($1,$2,false,$3,'task9 contiguous chain',$4,$5,'task9','8','1',1,'miss','1',$6,$7::jsonb,1,'task9-rule')
RETURNING id`, fmt.Sprintf("T9C%d", stamp), base.memberID, base.schemeID, base.lotteryCode, base.source,
		fmt.Sprintf("task9-accepted-%d", stamp), snapshot).Scan(&f.recordID); err != nil {
		t.Fatal(err)
	}
	base.processor.SetBettingMode("gray", []string{base.lotteryCode})
	f.worker = &Worker{pool: base.pool, q: f.q, strategyProcessor: base.processor, bettingBacklog: availableRearmBacklog{}}
	f.boundaryPipe = &task9BoundaryPipeline{fixture: f, targetReadyStarted: make(chan struct{}, 8)}
	f.drawWorker = drawsync.NewWorker(base.pool, guaji.NewClient(guaji.Config{Enabled: true}), nil)
	if f.drawWorker == nil {
		t.Fatal("drawsync worker unavailable")
	}
	f.drawWorker.SetPeriodBoundaryPublisher(f.boundaryPipe)
	f.provider = &task9Provider{}
	f.dispatchOwner = fmt.Sprintf("t9-%010d", stamp%10_000_000_000)
	f.dispatcher, err = schemebettingdispatch.New(f.q, f.provider, schemebettingdispatch.Config{
		Mode: "gray", Owner: f.dispatchOwner,
		LotteryCodes: []string{base.lotteryCode}, Shards: []int32{base.shardNo},
		Batch: 32, Concurrency: 1, LeaseDuration: 2 * time.Second, PollInterval: time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	f.dispatcher.SetPeriodVerifier(task9PeriodVerifier{fixture: f})
	return f
}

func (f *productionContiguousChainE2EFixture) cleanupProductionRows() {
	f.cleanupExec("dispatcher shard lease", `DELETE FROM scheme_betting_shard_leases WHERE lease_kind='dispatcher' AND shard_no=$1 AND lease_owner=$2`, f.shardNo, f.dispatchOwner)
	f.cleanupExec("strategy evaluations", `DELETE FROM scheme_strategy_evaluations WHERE instance_id=$1`, f.schemeID)
	f.cleanupExec("cloud bet records", `DELETE FROM cloud_bet_records WHERE scheme_id=$1`, f.schemeID)
	f.cleanupExec("source lottery draw", `DELETE FROM lottery_draws WHERE lottery_code=$1 AND issue_no=$2`, f.lotteryCode, f.source)
	f.cleanupExec("expanded provider snapshots", `DELETE FROM provider_period_snapshots WHERE lottery_code=$1 AND period_no IN ($2,$3,$4)`,
		f.lotteryCode, f.target, incrementDecimal(f.target), incrementDecimal(incrementDecimal(f.target)))
}

func (f *productionContiguousChainE2EFixture) deliverBoundaryBeforePhaseOne() {
	f.t.Helper()
	var decisions, outboxes int
	if err := f.pool.QueryRow(f.ctx, `SELECT COUNT(*)::int FROM scheme_period_decisions WHERE scheme_id=$1`, f.schemeID).Scan(&decisions); err != nil {
		f.t.Fatal(err)
	}
	if err := f.pool.QueryRow(f.ctx, `SELECT COUNT(*)::int FROM scheme_bet_outbox WHERE scheme_id=$1`, f.schemeID).Scan(&outboxes); err != nil {
		f.t.Fatal(err)
	}
	if decisions != 0 || outboxes != 0 {
		f.t.Fatalf("pre-phase boundary saw decisions=%d outboxes=%d", decisions, outboxes)
	}
	event := guaji.DrawEvent{
		GameKey: f.lotteryCode, Periods: f.source, NextPeriods: f.target,
		DrawnAt: f.databaseNow(), Balls: guaji.DrawBalls{SSC: "92345"},
	}
	if err := f.drawWorker.Ingest(f.ctx, event); err != nil {
		f.t.Fatal(err)
	}
	if got := f.boundaryPipe.boundaryCount(); got != 1 {
		f.t.Fatalf("pre-phase production boundary publications=%d want 1", got)
	}
}

func (f *productionContiguousChainE2EFixture) processPhaseOneWithProviderSnapshotBlocked() {
	f.t.Helper()
	lockTx, err := f.pool.Begin(f.ctx)
	if err != nil {
		f.t.Fatal(err)
	}
	locked := false
	defer func() {
		if !locked {
			return
		}
		if rollbackErr := lockTx.Rollback(context.Background()); rollbackErr != nil {
			f.t.Errorf("release provider snapshot test lock: %v", rollbackErr)
		}
	}()
	if _, err := lockTx.Exec(f.ctx, `LOCK TABLE provider_period_snapshots IN ACCESS EXCLUSIVE MODE`); err != nil {
		f.t.Fatal(err)
	}
	locked = true
	resolveCtx, cancel := context.WithTimeout(f.ctx, 200*time.Millisecond)
	defer cancel()
	err = f.processor.ProcessStrategyReady(resolveCtx, f.recordID, f.schemeID, f.lotteryCode, f.source, 7)
	if !errors.Is(err, context.DeadlineExceeded) {
		f.t.Fatalf("phase-one resolver error=%v want provider snapshot timeout after phase-one commit", err)
	}
	if err := lockTx.Rollback(context.Background()); err != nil {
		f.t.Fatal(err)
	}
	locked = false
}

func (f *productionContiguousChainE2EFixture) assertAwaitingWithoutOutbox() {
	f.t.Helper()
	var status string
	var outboxes int
	if err := f.pool.QueryRow(f.ctx, `
SELECT d.status, (SELECT COUNT(*)::int FROM scheme_bet_outbox o WHERE o.decision_id=d.id)
FROM scheme_period_decisions d
WHERE d.scheme_id=$1 AND d.source_period_no=$2`, f.schemeID, f.source).Scan(&status, &outboxes); err != nil {
		f.t.Fatal(err)
	}
	if status != "awaiting_target" || outboxes != 0 {
		f.t.Fatalf("post-phase-one status/outboxes=%q/%d want awaiting_target/0", status, outboxes)
	}
}

func (f *productionContiguousChainE2EFixture) redeliverBoundaryToTwoActiveTargetReadyHandlers() {
	f.t.Helper()
	decisionID := f.decisionID()
	lockTx, err := f.pool.Begin(f.ctx)
	if err != nil {
		f.t.Fatal(err)
	}
	locked := false
	defer func() {
		if !locked {
			return
		}
		if rollbackErr := lockTx.Rollback(context.Background()); rollbackErr != nil {
			f.t.Errorf("release active target-ready test lock: %v", rollbackErr)
		}
	}()
	if _, err := lockTx.Exec(f.ctx, `SELECT id FROM scheme_period_decisions WHERE id=$1 FOR UPDATE`, decisionID); err != nil {
		f.t.Fatal(err)
	}
	locked = true
	errs := make(chan error, 2)
	for range 2 {
		go func() { errs <- f.boundaryPipe.redeliver(f.ctx) }()
	}
	f.boundaryPipe.waitForTargetReadyStarts(2)
	if err := lockTx.Commit(f.ctx); err != nil {
		f.t.Fatal(err)
	}
	locked = false
	for range 2 {
		if err := <-errs; err != nil {
			f.t.Fatal(err)
		}
	}
}

func (f *productionContiguousChainE2EFixture) deliverStrategyReady(deliveries int) {
	f.t.Helper()
	for range deliveries {
		if err := f.processor.ProcessStrategyReady(f.ctx, f.recordID, f.schemeID, f.lotteryCode, f.source, 7); err != nil {
			f.t.Fatal(err)
		}
	}
}

func (f *productionContiguousChainE2EFixture) decisionID() int64 {
	f.t.Helper()
	var id int64
	if err := f.pool.QueryRow(f.ctx, `SELECT id FROM scheme_period_decisions WHERE scheme_id=$1 AND source_period_no=$2`, f.schemeID, f.source).Scan(&id); err != nil {
		f.t.Fatal(err)
	}
	return id
}

func (f *productionContiguousChainE2EFixture) deliverBoundaryReady(deliveries int) {
	f.t.Helper()
	event := schemeeventbus.ContiguousTargetReady{DecisionID: f.decisionID(), SchemeID: f.schemeID, LotteryCode: f.lotteryCode, SourcePeriod: f.source, BoundaryGeneration: 1}
	for range deliveries {
		if err := f.worker.ProcessContiguousTargetReady(f.ctx, event); err != nil {
			f.t.Fatal(err)
		}
	}
}

func (f *productionContiguousChainE2EFixture) dispatchBetReady(deliveries int) {
	f.t.Helper()
	var event schemeeventbus.BetReady
	err := f.pool.QueryRow(f.ctx, `
SELECT id, request_id, shard_no, safe_deadline_at
FROM scheme_bet_outbox WHERE scheme_id=$1 ORDER BY id DESC LIMIT 1`, f.schemeID).
		Scan(&event.OutboxID, &event.RequestID, &event.ShardNo, &event.SafeDeadline)
	if errors.Is(err, pgx.ErrNoRows) {
		return
	}
	if err != nil {
		f.t.Fatal(err)
	}
	for range deliveries {
		if err := f.dispatcher.HandleBetReady(f.ctx, event); err != nil {
			f.t.Fatal(err)
		}
	}
}

func (f *productionContiguousChainE2EFixture) publishExactBoundary() {
	f.t.Helper()
	if state, ok := lottery.PeriodStateFor(f.lotteryCode); ok && state.CurrentIssue == f.source && state.NextIssue == f.target {
		return
	}
	if !lottery.UpdatePeriodState(f.lotteryCode, f.source, f.target, time.Now().UTC().Add(30*time.Second), 6) {
		f.t.Fatalf("failed to publish exact boundary %s -> %s", f.source, f.target)
	}
}

func (f *productionContiguousChainE2EFixture) publishBoundaryPastSource() {
	f.t.Helper()
	if !lottery.UpdatePeriodState(f.lotteryCode, f.target, incrementDecimal(f.target), time.Now().UTC().Add(30*time.Second), 6) {
		f.t.Fatal("failed to publish boundary past source")
	}
}

func (f *productionContiguousChainE2EFixture) publishLaterBoundary() {
	f.t.Helper()
	current := incrementDecimal(f.target)
	if !lottery.UpdatePeriodState(f.lotteryCode, current, incrementDecimal(current), time.Now().UTC().Add(30*time.Second), 6) {
		f.t.Fatal("failed to publish later boundary")
	}
}

func (f *productionContiguousChainE2EFixture) restartAndRecoverWaiting(runs int) {
	f.t.Helper()
	f.publishExactBoundary()
	owner := fmt.Sprintf("t9r-%09d", time.Now().UnixNano()%1_000_000_000)
	epoch, acquired, err := f.q.AcquireSchemeBettingShardLease(f.ctx, "strategy", f.shardNo, owner, time.Minute)
	if err != nil || !acquired {
		f.t.Fatalf("acquire strategy lease acquired=%v err=%v", acquired, err)
	}
	defer f.cleanupExec("recovery strategy shard lease", `DELETE FROM scheme_betting_shard_leases WHERE lease_kind='strategy' AND shard_no=$1 AND lease_owner=$2`, f.shardNo, owner)
	ctx := WithStrategyLeaseFence(f.ctx, StrategyLeaseFence{ShardNo: f.shardNo, Owner: owner, Epoch: epoch})
	for range runs {
		if _, err := runContiguousTargetRecoveryBatch(ctx, f.q, f.processor, []string{f.lotteryCode}, []int32{f.shardNo}, 32); err != nil {
			f.t.Fatal(err)
		}
	}
}

func (f *productionContiguousChainE2EFixture) expireWaitingDecision() {
	f.t.Helper()
	if _, err := f.pool.Exec(f.ctx, `UPDATE scheme_period_decisions SET target_deadline_at=clock_timestamp()-interval '1 millisecond' WHERE id=$1`, f.decisionID()); err != nil {
		f.t.Fatal(err)
	}
}

// raceResolverCompletionAndExpiry starts the real resolver while completion is
// still legal, then starts the production expiry transition at the database
// deadline. Either conditional terminal update may win; a pre-expired fixture
// is deliberately not used because it would only race two expiry paths.
func (f *productionContiguousChainE2EFixture) raceResolverCompletionAndExpiry() {
	f.t.Helper()
	id := f.decisionID()
	deadline := f.databaseNow().Add(300 * time.Millisecond)
	if _, err := f.pool.Exec(f.ctx, `UPDATE scheme_period_decisions SET target_deadline_at=$2 WHERE id=$1`, id, deadline); err != nil {
		f.t.Fatal(err)
	}
	start := make(chan struct{})
	errs := make(chan error, 2)
	go func() {
		<-start
		// Keep CompleteAwaitingContiguousTarget viable but close enough to the
		// deadline that the independently scheduled expiry transition competes
		// for the same locked decision/instance rows.
		wait := time.Until(deadline.Add(-20 * time.Millisecond))
		if wait > 0 {
			time.Sleep(wait)
		}
		errs <- f.processor.ResolveAwaitingTarget(f.ctx, id)
	}()
	go func() {
		<-start
		wait := time.Until(deadline)
		if wait > 0 {
			time.Sleep(wait)
		}
		_, err := f.q.MissAwaitingContiguousTarget(f.ctx, sqlcdb.MissAwaitingContiguousTargetParams{DecisionID: id, FailureReason: "target_deadline_elapsed", Diagnostics: []byte(`{"source":"task9-race"}`)})
		errs <- err
	}()
	close(start)
	for range 2 {
		if err := <-errs; err != nil {
			f.t.Fatal(err)
		}
	}
}

func (f *productionContiguousChainE2EFixture) makeMissedChain() {
	f.deliverStrategyReady(1)
	f.publishBoundaryPastSource()
	f.deliverBoundaryReady(1)
}

func (f *productionContiguousChainE2EFixture) manualRearm() {
	f.t.Helper()
	if err := f.worker.RearmEventScheme(f.ctx, f.schemeID, "task9", "explicit task9 rearm"); err != nil {
		f.t.Fatal(err)
	}
}

func (f *productionContiguousChainE2EFixture) completePhaseOneAndResolve() {
	f.deliverBoundaryBeforePhaseOne()
	f.deliverStrategyReady(2)
	f.deliverBoundaryReady(2)
}

func (f *productionContiguousChainE2EFixture) dispatchWrongProviderPeriod(deliveries int) {
	f.provider.configure(incrementDecimal(f.target), nil)
	f.dispatchBetReady(deliveries)
}

func (f *productionContiguousChainE2EFixture) dispatchUnknownProviderResult(deliveries int) {
	f.provider.configure("", errors.New("provider response lost after request write"))
	f.dispatchBetReady(deliveries)
}

func (f *productionContiguousChainE2EFixture) reconcileUnknownFingerprint(runs int) {
	f.t.Helper()
	finalizer := schemebettingdispatch.NewAcceptanceFinalizer(f.pool, f.provider)
	for range runs {
		if _, err := f.pool.Exec(f.ctx, `UPDATE scheme_bet_outbox SET provider_reconcile_next_at=clock_timestamp() WHERE scheme_id=$1`, f.schemeID); err != nil {
			f.t.Fatal(err)
		}
		if err := finalizer.RecoverUnknown(f.ctx, 32); err != nil {
			f.t.Fatal(err)
		}
	}
	if f.provider.lookupCount() != runs {
		f.t.Fatalf("fingerprint lookups=%d want %d", f.provider.lookupCount(), runs)
	}
}

func (f *productionContiguousChainE2EFixture) assertOneStrategyAdvance() {
	f.t.Helper()
	var version int64
	var round, pick, evaluations int
	var marker bool
	if err := f.pool.QueryRow(f.ctx, `
SELECT i.state_version, i.round_index, i.pick_index,
       (SELECT COUNT(*)::int FROM scheme_strategy_evaluations e WHERE e.instance_id=i.id AND e.period_no=$2),
       (SELECT strategy_evaluated_at IS NOT NULL FROM cloud_bet_records c WHERE c.id=$3)
FROM scheme_instances i WHERE i.id=$1`, f.schemeID, f.source, f.recordID).Scan(&version, &round, &pick, &evaluations, &marker); err != nil {
		f.t.Fatal(err)
	}
	if version != 8 || round != 1 || pick != 1 || evaluations != 1 || !marker {
		f.t.Fatalf("state version/round/pick/evaluations/marker=%d/%d/%d/%d/%v", version, round, pick, evaluations, marker)
	}
}

func (f *productionContiguousChainE2EFixture) assertOneDecisionForSource() {
	f.t.Helper()
	var count int
	var before, after int64
	var localHit bool
	if err := f.pool.QueryRow(f.ctx, `SELECT COUNT(*)::int, MIN(state_version_before), MAX(state_version_after), BOOL_OR(local_hit) FROM scheme_period_decisions WHERE scheme_id=$1 AND source_period_no=$2`, f.schemeID, f.source).Scan(&count, &before, &after, &localHit); err != nil {
		f.t.Fatal(err)
	}
	if count != 1 || before != 7 || after != 8 || localHit {
		f.t.Fatalf("decision count/version/localHit=%d/%d/%d/%v want 1/7/8/false", count, before, after, localHit)
	}
}

func (f *productionContiguousChainE2EFixture) assertOneOutboxWithStableRequestID() {
	f.t.Helper()
	var count int
	var minID, maxID, frozenRaw string
	if err := f.pool.QueryRow(f.ctx, `
SELECT COUNT(*)::int, COALESCE(MIN(request_id),''), COALESCE(MAX(request_id),''),
       COALESCE(MAX(frozen_request::text),'{}')
FROM scheme_bet_outbox WHERE scheme_id=$1 AND source_period_no=$2`, f.schemeID, f.source).Scan(&count, &minID, &maxID, &frozenRaw); err != nil {
		f.t.Fatal(err)
	}
	var frozen schemebettingdispatch.FrozenGuajiRequest
	if err := json.Unmarshal([]byte(frozenRaw), &frozen); err != nil {
		f.t.Fatal(err)
	}
	if count != 1 || minID == "" || minID != maxID || frozen.RequestID != minID {
		f.t.Fatalf("outbox count/request/frozen=%d/%q/%q/%q", count, minID, maxID, frozen.RequestID)
	}
}

func (f *productionContiguousChainE2EFixture) assertFrozenStrategyState(multiplier, round int, content string) {
	f.t.Helper()
	var raw []byte
	if err := f.pool.QueryRow(f.ctx, `SELECT frozen_request FROM scheme_bet_outbox WHERE scheme_id=$1 AND source_period_no=$2`, f.schemeID, f.source).Scan(&raw); err != nil {
		f.t.Fatal(err)
	}
	var frozen schemebettingdispatch.FrozenGuajiRequest
	if err := json.Unmarshal(raw, &frozen); err != nil {
		f.t.Fatal(err)
	}
	if frozen.Request.Multiplier != multiplier || frozen.RoundLabel != fmt.Sprint(round) || frozen.BetContent != content {
		f.t.Fatalf("frozen multiplier/round/content=%d/%q/%q", frozen.Request.Multiplier, frozen.RoundLabel, frozen.BetContent)
	}
}

func (f *productionContiguousChainE2EFixture) assertOneProviderSubmission() {
	f.t.Helper()
	if got := f.provider.submissionCount(); got != 1 {
		f.t.Fatalf("provider submissions=%d want 1", got)
	}
}

func (f *productionContiguousChainE2EFixture) assertNoProviderSubmission() {
	f.t.Helper()
	if got := f.provider.submissionCount(); got != 0 {
		f.t.Fatalf("provider submissions=%d want 0", got)
	}
}

func (f *productionContiguousChainE2EFixture) assertMissedWithoutOutbox() {
	f.t.Helper()
	var status, chainState string
	var outboxes int
	if err := f.pool.QueryRow(f.ctx, `
SELECT d.status, i.strict_chain_state,
       (SELECT COUNT(*)::int FROM scheme_bet_outbox o WHERE o.decision_id=d.id)
FROM scheme_period_decisions d JOIN scheme_instances i ON i.id=d.scheme_id
WHERE d.scheme_id=$1 AND d.source_period_no=$2`, f.schemeID, f.source).Scan(&status, &chainState, &outboxes); err != nil {
		f.t.Fatal(err)
	}
	if status != "missed_contiguous_period" || chainState != "blocked_requires_rearm" || outboxes != 0 {
		f.t.Fatalf("missed status/chain/outboxes=%q/%q/%d", status, chainState, outboxes)
	}
}

func (f *productionContiguousChainE2EFixture) assertOneTerminalWinnerAndAtMostOneOutbox() {
	f.t.Helper()
	var status string
	var decisions, outboxes int
	if err := f.pool.QueryRow(f.ctx, `
SELECT COUNT(*)::int, MAX(status),
       (SELECT COUNT(*)::int FROM scheme_bet_outbox o WHERE o.scheme_id=$1 AND o.source_period_no=$2)
FROM scheme_period_decisions WHERE scheme_id=$1 AND source_period_no=$2`, f.schemeID, f.source).Scan(&decisions, &status, &outboxes); err != nil {
		f.t.Fatal(err)
	}
	if decisions != 1 || (status != "completed" && status != "missed_contiguous_period") || outboxes > 1 || (status == "completed" && outboxes != 1) || (status == "missed_contiguous_period" && outboxes != 0) {
		f.t.Fatalf("terminal decisions/status/outboxes=%d/%q/%d", decisions, status, outboxes)
	}
}

func (f *productionContiguousChainE2EFixture) assertNewChainStartsAtRoundOne() {
	f.t.Helper()
	var chainID, chainState string
	var round, pick, initialOutboxes int
	if err := f.pool.QueryRow(f.ctx, `
SELECT chain_id, strict_chain_state, round_index, pick_index,
       (SELECT COUNT(*)::int FROM scheme_bet_outbox o WHERE o.scheme_id=i.id AND o.initial_bet)
FROM scheme_instances i WHERE id=$1`, f.schemeID).Scan(&chainID, &chainState, &round, &pick, &initialOutboxes); err != nil {
		f.t.Fatal(err)
	}
	if chainID == f.originalChain || chainState != "active" || round != 0 || pick != 0 || initialOutboxes != 1 {
		f.t.Fatalf("rearm chain/state/round/pick/outboxes=%q/%q/%d/%d/%d", chainID, chainState, round, pick, initialOutboxes)
	}
}

func (f *productionContiguousChainE2EFixture) assertProviderFaultBlocked(reason string) {
	f.t.Helper()
	var state, outcome, chainState string
	if err := f.pool.QueryRow(f.ctx, `
SELECT o.state, COALESCE(o.outcome_reason,''), i.strict_chain_state
FROM scheme_bet_outbox o JOIN scheme_instances i ON i.id=o.scheme_id
WHERE o.scheme_id=$1 ORDER BY o.id DESC LIMIT 1`, f.schemeID).Scan(&state, &outcome, &chainState); err != nil {
		f.t.Fatal(err)
	}
	wantState := "sent_unknown"
	if reason == "accepted_wrong_period" {
		wantState = "accepted_wrong_period"
	}
	if state != wantState || outcome != reason || chainState != "blocked_requires_rearm" {
		f.t.Fatalf("provider fault state/outcome/chain=%q/%q/%q", state, outcome, chainState)
	}
}

// task9BoundaryPipeline preserves the real production event sequence inside
// the isolated database composition test. It replaces only JetStream's
// external transport: drawsync publishes PeriodBoundary, the production
// expander rereads durable waits, and target-ready invokes the real worker.
type task9BoundaryPipeline struct {
	fixture *productionContiguousChainE2EFixture

	mu                 sync.Mutex
	boundaries         []schemeeventbus.PeriodBoundary
	targetReadyStarted chan struct{}
}

func (pipeline *task9BoundaryPipeline) PublishPeriodBoundary(ctx context.Context, event schemeeventbus.PeriodBoundary) error {
	pipeline.mu.Lock()
	pipeline.boundaries = append(pipeline.boundaries, event)
	pipeline.mu.Unlock()
	return schemeeventbus.ExpandContiguousTargetBoundary(
		ctx, event, pipeline.fixture.q, pipeline, []int32{pipeline.fixture.shardNo}, shadowOutboxShardCount,
	)
}

func (pipeline *task9BoundaryPipeline) PublishContiguousTargetReady(ctx context.Context, event schemeeventbus.ContiguousTargetReady, _ uint32) error {
	select {
	case pipeline.targetReadyStarted <- struct{}{}:
	default:
	}
	return pipeline.fixture.worker.ProcessContiguousTargetReady(ctx, event)
}

func (pipeline *task9BoundaryPipeline) boundaryCount() int {
	pipeline.mu.Lock()
	defer pipeline.mu.Unlock()
	return len(pipeline.boundaries)
}

func (pipeline *task9BoundaryPipeline) redeliver(ctx context.Context) error {
	pipeline.mu.Lock()
	if len(pipeline.boundaries) == 0 {
		pipeline.mu.Unlock()
		return errors.New("no production period boundary to redeliver")
	}
	event := pipeline.boundaries[len(pipeline.boundaries)-1]
	pipeline.mu.Unlock()
	return schemeeventbus.ExpandContiguousTargetBoundary(
		ctx, event, pipeline.fixture.q, pipeline, []int32{pipeline.fixture.shardNo}, shadowOutboxShardCount,
	)
}

func (pipeline *task9BoundaryPipeline) waitForTargetReadyStarts(want int) {
	pipeline.fixture.t.Helper()
	for range want {
		select {
		case <-pipeline.targetReadyStarted:
		case <-time.After(time.Second):
			pipeline.fixture.t.Fatalf("timed out waiting for target-ready handler %d/%d", want, want)
		}
	}
}

type task9PeriodVerifier struct {
	fixture *productionContiguousChainE2EFixture
}

func (v task9PeriodVerifier) VerifyOpenPeriodForMember(context.Context, string, string) (string, time.Time, error) {
	return v.fixture.target, time.Now().UTC().Add(time.Minute), nil
}

type task9Provider struct {
	mu           sync.Mutex
	submissions  int
	lookups      int
	resultPeriod string
	placeErr     error
}

func (p *task9Provider) configure(period string, err error) {
	p.mu.Lock()
	p.resultPeriod, p.placeErr = period, err
	p.mu.Unlock()
}

func (p *task9Provider) Enabled() bool { return true }

func (p *task9Provider) PlaceRealBetOnce(ctx context.Context, _ string, req guajibet.Request) (guajibet.Result, error) {
	p.mu.Lock()
	p.submissions++
	period, err := p.resultPeriod, p.placeErr
	call := p.submissions
	p.mu.Unlock()
	guaji.ReportRequestProgress(ctx, guaji.RequestProgress{Operation: "POST /api/web_bets/lott", Phase: "response", RequestWritten: true})
	if err != nil {
		return guajibet.Result{}, err
	}
	if period == "" {
		period = req.IssueNo
	}
	return guajibet.Result{ThirdPartyBetID: fmt.Sprintf("task9-provider-%d", call), Periods: period, Amount: req.Amount, GuajiAccountID: 99, Currency: req.Currency}, nil
}

func (p *task9Provider) PlaceRealBet(ctx context.Context, account string, req guajibet.Request) (guajibet.Result, error) {
	return p.PlaceRealBetOnce(ctx, account, req)
}

func (*task9Provider) MirrorBetDebitLedger(context.Context, *sqlcdb.Queries, int64, string, float64, int64, string) error {
	return nil
}

func (p *task9Provider) ResolveAcceptedBet(context.Context, string, guajibet.Request) (guajibet.Result, error) {
	p.mu.Lock()
	p.lookups++
	p.mu.Unlock()
	return guajibet.Result{}, errors.New("exact provider fingerprint not found yet")
}

func (p *task9Provider) submissionCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.submissions
}

func (p *task9Provider) lookupCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.lookups
}
