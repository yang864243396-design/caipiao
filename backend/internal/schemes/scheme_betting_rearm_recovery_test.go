package schemes

import (
	"context"
	"errors"
	"testing"

	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/schemeeventbus"
)

type fakeAutomaticRearmSource struct {
	candidate  sqlcdb.AutomaticRearmCandidate
	candidates []sqlcdb.AutomaticRearmCandidate
	found      bool
	err        error
}

func (source *fakeAutomaticRearmSource) GetAutomaticRearmCandidate(_ context.Context, outboxID int64) (sqlcdb.AutomaticRearmCandidate, bool, error) {
	for _, candidate := range source.candidates {
		if candidate.OutboxID == outboxID {
			return candidate, true, source.err
		}
	}
	return source.candidate, source.found, source.err
}

func (source *fakeAutomaticRearmSource) ListAutomaticRearmCandidates(
	context.Context, []string, []int32, int32,
) ([]sqlcdb.AutomaticRearmCandidate, error) {
	return source.candidates, source.err
}

func TestAutomaticRearmConsumesOnlyAuthoritativeSafeBlockedCandidate(t *testing.T) {
	source := &fakeAutomaticRearmSource{
		found: true,
		candidate: sqlcdb.AutomaticRearmCandidate{
			OutboxID: 7, RequestID: "request-7", SchemeID: "scheme-7", LotteryCode: "tron_ffc_6s", ShardNo: 2,
			State: "rejected", Reason: "provider_pre_send_failed",
		},
	}
	enabler := &fakeFormalEventEnabler{}
	event := schemeeventbus.BetReconcile{
		OutboxID: 7, RequestID: "request-7", ShardNo: 2,
		State: "rejected", Reason: "provider_pre_send_failed",
	}

	if err := handleAutomaticRearmEvent(context.Background(), event, source, enabler); err != nil {
		t.Fatal(err)
	}
	if enabler.rearmCalls != 1 || enabler.scheme != "scheme-7" {
		t.Fatalf("unexpected automatic rearm call: %+v", enabler)
	}
}

func TestAutomaticRearmIgnoresAmbiguousOrStaleEvents(t *testing.T) {
	for _, event := range []schemeeventbus.BetReconcile{
		{OutboxID: 7, RequestID: "request-7", ShardNo: 2, State: "sent_unknown", Reason: "provider_acceptance_pending_reconciliation"},
		{OutboxID: 7, RequestID: "request-7", ShardNo: 2, State: "accepted_wrong_period", Reason: "accepted_wrong_period"},
	} {
		source := &fakeAutomaticRearmSource{found: true, candidate: sqlcdb.AutomaticRearmCandidate{OutboxID: 7, SchemeID: "scheme-7"}}
		enabler := &fakeFormalEventEnabler{}
		if err := handleAutomaticRearmEvent(context.Background(), event, source, enabler); err != nil {
			t.Fatal(err)
		}
		if enabler.rearmCalls != 0 {
			t.Fatalf("ambiguous event triggered rearm: %+v", event)
		}
	}

	source := &fakeAutomaticRearmSource{found: false}
	enabler := &fakeFormalEventEnabler{}
	if err := handleAutomaticRearmEvent(context.Background(), schemeeventbus.BetReconcile{
		OutboxID: 7, RequestID: "request-7", ShardNo: 2,
		State: "rejected", Reason: "provider_pre_send_failed",
	}, source, enabler); err != nil {
		t.Fatal(err)
	}
	if enabler.rearmCalls != 0 {
		t.Fatal("stale event must be idempotent")
	}
}

func TestAutomaticRearmPropagatesTransientFailureForJetStreamRetry(t *testing.T) {
	source := &fakeAutomaticRearmSource{
		found: true,
		candidate: sqlcdb.AutomaticRearmCandidate{
			OutboxID: 7, RequestID: "request-7", SchemeID: "scheme-7", LotteryCode: "tron_ffc_6s", ShardNo: 2,
			State: "rejected", Reason: "provider_pre_send_failed",
		},
	}
	want := errors.New("no_fresh_provider_target")
	enabler := &fakeFormalEventEnabler{err: want}
	err := handleAutomaticRearmEvent(context.Background(), schemeeventbus.BetReconcile{
		OutboxID: 7, RequestID: "request-7", ShardNo: 2,
		State: "rejected", Reason: "provider_pre_send_failed",
	}, source, enabler)
	if !errors.Is(err, want) {
		t.Fatalf("err=%v want=%v", err, want)
	}
}

func TestAutomaticRearmRecoveryBatchIsBoundedAndUsesSameSafeHandler(t *testing.T) {
	source := &fakeAutomaticRearmSource{candidates: []sqlcdb.AutomaticRearmCandidate{
		{OutboxID: 7, RequestID: "request-7", SchemeID: "scheme-7", LotteryCode: "tron_ffc_6s", ShardNo: 2, State: "rejected", Reason: "provider_pre_send_failed"},
		{OutboxID: 8, RequestID: "request-8", SchemeID: "scheme-8", LotteryCode: "tron_ffc_6s", ShardNo: 2, State: "expired", Reason: "safe_deadline_elapsed"},
	}}
	enabler := &fakeFormalEventEnabler{}

	processed, err := runAutomaticRearmBatch(
		context.Background(), source, enabler, []string{"tron_ffc_6s"}, []int32{2}, 32, 1,
	)
	if err != nil {
		t.Fatal(err)
	}
	if processed != 2 || enabler.rearmCalls != 2 {
		t.Fatalf("processed=%d calls=%d", processed, enabler.rearmCalls)
	}
}
