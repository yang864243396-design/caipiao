package lottery

import (
	"testing"
	"time"
)

func TestParseDrawIntervalSec(t *testing.T) {
	cases := map[string]int{
		"1m": 60, "3m": 180, "5m": 300,
		"3s": 3, "15s": 15,
		"jisu": 60,
		"": 0, "x": 0,
	}
	for in, want := range cases {
		if got := ParseDrawIntervalSec(in); got != want {
			t.Fatalf("%q: got %d want %d", in, got, want)
		}
	}
}

func TestPeriodCountdownSec_withLastDraw(t *testing.T) {
	drawn := time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)
	now := drawn.Add(25 * time.Second)
	if got := PeriodCountdownSec(now, drawn, 60); got != 35 {
		t.Fatalf("got %d want 35", got)
	}
}

func TestPeriodCountdownSec_overdueModulo(t *testing.T) {
	drawn := time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)
	now := drawn.Add(65 * time.Second)
	if got := PeriodCountdownSec(now, drawn, 60); got != 55 {
		t.Fatalf("got %d want 55", got)
	}
}

func TestPeriodCountdownBundleForLottery_zeroWithoutDB(t *testing.T) {
	interval, countdown, err := PeriodCountdownBundleForLottery(nil, nil, "tron_ffc_1m", time.Unix(100, 0))
	if err != nil || interval != 0 || countdown != 0 {
		t.Fatalf("interval=%d countdown=%d err=%v", interval, countdown, err)
	}
	sec, err := PeriodCountdownSecForLottery(nil, nil, "tron_ffc_1m", time.Unix(100, 0))
	if err != nil || sec != 0 {
		t.Fatalf("sec=%d err=%v", sec, err)
	}
}

func TestPeriodCountdownBundleForLottery_usesGuajiCloseAt(t *testing.T) {
	code := "bundle_guaji_close_test"
	closeAt := time.Now().UTC().Add(47 * time.Second)
	UpdatePeriodsScheduleFullWithDuration(code, "P1", "P1", closeAt, closeAt, 60, "", time.Time{})

	interval, countdown, err := PeriodCountdownBundleForLottery(nil, nil, code, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if interval != 60 || countdown < 45 || countdown > 47 {
		t.Fatalf("interval=%d countdown=%d", interval, countdown)
	}
}
