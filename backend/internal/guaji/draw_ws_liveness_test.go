package guaji

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

func TestDrawWSLivenessTimesOutSilentHalfOpenConnection(t *testing.T) {
	conn, clock := newFakeDrawWSConn(), newFakeClock()
	live := newDrawWSLiveness(conn, clock.Now)
	go func() { _ = live.Run(context.Background()) }()

	clock.Advance(drawWSReadIdleTimeout + time.Millisecond)
	eventually(t, time.Second, time.Millisecond, conn.WasClosed)
}

func TestDrawWSLivenessPongExtendsReadDeadline(t *testing.T) {
	conn, clock := newFakeDrawWSConn(), newFakeClock()
	_ = newDrawWSLiveness(conn, clock.Now)

	conn.EmitPong()
	if got, want := conn.LastReadDeadline(), clock.Now().Add(drawWSReadIdleTimeout); !got.Equal(want) {
		t.Fatalf("read deadline after pong = %s, want %s", got, want)
	}
}

func TestDrawWSLivenessStopsAllGoroutinesOnCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	live := newDrawWSLiveness(newFakeDrawWSConn(), time.Now)
	done := make(chan error, 1)
	go func() { done <- live.Run(ctx) }()

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run() error = %v, want nil", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Run() did not stop after context cancellation")
	}
}

func eventually(t *testing.T, timeout, interval time.Duration, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(interval)
	}
	if !condition() {
		t.Fatal("condition was not met before timeout")
	}
}

type fakeDrawWSConn struct {
	mu           sync.Mutex
	readDeadline time.Time
	pongHandler  func(string) error
	controls     []int
	closed       bool
	closedCh     chan struct{}
}

func newFakeDrawWSConn() *fakeDrawWSConn {
	return &fakeDrawWSConn{closedCh: make(chan struct{})}
}

func (c *fakeDrawWSConn) ReadMessage() (int, []byte, error) {
	<-c.closedCh
	return 0, nil, errors.New("fake websocket closed")
}

func (c *fakeDrawWSConn) WriteControl(messageType int, _ []byte, _ time.Time) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return errors.New("fake websocket closed")
	}
	c.controls = append(c.controls, messageType)
	return nil
}

func (c *fakeDrawWSConn) SetReadDeadline(deadline time.Time) error {
	c.mu.Lock()
	c.readDeadline = deadline
	c.mu.Unlock()
	return nil
}

func (c *fakeDrawWSConn) SetPongHandler(handler func(string) error) {
	c.mu.Lock()
	c.pongHandler = handler
	c.mu.Unlock()
}

func (c *fakeDrawWSConn) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.closed {
		c.closed = true
		close(c.closedCh)
	}
	return nil
}

func (c *fakeDrawWSConn) WasClosed() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.closed
}

func (c *fakeDrawWSConn) LastReadDeadline() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.readDeadline
}

func (c *fakeDrawWSConn) EmitPong() {
	c.mu.Lock()
	handler := c.pongHandler
	c.mu.Unlock()
	if handler != nil {
		_ = handler("")
	}
}

func (c *fakeDrawWSConn) waitForReadDeadline(deadline time.Time) <-chan time.Time {
	fakeClockMu.Lock()
	clock := activeFakeClock
	fakeClockMu.Unlock()
	if clock == nil {
		return nil
	}
	return clock.waitUntil(deadline)
}

type fakeClock struct {
	mu      sync.Mutex
	now     time.Time
	waiters map[chan time.Time]time.Time
}

var (
	fakeClockMu     sync.Mutex
	activeFakeClock *fakeClock
)

func newFakeClock() *fakeClock {
	clock := &fakeClock{
		now:     time.Date(2026, time.August, 21, 0, 0, 0, 0, time.UTC),
		waiters: make(map[chan time.Time]time.Time),
	}
	fakeClockMu.Lock()
	activeFakeClock = clock
	fakeClockMu.Unlock()
	return clock
}

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *fakeClock) Advance(duration time.Duration) {
	c.mu.Lock()
	c.now = c.now.Add(duration)
	for waiter, deadline := range c.waiters {
		if !c.now.Before(deadline) {
			waiter <- c.now
			delete(c.waiters, waiter)
		}
	}
	c.mu.Unlock()
}

func (c *fakeClock) waitUntil(deadline time.Time) <-chan time.Time {
	waiter := make(chan time.Time, 1)
	c.mu.Lock()
	if !c.now.Before(deadline) {
		waiter <- c.now
	} else {
		c.waiters[waiter] = deadline
	}
	c.mu.Unlock()
	return waiter
}
