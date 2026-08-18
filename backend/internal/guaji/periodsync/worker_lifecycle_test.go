package periodsync

import (
	"context"
	"testing"
	"time"
)

func TestWorkerRunWaitsForInFlightRefreshBeforeReturning(t *testing.T) {
	refreshStarted := make(chan struct{})
	releaseRefresh := make(chan struct{})
	refreshReturned := make(chan struct{})
	worker := &Worker{
		interval:           time.Hour,
		refreshConcurrency: 1,
		refreshTimeout:     time.Hour,
		refreshFunc: func(_ context.Context, lotteryCode string) error {
			if lotteryCode != "lifecycle-test" {
				t.Errorf("lotteryCode=%q", lotteryCode)
			}
			close(refreshStarted)
			<-releaseRefresh
			close(refreshReturned)
			return nil
		},
	}
	ctx, cancel := context.WithCancel(context.Background())
	runReturned := make(chan struct{})
	go func() {
		worker.Run(ctx)
		close(runReturned)
	}()
	worker.RequestRefresh("lifecycle-test")

	select {
	case <-refreshStarted:
	case <-time.After(time.Second):
		cancel()
		close(releaseRefresh)
		t.Fatal("refresh child did not start")
	}
	cancel()
	returnedBeforeRefresh := false
	select {
	case <-runReturned:
		returnedBeforeRefresh = true
	case <-time.After(100 * time.Millisecond):
	}
	close(releaseRefresh)
	select {
	case <-refreshReturned:
	case <-time.After(time.Second):
		t.Fatal("refresh child did not finish")
	}
	if !returnedBeforeRefresh {
		select {
		case <-runReturned:
		case <-time.After(time.Second):
			t.Fatal("Run did not return after refresh child finished")
		}
	}
	if returnedBeforeRefresh {
		t.Fatal("Run returned before its in-flight refresh child finished")
	}
}
