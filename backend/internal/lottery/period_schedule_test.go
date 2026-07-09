package lottery

import (
	"testing"
	"time"
)

func TestPeriodsDisplayCloseAt_allowsStaleCache(t *testing.T) {
	code := "display_close_stale_test"
	closeAt := time.Now().UTC().Add(40 * time.Second)
	UpdatePeriodsSchedule(code, "P1", closeAt)
	// 模拟缓存已写入超过展示新鲜度阈值
	if ps, ok := PeriodsScheduleFor(code); ok {
		ps.UpdatedAt = time.Now().UTC().Add(-30 * time.Second)
		periodsSchedule.Store(code, ps)
	}

	if _, ok := PeriodsBetCloseAt(code, time.Now()); ok {
		t.Fatal("bet closeAt should require fresh cache")
	}
	ca, ok := PeriodsDisplayCloseAt(code, time.Now())
	if !ok {
		t.Fatal("display closeAt should work with stale cache when closeAt still future")
	}
	sec, ok := PeriodsDisplayCountdownSec(code, time.Now())
	if !ok || sec < 35 || sec > 41 {
		t.Fatalf("display countdown=%d ok=%v want ~40", sec, ok)
	}
	if ca.IsZero() {
		t.Fatal("closeAt missing")
	}
}

func TestPeriodsBetCloseAt_rejectsPastEndTime(t *testing.T) {
	code := "bet_close_past_test"
	closeAt := time.Now().UTC().Add(-5 * time.Second)
	UpdatePeriodsSchedule(code, "P1", closeAt)

	if _, ok := PeriodsBetCloseAt(code, time.Now()); ok {
		t.Fatal("expected stale closeAt to be rejected")
	}
}

func TestPeriodsScheduleNeedsRefresh_whenCloseAtPassed(t *testing.T) {
	code := "needs_refresh_past_test"
	closeAt := time.Now().UTC().Add(-2 * time.Second)
	UpdatePeriodsSchedule(code, "P1", closeAt)

	if !PeriodsScheduleNeedsRefresh(code, time.Now()) {
		t.Fatal("expected refresh when closeAt passed")
	}
}

func TestPeriodsScheduleNeedsRefresh_whenFreshFuture(t *testing.T) {
	code := "needs_refresh_future_test"
	closeAt := time.Now().UTC().Add(45 * time.Second)
	UpdatePeriodsSchedule(code, "P2", closeAt)

	if PeriodsScheduleNeedsRefresh(code, time.Now()) {
		t.Fatal("expected no refresh for fresh future closeAt")
	}
}

func TestGuajiBetWindowOpen_staleCloseAtAllowsRetry(t *testing.T) {
	code := "bet_window_stale_test"
	closeAt := time.Now().UTC().Add(-3 * time.Second)
	UpdatePeriodsSchedule(code, "P1", closeAt)

	if !GuajiBetWindowOpen(code, time.Now()) {
		t.Fatal("stale closeAt should allow worker retry path")
	}
	if _, ok := StrictOpenIssueForGuajiBet(code); ok {
		t.Fatal("strict open issue should remain false until cache refreshed")
	}
}

func TestClearPeriodsSchedule(t *testing.T) {
	code := "clear_schedule_test"
	UpdatePeriodsSchedule(code, "P1", time.Now().UTC().Add(time.Minute))
	ClearPeriodsSchedule(code)
	if _, ok := PeriodsScheduleFor(code); ok {
		t.Fatal("expected schedule cleared")
	}
}

