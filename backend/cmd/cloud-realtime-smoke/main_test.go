package main

import (
	"bytes"
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"caipiao/backend/internal/realtimebus"
)

func TestHelpDoesNotExposeNATSEnvironmentCredentials(t *testing.T) {
	const secretURL = "nats://smoke-user:super-secret-password@127.0.0.1:4222"
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run(context.Background(), []string{"-h"}, runDependencies{
		stdout: &stdout,
		stderr: &stderr,
		getenv: func(key string) string {
			if key == "NATS_URL" {
				return secretURL
			}
			return ""
		},
		newBus: func(realtimebus.NATSConfig) (realtimebus.Bus, error) {
			t.Fatal("help must not create a NATS connection")
			return nil, nil
		},
	})

	if code != 0 {
		t.Fatalf("help exit code = %d, want 0", code)
	}
	help := stdout.String() + stderr.String()
	if strings.Contains(help, secretURL) || strings.Contains(help, "super-secret-password") {
		t.Fatalf("help exposed NATS credentials: %q", help)
	}
}

func TestCancellationUsesControlledCleanupPath(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	bus := newSmokeTestBus()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	done := make(chan int, 1)
	go func() {
		done <- run(ctx, []string{"-member-id", "7", "-timeout", "1m"}, runDependencies{
			stdout: &stdout,
			stderr: &stderr,
			getenv: func(string) string { return "" },
			newBus: func(realtimebus.NATSConfig) (realtimebus.Bus, error) {
				return bus, nil
			},
		})
	}()

	select {
	case <-bus.subscriptionsReady:
	case <-time.After(time.Second):
		t.Fatal("smoke command did not establish both subscriptions")
	}
	cancel()

	select {
	case code := <-done:
		if code != 130 {
			t.Fatalf("canceled exit code = %d, want 130", code)
		}
	case <-time.After(time.Second):
		t.Fatal("smoke command did not stop after context cancellation")
	}

	closed, unsubscribed := bus.cleanupCounts()
	if closed != 1 || unsubscribed != 2 {
		t.Fatalf("cleanup counts: close=%d unsubscribe=%d, want close=1 unsubscribe=2", closed, unsubscribed)
	}
	if !strings.Contains(stderr.String(), "canceled") {
		t.Fatalf("cancellation diagnostic missing: %q", stderr.String())
	}
}

type smokeTestBus struct {
	mu                 sync.Mutex
	closed             int
	unsubscribed       int
	subscriptions      int
	subscriptionsReady chan struct{}
	readyOnce          sync.Once
}

func newSmokeTestBus() *smokeTestBus {
	return &smokeTestBus{subscriptionsReady: make(chan struct{})}
}

func (b *smokeTestBus) Publish(context.Context, string, []byte) error { return nil }

func (b *smokeTestBus) Subscribe(string, realtimebus.Handler) (realtimebus.Subscription, error) {
	b.mu.Lock()
	b.subscriptions++
	if b.subscriptions == 2 {
		b.readyOnce.Do(func() { close(b.subscriptionsReady) })
	}
	b.mu.Unlock()
	return &smokeTestSubscription{bus: b}, nil
}

func (b *smokeTestBus) OnConnectionChange(func(bool)) {}

func (b *smokeTestBus) Diagnostics() realtimebus.Diagnostics {
	return realtimebus.Diagnostics{Kind: "test", Connected: true}
}

func (b *smokeTestBus) Close() error {
	b.mu.Lock()
	b.closed++
	b.mu.Unlock()
	return nil
}

func (b *smokeTestBus) cleanupCounts() (closed, unsubscribed int) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.closed, b.unsubscribed
}

type smokeTestSubscription struct {
	bus  *smokeTestBus
	once sync.Once
}

func (s *smokeTestSubscription) Unsubscribe() error {
	s.once.Do(func() {
		s.bus.mu.Lock()
		s.bus.unsubscribed++
		s.bus.mu.Unlock()
	})
	return nil
}
