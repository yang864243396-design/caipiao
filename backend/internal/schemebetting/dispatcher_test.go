package schemebetting

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"
)

type fakeDispatchStore struct {
	started  int
	finished []FinishDispatch
	startOK  bool
	released int
	mu       sync.Mutex
	renewed  int
	renewCh  chan struct{}
}

func (s *fakeDispatchStore) ReleaseLease(context.Context, LeasedCommand, string, time.Time) (bool, error) {
	s.released++
	return true, nil
}

type fakeDispatchLimiter struct {
	allowed bool
}

func (l fakeDispatchLimiter) Allow(context.Context, LeasedCommand, time.Time) (bool, error) {
	return l.allowed, nil
}

func (s *fakeDispatchStore) StartAttempt(context.Context, LeasedCommand, time.Time) (bool, error) {
	s.started++
	return s.startOK, nil
}

func (s *fakeDispatchStore) FinishAttempt(_ context.Context, finish FinishDispatch) (bool, error) {
	s.finished = append(s.finished, finish)
	return true, nil
}

func (s *fakeDispatchStore) RenewLease(_ context.Context, _ LeasedCommand, _, _ time.Time) (bool, error) {
	s.mu.Lock()
	s.renewed++
	ch := s.renewCh
	s.mu.Unlock()
	if ch != nil {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
	return true, nil
}

func (s *fakeDispatchStore) renewalCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.renewed
}

type fakeSingleAttemptTransport struct {
	calls  int
	result ProviderAcceptance
	err    error
}

type blockingUntilRenewedTransport struct {
	renewed <-chan struct{}
	calls   int
}

func (t *blockingUntilRenewedTransport) PlaceOnce(ctx context.Context, _ LeasedCommand) (ProviderAcceptance, error) {
	t.calls++
	select {
	case <-ctx.Done():
		return ProviderAcceptance{}, ctx.Err()
	case <-t.renewed:
		return ProviderAcceptance{OrderID: "order-after-renewal", PeriodNo: "T", Amount: 2, AccountID: 8, Currency: "CNY"}, nil
	case <-time.After(time.Second):
		return ProviderAcceptance{}, errors.New("lease was not renewed while provider call was in flight")
	}
}

type testDefinitelyNotSentError struct{ err error }

func (e testDefinitelyNotSentError) Error() string           { return e.err.Error() }
func (e testDefinitelyNotSentError) Unwrap() error           { return e.err }
func (e testDefinitelyNotSentError) DefinitelyNotSent() bool { return true }

func (t *fakeSingleAttemptTransport) PlaceOnce(context.Context, LeasedCommand) (ProviderAcceptance, error) {
	t.calls++
	return t.result, t.err
}

func TestDispatcherMakesExactlyOneCallAndCommitsAccepted(t *testing.T) {
	now := time.Date(2026, 8, 19, 12, 0, 0, 0, time.UTC)
	store := &fakeDispatchStore{startOK: true}
	transport := &fakeSingleAttemptTransport{result: ProviderAcceptance{OrderID: "order-1", PeriodNo: "T", Amount: 2, AccountID: 8, Currency: "CNY"}}
	d := Dispatcher{Store: store, Transport: transport, Now: func() time.Time { return now }}
	command := LeasedCommand{
		ID: 7, SchemeID: "scheme-1", TargetPeriod: "T", FrozenRequest: []byte(`{"issue":"T"}`), FrozenRequestHash: PayloadHash([]byte(`{"issue":"T"}`)),
		SafeDeadline: now.Add(time.Second), Lease: LeaseFence{Owner: "node-a", Token: 9, Until: now.Add(2 * time.Second)},
	}
	if err := d.Dispatch(context.Background(), command); err != nil {
		t.Fatal(err)
	}
	if transport.calls != 1 || store.started != 1 || len(store.finished) != 1 {
		t.Fatalf("calls=%d started=%d finished=%d", transport.calls, store.started, len(store.finished))
	}
	if store.finished[0].State != OutboxAccepted || store.finished[0].ProviderOrderID != "order-1" {
		t.Fatalf("finish=%+v", store.finished[0])
	}
}

