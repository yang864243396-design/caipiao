package server

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/schemeeventbus"
	"caipiao/backend/internal/schemes"
)

func TestPeriodBoundaryExpanderPublishesOneBoundedPageWithStableDedup(t *testing.T) {
	rows := make([]sqlcdb.AwaitingContiguousTargetRow, 33)
	for index := range rows {
		rows[index] = sqlcdb.AwaitingContiguousTargetRow{
			DecisionID: int64(index + 1), SchemeID: "scheme-" + string(rune('a'+index)),
			LotteryCode: "tron_ffc_6s", SourcePeriodNo: "100",
		}
	}
	source := &recordingBoundaryAwaitingSource{rows: rows}
	publisher := &recordingContiguousTargetPublisher{}
	event := schemeeventbus.PeriodBoundary{LotteryCode: "tron_ffc_6s", CurrentIssue: "100", NextIssue: "101", Generation: 9}

	if err := expandSchemePeriodBoundary(context.Background(), event, source, publisher, []int32{0, 1}, 64); err != nil {
		t.Fatalf("expandSchemePeriodBoundary() error = %v", err)
	}
	if source.limit != 32 || len(publisher.events) != 32 {
		t.Fatalf("listLimit=%d published=%d, want one bounded page", source.limit, len(publisher.events))
	}
	for index, ready := range publisher.events {
		if ready.DecisionID != int64(index+1) || ready.BoundaryGeneration != 9 || ready.MessageID() != "contiguous-target:"+itoa(index+1)+":9" {
			t.Fatalf("ready[%d]=%+v", index, ready)
		}
	}
}

func TestLeasedContiguousTargetWorkerCarriesStrategyFence(t *testing.T) {
	capture := &contiguousTargetWorkerCapture{}
	worker := leasedContiguousTargetWorker{
		worker: capture,
		fence:  schemes.StrategyLeaseFence{ShardNo: 3, Owner: "node-a", Epoch: 7},
	}
	event := schemeeventbus.ContiguousTargetReady{DecisionID: 41}
	if err := worker.ProcessContiguousTargetReady(context.Background(), event); err != nil {
		t.Fatalf("ProcessContiguousTargetReady() error = %v", err)
	}
	if capture.decisionID != 41 {
		t.Fatalf("capture decision=%d, want 41", capture.decisionID)
	}
}

func TestLeasedContiguousTargetWorkerCarriesStrategyFenceForRecovery(t *testing.T) {
	wantErr := errors.New("lease assertion failed")
	capture := &contiguousTargetWorkerCapture{recoveryErr: wantErr}
	worker := leasedContiguousTargetWorker{
		worker: capture,
		fence:  schemes.StrategyLeaseFence{ShardNo: 3, Owner: "node-a", Epoch: 7},
	}
	baseContext := context.Background()
	err := worker.RunContiguousTargetRecovery(
		baseContext, []string{"tron_ffc_6s"}, []int32{3}, 32, 1, time.Second,
	)
	if !errors.Is(err, wantErr) {
		t.Fatalf("error=%v want recovery error", err)
	}
	if capture.recoveryContext == nil || capture.recoveryContext == baseContext {
		t.Fatal("recovery did not receive the lease-wrapped context")
	}
}

func TestContiguousRecoveryWaitsForEveryRuntimeReadinessSignalAndStartsOnce(t *testing.T) {
	readiness := newContiguousRecoveryReadiness([]int32{3, 7})
	localSubscription := make(chan struct{})
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	started := make(chan struct{})
	var starts atomic.Int32
	go func() {
		done <- runContiguousTargetRecoveryWhenReady(ctx, readiness, localSubscription, func(runCtx context.Context) error {
			starts.Add(1)
			close(started)
			<-runCtx.Done()
			return nil
		})
	}()

	assertRecoveryNotStarted(t, started)
	readiness.SignalDrawWS()
	assertRecoveryNotStarted(t, started)
	readiness.SignalExpander()
	assertRecoveryNotStarted(t, started)
	readiness.SignalLease(3)
	readiness.SignalConsumer(3)
	close(localSubscription)
	assertRecoveryNotStarted(t, started)
	readiness.SignalLease(7)
	assertRecoveryNotStarted(t, started)
	readiness.SignalConsumer(7)
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("recovery did not start after all runtime readiness signals")
	}
	if got := starts.Load(); got != 1 {
		t.Fatalf("startup recovery starts=%d, want 1", got)
	}
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("recovery readiness runner returned %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("recovery readiness goroutine leaked after cancellation")
	}
}

