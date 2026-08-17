package lottery

import (
	"testing"
	"time"
)

func TestInspectPeriodRuntimeReportsPeriodsAndWebsocketState(t *testing.T) {
	const code = "runtime-diag-test"
	ClearPeriodsSchedule(code)
	t.Cleanup(func() { ClearPeriodsSchedule(code) })

	now := time.Now().UTC()
	UpdatePeriodsScheduleFullWithDuration(code, "85342626", "85342625", now.Add(-2*time.Second), now.Add(8*time.Second), 3, "", now.Add(-5*time.Second))
	UpdatePeriodState(code, "85342625", "85342626", now.Add(-time.Second), 3)

	got := InspectPeriodRuntime(code, now)
	if !got.HasPeriodsSnapshot || !got.PeriodsFresh || !got.BetWindowOpen {
		t.Fatalf("expected fresh open periods snapshot, got %+v", got)
	}
	if got.CurrentOpenPeriod != "85342626" || got.StartSkipPeriod != "85342625" {
		t.Fatalf("unexpected periods fields: %+v", got)
	}
	if got.WebsocketCurrentIssue != "85342625" || got.WebsocketNextIssue != "85342626" {
		t.Fatalf("unexpected websocket fields: %+v", got)
	}
}
