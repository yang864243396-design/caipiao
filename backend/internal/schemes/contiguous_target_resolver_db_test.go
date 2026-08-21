package schemes

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/joho/godotenv"

	"caipiao/backend/internal/config"
	"caipiao/backend/internal/db"
	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/lottery"
	"caipiao/backend/internal/playrules"
	"caipiao/backend/internal/schemebetting"
	"caipiao/backend/internal/schemebettingdispatch"
)

var resolverEntryIssueCounter atomic.Int64

func TestResolveAwaitingTargetDBExactSuccessCreatesOneOutboxAndAdvancesChain(t *testing.T) {
	f := newResolverEntryFixture(t)
	f.publishExactBoundary()
	decisionID := f.seed(f.databaseNow().Add(10 * time.Second))

	if err := f.processor.ResolveAwaitingTarget(f.ctx, decisionID); err != nil {
		t.Fatal(err)
	}
	f.assertCompletedOnce(decisionID)
}

func TestResolveAwaitingTargetDBEightConcurrentCallsHaveOneWinner(t *testing.T) {
	f := newResolverEntryFixture(t)
	f.publishExactBoundary()
	decisionID := f.seed(f.databaseNow().Add(10 * time.Second))

	start := make(chan struct{})
	errs := make(chan error, 8)
	var ready sync.WaitGroup
	ready.Add(8)
	for range 8 {
		go func() {
			ready.Done()
			<-start
			errs <- f.processor.ResolveAwaitingTarget(f.ctx, decisionID)
		}()
	}
	ready.Wait()
	close(start)
	for range 8 {
		if err := <-errs; err != nil {
			t.Fatal(err)
		}
	}
	f.assertCompletedOnce(decisionID)
}

func TestResolveAwaitingTargetDBDeadlineWinnerIsTerminalWithoutOutbox(t *testing.T) {
	f := newResolverEntryFixture(t)
	f.publishExactBoundary()
	decisionID := f.seed(f.databaseNow().Add(-time.Millisecond))

	start := make(chan struct{})
	errs := make(chan error, 2)
	go func() {
		<-start
		errs <- f.processor.ResolveAwaitingTarget(f.ctx, decisionID)
	}()
	go func() {
		<-start
		_, err := sqlcdb.New(f.pool).MissAwaitingContiguousTarget(f.ctx, sqlcdb.MissAwaitingContiguousTargetParams{
			DecisionID: decisionID, FailureReason: "target_deadline_elapsed", Diagnostics: []byte(`{"source":"deadline-race"}`),
		})
		errs <- err
	}()
	close(start)
	for range 2 {
		if err := <-errs; err != nil {
			t.Fatal(err)
		}
	}
	f.assertMissedWithoutOutbox(decisionID)
}

func TestResolveAwaitingTargetDBFreezesAlreadyAdvancedInstanceState(t *testing.T) {
	f := newResolverEntryFixture(t)
	f.publishExactBoundary()
	decisionID := f.seed(f.databaseNow().Add(10 * time.Second))

	if err := f.processor.ResolveAwaitingTarget(f.ctx, decisionID); err != nil {
		t.Fatal(err)
	}
	frozen := f.assertCompletedOnce(decisionID)
	if frozen.Request.Multiplier != 12 || frozen.RoundLabel != "2" || frozen.BetContent != "8" {
		t.Fatalf("frozen multiplier=%d round=%q picks=%q, want 12/2/8", frozen.Request.Multiplier, frozen.RoundLabel, frozen.BetContent)
	}
	var roundIndex, pickIndex int32
	var stateVersion, chainSeq int64
	var multiplier string
	if err := f.pool.QueryRow(f.ctx, `
SELECT round_index, pick_index, multiplier::text, state_version, chain_seq
FROM scheme_instances WHERE id = $1`, f.schemeID).Scan(&roundIndex, &pickIndex, &multiplier, &stateVersion, &chainSeq); err != nil {
		t.Fatal(err)
	}
	if roundIndex != 1 || pickIndex != 1 || multiplier != "4" || stateVersion != 8 || chainSeq != 5 {
		t.Fatalf("instance round=%d pick=%d multiplier=%q state=%d chain=%d", roundIndex, pickIndex, multiplier, stateVersion, chainSeq)
	}
}

