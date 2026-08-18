package ws

import (
	"sync"
	"testing"
	"time"
)

func TestConnOverflowClosesOnlySlowConnectionAsynchronously(t *testing.T) {
	h := NewHub()
	closeEntered := make(chan struct{})
	releaseClose := make(chan struct{})
	closeResult := make(chan struct {
		code   int
		reason string
	}, 1)
	c := &Conn{
		hub:    h,
		kind:   KindClient,
		topics: make(map[string]struct{}),
		send:   make(chan Envelope, 1),
		closeFn: func(code int, reason string) {
			closeResult <- struct {
				code   int
				reason string
			}{code: code, reason: reason}
			close(closeEntered)
			<-releaseClose
		},
	}
	c.send <- SystemFrame(NameConnected, nil)

	tryDone := make(chan bool, 1)
	go func() { tryDone <- c.TrySend(NewEvent("overflow", TopicClientSchemeInstance, nil)) }()
	select {
	case sent := <-tryDone:
		if sent {
			t.Fatal("overflow frame reported as sent")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("TrySend waited for network close")
	}
	waitClosed(t, closeEntered, "asynchronous backpressure close")
	result := <-closeResult
	if result.code != 4010 || result.reason != "realtime_backpressure" {
		t.Fatalf("close=%d/%q", result.code, result.reason)
	}
	if got := h.Diagnostics().BackpressureCloses; got != 1 {
		t.Fatalf("backpressure closes=%d", got)
	}
	if c.TrySend(NewEvent("late", TopicClientSchemeInstance, nil)) {
		t.Fatal("closed connection accepted a frame")
	}
	close(releaseClose)
	select {
	case _, ok := <-c.send:
		if !ok {
			t.Fatal("producer closed send channel")
		}
	default:
	}
}

func TestConnOverflowConcurrentSendersCloseOnceWithoutClosingSendChannel(t *testing.T) {
	h := NewHub()
	closed := make(chan struct{}, 1)
	c := &Conn{
		hub:    h,
		kind:   KindClient,
		topics: make(map[string]struct{}),
		send:   make(chan Envelope, 1),
		closeFn: func(int, string) {
			closed <- struct{}{}
		},
	}
	c.send <- SystemFrame(NameConnected, nil)

	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < 64; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_ = c.TrySend(NewEvent("overflow", TopicClientSchemeInstance, nil))
		}()
	}
	close(start)
	wg.Wait()
	waitClosed(t, closed, "single overflow close")
	if got := h.Diagnostics().BackpressureCloses; got != 1 {
		t.Fatalf("backpressure closes=%d", got)
	}
	select {
	case <-closed:
		t.Fatal("close function called more than once")
	default:
	}
	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("send channel was closed: %v", recovered)
		}
	}()
	select {
	case <-c.send:
	default:
	}
	c.send <- SystemFrame(NameConnected, nil)
}

func TestConnAsyncCloseReservesBackpressureCodeBeforeReturning(t *testing.T) {
	closed := make(chan struct {
		code   int
		reason string
	}, 1)
	c := &Conn{
		send: make(chan Envelope, 1),
		closeFn: func(code int, reason string) {
			closed <- struct {
				code   int
				reason string
			}{code: code, reason: reason}
		},
	}

	c.closeAsync(4010, "realtime_backpressure")
	c.Close(1000, "normal")
	select {
	case got := <-closed:
		if got.code != 4010 || got.reason != "realtime_backpressure" {
			t.Fatalf("close=%d/%q", got.code, got.reason)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("asynchronous close did not run")
	}
}

func TestConnCloseSignalsPumpWithoutClosingSendChannel(t *testing.T) {
	c := &Conn{
		done: make(chan struct{}),
		send: make(chan Envelope, 1),
	}
	c.Close(1000, "normal")
	select {
	case <-c.done:
	default:
		t.Fatal("pump stop signal remains open")
	}
	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("send channel was closed: %v", recovered)
		}
	}()
	c.send <- SystemFrame(NameConnected, nil)
}