func TestContiguousRecoveryReadinessCancellationExitsWithoutStarting(t *testing.T) {
	readiness := newContiguousRecoveryReadiness([]int32{3})
	localSubscription := make(chan struct{})
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	var starts atomic.Int32
	go func() {
		done <- runContiguousTargetRecoveryWhenReady(ctx, readiness, localSubscription, func(context.Context) error {
			starts.Add(1)
			return nil
		})
	}()
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("cancellation returned %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("readiness wait leaked after cancellation")
	}
	if got := starts.Load(); got != 0 {
		t.Fatalf("recovery started %d times before readiness", got)
	}
}

func TestContiguousShardRuntimeAssertsLeaseAndKeepsConsumerActiveBeforeRecovery(t *testing.T) {
	readiness := newContiguousRecoveryReadiness([]int32{3})
	readiness.SignalDrawWS()
	readiness.SignalExpander()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	consumerActive := make(chan struct{})
	recoveryStarted := make(chan struct{})
	var leaseAssertions atomic.Int32
	go func() {
		done <- runContiguousTargetShardRuntime(
			ctx, readiness, 3,
			func(context.Context) error {
				leaseAssertions.Add(1)
				return nil
			},
			func(consumeCtx context.Context, ready func()) error {
				ready()
				close(consumerActive)
				<-consumeCtx.Done()
				return nil
			},
			func(recoveryCtx context.Context) error {
				close(recoveryStarted)
				<-recoveryCtx.Done()
				return nil
			},
		)
	}()
	select {
	case <-consumerActive:
	case <-time.After(time.Second):
		t.Fatal("target-ready consumer did not remain active")
	}
	select {
	case <-recoveryStarted:
	case <-time.After(time.Second):
		t.Fatal("recovery did not start after lease and subscription readiness")
	}
	if got := leaseAssertions.Load(); got != 1 {
		t.Fatalf("lease assertions=%d, want 1", got)
	}
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("shard runtime leaked after cancellation")
	}
}

func TestContiguousShardRuntimeLeaseAssertionFailureStartsNothing(t *testing.T) {
	readiness := newContiguousRecoveryReadiness([]int32{3})
	wantErr := errors.New("stale lease")
	var consumers, recoveries atomic.Int32
	err := runContiguousTargetShardRuntime(
		context.Background(), readiness, 3,
		func(context.Context) error { return wantErr },
		func(context.Context, func()) error {
			consumers.Add(1)
			return nil
		},
		func(context.Context) error {
			recoveries.Add(1)
			return nil
		},
	)
	if !errors.Is(err, wantErr) {
		t.Fatalf("error=%v, want stale lease", err)
	}
	if consumers.Load() != 0 || recoveries.Load() != 0 {
		t.Fatalf("consumer starts=%d recovery starts=%d", consumers.Load(), recoveries.Load())
	}
}

func assertRecoveryNotStarted(t *testing.T, started <-chan struct{}) {
	t.Helper()
	select {
	case <-started:
		t.Fatal("recovery started before all runtime readiness signals")
	default:
	}
}

type recordingBoundaryAwaitingSource struct {
	rows  []sqlcdb.AwaitingContiguousTargetRow
	limit int32
}

func (source *recordingBoundaryAwaitingSource) ListAwaitingContiguousTargets(
	_ context.Context, _ []string, _ []int32, _ int64, limit int32,
) ([]sqlcdb.AwaitingContiguousTargetRow, error) {
	source.limit = limit
	if int(limit) < len(source.rows) {
		return source.rows[:limit], nil
	}
	return source.rows, nil
}

type recordingContiguousTargetPublisher struct {
	events []schemeeventbus.ContiguousTargetReady
}

func (publisher *recordingContiguousTargetPublisher) PublishContiguousTargetReady(
	_ context.Context, event schemeeventbus.ContiguousTargetReady, _ uint32,
) error {
	publisher.events = append(publisher.events, event)
	return nil
}

type contiguousTargetWorkerCapture struct {
	decisionID      int64
	recoveryContext context.Context
	recoveryErr     error
}

func (capture *contiguousTargetWorkerCapture) ProcessContiguousTargetReady(ctx context.Context, event schemeeventbus.ContiguousTargetReady) error {
	capture.decisionID = event.DecisionID
	return nil
}

func (capture *contiguousTargetWorkerCapture) RunContiguousTargetRecovery(
	ctx context.Context, _ []string, _ []int32, _, _ int, _ time.Duration,
) error {
	capture.recoveryContext = ctx
	return capture.recoveryErr
}

func itoa(value int) string {
	if value == 0 {
		return "0"
	}
	bytes := make([]byte, 0, 8)
	for value > 0 {
		bytes = append([]byte{byte('0' + value%10)}, bytes...)
		value /= 10
	}
	return string(bytes)
}
