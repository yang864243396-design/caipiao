package schemebettingdispatch

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/guaji"
	"caipiao/backend/internal/guajibet"
	"caipiao/backend/internal/schemebetting"
)

type fakeSinglePlacer struct {
	calls  int
	result guajibet.Result
	err    error
}

type progressReportingPlacer struct {
	progress guaji.RequestProgress
	result   guajibet.Result
}

func (placer *progressReportingPlacer) Enabled() bool { return true }
func (placer *progressReportingPlacer) PlaceRealBetOnce(ctx context.Context, _ string, _ guajibet.Request) (guajibet.Result, error) {
	guaji.ReportRequestProgress(ctx, placer.progress)
	return placer.result, nil
}

type fakeDefinitelyNotSentError struct{ err error }

func (err fakeDefinitelyNotSentError) Error() string           { return err.err.Error() }
func (err fakeDefinitelyNotSentError) Unwrap() error           { return err.err }
func (err fakeDefinitelyNotSentError) DefinitelyNotSent() bool { return true }

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

func transportTestContext(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	t.Cleanup(cancel)
	return ctx
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
	result, err := (Transport{Placer: placer, PeriodVerifier: verifier, Now: func() time.Time { return now }}).PlaceOnce(transportTestContext(t), schemebetting.LeasedCommand{
		TargetPeriod: "provider-T", FrozenRequest: frozen, CloseAt: now.Add(3 * time.Second), SafeDeadline: now.Add(1500 * time.Millisecond),
	})
	if err != nil {
		t.Fatal(err)
	}
	if verifier.calls != 1 || placer.calls != 1 || result.OrderID != "order-8" || result.PeriodNo != "provider-T" {
		t.Fatalf("calls=%d result=%+v", placer.calls, result)
	}
}

