package periodsync

import (
	"context"
	"sync"
	"time"
)

// prePlaceVerifyResult is the upstream-confirmed current open period. It is
// intentionally independent from the global display cache so a caller can
// reject a stale target before it reaches PlaceBet.
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
// lottery and third-party account. A busy account therefore causes one
// periods request, not one request for every running scheme.
type prePlaceVerifyCache struct {
	ttl time.Duration

	mu       sync.Mutex
	recent   map[string]prePlaceVerifyEntry
	inFlight map[string]*prePlaceVerifyFlight
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
