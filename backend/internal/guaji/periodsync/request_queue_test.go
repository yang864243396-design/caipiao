package periodsync

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestWorkerRequestRefresh_singleFlightPerLottery(t *testing.T) {
	started := make(chan struct{}, 1)
	release := make(chan struct{})
	var calls atomic.Int32
	w := &Worker{refreshConcurrency: 2, refreshFn: func(_ context.Context, code string) error {
		if code != "single" {
			t.Fatalf("code=%q", code)
		}
		calls.Add(1)
		started <- struct{}{}
		<-release
		return nil
	}}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	w.startRefreshWorkers(ctx)

	w.RequestRefresh("single")
	w.RequestRefresh("single")
	w.RequestRefresh("single")
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("refresh did not start")
	}
	time.Sleep(20 * time.Millisecond)
	if got := calls.Load(); got != 1 {
		t.Fatalf("calls=%d want 1", got)
	}
	close(release)
}

func TestWorkerRequestRefresh_slowLotteryDoesNotBlockAnother(t *testing.T) {
	slowStarted := make(chan struct{}, 1)
	releaseSlow := make(chan struct{})
	fastDone := make(chan struct{}, 1)
	w := &Worker{refreshConcurrency: 2, refreshFn: func(_ context.Context, code string) error {
		if code == "slow" {
			slowStarted <- struct{}{}
			<-releaseSlow
			return nil
		}
		if code == "fast" {
			fastDone <- struct{}{}
			return nil
		}
		t.Fatalf("unexpected code %q", code)
		return nil
	}}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	w.startRefreshWorkers(ctx)
	w.RequestRefresh("slow")
	select {
	case <-slowStarted:
	case <-time.After(time.Second):
		t.Fatal("slow refresh did not start")
	}
	w.RequestRefresh("fast")
	select {
	case <-fastDone:
	case <-time.After(time.Second):
		t.Fatal("fast refresh was blocked by slow refresh")
	}
	close(releaseSlow)
}

func TestWorkerRequestRefresh_respectsGlobalConcurrency(t *testing.T) {
	gate := make(chan struct{})
	var inFlight, peak atomic.Int32
	w := &Worker{refreshConcurrency: 2, refreshFn: func(_ context.Context, _ string) error {
		cur := inFlight.Add(1)
		for {
			old := peak.Load()
			if cur <= old || peak.CompareAndSwap(old, cur) {
				break
			}
		}
		<-gate
		inFlight.Add(-1)
		return nil
	}}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	w.startRefreshWorkers(ctx)
	for _, code := range []string{"a", "b", "c", "d"} {
		w.RequestRefresh(code)
	}
	time.Sleep(30 * time.Millisecond)
	if got := peak.Load(); got != 2 {
		t.Fatalf("peak=%d want 2", got)
	}
	close(gate)
}
