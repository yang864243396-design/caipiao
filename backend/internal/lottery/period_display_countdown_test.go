package lottery

import (
	"testing"
	"time"
)

func TestBuildPeriodsDisplayCountdown_openPeriod(t *testing.T) {
	code := "tron_ffc_1m_unified_cd"
	closeAt := time.Now().UTC().Add(45 * time.Second)
	UpdatePeriodsScheduleFullWithDuration(
		code,
		"1014017600305",
		"1014017600305",
		closeAt,
		closeAt,
		60,
		"2026-07-02 12:00:45",
		closeAt.Add(-60*time.Second),
	)

	cd := BuildPeriodsDisplayCountdown(code, time.Now().UTC())
	if cd.Period != "1014017600305" {
		t.Fatalf("period=%q", cd.Period)
	}
	if cd.Sec <= 0 || cd.Sec > 60 {
		t.Fatalf("sec=%d want 1..60", cd.Sec)
	}
	if cd.EndTimeRaw == "" {
		t.Fatal("expected end time raw")
	}
	if cd.WaitingLabel != "" {
		t.Fatalf("waiting=%q", cd.WaitingLabel)
	}
}

func TestBuildPeriodsDisplayCountdown_closedPeriod(t *testing.T) {
	code := "tron_ffc_1m_unified_cd_closed"
	closeAt := time.Now().UTC().Add(-5 * time.Second)
	UpdatePeriodsScheduleFullWithDuration(
		code,
		"1014017600306",
		"1014017600306",
		closeAt,
		closeAt,
		60,
		"2026-07-02 12:00:00",
		closeAt.Add(-60*time.Second),
	)

	cd := BuildPeriodsDisplayCountdown(code, time.Now().UTC())
	if cd.Sec != 0 || cd.WaitingLabel != PeriodsDisplayWaitingLabel {
		t.Fatalf("sec=%d label=%q", cd.Sec, cd.WaitingLabel)
	}
	if cd.Period != "1014017600306" {
		t.Fatalf("period=%q want closed cache period", cd.Period)
	}
	if cd.EndTimeRaw != "2026-07-02 12:00:00" {
		t.Fatalf("endTime=%q want closed cache end_time", cd.EndTimeRaw)
	}
	if cd.CloseAtRFC3339 == "" {
		t.Fatal("expected countdownCloseAt for closed cache")
	}
}
