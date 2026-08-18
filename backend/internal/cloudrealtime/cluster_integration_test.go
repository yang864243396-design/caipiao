package cloudrealtime

import (
	"context"
	"fmt"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"caipiao/backend/internal/realtimebus"
)

func TestNATSClusterIsolatesMembersAndRecoversAfterAPIReconnect(t *testing.T) {
	url := os.Getenv("NATS_TEST_URL")
	if url == "" {
		t.Skip("NATS_TEST_URL not set")
	}

	worker, err := realtimebus.NewNATS(realtimebus.NATSConfig{
		URL:           url,
		Name:          "cloud-realtime-cluster-test-worker",
		ReconnectWait: 25 * time.Millisecond,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = worker.Close() })
	waitForConnected(t, worker, 5*time.Second)

	api := newClusterTestAPI(t, url)
	prefix := fmt.Sprintf("caipiao-test-%d", time.Now().UnixNano())
	member7Subject, err := SchemeSubject(prefix, 7)
	if err != nil {
		t.Fatal(err)
	}
	member8Subject, err := SchemeSubject(prefix, 8)
	if err != nil {
		t.Fatal(err)
	}

	first := subscribeClusterMember(t, api, member7Subject)
	if err := worker.Publish(context.Background(), member8Subject, []byte(`{"schemaVersion":1,"member":8}`)); err != nil {
		t.Fatal(err)
	}
	assertNoClusterMessage(t, first, 150*time.Millisecond)
	waitForClusterSubscription(t, worker, member7Subject, first)
	assertSingleClusterMessage(t, worker, member7Subject, []byte(`{"schemaVersion":1,"member":7,"sequence":1}`), first)

	var closeBrowserRequests atomic.Int64
	api.OnConnectionChange(func(connected bool) {
		if !connected {
			closeBrowserRequests.Add(1)
		}
	})
	if err := api.Close(); err != nil {
		t.Fatal(err)
	}
	waitForCondition(t, 5*time.Second, func() bool { return closeBrowserRequests.Load() > 0 }, "disconnect callback did not request browser closure")

	reconnectedAPI := newClusterTestAPI(t, url)
	t.Cleanup(func() { _ = reconnectedAPI.Close() })
	second := subscribeClusterMember(t, reconnectedAPI, member7Subject)
	waitForClusterSubscription(t, worker, member7Subject, second)
	assertSingleClusterMessage(t, worker, member7Subject, []byte(`{"schemaVersion":1,"member":7,"sequence":2}`), second)
}

func newClusterTestAPI(t *testing.T, url string) *realtimebus.NATS {
	t.Helper()
	bus, err := realtimebus.NewNATS(realtimebus.NATSConfig{
		URL:           url,
		Name:          fmt.Sprintf("cloud-realtime-cluster-test-api-%d", time.Now().UnixNano()),
		ReconnectWait: 25 * time.Millisecond,
	})
	if err != nil {
		t.Fatal(err)
	}
	waitForConnected(t, bus, 5*time.Second)
	return bus
}

func subscribeClusterMember(t *testing.T, bus realtimebus.Bus, subject string) <-chan string {
	t.Helper()
	received := make(chan string, 4)
	subscription, err := bus.Subscribe(subject, func(_ string, payload []byte) {
		received <- string(payload)
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = subscription.Unsubscribe() })
	return received
}

func waitForClusterSubscription(t *testing.T, publisher realtimebus.Bus, subject string, received <-chan string) {
	t.Helper()
	payload := []byte(`{"schemaVersion":1,"probe":true}`)
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if err := publisher.Publish(context.Background(), subject, payload); err != nil {
			t.Fatal(err)
		}
		select {
		case got := <-received:
			if got != string(payload) {
				t.Fatalf("got %q want %q", got, payload)
			}
			return
		case <-time.After(25 * time.Millisecond):
		}
	}
	t.Fatal("timed out waiting for member subscription readiness")

}

func assertSingleClusterMessage(t *testing.T, publisher realtimebus.Bus, subject string, payload []byte, received <-chan string) {
	t.Helper()
	if err := publisher.Publish(context.Background(), subject, payload); err != nil {
		t.Fatal(err)
	}
	select {
	case got := <-received:
		if got != string(payload) {
			t.Fatalf("got %q want %q", got, payload)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for member-scoped snapshot")
	}
	assertNoClusterMessage(t, received, 150*time.Millisecond)
}

func assertNoClusterMessage(t *testing.T, received <-chan string, duration time.Duration) {
	t.Helper()
	select {
	case got := <-received:
		t.Fatalf("unexpected extra snapshot: %s", got)
	case <-time.After(duration):
	}
}

func waitForConnected(t *testing.T, bus realtimebus.Bus, timeout time.Duration) {
	t.Helper()
	waitForCondition(t, timeout, func() bool { return bus.Diagnostics().Connected }, "NATS client did not connect")
}

func waitForCondition(t *testing.T, timeout time.Duration, condition func() bool, message string) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal(message)
}
