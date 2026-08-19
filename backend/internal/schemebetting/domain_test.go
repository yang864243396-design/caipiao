package schemebetting

import (
	"testing"
	"time"
)

func TestSelectTargetPeriodUsesProviderSnapshotWithoutArithmetic(t *testing.T) {
	now := time.Date(2026, 8, 19, 12, 0, 3, 0, time.UTC)
	snapshots := []PeriodSnapshot{
		{PeriodNo: "block-A9", OpenAt: now.Add(-time.Second), CloseAt: now.Add(8 * time.Second), ObservedAt: now.Add(-100 * time.Millisecond)},
		{PeriodNo: "block-B1", OpenAt: now.Add(8 * time.Second), CloseAt: now.Add(11 * time.Second), ObservedAt: now.Add(-100 * time.Millisecond)},
	}

	got, ok := SelectTargetPeriod(snapshots, "block-A8", now, 2*time.Second)
	if !ok {
		t.Fatal("expected provider target")
	}
	if got.PeriodNo != "block-A9" {
		t.Fatalf("target=%q want provider period block-A9", got.PeriodNo)
	}
}

func TestSelectTargetPeriodRejectsStaleOrClosedSnapshot(t *testing.T) {
	now := time.Date(2026, 8, 19, 12, 0, 3, 0, time.UTC)
	snapshots := []PeriodSnapshot{
		{PeriodNo: "closed", OpenAt: now.Add(-5 * time.Second), CloseAt: now, ObservedAt: now},
		{PeriodNo: "stale", OpenAt: now.Add(-time.Second), CloseAt: now.Add(10 * time.Second), ObservedAt: now.Add(-3 * time.Second)},
	}
	if got, ok := SelectTargetPeriod(snapshots, "source", now, 2*time.Second); ok {
		t.Fatalf("unexpected target %+v", got)
	}
}

func TestSafeDeadlineReservesAllBudgets(t *testing.T) {
	closeAt := time.Date(2026, 8, 19, 12, 0, 10, 0, time.UTC)
	budget := DeadlineBudget{
		ClockSkew: 200 * time.Millisecond,
		Queue:     300 * time.Millisecond,
		Dispatch:  250 * time.Millisecond,
		Network:   1200 * time.Millisecond,
	}
	want := closeAt.Add(-1950 * time.Millisecond)
	if got := SafeDeadline(closeAt, budget); !got.Equal(want) {
		t.Fatalf("deadline=%s want=%s", got, want)
	}
	if IsSafeToCreate(want, want) {
		t.Fatal("deadline equality must not be considered safe")
	}
}

func TestOutboxStateTransitionsRejectBlindUnknownRetry(t *testing.T) {
	allowed := [][2]OutboxState{
		{OutboxPending, OutboxLeased},
		{OutboxLeased, OutboxSentUnknown},
		{OutboxLeased, OutboxAccepted},
		{OutboxLeased, OutboxRejected},
		{OutboxSentUnknown, OutboxAccepted},
		{OutboxSentUnknown, OutboxRejected},
	}
	for _, pair := range allowed {
		if !CanTransition(pair[0], pair[1]) {
			t.Fatalf("expected transition %s -> %s", pair[0], pair[1])
		}
	}
	if CanTransition(OutboxSentUnknown, OutboxPending) || CanTransition(OutboxSentUnknown, OutboxLeased) {
		t.Fatal("sent_unknown must reconcile instead of being blindly retried")
	}
}

func TestCommandIdentityAndShardAreDeterministic(t *testing.T) {
	a := CommandIdentity("scheme-9", "source-A", "target-Z", 17)
	b := CommandIdentity("scheme-9", "source-A", "target-Z", 17)
	if a == "" || a != b {
		t.Fatalf("request ids differ: %q %q", a, b)
	}
	if a == CommandIdentity("scheme-9", "source-A", "target-Z", 18) {
		t.Fatal("state version must participate in request identity")
	}
	if ShardForScheme("scheme-9", 64) != ShardForScheme("scheme-9", 64) {
		t.Fatal("scheme shard must be stable")
	}
}
