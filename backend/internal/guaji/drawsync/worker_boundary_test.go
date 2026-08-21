package drawsync

import (
	"context"
	"sync"
	"sync/atomic"
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

func TestDrawWorkerPublishesAcceptedBoundariesAndRejectsReplay(t *testing.T) {
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
	if publisher.events[0].Generation == 0 || publisher.events[1].Generation == 0 {
		t.Fatalf("boundary tokens must be non-zero: %+v", publisher.events)
	}
	if publisher.events[0].Generation == publisher.events[1].Generation {
		t.Fatalf("different boundaries shared token %d", publisher.events[0].Generation)
	}
}

func TestDrawWorkerBoundaryTokenSurvivesWorkerRestart(t *testing.T) {
	event := wsDraw("tron_ffc_6s", "P700", "P701")
	first := publishOneBoundary(t, event)
	second := publishOneBoundary(t, event)
	different := publishOneBoundary(t, wsDraw("tron_ffc_6s", "P701", "P702"))

	if first == 0 {
		t.Fatal("same-boundary token is zero")
	}
	if first != second {
		t.Fatalf("same boundary tokens differ across workers: %d != %d", first, second)
	}
	if first == different {
		t.Fatalf("different boundaries shared token %d", first)
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

func TestDrawWorkerSignalsRecoveryReadyAfterAcceptedConfiguredBoundaryAndHealthUpdate(t *testing.T) {
	worker := newDrawWorkerForTest(&fakeDrawStore{inserted: true}, &fakeBoundaryPublisher{})
	ready := make(chan struct{})
	var signals atomic.Int32
	worker.SetContiguousRecoveryReady([]string{"tron_ffc_6s"}, func() {
		if snapshot := worker.boundaryHealth.Snapshot("tron_ffc_6s"); snapshot.CurrentIssue != "P101" || snapshot.NextIssue != "P102" {
			t.Errorf("readiness observed before health update: %+v", snapshot)
		}
		signals.Add(1)
		select {
		case <-ready:
		default:
			close(ready)
		}
	})

	if err := worker.Ingest(context.Background(), wsDraw("tron_ffc_3s", "P100", "P101")); err != nil {
		t.Fatal(err)
	}
	select {
	case <-ready:
		t.Fatal("unconfigured formal lottery signaled readiness")
	default:
	}

	worker.periodStateUpdater = func(string, string, string, time.Time, int) bool { return false }
	if err := worker.Ingest(context.Background(), wsDraw("tron_ffc_6s", "P100", "P101")); err != nil {
		t.Fatal(err)
	}
	select {
	case <-ready:
		t.Fatal("rejected boundary signaled readiness")
	default:
	}

	worker.periodStateUpdater = func(string, string, string, time.Time, int) bool { return true }
	if err := worker.Ingest(context.Background(), wsDraw("tron_ffc_6s", "P101", "P102")); err != nil {
		t.Fatal(err)
	}
	select {
	case <-ready:
	case <-time.After(time.Second):
		t.Fatal("accepted configured boundary did not signal readiness")
	}
	if err := worker.Ingest(context.Background(), wsDraw("tron_ffc_6s", "P102", "P103")); err != nil {
		t.Fatal(err)
	}
	if got := signals.Load(); got != 1 {
		t.Fatalf("readiness signals=%d, want exactly one", got)
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

type fakePeriodStateUpdater struct {
	mu   sync.Mutex
	seen map[string]string
}

func newFakePeriodStateUpdater() *fakePeriodStateUpdater {
	return &fakePeriodStateUpdater{seen: make(map[string]string)}
}

func (updater *fakePeriodStateUpdater) Update(lotteryCode, currentIssue, nextIssue string, _ time.Time, _ int) bool {
	updater.mu.Lock()
	defer updater.mu.Unlock()
	boundary := currentIssue + "\x00" + nextIssue
	if updater.seen[lotteryCode] == boundary {
		return false
	}
	updater.seen[lotteryCode] = boundary
	return true
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
	updater := newFakePeriodStateUpdater()
	return &Worker{
		store: store, boundaryPublisher: publisher, periodStateUpdater: updater.Update,
		boundaryHealth: guaji.NewBoundaryHealth(lottery.FormalShortPeriodLotteryCodes()),
	}
}

func publishOneBoundary(t *testing.T, event guaji.DrawEvent) uint64 {
	t.Helper()
	publisher := &fakeBoundaryPublisher{}
	worker := newDrawWorkerForTest(&fakeDrawStore{inserted: true}, publisher)
	if err := worker.Ingest(context.Background(), event); err != nil {
		t.Fatalf("ingest boundary: %v", err)
	}
	publisher.mu.Lock()
	defer publisher.mu.Unlock()
	if len(publisher.events) != 1 {
		t.Fatalf("published boundaries = %d, want 1", len(publisher.events))
	}
	return publisher.events[0].Generation
}
