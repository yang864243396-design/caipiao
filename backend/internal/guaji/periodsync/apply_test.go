package periodsync

import (
	"testing"
	"time"

	"caipiao/backend/internal/guaji"
	"caipiao/backend/internal/lottery"
)

func TestApplyPeriodsListToCache_startSkipAndOpen(t *testing.T) {
	now := time.Date(2026, 6, 10, 15, 0, 0, 0, time.UTC)
	periods := []guaji.LottPeriod{
		{Period: "skip_me", StartTime: "2026-06-10 14:58:59", EndTime: "2026-06-10 14:59:59"},
		{Period: "open_me", StartTime: "2026-06-10 15:00:00", EndTime: "2026-06-10 15:01:00"},
	}
	applyPeriodsListToCache("hash_ffc_1m", periods, now)

	ps, ok := lottery.PeriodsScheduleFor("hash_ffc_1m")
	if !ok {
		t.Fatal("schedule missing")
	}
	if ps.PeriodDurationSec != 60 {
		t.Fatalf("duration=%d want 60", ps.PeriodDurationSec)
	}
	if ps.StartSkipPeriod != "open_me" {
		t.Fatalf("skip=%q", ps.StartSkipPeriod)
	}
	if ps.StartSkipCloseAt.IsZero() {
		t.Fatal("skip close at missing")
	}
	if ps.CurrentPeriod != "open_me" {
		t.Fatalf("current=%q", ps.CurrentPeriod)
	}
	skip, ok := lottery.StartSkipPeriodFromCache("hash_ffc_1m")
	if !ok || skip != "open_me" {
		t.Fatalf("skip cache=%q ok=%v", skip, ok)
	}
}

func TestApplyPeriodsListToCache_keepsCurrentPeriodWhileOpen(t *testing.T) {
	code := "stick_period_test"
	now := time.Date(2026, 6, 10, 15, 0, 30, 0, time.UTC)
	first := []guaji.LottPeriod{
		{Period: "100", StartTime: "2026-06-10 15:00:00", EndTime: "2026-06-10 15:01:00"},
		{Period: "101", StartTime: "2026-06-10 15:01:00", EndTime: "2026-06-10 15:02:00"},
	}
	applyPeriodsListToCache(code, first, now)
	ps, ok := lottery.PeriodsScheduleFor(code)
	if !ok || ps.CurrentPeriod != "100" {
		t.Fatalf("first current=%q ok=%v", ps.CurrentPeriod, ok)
	}

	// PickOpenLottPeriod 可能选 101（start 未到但 end-only 兜底）；缓存仍应保持 100
	second := []guaji.LottPeriod{
		{Period: "100", StartTime: "2026-06-10 15:00:00", EndTime: "2026-06-10 15:01:00"},
		{Period: "101", StartTime: "2026-06-10 15:01:00", EndTime: "2026-06-10 15:02:00"},
	}
	applyPeriodsListToCache(code, second, now)
	ps, ok = lottery.PeriodsScheduleFor(code)
	if !ok || ps.CurrentPeriod != "100" {
		t.Fatalf("after stick current=%q want 100", ps.CurrentPeriod)
	}
}

func TestApplyPeriodsListToCache_samePeriodNoCloseAtDriftAfterClear(t *testing.T) {
	code := "tron_ffc_1m"
	period := "1014046700026"
	now := time.Date(2026, 7, 8, 11, 50, 46, 0, time.UTC)
	first := []guaji.LottPeriod{
		{Period: period, StartTime: "2026-07-08 19:49:47", EndTime: "2026-07-08 19:50:47"},
	}
	applyPeriodsListToCache(code, first, now)
	ps, ok := lottery.PeriodsScheduleFor(code)
	if !ok || !ps.CloseAt.Equal(time.Date(2026, 7, 8, 11, 50, 47, 0, time.UTC)) {
		t.Fatalf("first closeAt=%v ok=%v", ps.CloseAt, ok)
	}

	lottery.ClearPeriodsSchedule(code)
	second := []guaji.LottPeriod{
		{Period: period, StartTime: "2026-07-08 19:49:47", EndTime: "2026-07-08 19:50:51"},
	}
	applyPeriodsListToCache(code, second, now)

	ps, ok = lottery.PeriodsScheduleFor(code)
	if !ok {
		t.Fatal("schedule missing after second apply")
	}
	wantClose := time.Date(2026, 7, 8, 11, 50, 47, 0, time.UTC)
	if !ps.CloseAt.Equal(wantClose) {
		t.Fatalf("closeAt=%v want %v", ps.CloseAt, wantClose)
	}
	if ps.CloseEndTimeRaw != "2026-07-08 19:50:47" {
		t.Fatalf("raw=%q", ps.CloseEndTimeRaw)
	}
}

