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

func TestBuildProviderPeriodSnapshotsHashChangesWithCloseTime(t *testing.T) {
	now := time.Date(2026, 8, 19, 4, 0, 0, 0, time.UTC)
	a := buildProviderPeriodSnapshots("hash_ffc_3s", []guaji.LottPeriod{{Period: "900-A", StartTime: "2026-08-19 12:00:00", EndTime: "2026-08-19 12:00:03"}}, now)
	b := buildProviderPeriodSnapshots("hash_ffc_3s", []guaji.LottPeriod{{Period: "900-A", StartTime: "2026-08-19 12:00:00", EndTime: "2026-08-19 12:00:04"}}, now)
	if len(a) != 1 || len(b) != 1 || a[0].SnapshotHash == b[0].SnapshotHash {
		t.Fatal("provider correction must append a distinct immutable snapshot")
	}
}
