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

func TestPayoutDiagnosticStoreFailureAfterSuccessClearsAttemptCounters(t *testing.T) {
	store := newPayoutDiagnosticStore()
	t1 := time.Unix(100, 0).UTC()
	store.begin(9, 4, t1)
	store.succeed(9, 50, 2, 2, t1.Add(time.Second))

	t2 := t1.Add(10 * time.Second)
	store.begin(9, 3, t2)
	store.fail(9, errors.New("historical provider timeout"), t2.Add(time.Second))

	failed, ok := store.snapshot(9)
	if !ok || failed.ProviderListCount != 0 || failed.SettledCount != 0 || failed.ProviderUnsettledCount != 0 {
		t.Fatalf("failed=%+v ok=%v, want zero counters for the failed attempt", failed, ok)
	}
}

func TestPayoutDiagnosticStoreEvictsOldestAttemptAtCapacity(t *testing.T) {
	store := newPayoutDiagnosticStore()
	base := time.Unix(100, 0).UTC()
	for accountID := int64(1); accountID <= payoutDiagnosticCapacity; accountID++ {
		store.begin(accountID, 1, base.Add(time.Duration(accountID)*time.Second))
	}
	store.begin(payoutDiagnosticCapacity+1, 1, base.Add(time.Duration(payoutDiagnosticCapacity+1)*time.Second))

	if _, ok := store.snapshot(1); ok {
		t.Fatal("oldest account snapshot was not evicted")
	}
	if _, ok := store.snapshot(2); !ok {
		t.Fatal("newer account snapshot was unexpectedly evicted")
	}
	if _, ok := store.snapshot(payoutDiagnosticCapacity + 1); !ok {
		t.Fatal("new account snapshot was not stored")
	}
}

func TestPayoutDiagnosticStoreEvictionBreaksTimestampTiesByAccountID(t *testing.T) {
	store := newPayoutDiagnosticStore()
	at := time.Unix(100, 0).UTC()
	for accountID := int64(payoutDiagnosticCapacity); accountID >= 1; accountID-- {
		store.begin(accountID, 1, at)
	}
	store.begin(payoutDiagnosticCapacity+1, 1, at.Add(time.Second))

	if _, ok := store.snapshot(1); ok {
		t.Fatal("lowest account ID was not evicted for equal attempt timestamps")
	}
	if _, ok := store.snapshot(2); !ok {
		t.Fatal("higher account ID was unexpectedly evicted for equal attempt timestamps")
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
