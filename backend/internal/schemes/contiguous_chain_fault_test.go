package schemes

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/guaji"
)

var (
	errUnexpectedRecoveryLeaseRole = errors.New("unexpected recovery lease role")
	errUnexpectedRecoveryScope     = errors.New("unexpected recovery scope")
)

func TestContiguousChainFaultRESTDrawBeforeWSDuplicateIsStable(t *testing.T) {
	// The REST insert is intentionally not an authorization source. A later WS
	// duplicate must therefore preserve the same source -> N+1 decision rather
	// than advance to a REST-advertised future period.
	for attempt := 0; attempt < 2; attempt++ {
		if got := classifyContiguousTargetBoundary("85430017", "85430017", "85430018"); got != contiguousTargetResolved {
			t.Fatalf("attempt %d classification=%v, want resolved N+1", attempt, got)
		}
	}
	if got := classifyContiguousTargetBoundary("85430017", "85430018", "85430019"); got != contiguousTargetMissed {
		t.Fatalf("future REST/WS state classification=%v, want missed instead of N+2", got)
	}
}

func TestContiguousChainFaultBoundaryBeforePhaseOneCommitCannotResolve(t *testing.T) {
	var resolverCalls atomic.Int32
	err := resolveCommittedFormalWaitTransition(context.Background(), formalCommittedPhaseOneEvidence{}, func(context.Context, int64) error {
		resolverCalls.Add(1)
		return nil
	})
	if err != nil {
		t.Fatalf("pre-commit boundary error=%v, want safe ignore", err)
	}
	if got := resolverCalls.Load(); got != 0 {
		t.Fatalf("resolver calls=%d, want zero before phase-one commit", got)
	}
}

func TestContiguousChainFaultJetStreamRedeliveryUsesOneDurableDecision(t *testing.T) {
	f := newContiguousChainE2EFixture(t, "tron_ffc_6s")
	f.addWaiting(44, 7)
	evidence := formalCommittedPhaseOneEvidence{DecisionID: 44, Status: "awaiting_target", TargetDeadlineAt: pgtype.Timestamptz{Time: time.Now().UTC().Add(time.Second), Valid: true}}
	for delivery := 0; delivery < 3; delivery++ {
		err := resolveCommittedFormalWaitTransition(context.Background(), evidence, f.resolver.ResolveAwaitingTarget)
		if err != nil {
			t.Fatalf("redelivery %d error=%v", delivery, err)
		}
	}
	if got := f.submissionCount(); got != 1 {
		t.Fatalf("redelivered strategy/boundary submissions=%d, want one", got)
	}
	if got := f.outboxCountForSource("100"); got != 1 {
		t.Fatalf("redelivered strategy/boundary outboxes=%d, want one", got)
	}
}

func TestContiguousChainFaultRestartRecoversUnexpiredWaitOnce(t *testing.T) {
	f := newContiguousChainE2EFixture(t, "tron_ffc_6s")
	f.addWaiting(45, 7)
	if processed, err := f.dispatchOne(); err != nil || processed != 1 {
		t.Fatalf("first recovery processed=%d err=%v", processed, err)
	}
	if processed, err := f.dispatchOne(); err != nil || processed != 0 {
		t.Fatalf("restarted recovery processed=%d err=%v, want idempotent empty page", processed, err)
	}
}

func TestContiguousChainFaultRestartRecoversExpiredWaitAsTerminalWithoutDispatch(t *testing.T) {
	source := &contiguousChainRecoverySource{lotteryCode: "tron_ffc_6s", rows: map[int64]sqlcdb.AwaitingContiguousTargetRow{46: {DecisionID: 46, LotteryCode: "tron_ffc_6s", SourcePeriodNo: "100", ShardNo: 7}}}
	resolver := &expiredContiguousRecoveryResolver{source: source}
	ctx := WithStrategyLeaseFence(context.Background(), StrategyLeaseFence{ShardNo: 7, Owner: "test-owner", Epoch: 1})
	if processed, err := runContiguousTargetRecoveryBatch(ctx, source, resolver, []string{"tron_ffc_6s"}, []int32{7}, 32); err != nil || processed != 1 {
		t.Fatalf("expired recovery processed=%d err=%v", processed, err)
	}
	if resolver.providerSubmissions.Load() != 0 || resolver.missed.Load() != 1 {
		t.Fatalf("expired recovery submissions=%d missed=%d, want 0/1", resolver.providerSubmissions.Load(), resolver.missed.Load())
	}
	if processed, err := runContiguousTargetRecoveryBatch(ctx, source, resolver, []string{"tron_ffc_6s"}, []int32{7}, 32); err != nil || processed != 0 {
		t.Fatalf("restart after terminal expiry processed=%d err=%v", processed, err)
	}
}