func TestDispatcherAcceptsFrozenRequestAfterJSONBKeyReorder(t *testing.T) {
	now := time.Date(2026, 8, 20, 3, 31, 30, 0, time.UTC)
	written := []byte(`{"requestId":"sb-1","request":{"issueNo":"10114251404243","amount":0.2},"rule":{"version":1}}`)
	readBack := []byte(`{"rule":{"version":1},"request":{"amount":0.2,"issueNo":"10114251404243"},"requestId":"sb-1"}`)
	store := &fakeDispatchStore{startOK: true}
	transport := &fakeSingleAttemptTransport{result: ProviderAcceptance{OrderID: "order-1", PeriodNo: "10114251404243"}}
	d := Dispatcher{Store: store, Transport: transport, Now: func() time.Time { return now }}
	command := LeasedCommand{
		ID: 11, SchemeID: "scheme-1", TargetPeriod: "10114251404243",
		FrozenRequest: readBack, FrozenRequestHash: CanonicalJSONPayloadHash(written),
		SafeDeadline: now.Add(time.Second), Lease: LeaseFence{Owner: "node-a", Token: 1, Until: now.Add(2 * time.Second)},
	}

	if err := d.Dispatch(context.Background(), command); err != nil {
		t.Fatal(err)
	}
	if store.started != 1 || transport.calls != 1 {
		t.Fatalf("start=%d providerCalls=%d", store.started, transport.calls)
	}
}

func TestDispatcherDoesNotCallProviderAfterDeadlineOrStaleFence(t *testing.T) {
	now := time.Date(2026, 8, 19, 12, 0, 0, 0, time.UTC)
	for _, command := range []LeasedCommand{
		{ID: 1, SafeDeadline: now, Lease: LeaseFence{Owner: "node", Token: 1, Until: now.Add(time.Second)}},
		{ID: 2, SafeDeadline: now.Add(time.Second), Lease: LeaseFence{Owner: "node", Token: 1, Until: now}},
	} {
		store := &fakeDispatchStore{startOK: true}
		transport := &fakeSingleAttemptTransport{}
		d := Dispatcher{Store: store, Transport: transport, Now: func() time.Time { return now }}
		err := d.Dispatch(context.Background(), command)
		if command.ID == 1 && err != nil {
			t.Fatal(err)
		}
		if command.ID == 2 && !errors.Is(err, ErrStaleLease) {
			t.Fatalf("stale lease error=%v", err)
		}
		if transport.calls != 0 {
			t.Fatalf("provider called for command %d", command.ID)
		}
	}
}

func TestDispatcherMapsTransportTimeoutToUnknownWithoutRetry(t *testing.T) {
	now := time.Date(2026, 8, 19, 12, 0, 0, 0, time.UTC)
	store := &fakeDispatchStore{startOK: true}
	transport := &fakeSingleAttemptTransport{err: context.DeadlineExceeded}
	d := Dispatcher{Store: store, Transport: transport, Now: func() time.Time { return now }}
	command := LeasedCommand{ID: 8, TargetPeriod: "T", FrozenRequest: []byte(`{}`), FrozenRequestHash: PayloadHash([]byte(`{}`)), SafeDeadline: now.Add(time.Second), Lease: LeaseFence{Owner: "node", Token: 3, Until: now.Add(time.Second)}}
	if err := d.Dispatch(context.Background(), command); err != nil {
		t.Fatal(err)
	}
	if transport.calls != 1 || len(store.finished) != 1 || store.finished[0].State != OutboxSentUnknown {
		t.Fatalf("calls=%d finish=%+v", transport.calls, store.finished)
	}
	store.startOK = false
	if err := d.Dispatch(context.Background(), command); !errors.Is(err, ErrStaleLease) {
		t.Fatalf("second dispatch error=%v", err)
	}
	if transport.calls != 1 {
		t.Fatalf("provider was called again after ambiguous acceptance: %d", transport.calls)
	}
}

func TestDispatcherPreservesAmbiguousTransportErrorDetail(t *testing.T) {
	now := time.Date(2026, 8, 20, 13, 36, 18, 0, time.UTC)
	store := &fakeDispatchStore{startOK: true}
	transportErr := errors.New("provider response timeout after request write")
	transport := &fakeSingleAttemptTransport{err: transportErr}
	d := Dispatcher{Store: store, Transport: transport, Now: func() time.Time { return now }}
	command := LeasedCommand{
		ID: 14, SchemeID: "scheme-1", TargetPeriod: "T", FrozenRequest: []byte(`{}`), FrozenRequestHash: PayloadHash([]byte(`{}`)),
		SafeDeadline: now.Add(time.Second), Lease: LeaseFence{Owner: "node", Token: 6, Until: now.Add(time.Second)},
	}

	if err := d.Dispatch(context.Background(), command); err != nil {
		t.Fatal(err)
	}
	if len(store.finished) != 1 || store.finished[0].State != OutboxSentUnknown {
		t.Fatalf("finish=%+v", store.finished)
	}
	if store.finished[0].ErrorDetail != transportErr.Error() {
		t.Fatalf("error detail=%q", store.finished[0].ErrorDetail)
	}
}

