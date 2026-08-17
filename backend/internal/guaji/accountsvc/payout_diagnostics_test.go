package accountsvc

import (
	"errors"
	"sync"
	"testing"
	"time"
)

func TestPayoutDiagnosticStoreTracksFailureThenRecovery(t *testing.T) {
	store := newPayoutDiagnosticStore()
	t1 := time.Unix(100, 0).UTC()
	store.begin(9, 4, t1)
	store.fail(9, errors.New("tls handshake timeout"), t1.Add(time.Second))
	failed, ok := store.snapshot(9)
	if !ok || failed.PendingCount != 4 || failed.LastError == "" || failed.LastErrorAt == nil {
		t.Fatalf("failed=%+v", failed)
	}

	t2 := t1.Add(10 * time.Second)
	store.begin(9, 4, t2)
	store.succeed(9, 50, 2, 2, t2.Add(time.Second))
	recovered, _ := store.snapshot(9)
	if recovered.LastError != "" || recovered.LastErrorAt != nil || recovered.LastSuccessAt == nil || recovered.SettledCount != 2 || recovered.ProviderUnsettledCount != 2 {
		t.Fatalf("recovered=%+v", recovered)
	}
}

func TestPayoutDiagnosticStoreConcurrentSnapshots(t *testing.T) {
	store := newPayoutDiagnosticStore()
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			now := time.Unix(int64(n+1), 0).UTC()
			store.begin(9, n, now)
			store.succeed(9, n, n/2, n-n/2, now)
			if got, ok := store.snapshot(9); ok && (got.PendingCount < 0 || got.SettledCount < 0 || got.ProviderUnsettledCount < 0) {
				t.Errorf("invalid snapshot: %+v", got)
			}
		}(i)
	}
	wg.Wait()
}

func TestPayoutDiagnosticStoreReturnsValueCopies(t *testing.T) {
	store := newPayoutDiagnosticStore()
	now := time.Unix(100, 0).UTC()
	store.begin(9, 1, now)

	first, ok := store.snapshot(9)
	if !ok || first.LastAttemptAt == nil {
		t.Fatalf("first=%+v ok=%v", first, ok)
	}
	*first.LastAttemptAt = first.LastAttemptAt.Add(time.Hour)

	second, _ := store.snapshot(9)
	if second.LastAttemptAt == nil || !second.LastAttemptAt.Equal(now) {
		t.Fatalf("snapshot shared mutable timestamp: %+v", second)
	}
}

func TestServicePayoutSyncDiagnosticsReturnsAccountSnapshot(t *testing.T) {
	store := newPayoutDiagnosticStore()
	now := time.Unix(100, 0).UTC()
	store.begin(9, 3, now)
	svc := &Service{payoutDiagnostics: store}

	got, ok := svc.PayoutSyncDiagnostics(9)
	if !ok || got.AccountID != 9 || got.PendingCount != 3 {
		t.Fatalf("got=%+v ok=%v", got, ok)
	}
}
