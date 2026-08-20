package schemebetting

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"
)

type fakeDispatchStore struct {
	started          int
	finished         []FinishDispatch
	finishErr        error
	recordedFailure  []string
	recordContextErr error
	startOK          bool
	startSafeWindow  time.Duration
	released         int
	mu               sync.Mutex
	renewed          int
	renewCh          chan struct{}
	renewBlock       chan struct{}
	renewLost        bool
	renewErr         error
}

func (s *fakeDispatchStore) ReleaseLease(context.Context, LeasedCommand, string, time.Time) (bool, error) {
	s.released++
	return true, nil
}

type fakeDispatchLimiter struct {
	allowed bool
}

type fakePreSendFailureHandler struct {
	calls    int
	outboxID int64
	err      error
}

func (handler *fakePreSendFailureHandler) HandlePreSendFailure(_ context.Context, outboxID int64) error {
	handler.calls++
	handler.outboxID = outboxID
	return handler.err
}

func (l fakeDispatchLimiter) Allow(context.Context, LeasedCommand, time.Time) (bool, error) {
	return l.allowed, nil
}

func (s *fakeDispatchStore) StartAttempt(context.Context, LeasedCommand, time.Duration) (AttemptStart, error) {
	s.started++
	safeWindow := s.startSafeWindow
	if safeWindow <= 0 {
		safeWindow = time.Second
	}
	return AttemptStart{Started: s.startOK, SafeWindow: safeWindow}, nil
}

func (s *fakeDispatchStore) FinishAttempt(_ context.Context, finish FinishDispatch) (bool, error) {
	s.finished = append(s.finished, finish)
	if s.finishErr != nil {
		return false, s.finishErr
	}
	return true, nil
}

func (s *fakeDispatchStore) RecordFinishAttemptFailure(ctx context.Context, _ LeasedCommand, detail string) (bool, error) {
	s.recordContextErr = ctx.Err()
	s.recordedFailure = append(s.recordedFailure, detail)
	return true, nil
}

