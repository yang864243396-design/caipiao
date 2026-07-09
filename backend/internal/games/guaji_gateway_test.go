package games

import (
	"context"
	"testing"

	"caipiao/backend/internal/db/sqlcdb"
)

type stubBetPlacer struct {
	enabled bool
	calls   int
	res     GuajiBetResult
	err     error
}

func (s *stubBetPlacer) Enabled() bool { return s.enabled }

func (s *stubBetPlacer) PlaceRealBet(_ context.Context, _ string, _ GuajiBetRequest) (GuajiBetResult, error) {
	s.calls++
	return s.res, s.err
}

func (s *stubBetPlacer) MirrorBetDebitLedger(_ context.Context, _ *sqlcdb.Queries, _ int64, _ string, _ float64, _ int64, _ string) error {
	return nil
}

func TestGuajiRealEnabledToggle(t *testing.T) {
	svc := &Service{}
	if svc.guajiRealEnabled() {
		t.Fatal("nil placer 应为未启用")
	}

	svc.SetGuajiBetPlacer(&stubBetPlacer{enabled: false})
	if svc.guajiRealEnabled() {
		t.Fatal("placer.Enabled=false 应为未启用（real 走本地降级）")
	}

	svc.SetGuajiBetPlacer(&stubBetPlacer{enabled: true})
	if !svc.guajiRealEnabled() {
		t.Fatal("placer.Enabled=true 应启用第三方下单")
	}

	svc.SetGuajiBetPlacer(nil)
	if svc.guajiRealEnabled() {
		t.Fatal("清空 placer 后应回退未启用")
	}
}
