package lottery

import (
	"strings"
	"time"
)

// PeriodRuntimeDiagnostics is a read-only view of the in-process period and
// draw websocket caches used by the scheme worker.
type PeriodRuntimeDiagnostics struct {
	LotteryCode           string    `json:"lotteryCode"`
	HasPeriodsSnapshot    bool      `json:"hasPeriodsSnapshot"`
	PeriodsFresh          bool      `json:"periodsFresh"`
	BetWindowOpen         bool      `json:"betWindowOpen"`
	CurrentOpenPeriod     string    `json:"currentOpenPeriod,omitempty"`
	StartSkipPeriod       string    `json:"startSkipPeriod,omitempty"`
	CloseAt               time.Time `json:"closeAt,omitempty"`
	UpdatedAt             time.Time `json:"updatedAt,omitempty"`
	PeriodDurationSec     int       `json:"periodDurationSec,omitempty"`
	WebsocketCurrentIssue string    `json:"websocketCurrentIssue,omitempty"`
	WebsocketNextIssue    string    `json:"websocketNextIssue,omitempty"`
	WebsocketUpdatedAt    time.Time `json:"websocketUpdatedAt,omitempty"`
}

// InspectPeriodRuntime reports cache state only. It never refreshes periods
// and never opens a third-party connection.
func InspectPeriodRuntime(lotteryCode string, now time.Time) PeriodRuntimeDiagnostics {
	lotteryCode = strings.TrimSpace(lotteryCode)
	if now.IsZero() {
		now = time.Now().UTC()
	} else {
		now = now.UTC()
	}
	out := PeriodRuntimeDiagnostics{LotteryCode: lotteryCode}
	if lotteryCode == "" {
		return out
	}
	if ps, ok := PeriodsScheduleFor(lotteryCode); ok {
		out.HasPeriodsSnapshot = true
		out.PeriodsFresh = PeriodsScheduleFresh(lotteryCode, periodsScheduleMaxAge, now)
		out.CurrentOpenPeriod = strings.TrimSpace(ps.CurrentPeriod)
		out.StartSkipPeriod = strings.TrimSpace(ps.StartSkipPeriod)
		out.CloseAt = ps.CloseAt.UTC()
		out.UpdatedAt = ps.UpdatedAt.UTC()
		out.PeriodDurationSec = ps.PeriodDurationSec
	}
	out.BetWindowOpen = GuajiBetWindowOpen(lotteryCode, now)
	if ws, ok := PeriodStateFor(lotteryCode); ok {
		out.WebsocketCurrentIssue = strings.TrimSpace(ws.CurrentIssue)
		out.WebsocketNextIssue = strings.TrimSpace(ws.NextIssue)
		out.WebsocketUpdatedAt = ws.UpdatedAt.UTC()
	}
	return out
}
