package server

import (
	"testing"
	"time"

	"caipiao/backend/internal/guaji"
	"caipiao/backend/internal/handler"
)

type recordingRuntimeDiagnosticsSetter struct {
	provider handler.SchemeRuntimeDiagnosticsProvider
}

func (s *recordingRuntimeDiagnosticsSetter) SetSchemeRuntimeDiagnostics(provider handler.SchemeRuntimeDiagnosticsProvider) {
	s.provider = provider
}

type fixedRuntimeDiagnosticsProvider struct{}

func (fixedRuntimeDiagnosticsProvider) DrawWSHealth() guaji.DrawWSHealthSnapshot {
	return guaji.DrawWSHealthSnapshot{Connected: true}
}

func (fixedRuntimeDiagnosticsProvider) PeriodBoundaryHealth(string, time.Time) guaji.LotteryBoundaryHealthSnapshot {
	return guaji.LotteryBoundaryHealthSnapshot{}
}

func TestWireSchemeRuntimeDiagnosticsAttachesReadOnlyProvider(t *testing.T) {
	setter := &recordingRuntimeDiagnosticsSetter{}
	provider := fixedRuntimeDiagnosticsProvider{}

	wireSchemeRuntimeDiagnostics(setter, provider)

	if setter.provider == nil || !setter.provider.DrawWSHealth().Connected {
		t.Fatalf("runtime diagnostics provider was not attached: %#v", setter.provider)
	}
}