func TestApplyPeriodsListToCache_skipsAnchoredClosedPeriod(t *testing.T) {
	code := "tron_ffc_1m"
	period042 := "1014046800042"
	period043 := "1014046800043"
	close042 := time.Date(2026, 7, 8, 12, 6, 36, 0, time.UTC)
	lottery.UpdatePeriodsScheduleFullWithDuration(
		code, period042, period042, close042, close042, 60, "2026-07-08 20:06:36", close042.Add(-60*time.Second),
	)
	lottery.ClearPeriodsSchedule(code)

	now := time.Date(2026, 7, 8, 12, 6, 40, 0, time.UTC)
	periods := []guaji.LottPeriod{
		{Period: period042, StartTime: "2026-07-08 20:05:36", EndTime: "2026-07-08 20:06:45"},
		{Period: period043, StartTime: "2026-07-08 20:06:36", EndTime: "2026-07-08 20:07:36"},
	}
	applyPeriodsListToCache(code, periods, now)

	ps, ok := lottery.PeriodsScheduleFor(code)
	if !ok {
		t.Fatal("schedule missing")
	}
	if ps.CurrentPeriod != period043 {
		t.Fatalf("period=%q want %q", ps.CurrentPeriod, period043)
	}
	if ps.CloseEndTimeRaw != "2026-07-08 20:07:36" {
		t.Fatalf("raw=%q", ps.CloseEndTimeRaw)
	}
	cd := lottery.BuildPeriodsDisplayCountdown(code, now)
	if cd.EndTimeRaw == "" {
		t.Fatal("expected countdownEndTime in display")
	}
	if cd.Period != period043 {
		t.Fatalf("display period=%q", cd.Period)
	}
}

func TestApplyPeriodsListToCache_keepsClosedSnapshotDuringGap(t *testing.T) {
	code := "tron_ffc_1m_gap_keep"
	period042 := "1014046800042"
	close042 := time.Date(2026, 7, 8, 12, 6, 36, 0, time.UTC)
	lottery.UpdatePeriodsScheduleFullWithDuration(
		code, period042, period042, close042, close042, 60, "2026-07-08 20:06:36", close042.Add(-60*time.Second),
	)

	// 封盘后、下一期尚未开盘：第三方列表可能只剩已封盘期
	now := time.Date(2026, 7, 8, 12, 6, 40, 0, time.UTC)
	periods := []guaji.LottPeriod{
		{Period: period042, StartTime: "2026-07-08 20:05:36", EndTime: "2026-07-08 20:06:36"},
	}
	applyPeriodsListToCache(code, periods, now)

	ps, ok := lottery.PeriodsScheduleFor(code)
	if !ok {
		t.Fatal("closed snapshot should be kept during gap")
	}
	if ps.CurrentPeriod != period042 {
		t.Fatalf("period=%q", ps.CurrentPeriod)
	}
	if ps.CloseEndTimeRaw != "2026-07-08 20:06:36" {
		t.Fatalf("raw=%q", ps.CloseEndTimeRaw)
	}

	cd := lottery.BuildPeriodsDisplayCountdown(code, now)
	if cd.Sec != 0 || cd.WaitingLabel != lottery.PeriodsDisplayWaitingLabel {
		t.Fatalf("sec=%d label=%q", cd.Sec, cd.WaitingLabel)
	}
	if cd.EndTimeRaw != "2026-07-08 20:06:36" {
		t.Fatalf("countdownEndTime=%q", cd.EndTimeRaw)
	}
	if cd.Period != period042 {
		t.Fatalf("countdownPeriod=%q", cd.Period)
	}
}
