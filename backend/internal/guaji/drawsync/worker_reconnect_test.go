package drawsync

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"caipiao/backend/internal/guaji"
)

func TestWorkerRunUsesCappedReconnectBackoff(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var waits []time.Duration
	worker := &Worker{
		client: drawSubscriberFunc(func(context.Context, func([]guaji.DrawEvent)) error {
			return errors.New("draw websocket disconnected")
		}),
		reconnectJitter: func(delay time.Duration) time.Duration { return delay },
		waitRetry: func(_ context.Context, delay time.Duration) bool {
			waits = append(waits, delay)
			if len(waits) == 6 {
				cancel()
				return false
			}
			return true
		},
	}

	worker.Run(ctx)

	want := []time.Duration{time.Second, 2 * time.Second, 4 * time.Second, 8 * time.Second, 16 * time.Second, 30 * time.Second}
	if !reflect.DeepEqual(waits, want) {
		t.Fatalf("reconnect waits = %v, want %v", waits, want)
	}
}

func TestWorkerRunResetsReconnectBackoffAfterValidDrawFrame(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	attempts := 0
	var waits []time.Duration
	worker := &Worker{
		client: drawSubscriberFunc(func(_ context.Context, handler func([]guaji.DrawEvent)) error {
			attempts++
			if attempts == 3 {
				handler([]guaji.DrawEvent{{GameKey: "lottery_log001", Periods: "123"}})
			}
			return errors.New("draw websocket disconnected")
		}),
		reconnectJitter: func(delay time.Duration) time.Duration { return delay },
		waitRetry: func(_ context.Context, delay time.Duration) bool {
			waits = append(waits, delay)
			if len(waits) == 3 {
				cancel()
				return false
			}
			return true
		},
	}

	worker.Run(ctx)

	want := []time.Duration{time.Second, 2 * time.Second, time.Second}
	if !reflect.DeepEqual(waits, want) {
		t.Fatalf("reconnect waits after valid draw = %v, want %v", waits, want)
	}
	if attempts != 3 {
		t.Fatalf("subscription attempts = %d, want 3", attempts)
	}
}

type drawSubscriberFunc func(context.Context, func([]guaji.DrawEvent)) error

func (f drawSubscriberFunc) SubscribeDraws(ctx context.Context, handler func([]guaji.DrawEvent)) error {
	return f(ctx, handler)
}