func TestUpdatePeriodsScheduleFullWithDuration_samePeriodNoCloseAtDrift(t *testing.T) {
	code := "same_period_drift_test"
	period := "9001"
	start := time.Date(2026, 6, 21, 6, 0, 0, 0, time.UTC)
	close1 := time.Date(2026, 6, 21, 6, 1, 0, 0, time.UTC)
	UpdatePeriodsScheduleFullWithDuration(code, period, period, close1, close1, 60, "2026-06-21 06:01:00", start)

	close2 := close1.Add(18 * time.Second)
	UpdatePeriodsScheduleFullWithDuration(code, period, period, close2, close2, 60, "2026-06-21 06:01:18", start)

	ps, ok := PeriodsScheduleFor(code)
	if !ok {
		t.Fatal("schedule missing")
	}
	if !ps.CloseAt.Equal(close1) {
		t.Fatalf("closeAt=%v want %v (same period must not drift forward)", ps.CloseAt, close1)
	}
	if ps.CloseEndTimeRaw != "2026-06-21 06:01:00" {
		t.Fatalf("raw=%q", ps.CloseEndTimeRaw)
	}
}

func TestPeriodCloseAnchor_survivesCacheClear(t *testing.T) {
	code := "anchor_after_clear_test"
	period := "1014046700026"
	start := time.Date(2026, 7, 8, 11, 49, 47, 0, time.UTC)
	close1 := time.Date(2026, 7, 8, 11, 50, 47, 0, time.UTC)
	UpdatePeriodsScheduleFullWithDuration(
		code, period, period, close1, close1, 60, "2026-07-08 19:50:47", start,
	)

	ClearPeriodsSchedule(code)

	close2 := close1.Add(4 * time.Second)
	UpdatePeriodsScheduleFullWithDuration(
		code, period, period, close2, close2, 60, "2026-07-08 19:50:51", start,
	)

	ps, ok := PeriodsScheduleFor(code)
	if !ok {
		t.Fatal("schedule missing after re-write")
	}
	if !ps.CloseAt.Equal(close1) {
		t.Fatalf("closeAt=%v want %v (anchor must block drift after cache clear)", ps.CloseAt, close1)
	}
	if ps.CloseEndTimeRaw != "2026-07-08 19:50:47" {
		t.Fatalf("raw=%q", ps.CloseEndTimeRaw)
	}
}

func TestPeriodCloseAnchor_allowsNewPeriod(t *testing.T) {
	code := "anchor_new_period_test"
	start := time.Date(2026, 7, 8, 11, 49, 0, 0, time.UTC)
	close1 := time.Date(2026, 7, 8, 11, 50, 0, 0, time.UTC)
	UpdatePeriodsScheduleFullWithDuration(
		code, "1014046700026", "1014046700026", close1, close1, 60, "2026-07-08 19:50:00", start,
	)

	close2 := time.Date(2026, 7, 8, 11, 51, 0, 0, time.UTC)
	UpdatePeriodsScheduleFullWithDuration(
		code, "1014046700027", "1014046700027", close2, close2, 60, "2026-07-08 19:51:00", start.Add(time.Minute),
	)

	ps, ok := PeriodsScheduleFor(code)
	if !ok {
		t.Fatal("schedule missing")
	}
	if ps.CurrentPeriod != "1014046700027" {
		t.Fatalf("period=%q", ps.CurrentPeriod)
	}
	if !ps.CloseAt.Equal(close2) {
		t.Fatalf("closeAt=%v want %v", ps.CloseAt, close2)
	}
}

func TestTryUpdatePeriodsScheduleFullWithDurationAt_rejectsAnchoredPastClose(t *testing.T) {
	code := "try_update_past_test"
	period := "9001"
	start := time.Date(2026, 7, 8, 12, 0, 0, 0, time.UTC)
	close1 := time.Date(2026, 7, 8, 12, 1, 0, 0, time.UTC)
	UpdatePeriodsScheduleFullWithDuration(code, period, period, close1, close1, 60, "2026-07-08 20:01:00", start)

	now := close1.Add(5 * time.Second)
	close2 := close1.Add(10 * time.Second)
	ok := TryUpdatePeriodsScheduleFullWithDurationAt(
		code, period, period, close2, close2, 60, "2026-07-08 20:01:10", start, now,
	)
	if ok {
		t.Fatal("expected reject when anchor clamps to past closeAt")
	}
}