func (s *fakeDispatchStore) RenewLease(_ context.Context, _ LeasedCommand, _ time.Duration) (bool, error) {
	s.mu.Lock()
	s.renewed++
	ch := s.renewCh
	block := s.renewBlock
	lost := s.renewLost
	err := s.renewErr
	s.mu.Unlock()
	if ch != nil {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
	if block != nil {
		<-block
	}
	if err != nil {
		return false, err
	}
	return !lost, nil
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

type deadlineObservingTransport struct {
	calls       int
	hadDeadline bool
}

type blockingProgressTransport struct {
	stage          string
	requestWritten bool
	writeKnown     bool
	started        chan struct{}
	release        chan struct{}
}

func (transport *blockingProgressTransport) PlaceOnce(ctx context.Context, _ LeasedCommand) (ProviderAcceptance, error) {
	close(transport.started)
	<-transport.release
	return ProviderAcceptance{}, ctx.Err()
}

func (transport *blockingProgressTransport) PlaceOnceWithProgress(
	ctx context.Context,
	_ LeasedCommand,
	report func(stage string, requestWritten, writeKnown bool),
) (ProviderAcceptance, error) {
	report(transport.stage, transport.requestWritten, transport.writeKnown)
	close(transport.started)
	<-transport.release
	return ProviderAcceptance{}, ctx.Err()
}

func (transport *deadlineObservingTransport) PlaceOnce(ctx context.Context, _ LeasedCommand) (ProviderAcceptance, error) {
	transport.calls++
	_, transport.hadDeadline = ctx.Deadline()
	<-ctx.Done()
	return ProviderAcceptance{}, ctx.Err()
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

func TestDispatcherPersistsFinishAttemptFailureWithIndependentContext(t *testing.T) {
	now := time.Date(2026, 8, 20, 17, 18, 50, 0, time.UTC)
	finishErr := errors.New("deadlock detected while updating scheme terminal state")
	store := &fakeDispatchStore{startOK: true, finishErr: finishErr}
	transport := &fakeSingleAttemptTransport{result: ProviderAcceptance{OrderID: "order-1", PeriodNo: "T", Amount: 2, AccountID: 8, Currency: "USDT"}}
	d := Dispatcher{Store: store, Transport: transport, Now: func() time.Time { return now }}
	command := LeasedCommand{
		ID: 639, SchemeID: "scheme-1", TargetPeriod: "T", FrozenRequest: []byte(`{}`), FrozenRequestHash: PayloadHash([]byte(`{}`)),
		SafeDeadline: now.Add(time.Second), Lease: LeaseFence{Owner: "node", Token: 13, Until: now.Add(time.Second)},
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := d.Dispatch(ctx, command)
	if !errors.Is(err, finishErr) {
		t.Fatalf("dispatch error=%v", err)
	}
	if len(store.recordedFailure) != 1 {
		t.Fatalf("recorded failures=%v", store.recordedFailure)
	}
	if store.recordContextErr != nil {
		t.Fatalf("failure recorder inherited cancelled dispatch context: %v", store.recordContextErr)
	}
	if detail := store.recordedFailure[0]; !strings.Contains(detail, "finish_attempt_failed") || !strings.Contains(detail, finishErr.Error()) {
		t.Fatalf("failure detail=%q", detail)
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

func TestDispatcherDoesNotCallProviderWhenDatabaseAttemptStartCASLoses(t *testing.T) {
	now := time.Date(2026, 8, 19, 12, 0, 0, 0, time.UTC)
	store := &fakeDispatchStore{startOK: false}
	transport := &fakeSingleAttemptTransport{}
	d := Dispatcher{Store: store, Transport: transport, Now: func() time.Time { return now }}
	command := LeasedCommand{
		ID: 2, TargetPeriod: "T", FrozenRequest: []byte(`{}`), FrozenRequestHash: PayloadHash([]byte(`{}`)),
		SafeDeadline: now.Add(-time.Second), Lease: LeaseFence{Owner: "node", Token: 1, Until: now.Add(-time.Second)},
	}
	err := d.Dispatch(context.Background(), command)
	if !errors.Is(err, ErrStaleLease) {
		t.Fatalf("stale lease error=%v", err)
	}
	if transport.calls != 0 {
		t.Fatal("provider called after database attempt-start CAS loss")
	}
}

func TestDispatcherUsesDatabaseSafeWindowInsteadOfHostClock(t *testing.T) {
	hostNow := time.Date(2026, 8, 20, 12, 0, 2, 0, time.UTC)
	store := &fakeDispatchStore{startOK: true, startSafeWindow: 20 * time.Millisecond}
	transport := &deadlineObservingTransport{}
	d := Dispatcher{Store: store, Transport: transport, Now: func() time.Time { return hostNow }}
	command := LeasedCommand{
		ID: 15, SchemeID: "scheme-1", TargetPeriod: "T", FrozenRequest: []byte(`{}`), FrozenRequestHash: PayloadHash([]byte(`{}`)),
		SafeDeadline: hostNow.Add(-time.Second), Lease: LeaseFence{Owner: "node", Token: 7, Until: hostNow.Add(-time.Second)},
	}

	if err := d.Dispatch(context.Background(), command); err != nil {
		t.Fatal(err)
	}
	if store.started != 1 || transport.calls != 1 || !transport.hadDeadline {
		t.Fatalf("started=%d calls=%d deadline=%v", store.started, transport.calls, transport.hadDeadline)
	}
	if len(store.finished) != 1 || store.finished[0].State != OutboxSentUnknown {
		t.Fatalf("finish=%+v", store.finished)
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

func TestDispatcherWatchdogReschedulesWhenBetRequestWasNotWritten(t *testing.T) {
	now := time.Now().UTC()
	store := &fakeDispatchStore{startOK: true, startSafeWindow: 15 * time.Millisecond}
	transport := &blockingProgressTransport{
		stage: "provider_account_preparation", writeKnown: true,
		started: make(chan struct{}), release: make(chan struct{}),
	}
	handler := &fakePreSendFailureHandler{}
	d := Dispatcher{Store: store, Transport: transport, PreSendFailureHandler: handler, Now: func() time.Time { return now }}
	command := LeasedCommand{
		ID: 19, SchemeID: "scheme-1", TargetPeriod: "T", FrozenRequest: []byte(`{}`), FrozenRequestHash: PayloadHash([]byte(`{}`)),
		SafeDeadline: now.Add(time.Second), Lease: LeaseFence{Owner: "node", Token: 11, Until: now.Add(time.Second)},
	}

	done := make(chan error, 1)
	go func() { done <- d.Dispatch(context.Background(), command) }()
	<-transport.started
	defer close(transport.release)
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("dispatcher remained blocked after the database safe window elapsed")
	}
	if len(store.finished) != 1 || store.finished[0].State != OutboxRejected || store.finished[0].Reason != "provider_pre_send_failed" {
		t.Fatalf("finish=%+v", store.finished)
	}
	if !strings.Contains(store.finished[0].ErrorDetail, "stage=provider_account_preparation") || !strings.Contains(store.finished[0].ErrorDetail, "request_written=false") {
		t.Fatalf("error detail=%q", store.finished[0].ErrorDetail)
	}
	if handler.calls != 1 || handler.outboxID != command.ID {
		t.Fatalf("replacement calls=%d outbox=%d", handler.calls, handler.outboxID)
	}
}

func TestDispatcherWatchdogBlocksWhenBetRequestWasWritten(t *testing.T) {
	now := time.Now().UTC()
	store := &fakeDispatchStore{startOK: true, startSafeWindow: 15 * time.Millisecond}
	transport := &blockingProgressTransport{
		stage: "provider_bet_response", requestWritten: true, writeKnown: true,
		started: make(chan struct{}), release: make(chan struct{}),
	}
	handler := &fakePreSendFailureHandler{}
	d := Dispatcher{Store: store, Transport: transport, PreSendFailureHandler: handler, Now: func() time.Time { return now }}
	command := LeasedCommand{
		ID: 20, SchemeID: "scheme-1", TargetPeriod: "T", FrozenRequest: []byte(`{}`), FrozenRequestHash: PayloadHash([]byte(`{}`)),
		SafeDeadline: now.Add(time.Second), Lease: LeaseFence{Owner: "node", Token: 12, Until: now.Add(time.Second)},
	}

	done := make(chan error, 1)
	go func() { done <- d.Dispatch(context.Background(), command) }()
	<-transport.started
	defer close(transport.release)
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("dispatcher remained blocked after a written request exceeded the database safe window")
	}
	if len(store.finished) != 1 || store.finished[0].State != OutboxSentUnknown || !store.finished[0].BlocksChain {
		t.Fatalf("finish=%+v", store.finished)
	}
	if !strings.Contains(store.finished[0].ErrorDetail, "stage=provider_bet_response") || !strings.Contains(store.finished[0].ErrorDetail, "request_written=true") {
		t.Fatalf("error detail=%q", store.finished[0].ErrorDetail)
	}
	if handler.calls != 0 {
		t.Fatalf("written request triggered replacement calls=%d", handler.calls)
	}
}

func TestDispatcherWatchdogDoesNotWaitForeverForHeartbeatShutdown(t *testing.T) {
	now := time.Now().UTC()
	heartbeatRelease := make(chan struct{})
	store := &fakeDispatchStore{
		startOK: true, startSafeWindow: 15 * time.Millisecond, renewBlock: heartbeatRelease,
	}
	transport := &blockingProgressTransport{
		stage: "provider_account_preparation", writeKnown: true,
		started: make(chan struct{}), release: make(chan struct{}),
	}
	handler := &fakePreSendFailureHandler{}
	d := Dispatcher{
		Store: store, Transport: transport, PreSendFailureHandler: handler, Now: func() time.Time { return now },
		LeaseDuration: 30 * time.Millisecond, LeaseHeartbeatInterval: time.Millisecond,
	}
	command := LeasedCommand{
		ID: 21, SchemeID: "scheme-1", TargetPeriod: "T", FrozenRequest: []byte(`{}`), FrozenRequestHash: PayloadHash([]byte(`{}`)),
		SafeDeadline: now.Add(time.Second), Lease: LeaseFence{Owner: "node", Token: 13, Until: now.Add(time.Second)},
	}

	done := make(chan error, 1)
	go func() { done <- d.Dispatch(context.Background(), command) }()
	<-transport.started
	defer close(transport.release)
	defer close(heartbeatRelease)
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("watchdog remained blocked waiting for heartbeat shutdown")
	}
	if len(store.finished) != 1 || store.finished[0].State != OutboxRejected {
		t.Fatalf("finish=%+v", store.finished)
	}
	if !strings.Contains(store.finished[0].ErrorDetail, "lease heartbeat shutdown timed out") {
		t.Fatalf("heartbeat shutdown evidence missing: %q", store.finished[0].ErrorDetail)
	}
}

func TestDispatcherCancelsProviderAndRecordsHeartbeatLoss(t *testing.T) {
	now := time.Now().UTC()
	store := &fakeDispatchStore{startOK: true, startSafeWindow: time.Second, renewLost: true}
	transport := &deadlineObservingTransport{}
	d := Dispatcher{
		Store: store, Transport: transport, Now: func() time.Time { return now },
		LeaseDuration: 30 * time.Millisecond, LeaseHeartbeatInterval: time.Millisecond,
	}
	command := LeasedCommand{
		ID: 16, SchemeID: "scheme-1", TargetPeriod: "T", FrozenRequest: []byte(`{}`), FrozenRequestHash: PayloadHash([]byte(`{}`)),
		SafeDeadline: now.Add(time.Second), Lease: LeaseFence{Owner: "node-a", Token: 8, Until: now.Add(30 * time.Millisecond)},
	}

	if err := d.Dispatch(context.Background(), command); err != nil {
		t.Fatal(err)
	}
	if transport.calls != 1 || store.renewalCount() != 1 {
		t.Fatalf("provider calls=%d renewals=%d", transport.calls, store.renewalCount())
	}
	if len(store.finished) != 1 || store.finished[0].State != OutboxSentUnknown {
		t.Fatalf("finish=%+v", store.finished)
	}
	detail := store.finished[0].ErrorDetail
	if !strings.Contains(detail, "lease heartbeat lost") || !strings.Contains(detail, "owner=node-a") || !strings.Contains(detail, "token=8") {
		t.Fatalf("heartbeat evidence missing: %q", detail)
	}
}

func TestDispatcherRecognizesWrappedDefinitelyNotSentError(t *testing.T) {
	now := time.Date(2026, 8, 20, 12, 0, 0, 0, time.UTC)
	store := &fakeDispatchStore{startOK: true}
	transport := &fakeSingleAttemptTransport{err: fmt.Errorf("place bet: %w", testDefinitelyNotSentError{err: errors.New("tls handshake failed before write")})}
	handler := &fakePreSendFailureHandler{}
	d := Dispatcher{Store: store, Transport: transport, PreSendFailureHandler: handler, Now: func() time.Time { return now }}
	command := LeasedCommand{
		ID: 13, SchemeID: "scheme-1", TargetPeriod: "T", FrozenRequest: []byte(`{}`), FrozenRequestHash: PayloadHash([]byte(`{}`)),
		SafeDeadline: now.Add(time.Second), Lease: LeaseFence{Owner: "node", Token: 5, Until: now.Add(time.Second)},
	}

	if err := d.Dispatch(context.Background(), command); err != nil {
		t.Fatal(err)
	}
	if len(store.finished) != 1 || store.finished[0].State != OutboxRejected || store.finished[0].Reason != "provider_pre_send_failed" || store.finished[0].BlocksChain {
		t.Fatalf("finish=%+v", store.finished)
	}
	if handler.calls != 1 || handler.outboxID != command.ID {
		t.Fatalf("replacement handler calls=%d outbox=%d", handler.calls, handler.outboxID)
	}
}

func TestDispatcherBlocksPreSendFailureWhenReplacementHandlerIsUnavailable(t *testing.T) {
	now := time.Date(2026, 8, 20, 12, 0, 0, 0, time.UTC)
	store := &fakeDispatchStore{startOK: true}
	transport := &fakeSingleAttemptTransport{err: testDefinitelyNotSentError{err: errors.New("connect failed before write")}}
	d := Dispatcher{Store: store, Transport: transport, Now: func() time.Time { return now }}
	command := LeasedCommand{
		ID: 17, SchemeID: "scheme-1", TargetPeriod: "T", FrozenRequest: []byte(`{}`), FrozenRequestHash: PayloadHash([]byte(`{}`)),
		SafeDeadline: now.Add(time.Second), Lease: LeaseFence{Owner: "node", Token: 9, Until: now.Add(time.Second)},
	}

	if err := d.Dispatch(context.Background(), command); err != nil {
		t.Fatal(err)
	}
	if len(store.finished) != 1 || !store.finished[0].BlocksChain {
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
	if err := d.Dispatch(context.Background(), command); !errors.Is(err, ErrDispatchDeferred) {
		t.Fatalf("error=%v want deferred", err)
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