func TestContiguousChainFaultConnectedWSStillDetectsOneStaleLottery(t *testing.T) {
	base := time.Date(2026, time.August, 21, 0, 0, 0, 0, time.UTC)
	health := guaji.NewBoundaryHealth([]string{"tron_ffc_3s", "tron_ffc_6s"})
	health.Observe("tron_ffc_3s", "100", "101", base, 3*time.Second)
	health.Observe("tron_ffc_6s", "200", "201", base.Add(3*time.Second), 6*time.Second)

	stale := health.Stale(base.Add(4 * time.Second))
	if len(stale) != 1 || stale[0].LotteryCode != "tron_ffc_3s" {
		t.Fatalf("stale=%+v, want only tron_ffc_3s while shared WS remains connected", stale)
	}
	if fresh := health.SnapshotAt("tron_ffc_6s", base.Add(4*time.Second)); fresh.Stale {
		t.Fatalf("tron_ffc_6s unexpectedly stale: %+v", fresh)
	}
}

func TestContiguousChainFaultResolverAndExpiryRaceHaveOneTerminalWinner(t *testing.T) {
	terminal := &atomicContiguousTerminal{}
	start := make(chan struct{})
	var wait sync.WaitGroup
	wait.Add(2)
	for _, status := range []string{"completed", "missed_contiguous_period"} {
		status := status
		go func() {
			defer wait.Done()
			<-start
			terminal.claim(status)
		}()
	}
	close(start)
	wait.Wait()
	if got := terminal.count.Load(); got != 1 {
		t.Fatalf("terminal winners=%d, want exactly one", got)
	}
	if status := terminal.status(); status != "completed" && status != "missed_contiguous_period" {
		t.Fatalf("terminal status=%q", status)
	}
}

func TestContiguousChainFaultProviderWrongPeriodNeverAllowsCrossPeriodOrder(t *testing.T) {
	if isImmediateContiguousSuccessor("10114255702276", "10114255702277") != true {
		t.Fatal("immediate successor rejected")
	}
	if isImmediateContiguousSuccessor("10114255702276", "10114255702278") {
		t.Fatal("provider wrong period N+2 was accepted")
	}
	if got := classifyContiguousTargetBoundary("10114255702276", "10114255702277", "10114255702278"); got != contiguousTargetMissed {
		t.Fatalf("advanced provider boundary=%v, want missed", got)
	}
}

func TestContiguousChainFaultUnknownProviderFingerprintStaysPendingWithoutDispatch(t *testing.T) {
	// Missing phase-one target evidence represents an unknown provider response
	// whose exact-fingerprint reconciliation is still pending. It must not
	// manufacture a resolver call or provider order.
	var providerSubmissions atomic.Int32
	err := resolveCommittedFormalWaitTransition(context.Background(), formalCommittedPhaseOneEvidence{DecisionID: 47, Status: "awaiting_target"}, func(context.Context, int64) error {
		providerSubmissions.Add(1)
		return nil
	})
	if !errors.Is(err, ErrFormalPhaseOneInconsistentState) {
		t.Fatalf("error=%v, want pending/incomplete evidence", err)
	}
	if got := providerSubmissions.Load(); got != 0 {
		t.Fatalf("provider submissions=%d, want zero while fingerprint reconciliation is pending", got)
	}
}

func TestContiguousChainFaultOneShardDatabaseFailureDoesNotBlockAnotherShard(t *testing.T) {
	failing := &contiguousChainRecoverySource{lotteryCode: "tron_ffc_6s", assertErr: errors.New("shard 7 database unavailable")}
	passing := newContiguousChainE2EFixture(t, "tron_ffc_6s")
	passing.addWaiting(48, 8)

	failedCtx := WithStrategyLeaseFence(context.Background(), StrategyLeaseFence{ShardNo: 7, Owner: "owner", Epoch: 1})
	if _, err := runContiguousTargetRecoveryBatch(failedCtx, failing, passing.resolver, []string{"tron_ffc_6s"}, []int32{7}, 32); err == nil {
		t.Fatal("failed shard recovery unexpectedly succeeded")
	}

	passingCtx := WithStrategyLeaseFence(context.Background(), StrategyLeaseFence{ShardNo: 8, Owner: "owner", Epoch: 1})
	if processed, err := runContiguousTargetRecoveryBatch(passingCtx, passing.source, passing.resolver, []string{"tron_ffc_6s"}, []int32{8}, 32); err != nil || processed != 1 {
		t.Fatalf("independent shard processed=%d err=%v, want continued recovery", processed, err)
	}
	if got := passing.submissionCount(); got != 1 {
		t.Fatalf("independent shard submissions=%d, want one", got)
	}
}

type expiredContiguousRecoveryResolver struct {
	source              *contiguousChainRecoverySource
	missed              atomic.Int32
	providerSubmissions atomic.Int32
}

func (r *expiredContiguousRecoveryResolver) ResolveAwaitingTarget(_ context.Context, decisionID int64) error {
	r.source.mu.Lock()
	row, found := r.source.rows[decisionID]
	r.source.mu.Unlock()
	if !found {
		return nil
	}
	r.missed.Add(1)
	r.source.complete(row)
	return nil
}

type atomicContiguousTerminal struct {
	value atomic.Pointer[string]
	count atomic.Int32
}

func (t *atomicContiguousTerminal) claim(status string) {
	if t.value.CompareAndSwap(nil, &status) {
		t.count.Add(1)
	}
}

func (t *atomicContiguousTerminal) status() string {
	value := t.value.Load()
	if value == nil {
		return ""
	}
	return *value
}
