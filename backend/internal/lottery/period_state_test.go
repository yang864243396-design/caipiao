package lottery

import (
	"testing"
	"time"
)

func TestPeriodCountdownSecForLottery_wallClockFallback(t *testing.T) {	// 无 DB：DrawIntervalSecForLottery 返回 0，CountdownSecForLottery 返回 0
	sec, err := PeriodCountdownSecForLottery(nil, nil, "tron_ffc_1m", time.Unix(100, 0))
	if err != nil || sec != 0 {
		t.Fatalf("got %d err=%v want 0", sec, err)
	}
}

func TestSmoothCountdown_noBackwardJump(t *testing.T) {
	code := "test_smooth_countdown"
	now := time.Date(2026, 6, 10, 12, 0, 10, 0, time.UTC)
	UpdatePeriodState(code, "100", "101", now.Add(-10*time.Second), 60)

	first := smoothCountdown(code, now, 50)
	second := smoothCountdown(code, now.Add(1*time.Second), 55) // 模拟 WS 与墙钟不一致
	if second > first {
		t.Fatalf("got backward jump %d -> %d", first, second)
	}
	if second != 49 {
		t.Fatalf("got %d want 49", second)
	}
}

func TestSmoothCountdown_newPeriodReset(t *testing.T) {
	code := "test_smooth_new_period"
	now := time.Date(2026, 6, 10, 12, 0, 59, 0, time.UTC)
	UpdatePeriodState(code, "100", "101", now, 60)
	_ = smoothCountdown(code, now, 1)

	UpdatePeriodState(code, "101", "102", now.Add(2*time.Second), 60)
	got := smoothCountdown(code, now.Add(2*time.Second), 58)
	if got != 58 {
		t.Fatalf("new period should reset countdown, got %d", got)
	}
}

func TestBetCloseSec(t *testing.T) {
	cases := map[int]int{
		3: 1, 6: 1, 15: 2, 60: 8, 180: 15, 300: 15,
	}
	for in, want := range cases {
		if got := BetCloseSec(in); got != want {
			t.Fatalf("interval %d: got %d want %d", in, got, want)
		}
	}
}

func TestPeriodCountdownWallClockMonotonic(t *testing.T) {
	interval := 60
	base := time.Date(2026, 6, 10, 12, 0, 9, 0, time.UTC)
	prev := PeriodCountdownSec(base, time.Time{}, interval)
	for i := 1; i <= 5; i++ {
		now := base.Add(time.Duration(i) * time.Second)
		cur := PeriodCountdownSec(now, time.Time{}, interval)
		if cur >= prev {
			t.Fatalf("t+%ds: %d should be < %d", i, cur, prev)
		}
		prev = cur
	}
}