func TestResolveAwaitingTargetDBRecoveryTwiceIsIdempotent(t *testing.T) {
	f := newResolverEntryFixture(t)
	f.publishExactBoundary()
	decisionID := f.seed(f.databaseNow().Add(10 * time.Second))
	owner := fmt.Sprintf("resolver-recovery-%d", time.Now().UnixNano())
	epoch, acquired, err := sqlcdb.New(f.pool).AcquireSchemeBettingShardLease(f.ctx, "strategy", f.shardNo, owner, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if !acquired {
		t.Skipf("strategy shard %d lease is held by another test or process", f.shardNo)
	}
	t.Cleanup(func() {
		_, _ = f.pool.Exec(context.Background(), `
DELETE FROM scheme_betting_shard_leases
WHERE lease_kind = 'strategy' AND shard_no = $1 AND lease_owner = $2 AND lease_epoch = $3`, f.shardNo, owner, epoch)
	})
	fenced := WithStrategyLeaseFence(f.ctx, StrategyLeaseFence{ShardNo: f.shardNo, Owner: owner, Epoch: epoch})
	q := sqlcdb.New(f.pool)

	first, err := runContiguousTargetRecoveryBatch(fenced, q, f.processor, []string{f.lotteryCode}, []int32{f.shardNo}, 32)
	if err != nil {
		t.Fatal(err)
	}
	second, err := runContiguousTargetRecoveryBatch(fenced, q, f.processor, []string{f.lotteryCode}, []int32{f.shardNo}, 32)
	if err != nil {
		t.Fatal(err)
	}
	if first != 1 || second != 0 {
		t.Fatalf("recovery processed first=%d second=%d, want 1 then 0", first, second)
	}
	f.assertCompletedOnce(decisionID)
}

type resolverEntryFixture struct {
	t           *testing.T
	ctx         context.Context
	pool        *db.Pool
	processor   *StrategyProcessor
	memberID    int64
	definition  string
	schemeID    string
	lotteryCode string
	source      string
	target      string
	shardNo     int32
}

func newResolverEntryFixture(t *testing.T) *resolverEntryFixture {
	t.Helper()
	_ = godotenv.Load("../../.env")
	cfg := config.Load()
	if cfg.DatabaseURL == "" {
		t.Skip("DATABASE_URL not set")
	}
	pool, err := db.Connect(context.Background(), cfg.DatabaseURL, 16, 0)
	if err != nil {
		t.Skipf("database unavailable: %v", err)
	}
	t.Cleanup(pool.Close)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `
SELECT d.target_deadline_at, d.target_period_no, d.failure_reason, d.shard_no,
       o.frozen_request, o.frozen_request_hash, i.chain_block_reason
FROM scheme_period_decisions d
JOIN scheme_instances i ON i.id = d.scheme_id
LEFT JOIN scheme_bet_outbox o ON o.decision_id = d.id
WHERE FALSE`); err != nil {
		t.Skipf("Task 1/4/6 schema not applied: %v", err)
	}
	var catalogTemplate string
	if err := pool.QueryRow(ctx, `SELECT COALESCE(play_template, '') FROM lottery_catalog WHERE code = 'tron_ffc_6s'`).Scan(&catalogTemplate); err != nil {
		t.Skipf("tron_ffc_6s catalog unavailable: %v", err)
	}
	if catalogTemplate != "fast_ssc_std" {
		t.Skipf("tron_ffc_6s catalog template=%q, want fast_ssc_std", catalogTemplate)
	}
	var catalogReady bool
	if err := pool.QueryRow(ctx, `
SELECT EXISTS (
  SELECT 1 FROM sub_plays
  WHERE template_code = 'fast_ssc_std' AND type_id = 'g006' AND sub_id = '13' AND enabled
)`).Scan(&catalogReady); err != nil || !catalogReady {
		t.Skipf("fast SSC g006/13 catalog unavailable: ready=%v err=%v", catalogReady, err)
	}

	stamp := time.Now().UnixNano()
	source, target := nextResolverEntryBoundary()
	f := &resolverEntryFixture{
		t: t, ctx: ctx, pool: pool, lotteryCode: "tron_ffc_6s", source: source, target: target,
		definition: fmt.Sprintf("resolver-entry-def-%d", stamp), schemeID: fmt.Sprintf("resolver-entry-inst-%d", stamp),
	}
	f.shardNo = int32(schemebetting.ShardForScheme(f.schemeID, shadowOutboxShardCount))
	t.Cleanup(f.cleanup)
	if err := pool.QueryRow(ctx, `
INSERT INTO members (account, password_hash, display_name, status)
VALUES ($1, 'test', 'resolver entry test', 'active') RETURNING id`, fmt.Sprintf("resolver-entry-%d", stamp)).Scan(&f.memberID); err != nil {
		t.Fatal(err)
	}
	definitionConfig := []byte(`{
  "runTypeId":"fixed_rotate",
  "playTemplate":"fast_ssc_std",
  "playTypeId":"g006",
  "subPlayId":"13",
  "typeId":"g006",
  "subId":"13",
  "betMode":"dingwei",
  "schemeGroups":["1","8"],
  "rounds":[
    {"mult":2,"afterHit":0,"afterMiss":0},
    {"mult":3,"afterHit":1,"afterMiss":1}
  ]
}`)
	if _, err := pool.Exec(ctx, `
INSERT INTO scheme_definitions
    (id, member_id, kind, scheme_name, lottery_code, lottery_label, share_status, config)
VALUES ($1, $2, 'custom', 'resolver entry test', $3, 'test', 'private', $4::jsonb)`, f.definition, f.memberID, f.lotteryCode, definitionConfig); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
INSERT INTO scheme_instances
    (id, definition_id, member_id, kind, scheme_name, lottery_code, lottery_label, status,
     sim_bet, betting_owner, strict_chain_state, chain_id, chain_seq, state_version,
     round_index, pick_index, multiplier)
VALUES ($1, $2, $3, 'custom', 'resolver entry test', $4, 'test', 'running',
        false, 'event', 'active', $5, 4, 8, 1, 1, 4)`, f.schemeID, f.definition, f.memberID, f.lotteryCode, "resolver-entry-chain-"+f.schemeID); err != nil {
		t.Fatal(err)
	}
	spec, err := json.Marshal(playrules.EvaluationSpec{
		Mode: "dingwei", NumberMin: 0, NumberMax: 9, SegmentLen: 1, BetMode: "dingwei", CatalogSubID: "13",
	})
	if err != nil {
		t.Fatal(err)
	}
	registry, err := playrules.NewRegistry([]playrules.PublishedSpec{{
		Locator:     playrules.Locator{TemplateCode: "fast_ssc_std", TypeID: "g006", SubID: "13"},
		RuleVersion: 1, EvaluatorVersion: 1, EvaluatorKey: "ssc.direct", EvaluationSpec: spec, StrategyEnabled: true,
	}})
	if err != nil {
		t.Fatal(err)
	}
	f.processor = NewStrategyProcessor(pool)
	f.processor.ruleRegistry = playrules.NewRegistryStore(registry)
	return f
}

func nextResolverEntryBoundary() (string, string) {
	n := resolverEntryIssueCounter.Add(1)
	source := fmt.Sprintf("99999999999999999999999%07d", n)
	return source, incrementDecimal(source)
}

func (f *resolverEntryFixture) publishExactBoundary() {
	f.t.Helper()
	if !lottery.UpdatePeriodState(f.lotteryCode, f.source, f.target, time.Now().UTC(), 15) {
		f.t.Fatalf("publish websocket boundary %s -> %s", f.source, f.target)
	}
}

func (f *resolverEntryFixture) databaseNow() time.Time {
	f.t.Helper()
	var now time.Time
	if err := f.pool.QueryRow(f.ctx, `SELECT clock_timestamp()`).Scan(&now); err != nil {
		f.t.Fatal(err)
	}
	return now
}

func (f *resolverEntryFixture) seed(deadline time.Time) int64 {
	f.t.Helper()
	var decisionID int64
	if err := f.pool.QueryRow(f.ctx, `
INSERT INTO scheme_period_decisions
    (scheme_id, lottery_code, source_period_no, state_version_before, state_version_after,
     rule_snapshot_hash, local_hit, status, target_deadline_at, shard_no)
VALUES ($1, $2, $3, 7, 8, 'resolver-entry-rule', false, 'awaiting_target', $4, $5)
RETURNING id`, f.schemeID, f.lotteryCode, f.source, deadline, f.shardNo).Scan(&decisionID); err != nil {
		f.t.Fatal(err)
	}
	return decisionID
}

func (f *resolverEntryFixture) assertCompletedOnce(decisionID int64) schemebettingdispatch.FrozenGuajiRequest {
	f.t.Helper()
	var status, target string
	var instanceChainSeq, outboxChainSeq int64
	var outboxCount int
	var frozenRaw string
	if err := f.pool.QueryRow(f.ctx, `
SELECT d.status, d.target_period_no, i.chain_seq,
       (SELECT COUNT(*)::int FROM scheme_bet_outbox o WHERE o.decision_id = d.id),
       COALESCE((SELECT o.chain_seq FROM scheme_bet_outbox o WHERE o.decision_id = d.id), 0),
       COALESCE((SELECT o.frozen_request::text FROM scheme_bet_outbox o WHERE o.decision_id = d.id), '{}')
FROM scheme_period_decisions d
JOIN scheme_instances i ON i.id = d.scheme_id
WHERE d.id = $1`, decisionID).Scan(&status, &target, &instanceChainSeq, &outboxCount, &outboxChainSeq, &frozenRaw); err != nil {
		f.t.Fatal(err)
	}
	if status != "completed" || target != f.target || instanceChainSeq != 5 || outboxCount != 1 || outboxChainSeq != 5 {
		f.t.Fatalf("status=%q target=%q chain=%d outboxes=%d outboxChain=%d", status, target, instanceChainSeq, outboxCount, outboxChainSeq)
	}
	var frozen schemebettingdispatch.FrozenGuajiRequest
	if err := json.Unmarshal([]byte(frozenRaw), &frozen); err != nil {
		f.t.Fatal(err)
	}
	if frozen.Request.IssueNo != f.target {
		f.t.Fatalf("frozen issue=%q want %q", frozen.Request.IssueNo, f.target)
	}
	return frozen
}

func (f *resolverEntryFixture) assertMissedWithoutOutbox(decisionID int64) {
	f.t.Helper()
	var decisionStatus, instanceStatus, reason, chainState string
	var chainSeq int64
	var outboxCount int
	if err := f.pool.QueryRow(f.ctx, `
SELECT d.status, i.status, i.status_reason, i.strict_chain_state, i.chain_seq,
       (SELECT COUNT(*)::int FROM scheme_bet_outbox o WHERE o.decision_id = d.id)
FROM scheme_period_decisions d
JOIN scheme_instances i ON i.id = d.scheme_id
WHERE d.id = $1`, decisionID).Scan(&decisionStatus, &instanceStatus, &reason, &chainState, &chainSeq, &outboxCount); err != nil {
		f.t.Fatal(err)
	}
	if decisionStatus != "missed_contiguous_period" || instanceStatus != "paused" || reason != "bet_failed" || chainState != "blocked_requires_rearm" || chainSeq != 4 || outboxCount != 0 {
		f.t.Fatalf("decision=%q instance=%q reason=%q chain=%q seq=%d outboxes=%d", decisionStatus, instanceStatus, reason, chainState, chainSeq, outboxCount)
	}
}

func (f *resolverEntryFixture) cleanup() {
	_, _ = f.pool.Exec(context.Background(), `DELETE FROM scheme_bet_outbox WHERE scheme_id = $1`, f.schemeID)
	_, _ = f.pool.Exec(context.Background(), `DELETE FROM scheme_period_decisions WHERE scheme_id = $1`, f.schemeID)
	_, _ = f.pool.Exec(context.Background(), `DELETE FROM provider_period_snapshots WHERE lottery_code = $1 AND period_no = $2`, f.lotteryCode, f.target)
	_, _ = f.pool.Exec(context.Background(), `DELETE FROM scheme_instances WHERE id = $1`, f.schemeID)
	_, _ = f.pool.Exec(context.Background(), `DELETE FROM scheme_definitions WHERE id = $1`, f.definition)
	if f.memberID > 0 {
		_, _ = f.pool.Exec(context.Background(), `DELETE FROM members WHERE id = $1`, f.memberID)
	}
}