func TestDispatcherRenewsLeaseWhileProviderCallIsInFlight(t *testing.T) {
	now := time.Now().UTC()
	renewed := make(chan struct{}, 1)
	store := &fakeDispatchStore{startOK: true, renewCh: renewed}
	transport := &blockingUntilRenewedTransport{renewed: renewed}
	d := Dispatcher{
		Store: store, Transport: transport, Now: func() time.Time { return now },
		LeaseDuration: 30 * time.Millisecond, LeaseHeartbeatInterval: time.Millisecond,
	}
	command := LeasedCommand{
		ID: 12, SchemeID: "scheme-1", TargetPeriod: "T", FrozenRequest: []byte(`{}`), FrozenRequestHash: PayloadHash([]byte(`{}`)),
		SafeDeadline: now.Add(time.Second), Lease: LeaseFence{Owner: "node", Token: 4, Until: now.Add(30 * time.Millisecond)},
	}

	if err := d.Dispatch(context.Background(), command); err != nil {
		t.Fatal(err)
	}
	if transport.calls != 1 || store.renewalCount() == 0 || len(store.finished) != 1 || store.finished[0].State != OutboxAccepted {
		t.Fatalf("calls=%d renewals=%d finish=%+v", transport.calls, store.renewalCount(), store.finished)
	}
}

func TestDispatcherRecognizesWrappedDefinitelyNotSentError(t *testing.T) {
	now := time.Date(2026, 8, 20, 12, 0, 0, 0, time.UTC)
	store := &fakeDispatchStore{startOK: true}
	transport := &fakeSingleAttemptTransport{err: fmt.Errorf("place bet: %w", testDefinitelyNotSentError{err: errors.New("tls handshake failed before write")})}
	d := Dispatcher{Store: store, Transport: transport, Now: func() time.Time { return now }}
	command := LeasedCommand{
		ID: 13, SchemeID: "scheme-1", TargetPeriod: "T", FrozenRequest: []byte(`{}`), FrozenRequestHash: PayloadHash([]byte(`{}`)),
		SafeDeadline: now.Add(time.Second), Lease: LeaseFence{Owner: "node", Token: 5, Until: now.Add(time.Second)},
	}

	if err := d.Dispatch(context.Background(), command); err != nil {
		t.Fatal(err)
	}
	if len(store.finished) != 1 || store.finished[0].State != OutboxRejected || store.finished[0].Reason != "definitive_pre_send_failure" {
		t.Fatalf("finish=%+v", store.finished)
	}
}

func TestDispatcherReleasesLeaseWhenRateLimitIsUnavailable(t *testing.T) {
	now := time.Date(2026, 8, 19, 12, 0, 0, 0, time.UTC)
	store := &fakeDispatchStore{startOK: true}
	transport := &fakeSingleAttemptTransport{}
	d := Dispatcher{
		Store: store, Transport: transport, Limiter: fakeDispatchLimiter{allowed: false},
		Now: func() time.Time { return now },
	}
	command := LeasedCommand{
		ID: 10, TargetPeriod: "T", FrozenRequest: []byte(`{}`), FrozenRequestHash: PayloadHash([]byte(`{}`)),
		SafeDeadline: now.Add(time.Second), Lease: LeaseFence{Owner: "node", Token: 3, Until: now.Add(time.Second)},
	}
	if err := d.Dispatch(context.Background(), command); err != nil {
		t.Fatal(err)
	}
	if store.released != 1 || store.started != 0 || transport.calls != 0 || len(store.finished) != 0 {
		t.Fatalf("released=%d started=%d calls=%d finished=%d", store.released, store.started, transport.calls, len(store.finished))
	}
}

func TestDispatcherDoesNotSendWhenAttemptStartCASLoses(t *testing.T) {
	now := time.Date(2026, 8, 19, 12, 0, 0, 0, time.UTC)
	store := &fakeDispatchStore{startOK: false}
	transport := &fakeSingleAttemptTransport{}
	d := Dispatcher{Store: store, Transport: transport, Now: func() time.Time { return now }}
	command := LeasedCommand{ID: 9, TargetPeriod: "T", FrozenRequest: []byte(`{}`), FrozenRequestHash: PayloadHash([]byte(`{}`)), SafeDeadline: now.Add(time.Second), Lease: LeaseFence{Owner: "node", Token: 3, Until: now.Add(time.Second)}}
	if err := d.Dispatch(context.Background(), command); !errors.Is(err, ErrStaleLease) {
		t.Fatalf("error=%v", err)
	}
	if transport.calls != 0 {
		t.Fatal("provider called after CAS loss")
	}
}
