package lottery

import (
	"testing"
	"time"
)

func TestBetCountdownSecFromSchedule_capsBeforeStart(t *testing.T) {
	start := time.Date(2026, 6, 21, 5, 30, 39, 0, time.UTC)
	closeAt := time.Date(2026, 6, 21, 5, 31, 39, 0, time.UTC)
	ps := PeriodsSchedule{
		CloseAt:           closeAt,
		OpenStartAt:       start,
		PeriodDurationSec: 60,
	}
	// 距 start 还有 29s，距 end 还有 89s；展示应封顶为 60s
	now := time.Date(2026, 6, 21, 5, 30, 10, 0, time.UTC)
	sec, ok := BetCountdownSecFromSchedule(ps, now)
	if !ok || sec != 60 {
		t.Fatalf("sec=%d ok=%v want 60", sec, ok)
	}
}

func TestBetCountdownSecFromSchedule_afterStart(t *testing.T) {
	start := time.Date(2026, 6, 21, 5, 30, 39, 0, time.UTC)
	closeAt := time.Date(2026, 6, 21, 5, 31, 39, 0, time.UTC)
	ps := PeriodsSchedule{
		CloseAt:           closeAt,
		OpenStartAt:       start,
		PeriodDurationSec: 60,
	}
	now := time.Date(2026, 6, 21, 5, 31, 10, 0, time.UTC)
	sec, ok := BetCountdownSecFromSchedule(ps, now)
	if !ok || sec != 29 {
		t.Fatalf("sec=%d ok=%v want 29", sec, ok)
	}
}