func TestTransportReportsOnlyBetRequestWriteProgress(t *testing.T) {
	now := time.Date(2026, 8, 20, 16, 45, 41, 0, time.UTC)
	placer := &progressReportingPlacer{
		progress: guaji.RequestProgress{Operation: "POST /api/web_bets/lott", Phase: "response", RequestWritten: true},
		result:   guajibet.Result{ThirdPartyBetID: "order-progress", Periods: "T"},
	}
	frozen, _ := json.Marshal(FrozenGuajiRequest{
		RequestID: "request-progress", MemberAccount: "member-1", Request: guajibet.Request{LotteryCode: "lottery", IssueNo: "T"},
	})
	command := schemebetting.LeasedCommand{TargetPeriod: "T", FrozenRequest: frozen, CloseAt: now.Add(3 * time.Second), SafeDeadline: now.Add(time.Second)}
	var stages []string
	var written, known bool

	result, err := (Transport{
		Placer: placer, PeriodVerifier: &fakePeriodVerifier{period: "T", closeAt: now.Add(3 * time.Second)}, Now: func() time.Time { return now },
	}).PlaceOnceWithProgress(transportTestContext(t), command, func(stage string, requestWritten, writeKnown bool) {
		stages = append(stages, stage)
		written, known = requestWritten, writeKnown
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.OrderID != "order-progress" || len(stages) < 2 {
		t.Fatalf("result=%+v stages=%v", result, stages)
	}
	if stages[len(stages)-1] != "provider_bet_response" || !written || !known {
		t.Fatalf("last stage=%q written=%v known=%v all=%v", stages[len(stages)-1], written, known, stages)
	}
}

func TestTransportMarksOnlyDefinitiveRejectAsNotSent(t *testing.T) {
	now := time.Date(2026, 8, 19, 12, 0, 0, 0, time.UTC)
	placer := &fakeSinglePlacer{err: guajibet.ErrInsufficient}
	frozen, _ := json.Marshal(FrozenGuajiRequest{RequestID: "request-1", MemberAccount: "member-1", Request: guajibet.Request{LotteryCode: "lottery", IssueNo: "T"}})
	transport := Transport{Placer: placer, PeriodVerifier: &fakePeriodVerifier{period: "T", closeAt: now.Add(3 * time.Second)}, Now: func() time.Time { return now }}
	command := schemebetting.LeasedCommand{TargetPeriod: "T", FrozenRequest: frozen, CloseAt: now.Add(3 * time.Second), SafeDeadline: now.Add(time.Second)}
	_, err := transport.PlaceOnce(transportTestContext(t), command)
	var safe interface{ DefinitelyNotSent() bool }
	if !errors.As(err, &safe) || !safe.DefinitelyNotSent() {
		t.Fatalf("definitive rejection not classified: %v", err)
	}

	placer.err = guajibet.ErrPlaceRejected
	_, err = transport.PlaceOnce(transportTestContext(t), command)
	if errors.As(err, &safe) {
		t.Fatalf("ambiguous placement error classified as safe: %v", err)
	}
}

func TestTransportQualifiesPlacementFailurePhaseWithoutBreakingErrorIdentity(t *testing.T) {
	now := time.Date(2026, 8, 20, 13, 36, 18, 0, time.UTC)
	placementErr := errors.New("response timeout after request write")
	placer := &fakeSinglePlacer{err: placementErr}
	frozen, _ := json.Marshal(FrozenGuajiRequest{RequestID: "request-1", MemberAccount: "member-1", Request: guajibet.Request{LotteryCode: "lottery", IssueNo: "T"}})
	transport := Transport{Placer: placer, PeriodVerifier: &fakePeriodVerifier{period: "T", closeAt: now.Add(3 * time.Second)}, Now: func() time.Time { return now }}
	command := schemebetting.LeasedCommand{TargetPeriod: "T", FrozenRequest: frozen, CloseAt: now.Add(3 * time.Second), SafeDeadline: now.Add(time.Second)}

	_, err := transport.PlaceOnce(transportTestContext(t), command)
	if !errors.Is(err, placementErr) {
		t.Fatalf("error identity lost: %v", err)
	}
	if err == nil || !strings.Contains(err.Error(), "provider placement") {
		t.Fatalf("placement phase missing: %v", err)
	}
}

func TestTransportPreservesWrappedDefinitelyNotSentPlacementEvidence(t *testing.T) {
	now := time.Date(2026, 8, 20, 13, 36, 18, 0, time.UTC)
	transportErr := errors.New("tls handshake timeout")
	placer := &fakeSinglePlacer{err: fmt.Errorf("account place: %w", fakeDefinitelyNotSentError{err: transportErr})}
	frozen, _ := json.Marshal(FrozenGuajiRequest{RequestID: "request-1", MemberAccount: "member-1", Request: guajibet.Request{LotteryCode: "lottery", IssueNo: "T"}})
	transport := Transport{Placer: placer, PeriodVerifier: &fakePeriodVerifier{period: "T", closeAt: now.Add(3 * time.Second)}, Now: func() time.Time { return now }}
	command := schemebetting.LeasedCommand{TargetPeriod: "T", FrozenRequest: frozen, CloseAt: now.Add(3 * time.Second), SafeDeadline: now.Add(time.Second)}

	_, err := transport.PlaceOnce(transportTestContext(t), command)
	if !errors.Is(err, transportErr) {
		t.Fatalf("transport error identity lost: %v", err)
	}
	if err == nil || !strings.Contains(err.Error(), "provider placement failed") {
		t.Fatalf("placement phase missing: %v", err)
	}
	var marker interface{ DefinitelyNotSent() bool }
	if !errors.As(err, &marker) || !marker.DefinitelyNotSent() {
		t.Fatalf("pre-send evidence lost: %T %v", err, err)
	}
}

func TestTransportQualifiesPeriodVerificationFailureWithDuration(t *testing.T) {
	now := time.Date(2026, 8, 20, 13, 36, 18, 0, time.UTC)
	verifyErr := errors.New("period refresh timeout")
	placer := &fakeSinglePlacer{}
	frozen, _ := json.Marshal(FrozenGuajiRequest{RequestID: "request-1", MemberAccount: "member-1", Request: guajibet.Request{LotteryCode: "lottery", IssueNo: "T"}})
	transport := Transport{Placer: placer, PeriodVerifier: &fakePeriodVerifier{err: verifyErr}, Now: func() time.Time { return now }}
	command := schemebetting.LeasedCommand{TargetPeriod: "T", FrozenRequest: frozen, CloseAt: now.Add(3 * time.Second), SafeDeadline: now.Add(time.Second)}

	_, err := transport.PlaceOnce(transportTestContext(t), command)
	if !errors.Is(err, verifyErr) {
		t.Fatalf("error identity lost: %v", err)
	}
	if err == nil || !strings.Contains(err.Error(), "provider period verification failed (verify_ms=") {
		t.Fatalf("verification phase duration missing: %v", err)
	}
	var safe interface{ DefinitelyNotSent() bool }
	if !errors.As(err, &safe) || !safe.DefinitelyNotSent() {
		t.Fatalf("verification failure not classified pre-send: %v", err)
	}
	if placer.calls != 0 {
		t.Fatalf("placement calls=%d", placer.calls)
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
		_, err := (Transport{Placer: placer, PeriodVerifier: verifier, Now: func() time.Time { return now }}).PlaceOnce(transportTestContext(t), command)
		var safe interface{ DefinitelyNotSent() bool }
		if !errors.As(err, &safe) || !safe.DefinitelyNotSent() {
			t.Fatalf("pre-send validation error=%v", err)
		}
		if placer.calls != 0 {
			t.Fatal("provider placement called after failed period validation")
		}
	}
}

func TestTransportUsesDispatcherDeadlineInsteadOfHostWallClock(t *testing.T) {
	providerNow := time.Date(2026, 8, 20, 12, 0, 0, 0, time.UTC)
	hostNow := providerNow.Add(3 * time.Second)
	closeAt := providerNow.Add(3 * time.Second)
	safeDeadline := providerNow.Add(time.Second)
	placer := &fakeSinglePlacer{result: guajibet.Result{ThirdPartyBetID: "order-9", Periods: "T"}}
	verifier := &fakePeriodVerifier{period: "T", closeAt: closeAt}
	frozen, _ := json.Marshal(FrozenGuajiRequest{RequestID: "request-1", MemberAccount: "member-1", Request: guajibet.Request{LotteryCode: "lottery", IssueNo: "T"}})
	command := schemebetting.LeasedCommand{TargetPeriod: "T", FrozenRequest: frozen, CloseAt: closeAt, SafeDeadline: safeDeadline}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	result, err := (Transport{Placer: placer, PeriodVerifier: verifier, Now: func() time.Time { return hostNow }}).PlaceOnce(ctx, command)
	if err != nil {
		t.Fatal(err)
	}
	if verifier.calls != 1 || placer.calls != 1 || result.OrderID != "order-9" {
		t.Fatalf("verify=%d place=%d result=%+v", verifier.calls, placer.calls, result)
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
