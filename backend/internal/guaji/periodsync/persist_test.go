package periodsync

import (
	"testing"
	"time"

	"caipiao/backend/internal/guaji"
)

func TestBuildProviderPeriodSnapshotsCanonicalAndStable(t *testing.T) {
	now := time.Date(2026, 8, 19, 4, 0, 0, 0, time.UTC)
	periods := []guaji.LottPeriod{
		{Period: " 900-A ", StartTime: "2026-08-19 12:00:00", EndTime: "2026-08-19 12:00:03"},
		{Period: "", StartTime: "2026-08-19 12:00:03", EndTime: "2026-08-19 12:00:06"},
	}

	a := buildProviderPeriodSnapshots("hash_ffc_3s", periods, now)
	b := buildProviderPeriodSnapshots("hash_ffc_3s", periods, now)
	if len(a) != 1 {
		t.Fatalf("snapshots=%d want=1", len(a))
	}
	if a[0].PeriodNo != "900-A" || a[0].SnapshotHash == "" {
		t.Fatalf("snapshot=%+v", a[0])
	}
	if a[0].SnapshotHash != b[0].SnapshotHash {
		t.Fatal("same provider fact must produce a stable hash")
	}
	if !a[0].ObservedAt.Equal(now) {
		t.Fatalf("observedAt=%s want=%s", a[0].ObservedAt, now)
	}
}

func TestBuildProviderPeriodSnapshotsTreatsNextStartAsCurrentBetClose(t *testing.T) {
	now := time.Date(2026, 8, 20, 2, 41, 2, 0, time.UTC)
	snapshots := buildProviderPeriodSnapshots("tron_ffc_6s", []guaji.LottPeriod{{
		Period: "10114251203740", StartTime: "2026-08-20 10:41:08", EndTime: "2026-08-20 10:41:14",
	}}, now)
	if len(snapshots) != 1 {
		t.Fatalf("snapshots=%d", len(snapshots))
	}
	if !snapshots[0].OpenAt.IsZero() {
		t.Fatalf("future provider start must represent an already-open current bet window: %s", snapshots[0].OpenAt)
	}
	wantClose := now.Add(6 * time.Second)
	if !snapshots[0].CloseAt.Equal(wantClose) {
		t.Fatalf("close=%s want next provider start %s", snapshots[0].CloseAt, wantClose)
	}
}

func TestBuildProviderPeriodSnapshotsHashChangesWithCloseTime(t *testing.T) {
	// Use an already-open provider period: before its start, the effective
	// current-window close is start_time and end_time is intentionally ignored.
	now := time.Date(2026, 8, 19, 12, 0, 1, 0, time.UTC)
	a := buildProviderPeriodSnapshots("hash_ffc_3s", []guaji.LottPeriod{{Period: "900-A", StartTime: "2026-08-19 12:00:00", EndTime: "2026-08-19 12:00:03"}}, now)
	b := buildProviderPeriodSnapshots("hash_ffc_3s", []guaji.LottPeriod{{Period: "900-A", StartTime: "2026-08-19 12:00:00", EndTime: "2026-08-19 12:00:04"}}, now)
	if len(a) != 1 || len(b) != 1 || a[0].SnapshotHash == b[0].SnapshotHash {
		t.Fatal("provider correction must append a distinct immutable snapshot")
	}
}
