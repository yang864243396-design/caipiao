package drawsync

import (
	"context"
	"errors"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"caipiao/backend/internal/guaji"
)

func TestWorkerRunUsesCappedReconnectBackoff(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var waits []time.Duration
	worker := &Worker{
		client: drawSubscriberFunc(func(context.Context, func([]guaji.DrawEvent)) error {
			return errors.New("draw websocket disconnected")
		}),
		reconnectJitter: func(delay time.Duration) time.Duration { return delay },
		waitRetry: func(_ context.Context, delay time.Duration) bool {
			waits = append(waits, delay)
			if len(waits) == 6 {
				cancel()
				return false
			}
			return true
		},
	}

	worker.Run(ctx)

	want := []time.Duration{time.Second, 2 * time.Second, 4 * time.Second, 8 * time.Second, 16 * time.Second, 30 * time.Second}
	if !reflect.DeepEqual(waits, want) {
		t.Fatalf("reconnect waits = %v, want %v", waits, want)
	}
}

func TestWorkerRunResetsReconnectBackoffAfterValidDrawFrame(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	attempts := 0
	var waits []time.Duration
	worker := &Worker{
		client: drawSubscriberFunc(func(_ context.Context, handler func([]guaji.DrawEvent)) error {
			attempts++
			if attempts == 3 {
				handler([]guaji.DrawEvent{{GameKey: "lottery_log001", Periods: "123"}})
			}
			return errors.New("draw websocket disconnected")
		}),
		reconnectJitter: func(delay time.Duration) time.Duration { return delay },
		waitRetry: func(_ context.Context, delay time.Duration) bool {
			waits = append(waits, delay)
			if len(waits) == 3 {
				cancel()
				return false
			}
			return true
		},
	}

	worker.Run(ctx)

	want := []time.Duration{time.Second, 2 * time.Second, time.Second}
	if !reflect.DeepEqual(waits, want) {
		t.Fatalf("reconnect waits after valid draw = %v, want %v", waits, want)
	}
	if attempts != 3 {
		t.Fatalf("subscription attempts = %d, want 3", attempts)
	}
}

func TestWorkerRunCancelsOneSharedSubscriptionForSimultaneousStaleLotteries(t *testing.T) {
	base := time.Unix(100, 0)
	health := guaji.NewBoundaryHealth([]string{"tron_ffc_3s", "tron_ffc_6s"})
	health.Observe("tron_ffc_3s", "10", "11", base, 3*time.Second)
	health.Observe("tron_ffc_6s", "20", "21", base, 6*time.Second)
	ticks := make(chan time.Time)
	subscriber := newSerialDrawSubscriber()
	var waits atomic.Int32
	worker := &Worker{
		client:         subscriber,
		boundaryHealth: health,
		boundaryHealthTicks: func() (<-chan time.Time, func()) {
			return ticks, func() {}
		},
		reconnectJitter: func(delay time.Duration) time.Duration { return delay },
		waitRetry: func(context.Context, time.Duration) bool {
			waits.Add(1)
			return true
		},
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		worker.Run(ctx)
		close(done)
	}()

	requireSubscriptionAttempt(t, subscriber.started, 1)
	ticks <- base.Add(10 * time.Second)
	requireSubscriptionAttempt(t, subscriber.started, 2)
	if got := subscriber.canceled.Load(); got != 1 {
		t.Fatalf("stale subscription cancellations=%d want=1", got)
	}
	if got := subscriber.maxActive.Load(); got != 1 {
		t.Fatalf("maximum parallel subscriptions=%d want=1", got)
	}
	if got := waits.Load(); got != 1 {
		t.Fatalf("serial reconnect waits=%d want=1", got)
	}
	if got := subscriber.attempts.Load(); got != 2 {
		t.Fatalf("subscription attempts=%d want=2", got)
	}
	if !health.Snapshot("tron_ffc_3s").ReconnectRequested || !health.Snapshot("tron_ffc_6s").ReconnectRequested {
		t.Fatal("simultaneously stale lotteries did not share one reconnect generation")
	}

	cancel()
	requireWorkerReturn(t, done)
	if got := subscriber.attempts.Load(); got != 2 {
		t.Fatalf("subscription storm after parent cancellation: attempts=%d", got)
	}
}

