package periodsync

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

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
