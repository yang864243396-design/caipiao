package schemebettingdispatch

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/guajibet"
	"caipiao/backend/internal/schemebetting"
)

type fakeSinglePlacer struct {
	calls  int
	result guajibet.Result
	err    error
}

type fakeAcceptedRecovery struct {
	calls int
	err   error
}

type fakePeriodVerifier struct {
	calls   int
	period  string
	closeAt time.Time
	err     error
}

func (verifier *fakePeriodVerifier) VerifyOpenPeriodForMember(context.Context, string, string) (string, time.Time, error) {
	verifier.calls++
	return verifier.period, verifier.closeAt, verifier.err
}

func (recovery *fakeAcceptedRecovery) RecoverAccepted(context.Context, int32) error {
	recovery.calls++
	return recovery.err
}

func (p *fakeSinglePlacer) Enabled() bool { return true }
func (p *fakeSinglePlacer) PlaceRealBetOnce(context.Context, string, guajibet.Request) (guajibet.Result, error) {
	p.calls++
	return p.result, p.err
}

func TestTransportUsesSingleAttemptPlacerAndPreservesProviderPeriod(t *testing.T) {
	now := time.Date(2026, 8, 19, 12, 0, 0, 0, time.UTC)
	placer := &fakeSinglePlacer{result: guajibet.Result{ThirdPartyBetID: "order-8", Periods: "provider-T"}}
	verifier := &fakePeriodVerifier{period: "provider-T", closeAt: now.Add(3 * time.Second)}
	frozen, err := json.Marshal(FrozenGuajiRequest{
		RequestID: "request-1", MemberAccount: "member-1", Request: guajibet.Request{LotteryCode: "lottery", IssueNo: "provider-T", Amount: 12},
	})
	if err != nil {
		t.Fatal(err)
	}
	result, err := (Transport{Placer: placer, PeriodVerifier: verifier, Now: func() time.Time { return now }}).PlaceOnce(context.Background(), schemebetting.LeasedCommand{
		TargetPeriod: "provider-T", FrozenRequest: frozen, CloseAt: now.Add(3 * time.Second), SafeDeadline: now.Add(1500 * time.Millisecond),
	})
	if err != nil {
		t.Fatal(err)
	}
	if verifier.calls != 1 || placer.calls != 1 || result.OrderID != "order-8" || result.PeriodNo != "provider-T" {
		t.Fatalf("calls=%d result=%+v", placer.calls, result)
	}
}

func TestTransportMarksOnlyDefinitiveRejectAsNotSent(t *testing.T) {
	now := time.Date(2026, 8, 19, 12, 0, 0, 0, time.UTC)
	placer := &fakeSinglePlacer{err: guajibet.ErrInsufficient}
	frozen, _ := json.Marshal(FrozenGuajiRequest{RequestID: "request-1", MemberAccount: "member-1", Request: guajibet.Request{LotteryCode: "lottery", IssueNo: "T"}})
	transport := Transport{Placer: placer, PeriodVerifier: &fakePeriodVerifier{period: "T", closeAt: now.Add(3 * time.Second)}, Now: func() time.Time { return now }}
	command := schemebetting.LeasedCommand{TargetPeriod: "T", FrozenRequest: frozen, CloseAt: now.Add(3 * time.Second), SafeDeadline: now.Add(time.Second)}
	_, err := transport.PlaceOnce(context.Background(), command)
	var safe interface{ DefinitelyNotSent() bool }
	if !errors.As(err, &safe) || !safe.DefinitelyNotSent() {
		t.Fatalf("definitive rejection not classified: %v", err)
	}

	placer.err = guajibet.ErrPlaceRejected
	_, err = transport.PlaceOnce(context.Background(), command)
	if errors.As(err, &safe) {
		t.Fatalf("ambiguous placement error classified as safe: %v", err)
	}
}

func TestTransportRejectsChangedOrUnsafeProviderPeriodBeforePlacement(t *testing.T) {
	now := time.Date(2026, 8, 19, 12, 0, 0, 0, time.UTC)
	frozen, _ := json.Marshal(FrozenGuajiRequest{
		RequestID: "request-1", MemberAccount: "member-1", Request: guajibet.Request{LotteryCode: "lottery", IssueNo: "T"},
	})
	command := schemebetting.LeasedCommand{
		TargetPeriod: "T", FrozenRequest: frozen, CloseAt: now.Add(3 * time.Second), SafeDeadline: now.Add(time.Second),
	}
	for _, verifier := range []*fakePeriodVerifier{
		{period: "T+1", closeAt: now.Add(3 * time.Second)},
		{period: "T", closeAt: now.Add(1500 * time.Millisecond)},
	} {
		placer := &fakeSinglePlacer{}
		_, err := (Transport{Placer: placer, PeriodVerifier: verifier, Now: func() time.Time { return now }}).PlaceOnce(context.Background(), command)
		var safe interface{ DefinitelyNotSent() bool }
		if !errors.As(err, &safe) || !safe.DefinitelyNotSent() {
			t.Fatalf("pre-send validation error=%v", err)
		}
		if placer.calls != 0 {
			t.Fatal("provider placement called after failed period validation")
		}
	}
}

func TestRuntimeRequiresFormalModeAndExplicitShards(t *testing.T) {
	q := &sqlcdb.Queries{}
	if _, err := New(q, &fakeSinglePlacer{}, Config{Mode: "shadow", Owner: "node", Shards: []int32{0}}); err == nil {
		t.Fatal("shadow mode must not start real dispatcher")
	}
	if _, err := New(q, &fakeSinglePlacer{}, Config{Mode: "gray", Owner: "node"}); err == nil {
		t.Fatal("implicit all-shard ownership must be rejected")
	}
}

func TestRuntimeConfiguresDispatchLeaseHeartbeat(t *testing.T) {
	leaseDuration := 9 * time.Second
	runtime, err := New(&sqlcdb.Queries{}, &fakeSinglePlacer{}, Config{
		Mode: "gray", Owner: "node", LotteryCodes: []string{"tron_ffc_6s"}, Shards: []int32{0}, LeaseDuration: leaseDuration,
	})
	if err != nil {
		t.Fatal(err)
	}
	if runtime.dispatcher.LeaseDuration != leaseDuration || runtime.dispatcher.LeaseHeartbeatInterval != 3*time.Second {
		t.Fatalf("lease=%v heartbeat=%v", runtime.dispatcher.LeaseDuration, runtime.dispatcher.LeaseHeartbeatInterval)
	}
}

func TestRunAcceptanceRecoveryAfterSuccessfulAbandonedDispatchSweep(t *testing.T) {
	recovery := &fakeAcceptedRecovery{}
	if err := runAcceptanceRecovery(context.Background(), nil, recovery, 32); err != nil {
		t.Fatal(err)
	}
	if recovery.calls != 1 {
		t.Fatalf("recovery calls=%d want 1", recovery.calls)
	}
}

func TestRunAcceptanceRecoveryStopsOnAbandonedDispatchSweepFailure(t *testing.T) {
	recovery := &fakeAcceptedRecovery{}
	sweepErr := errors.New("sweep failed")
	if err := runAcceptanceRecovery(context.Background(), sweepErr, recovery, 32); !errors.Is(err, sweepErr) {
		t.Fatalf("error=%v want sweep failure", err)
	}
	if recovery.calls != 0 {
		t.Fatalf("recovery calls=%d want 0", recovery.calls)
	}
}
