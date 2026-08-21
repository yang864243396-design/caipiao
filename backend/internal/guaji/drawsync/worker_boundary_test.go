package drawsync

import (
	"context"
	"sync"
	"testing"
	"time"

	"caipiao/backend/internal/guaji"
	"caipiao/backend/internal/lottery"
	"caipiao/backend/internal/schemeeventbus"
)

func TestDrawWorkerPublishesBoundaryWhenDrawInsertIsDuplicate(t *testing.T) {
	store := &fakeDrawStore{inserted: false}
	publisher := &fakeBoundaryPublisher{}
	worker := newDrawWorkerForTest(store, publisher)

	if err := worker.Ingest(context.Background(), wsDraw("tron_ffc_6s", "100", "101")); err != nil {
		t.Fatalf("ingest duplicate draw: %v", err)
	}
	if got := publisher.Count(); got != 1 {
		t.Fatalf("published boundaries = %d, want 1", got)
	}
	if got := store.PersistCount(); got != 1 {
		t.Fatalf("persist calls = %d, want 1", got)
	}
}

func TestDrawWorkerPublishesOneMonotonicGenerationPerAcceptedBoundary(t *testing.T) {
	store := &fakeDrawStore{inserted: true}
	publisher := &fakeBoundaryPublisher{}
	worker := newDrawWorkerForTest(store, publisher)

	for _, event := range []guaji.DrawEvent{
		wsDraw("tron_ffc_3s", "P100", "P101"),
		wsDraw("tron_ffc_3s", "P101", "P102"),
		wsDraw("tron_ffc_3s", "P101", "P102"),
	} {
		if err := worker.Ingest(context.Background(), event); err != nil {
			t.Fatalf("ingest boundary: %v", err)
		}
	}

	publisher.mu.Lock()
	defer publisher.mu.Unlock()
	if got := len(publisher.events); got != 2 {
		t.Fatalf("published boundaries = %d, want 2", got)
	}
	if got := publisher.events[0].Generation; got != 1 {
		t.Fatalf("first generation = %d, want 1", got)
	}
	if got := publisher.events[1].Generation; got != 2 {
		t.Fatalf("second generation = %d, want 2", got)
	}
}

func TestDrawWorkerPersistsAfterBoundaryPublishFailure(t *testing.T) {
	store := &fakeDrawStore{inserted: true}
	publisher := &fakeBoundaryPublisher{err: context.DeadlineExceeded}
	worker := newDrawWorkerForTest(store, publisher)

	if err := worker.Ingest(context.Background(), wsDraw("tron_ffc_15s", "P100", "P101")); err != nil {
		t.Fatalf("ingest after publish failure: %v", err)
	}
	if got := store.PersistCount(); got != 1 {
		t.Fatalf("persist calls = %d, want 1", got)
	}
}

type fakeBoundaryPublisher struct {
	mu     sync.Mutex
	events []schemeeventbus.PeriodBoundary
	err    error
}

func (publisher *fakeBoundaryPublisher) PublishPeriodBoundary(_ context.Context, event schemeeventbus.PeriodBoundary) error {
	publisher.mu.Lock()
	defer publisher.mu.Unlock()
	publisher.events = append(publisher.events, event)
	return publisher.err
}

func (publisher *fakeBoundaryPublisher) Count() int {
	publisher.mu.Lock()
	defer publisher.mu.Unlock()
	return len(publisher.events)
}

type fakeDrawStore struct {
	inserted bool

	mu           sync.Mutex
	persistCalls int
}

func (*fakeDrawStore) ResolveLotteries(_ context.Context, gameKey string) ([]lotteryTarget, error) {
	return []lotteryTarget{{code: gameKey, template: "fast_ssc_std"}}, nil
}

func (*fakeDrawStore) DrawIntervalSec(context.Context, string) int { return 6 }

func (store *fakeDrawStore) PersistDraw(_ context.Context, _ string, _ string, _ []string, _ time.Time, _ lottery.DrawFactMeta) (bool, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	store.persistCalls++
	return store.inserted, nil
}

func (store *fakeDrawStore) PersistCount() int {
	store.mu.Lock()
	defer store.mu.Unlock()
	return store.persistCalls
}

func wsDraw(lotteryCode, currentIssue, nextIssue string) guaji.DrawEvent {
	return guaji.DrawEvent{
		GameKey: lotteryCode, Periods: currentIssue, NextPeriods: nextIssue,
		DrawnAt: time.Unix(1_700_000_000, 0).UTC(), Balls: guaji.DrawBalls{SSC: "12345"},
	}
}

func newDrawWorkerForTest(store *fakeDrawStore, publisher *fakeBoundaryPublisher) *Worker {
	return &Worker{
		store: store, boundaryPublisher: publisher,
		boundaryHealth: guaji.NewBoundaryHealth(lottery.FormalShortPeriodLotteryCodes()),
	}
}
