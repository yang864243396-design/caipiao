package realtimebus

import (
	"context"
	"testing"
	"time"
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

func TestMemoryBusDoesNotBlockPublishOrOtherSubscribersOnSlowHandler(t *testing.T) {
	// This catches synchronous handler dispatch, where one stuck consumer
	// prevents Publish from returning and holds every later subscriber.
	bus := NewMemory()
	slowStarted := make(chan struct{})
	releaseSlow := make(chan struct{})
	defer close(releaseSlow)
	fastReceived := make(chan string, 1)

	if _, err := bus.Subscribe("caipiao.client.7.scheme", func(_ string, _ []byte) {
		close(slowStarted)
		<-releaseSlow
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := bus.Subscribe("caipiao.client.7.scheme", func(_ string, payload []byte) {
		fastReceived <- string(payload)
	}); err != nil {
		t.Fatal(err)
	}

	published := make(chan error, 1)
	go func() {
		published <- bus.Publish(context.Background(), "caipiao.client.7.scheme", []byte("snapshot"))
	}()

	select {
	case <-slowStarted:
	case <-time.After(time.Second):
		t.Fatal("slow handler did not start")
	}
	select {
	case err := <-published:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("Publish was blocked by slow handler")
	}
	select {
	case value := <-fastReceived:
		if value != "snapshot" {
			t.Fatalf("got %q", value)
		}
	case <-time.After(time.Second):
		t.Fatal("fast subscriber was blocked by slow handler")
	}
}
