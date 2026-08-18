package schemes

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestStrategyProcessorNotifyDrawIsNonBlockingAndCloseWaits(t *testing.T) {
	recoveryStarted := make(chan struct{}, 1)
	releaseRecovery := make(chan struct{})
	var recoveryCalls atomic.Int32
	processor := &StrategyProcessor{recoverFn: func(context.Context) error {
		recoveryCalls.Add(1)
		recoveryStarted <- struct{}{}
		<-releaseRecovery
		return nil
	}}

	notifyReturned := make(chan struct{})
	go func() {
		processor.NotifyDraw(context.Background(), "lottery", "period")
		close(notifyReturned)
	}()
	select {
	case <-notifyReturned:
	case <-time.After(100 * time.Millisecond):
		close(releaseRecovery)
		t.Fatal("NotifyDraw blocked on strategy recovery")
	}
	select {
	case <-recoveryStarted:
	case <-time.After(time.Second):
		close(releaseRecovery)
		t.Fatal("strategy recovery did not start")
	}

	closeReturned := make(chan struct{})
	go func() {
		processor.Close()
		close(closeReturned)
	}()
	select {
	case <-closeReturned:
		close(releaseRecovery)
		t.Fatal("Close returned before strategy recovery finished")
	case <-time.After(100 * time.Millisecond):
	}
	close(releaseRecovery)
	select {
	case <-closeReturned:
	case <-time.After(time.Second):
		t.Fatal("Close did not return after strategy recovery finished")
	}

	processor.NotifyDraw(context.Background(), "lottery", "later-period")
	select {
	case <-recoveryStarted:
		t.Fatal("NotifyDraw launched strategy recovery after Close")
	case <-time.After(50 * time.Millisecond):
	}
	if calls := recoveryCalls.Load(); calls != 1 {
		t.Fatalf("recovery calls=%d want 1", calls)
	}
}

func TestWorkerRunWaitsForStrategyRecoveryBeforeReturning(t *testing.T) {
	recoveryStarted := make(chan struct{})
	releaseRecovery := make(chan struct{})
	processor := &StrategyProcessor{recoverFn: func(context.Context) error {
		close(recoveryStarted)
		<-releaseRecovery
		return nil
	}}
	worker := &Worker{tickSec: 3600, strategyProcessor: processor}
	ctx, cancel := context.WithCancel(context.Background())
	runReturned := make(chan struct{})
	go func() {
		worker.Run(ctx)
		close(runReturned)
	}()
	worker.NotifyStrategyDraw(context.Background(), "lottery", "period")

	select {
	case <-recoveryStarted:
	case <-time.After(time.Second):
		cancel()
		close(releaseRecovery)
		t.Fatal("strategy recovery did not start")
	}
	cancel()
	returnedBeforeRecovery := false
	select {
	case <-runReturned:
		returnedBeforeRecovery = true
	case <-time.After(100 * time.Millisecond):
	}
	close(releaseRecovery)
	if !returnedBeforeRecovery {
		select {
		case <-runReturned:
		case <-time.After(time.Second):
			t.Fatal("Worker.Run did not return after strategy recovery finished")
		}
	}
	if returnedBeforeRecovery {
		t.Fatal("Worker.Run returned before its strategy recovery child finished")
	}
}
