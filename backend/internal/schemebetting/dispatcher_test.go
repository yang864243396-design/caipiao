package schemebetting

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeDispatchStore struct {
	started  int
	finished []FinishDispatch
	startOK  bool
	released int
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

type fakeSingleAttemptTransport struct {
	calls  int
	result ProviderAcceptance
	err    error
}

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