func TestWorkerRunStopsSupervisorAndSubscriptionOnParentCancellation(t *testing.T) {
	subscriber := newSerialDrawSubscriber()
	ticks := make(chan time.Time)
	var supervisorStops atomic.Int32
	var waits atomic.Int32
	worker := &Worker{
		client:         subscriber,
		boundaryHealth: guaji.NewBoundaryHealth([]string{"tron_ffc_15s"}),
		boundaryHealthTicks: func() (<-chan time.Time, func()) {
			return ticks, func() { supervisorStops.Add(1) }
		},
		waitRetry: func(context.Context, time.Duration) bool {
			waits.Add(1)
			return true
		},
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		worker.Run(ctx)
		close(done)
	}()

	requireSubscriptionAttempt(t, subscriber.started, 1)
	cancel()
	requireWorkerReturn(t, done)
	if got := supervisorStops.Load(); got != 1 {
		t.Fatalf("supervisor stops=%d want=1", got)
	}
	if got := subscriber.active.Load(); got != 0 {
		t.Fatalf("active subscriptions after Run return=%d want=0", got)
	}
	if got := waits.Load(); got != 0 {
		t.Fatalf("reconnect waits after parent cancellation=%d want=0", got)
	}
}

func TestWorkerRejectedFormalBoundaryDoesNotRefreshHealth(t *testing.T) {
	health := guaji.NewBoundaryHealth([]string{"tron_ffc_3s"})
	worker := &Worker{boundaryHealth: health}
	base := time.Unix(100, 0)
	health.Observe("tron_ffc_3s", "P100", "P101", base, 3*time.Second)
	if got := health.Stale(base.Add(4 * time.Second)); len(got) != 1 {
		t.Fatalf("first stale signal = %v, want one", got)
	}

	worker.observeAcceptedBoundary(false, "tron_ffc_3s", "P99", "P100", base.Add(4*time.Second), 3)
	snapshot := health.Snapshot("tron_ffc_3s")
	if !snapshot.ReconnectRequested || snapshot.CurrentIssue != "P100" || !snapshot.LastReceivedMono.Equal(base) {
		t.Fatalf("worker refreshed health for rejected replay: %+v", snapshot)
	}

	worker.observeAcceptedBoundary(true, "tron_ffc_3s", "P101", "P102", base.Add(5*time.Second), 3)
	snapshot = health.Snapshot("tron_ffc_3s")
	if snapshot.ReconnectRequested || snapshot.CurrentIssue != "P101" || !snapshot.LastReceivedMono.Equal(base.Add(5*time.Second)) {
		t.Fatalf("worker did not observe accepted advancement: %+v", snapshot)
	}
}

type serialDrawSubscriber struct {
	attempts  atomic.Int32
	active    atomic.Int32
	maxActive atomic.Int32
	canceled  atomic.Int32
	started   chan int
}

func newSerialDrawSubscriber() *serialDrawSubscriber {
	return &serialDrawSubscriber{started: make(chan int, 4)}
}

func (s *serialDrawSubscriber) SubscribeDraws(ctx context.Context, _ func([]guaji.DrawEvent)) error {
	attempt := s.attempts.Add(1)
	active := s.active.Add(1)
	for {
		maximum := s.maxActive.Load()
		if active <= maximum || s.maxActive.CompareAndSwap(maximum, active) {
			break
		}
	}
	s.started <- int(attempt)
	<-ctx.Done()
	s.active.Add(-1)
	s.canceled.Add(1)
	return ctx.Err()
}

func requireSubscriptionAttempt(t *testing.T, started <-chan int, want int) {
	t.Helper()
	select {
	case got := <-started:
		if got != want {
			t.Fatalf("subscription attempt=%d want=%d", got, want)
		}
	case <-time.After(time.Second):
		t.Fatalf("subscription attempt %d did not start", want)
	}
}

func requireWorkerReturn(t *testing.T, done <-chan struct{}) {
	t.Helper()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Worker.Run did not return")
	}
}

type drawSubscriberFunc func(context.Context, func([]guaji.DrawEvent)) error

func (f drawSubscriberFunc) SubscribeDraws(ctx context.Context, handler func([]guaji.DrawEvent)) error {
	return f(ctx, handler)
}
