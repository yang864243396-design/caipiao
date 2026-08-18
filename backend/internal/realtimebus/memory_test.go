package realtimebus

import (
	"context"
	"testing"
)

func TestMemoryBusIsolatesSubjectsAndUnsubscribes(t *testing.T) {
	// This catches a subject registry that broadcasts to every subscription or
	// leaves an unsubscribed handler active.
	bus := NewMemory()
	got := make(chan string, 2)
	sub, err := bus.Subscribe("caipiao.client.7.scheme", func(_ string, payload []byte) {
		got <- string(payload)
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := bus.Publish(context.Background(), "caipiao.client.8.scheme", []byte("wrong")); err != nil {
		t.Fatal(err)
	}
	if err := bus.Publish(context.Background(), "caipiao.client.7.scheme", []byte("right")); err != nil {
		t.Fatal(err)
	}
	if value := <-got; value != "right" {
		t.Fatalf("got %q", value)
	}
	if err := sub.Unsubscribe(); err != nil {
		t.Fatal(err)
	}
	if err := bus.Publish(context.Background(), "caipiao.client.7.scheme", []byte("late")); err != nil {
		t.Fatal(err)
	}
	select {
	case value := <-got:
		t.Fatalf("unexpected %q", value)
	default:
	}
}
