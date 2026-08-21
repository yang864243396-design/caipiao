package periodsync

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"caipiao/backend/internal/lottery"
)

const prePlaceSharedRefreshTimeout = 1200 * time.Millisecond

// prePlaceVerifyResult is the provider-derived current open period. It may
// come from the fresh lottery-wide schedule or a coalesced upstream refresh.
type prePlaceVerifyResult struct {
	Period  string
	CloseAt time.Time
}

type prePlaceVerifyEntry struct {
	result     prePlaceVerifyResult
	verifiedAt time.Time
}

type prePlaceVerifyFlight struct {
	done   chan struct{}
	result prePlaceVerifyResult
	err    error
}

// prePlaceVerifyCache coalesces a short-lived upstream confirmation by
// lottery. Provider periods are global for a lottery, so users and schemes
// must not amplify one stale snapshot into separate upstream requests.
type prePlaceVerifyCache struct {
	ttl time.Duration

	mu       sync.Mutex
	recent   map[string]prePlaceVerifyEntry
	inFlight map[string]*prePlaceVerifyFlight
}

func freshSharedOpenPeriod(lotteryCode string, now time.Time) (prePlaceVerifyResult, bool) {
	lotteryCode = strings.TrimSpace(lotteryCode)
	now = now.UTC()
	if lottery.RequiresFreshShortPeriodWSBetTarget(lotteryCode) {
		sourcePeriod, ok := lottery.FreshShortPeriodWSCurrentIssue(lotteryCode, now)
		if !ok {
			return prePlaceVerifyResult{}, false
		}
		state, ok := lottery.FreshShortPeriodWSBetTarget(lotteryCode, sourcePeriod, now)
		if ok {
			return prePlaceVerifyResult{Period: state.NextIssue, CloseAt: state.CloseAt.UTC()}, true
		}
		return prePlaceVerifyResult{}, false
	}
	if lotteryCode == "" || !lottery.PeriodsScheduleFresh(lotteryCode, lottery.PeriodsFallbackStaleAge, now) {
		return prePlaceVerifyResult{}, false
	}
	snapshot, ok := lottery.PeriodsScheduleFor(lotteryCode)
	if !ok || strings.TrimSpace(snapshot.CurrentPeriod) == "" || snapshot.CloseAt.IsZero() || !now.Before(snapshot.CloseAt.UTC()) {
		return prePlaceVerifyResult{}, false
	}
	closeAt := snapshot.CloseAt.UTC()
	if snapshot.ProvisionalClose && snapshot.RealCloseAt.UTC().After(closeAt) {
		closeAt = snapshot.RealCloseAt.UTC()
	}
	return prePlaceVerifyResult{Period: strings.TrimSpace(snapshot.CurrentPeriod), CloseAt: closeAt}, true
}

func (s *Syncer) verifyOpenPeriod(
	ctx context.Context,
	lotteryCode string,
	now time.Time,
	refresh func(context.Context) (prePlaceVerifyResult, error),
) (prePlaceVerifyResult, error) {
	if result, ok := freshSharedOpenPeriod(lotteryCode, now); ok {
		return result, nil
	}
	if lottery.RequiresFreshShortPeriodWSBetTarget(lotteryCode) {
		return prePlaceVerifyResult{}, fmt.Errorf("fresh websocket period unavailable for %s", strings.TrimSpace(lotteryCode))
	}
	cache := s.prePlaceVerifications
	if cache == nil {
		cache = newPrePlaceVerifyCache(time.Second)
		s.prePlaceVerifications = cache
	}
	return cache.getOrRefresh(ctx, strings.TrimSpace(lotteryCode), now, func(leaderCtx context.Context) (prePlaceVerifyResult, error) {
		sharedCtx, cancel := context.WithTimeout(context.WithoutCancel(leaderCtx), prePlaceSharedRefreshTimeout)
		defer cancel()
		return refresh(sharedCtx)
	})
}

func newPrePlaceVerifyCache(ttl time.Duration) *prePlaceVerifyCache {
	if ttl <= 0 {
		ttl = 500 * time.Millisecond
	}
	return &prePlaceVerifyCache{
		ttl:      ttl,
		recent:   make(map[string]prePlaceVerifyEntry),
		inFlight: make(map[string]*prePlaceVerifyFlight),
	}
}

func (c *prePlaceVerifyCache) getOrRefresh(
	ctx context.Context,
	key string,
	now time.Time,
	fetch func(context.Context) (prePlaceVerifyResult, error),
) (prePlaceVerifyResult, error) {
	if c == nil {
		return fetch(ctx)
	}
	now = now.UTC()

	c.mu.Lock()
	if entry, ok := c.recent[key]; ok && now.Sub(entry.verifiedAt) >= 0 && now.Sub(entry.verifiedAt) < c.ttl {
		c.mu.Unlock()
		return entry.result, nil
	}
	if flight, ok := c.inFlight[key]; ok {
		c.mu.Unlock()
		select {
		case <-ctx.Done():
			return prePlaceVerifyResult{}, ctx.Err()
		case <-flight.done:
			return flight.result, flight.err
		}
	}
	flight := &prePlaceVerifyFlight{done: make(chan struct{})}
	c.inFlight[key] = flight
	c.mu.Unlock()

	result, err := fetch(ctx)

	c.mu.Lock()
	flight.result = result
	flight.err = err
	if err == nil {
		c.recent[key] = prePlaceVerifyEntry{result: result, verifiedAt: now}
	}
	delete(c.inFlight, key)
	close(flight.done)
	c.mu.Unlock()
	return result, err
}
