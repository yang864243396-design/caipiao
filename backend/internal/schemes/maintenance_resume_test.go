package schemes

import (
	"testing"
	"time"

	"caipiao/backend/internal/db/sqlcdb"
)

func TestEvaluateSchemeScheduleGate_inWindow(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 6, 24, 12, 0, 0, 0, time.Local)
	cfg := []byte(`{"startTime":"2026-06-24 08:00:00","endTime":"2026-06-24 20:00:00"}`)

	if got := evaluateSchemeScheduleGate(cfg, now); got != schemeScheduleOK {
		t.Fatalf("got %v want OK", got)
	}
}

func TestEvaluateSchemeScheduleGate_pastEnd(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 6, 24, 21, 0, 0, 0, time.Local)
	cfg := []byte(`{"startTime":"2026-06-24 08:00:00","endTime":"2026-06-24 20:00:00"}`)

	if got := evaluateSchemeScheduleGate(cfg, now); got != schemeSchedulePastEnd {
		t.Fatalf("got %v want past end", got)
	}
}

func TestCanResumeAfterMaintenance_requiresMaintenanceReason(t *testing.T) {
	t.Parallel()
	inst := sqlcdb.SchemeInstance{
		Status:       "pending",
		StatusReason: StatusReasonBetFailed,
	}
	if canResumeAfterMaintenance(nil, nil, inst, nil, time.Now()) {
		t.Fatal("bet_failed pending must not resume as maintenance")
	}
}

func TestMaintenanceResumePastStartInWindowPassesScheduleGate(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 6, 24, 12, 0, 0, 0, time.Local)
	cfg := []byte(`{"startTime":"2026-06-24 08:00:00","endTime":"2026-06-24 20:00:00"}`)

	if err := validateSchemeStartTimeAfterNow(cfg, now); err == nil {
		t.Fatal("normal start path must reject past startTime")
	}
	if got := evaluateSchemeScheduleGate(cfg, now); got != schemeScheduleOK {
		t.Fatalf("maintenance resume uses schedule gate in window, got %v", got)
	}
}
