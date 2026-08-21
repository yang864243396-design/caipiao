package periodsync

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"caipiao/backend/internal/lottery"
)

func TestVerifyOpenPeriodUsesFreshSharedScheduleWithoutRefresh(t *testing.T) {
	code := fmt.Sprintf("test_shared_snapshot_ffc_%d", time.Now().UnixNano())
	lottery.ClearPeriodsSchedule(code)
	t.Cleanup(func() { lottery.ClearPeriodsSchedule(code) })
	now := time.Now().UTC()
	lottery.UpdatePeriodsScheduleFullWithDuration(code, "P200", "P200", now.Add(4*time.Second), now.Add(4*time.Second), 6, "", now.Add(-2*time.Second))
	syncer := &Syncer{prePlaceVerifications: newPrePlaceVerifyCache(time.Second)}
	var refreshCalls atomic.Int32

	result, err := syncer.verifyOpenPeriod(context.Background(), code, now, func(context.Context) (prePlaceVerifyResult, error) {
		refreshCalls.Add(1)
		return prePlaceVerifyResult{Period: "unexpected"}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Period != "P200" || !result.CloseAt.Equal(now.Add(4*time.Second)) {
		t.Fatalf("result=%+v", result)
	}
	if refreshCalls.Load() != 0 {
		t.Fatalf("refresh calls=%d, want 0", refreshCalls.Load())
	}
}

func TestVerifyOpenPeriodUsesFreshWSNextForTronSixSecondBet(t *testing.T) {
	code := "tron_ffc_6s"
	lottery.ClearPeriodsSchedule(code)
	now := time.Now().UTC()
	lottery.UpdatePeriodsScheduleFullWithDuration(
		code, "10114255902823", "10114255902823", now.Add(12*time.Second), now.Add(12*time.Second),
		6, "", now.Add(6*time.Second),
	)
	lottery.UpdatePeriodState(code, "10114255902821", "10114255902822", now, 6)
	syncer := &Syncer{prePlaceVerifications: newPrePlaceVerifyCache(time.Second)}
	var refreshCalls atomic.Int32

	result, err := syncer.verifyOpenPeriod(context.Background(), code, now.Add(100*time.Millisecond), func(context.Context) (prePlaceVerifyResult, error) {
		refreshCalls.Add(1)
		return prePlaceVerifyResult{Period: "unexpected"}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Period != "10114255902822" || !result.CloseAt.Equal(now.Add(6*time.Second)) {
		t.Fatalf("result=%+v want fresh WS next period", result)
	}
	if refreshCalls.Load() != 0 {
		t.Fatalf("refresh calls=%d want 0", refreshCalls.Load())
	}
}

func TestFreshSharedOpenPeriodUsesRememberedRealCloseForProvisionalPeriod(t *testing.T) {
	code := fmt.Sprintf("test_provisional_real_close_%d", time.Now().UnixNano())
	lottery.ClearPeriodsSchedule(code)
	t.Cleanup(func() { lottery.ClearPeriodsSchedule(code) })

	now := time.Now().UTC()
	openAt := now.Add(2 * time.Second)
	realCloseAt := openAt.Add(6 * time.Second)
	if !lottery.TryUpdatePeriodsScheduleFullWithDurationAt(
		code, "P250", "P250", openAt, openAt, 6, openAt.Format("2006-01-02 15:04:05"), openAt, now,
	) {
		t.Fatal("provisional schedule was not stored")
	}
	lottery.RememberPeriodsRealClose(code, "P250", realCloseAt, realCloseAt.Format("2006-01-02 15:04:05"))

	result, ok := freshSharedOpenPeriod(code, now)
	if !ok {
		t.Fatal("expected fresh provisional period")
	}
	if result.Period != "P250" || !result.CloseAt.Equal(realCloseAt) {
		t.Fatalf("result=%+v want real close %s", result, realCloseAt)
	}
}

func TestVerifyOpenPeriodCoalescesStaleRefreshByLottery(t *testing.T) {
	code := fmt.Sprintf("test_shared_refresh_ffc_%d", time.Now().UnixNano())
	lottery.ClearPeriodsSchedule(code)
	t.Cleanup(func() { lottery.ClearPeriodsSchedule(code) })
	syncer := &Syncer{prePlaceVerifications: newPrePlaceVerifyCache(time.Second)}
	var refreshCalls atomic.Int32
	started := make(chan struct{})
	release := make(chan struct{})
	refresh := func(context.Context) (prePlaceVerifyResult, error) {
		if refreshCalls.Add(1) == 1 {
			close(started)
		}
		<-release
		return prePlaceVerifyResult{Period: "P300", CloseAt: time.Now().Add(4 * time.Second)}, nil
	}

	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			result, err := syncer.verifyOpenPeriod(context.Background(), code, time.Now(), refresh)
			if err == nil && result.Period != "P300" {
				err = &unexpectedPeriodError{got: result.Period}
			}
			errs <- err
		}()
	}
	<-started
	close(release)
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	if refreshCalls.Load() != 1 {
		t.Fatalf("refresh calls=%d, want 1 for one lottery", refreshCalls.Load())
	}
}

func TestVerifyOpenPeriodLeaderCancellationDoesNotPoisonLotteryRefresh(t *testing.T) {
	code := fmt.Sprintf("test_shared_refresh_cancel_ffc_%d", time.Now().UnixNano())
	lottery.ClearPeriodsSchedule(code)
	t.Cleanup(func() { lottery.ClearPeriodsSchedule(code) })
	syncer := &Syncer{prePlaceVerifications: newPrePlaceVerifyCache(time.Second)}
	var refreshCalls atomic.Int32
	started := make(chan struct{})
	release := make(chan struct{})
	refresh := func(ctx context.Context) (prePlaceVerifyResult, error) {
		if refreshCalls.Add(1) == 1 {
			close(started)
		}
		for {
			if err := ctx.Err(); err != nil {
				return prePlaceVerifyResult{}, err
			}
			select {
			case <-release:
				return prePlaceVerifyResult{Period: "P400", CloseAt: time.Now().Add(4 * time.Second)}, nil
			case <-time.After(time.Millisecond):
			}
		}
	}

	leaderCtx, cancelLeader := context.WithCancel(context.Background())
	leaderDone := make(chan error, 1)
	go func() {
		_, err := syncer.verifyOpenPeriod(leaderCtx, code, time.Now(), refresh)
		leaderDone <- err
	}()
	<-started
	followerDone := make(chan error, 1)
	go func() {
		result, err := syncer.verifyOpenPeriod(context.Background(), code, time.Now(), refresh)
		if err == nil && result.Period != "P400" {
			err = &unexpectedPeriodError{got: result.Period}
		}
		followerDone <- err
	}()
	cancelLeader()
	time.Sleep(5 * time.Millisecond)
	close(release)

	select {
	case err := <-followerDone:
		if err != nil {
			t.Fatalf("follower inherited leader cancellation: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("follower did not receive shared refresh")
	}
	select {
	case <-leaderDone:
	case <-time.After(time.Second):
		t.Fatal("leader did not finish")
	}
	if refreshCalls.Load() != 1 {
		t.Fatalf("refresh calls=%d, want 1", refreshCalls.Load())
	}
}

func TestPrePlaceVerifyCacheCoalescesConcurrentRefreshes(t *testing.T) {
	cache := newPrePlaceVerifyCache(time.Second)
	var calls atomic.Int32
	started := make(chan struct{})
	release := make(chan struct{})

	fetch := func(context.Context) (prePlaceVerifyResult, error) {
		if calls.Add(1) == 1 {
			close(started)
		}
		<-release
		return prePlaceVerifyResult{Period: "P100"}, nil
	}

	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			result, err := cache.getOrRefresh(context.Background(), "ffc|member-a", time.Now(), fetch)
			if err == nil && result.Period != "P100" {
				err = &unexpectedPeriodError{got: result.Period}
			}
			errs <- err
		}()
	}
	<-started
	close(release)
	wg.Wait()
	close(errs)

	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("fetch calls=%d, want 1", got)
	}
}

func TestPrePlaceVerifyCacheReusesRecentResultOnlyForSameAccount(t *testing.T) {
	cache := newPrePlaceVerifyCache(time.Second)
	var calls atomic.Int32
	fetch := func(context.Context) (prePlaceVerifyResult, error) {
		return prePlaceVerifyResult{Period: "P"}, nil
	}
	now := time.Now()

	for _, key := range []string{"ffc|member-a", "ffc|member-a", "ffc|member-b"} {
		if _, err := cache.getOrRefresh(context.Background(), key, now, func(ctx context.Context) (prePlaceVerifyResult, error) {
			calls.Add(1)
			return fetch(ctx)
		}); err != nil {
			t.Fatal(err)
		}
	}
	if got := calls.Load(); got != 2 {
		t.Fatalf("fetch calls=%d, want 2 (one per account)", got)
	}
}

type unexpectedPeriodError struct{ got string }

func (e *unexpectedPeriodError) Error() string { return "unexpected period: " + e.got }
