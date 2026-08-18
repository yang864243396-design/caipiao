package main

import (
	"bytes"
	"context"
	"fmt"
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

func TestExternalNATSErrorsNeverExposeCredentialBearingURLs(t *testing.T) {
	const secretURL = "nats://smoke-user:super-secret-password@%zz"
	secretError := fmt.Errorf("invalid NATS URL %s", secretURL)
	tests := []struct {
		name       string
		args       []string
		newBus     func(realtimebus.NATSConfig) (realtimebus.Bus, error)
		wantOutput string
	}{
		{
			name: "connect",
			args: []string{"-nats", secretURL, "-member-id", "7"},
			newBus: func(config realtimebus.NATSConfig) (realtimebus.Bus, error) {
				if config.URL != secretURL {
					t.Fatalf("NATS URL = %q, want secret test URL", config.URL)
				}
				return nil, secretError
			},
			wantOutput: "error: nats_connect_failed\n",
		},
		{
			name: "subscribe scheme",
			args: []string{"-member-id", "7"},
			newBus: func(realtimebus.NATSConfig) (realtimebus.Bus, error) {
				bus := newSmokeTestBus()
				bus.subscribeErr = secretError
				bus.failSubscribeAt = 1
				return bus, nil
			},
			wantOutput: "error: nats_subscribe_failed kind=scheme\n",
		},
		{
			name: "subscribe stats",
			args: []string{"-member-id", "7"},
			newBus: func(realtimebus.NATSConfig) (realtimebus.Bus, error) {
				bus := newSmokeTestBus()
				bus.subscribeErr = secretError
				bus.failSubscribeAt = 2
				return bus, nil
			},
			wantOutput: "error: nats_subscribe_failed kind=stats\n",
		},
		{
			name: "publish",
			args: []string{"-member-id", "7", "-publish"},
			newBus: func(realtimebus.NATSConfig) (realtimebus.Bus, error) {
				bus := newSmokeTestBus()
				bus.publishErr = secretError
				return bus, nil
			},
			wantOutput: "error: nats_publish_failed\n",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			code := run(context.Background(), test.args, runDependencies{
				stdout: &stdout,
				stderr: &stderr,
				getenv: func(string) string { return "" },
				newBus: test.newBus,
			})

			if code != 1 {
				t.Fatalf("exit code = %d, want 1", code)
			}
			if got := stderr.String(); got != test.wantOutput {
				t.Fatalf("stderr = %q, want fixed sanitized output %q", got, test.wantOutput)
			}
			output := stdout.String() + stderr.String()
			if strings.Contains(output, secretURL) || strings.Contains(output, "super-secret-password") || strings.Contains(output, secretError.Error()) {
				t.Fatalf("external NATS failure exposed credentials or raw error: %q", output)
			}
		})
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
	subscribeErr       error
	failSubscribeAt    int
	publishErr         error
}

func newSmokeTestBus() *smokeTestBus {
	return &smokeTestBus{subscriptionsReady: make(chan struct{})}
}

func (b *smokeTestBus) Publish(context.Context, string, []byte) error { return b.publishErr }

func (b *smokeTestBus) Subscribe(string, realtimebus.Handler) (realtimebus.Subscription, error) {
	b.mu.Lock()
	b.subscriptions++
	if b.subscribeErr != nil && b.subscriptions == b.failSubscribeAt {
		err := b.subscribeErr
		b.mu.Unlock()
		return nil, err
	}
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
