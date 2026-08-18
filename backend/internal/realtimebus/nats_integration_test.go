package realtimebus

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"
)

func TestNATSErrorDiagnosticsUseSafeCategory(t *testing.T) {
	// This catches diagnostics retaining a transport error verbatim, which can
	// expose connection details supplied to the NATS client.
	if got := safeNATSError(errors.New("dial failed")); got != "nats transport error" {
		t.Fatalf("got %q", got)
	}
}

func TestNATSBusExchangesMemberScopedMessage(t *testing.T) {
	url := os.Getenv("NATS_TEST_URL")
	if url == "" {
		t.Skip("NATS_TEST_URL not set")
	}

	publisher, err := NewNATS(NATSConfig{URL: url, Name: "realtimebus-test-publisher", ReconnectWait: 10 * time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = publisher.Close() })

	consumer, err := NewNATS(NATSConfig{URL: url, Name: "realtimebus-test-consumer", ReconnectWait: 10 * time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = consumer.Close() })

	if !waitForNATSConnection(publisher, 5*time.Second) || !waitForNATSConnection(consumer, 5*time.Second) {
		t.Fatal("NATS buses did not connect")
	}

	got := make(chan string, 1)
	sub, err := consumer.Subscribe("caipiao.client.7.scheme", func(_ string, payload []byte) {
		got <- string(payload)
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sub.Unsubscribe() })
	if err := consumer.nc.FlushTimeout(5 * time.Second); err != nil {
		t.Fatal(err)
	}

	if err := publisher.Publish(context.Background(), "caipiao.client.7.scheme", []byte("snapshot")); err != nil {
		t.Fatal(err)
	}
	select {
	case value := <-got:
		if value != "snapshot" {
			t.Fatalf("got %q", value)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for member-scoped message")
	}

	if !publisher.Diagnostics().Connected || !consumer.Diagnostics().Connected {
		t.Fatal("NATS diagnostics did not report connected buses")
	}
}

func waitForNATSConnection(bus Bus, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if bus.Diagnostics().Connected {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return false
}
