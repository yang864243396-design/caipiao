package lottery

import (
	"testing"
	"time"
)

func TestPeriodCloseAtFromAnchor_andLatency(t *testing.T) {
	code := "tron_ffc_1m_diag"
	period := "1014049800465"
	closeAt := time.Date(2026, 7, 9, 3, 29, 53, 0, time.UTC)
	UpdatePeriodsScheduleFullWithDuration(
		code, period, period, closeAt, closeAt, 60, "2026-07-09 11:29:53", closeAt.Add(-60*time.Second),
	)
	got, ok := PeriodCloseAtFromAnchor(code, period)
	if !ok || !got.Equal(closeAt) {
		t.Fatalf("anchor closeAt=%v ok=%v want %v", got, ok, closeAt)
	}
	ingested := closeAt.Add(47 * time.Second)
	latencySec := int(ingested.Sub(got).Round(time.Second).Seconds())
	if latencySec != 47 {
		t.Fatalf("latencySec=%d want 47", latencySec)
	}
}
