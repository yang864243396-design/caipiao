package handler

import (
	"testing"

	"caipiao/backend/internal/lottery"
)

func TestSchemeRuntimeBlockReason(t *testing.T) {
	ready := lottery.PeriodRuntimeDiagnostics{HasPeriodsSnapshot: true, PeriodsFresh: true, BetWindowOpen: true}
	cases := []struct {
		name        string
		status      string
		runtime     lottery.PeriodRuntimeDiagnostics
		needsDraw   bool
		drawPresent bool
		want        string
	}{
		{name: "not running", status: "paused", runtime: ready, want: "scheme_not_running"},
		{name: "periods missing", status: "running", runtime: lottery.PeriodRuntimeDiagnostics{}, want: "periods_snapshot_missing"},
		{name: "periods stale", status: "running", runtime: lottery.PeriodRuntimeDiagnostics{HasPeriodsSnapshot: true}, want: "periods_snapshot_stale"},
		{name: "bet window closed", status: "running", runtime: lottery.PeriodRuntimeDiagnostics{HasPeriodsSnapshot: true, PeriodsFresh: true}, want: "bet_window_closed"},
		{name: "previous draw missing", status: "running", runtime: ready, needsDraw: true, want: "previous_draw_missing"},
		{name: "ready", status: "running", runtime: ready, needsDraw: true, drawPresent: true, want: "no_local_preflight_block"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := schemeRuntimeBlockReason(tc.status, tc.runtime, tc.needsDraw, tc.drawPresent); got != tc.want {
				t.Fatalf("got %q want %q", got, tc.want)
			}
		})
	}
}

func TestAcceptedPendingBlocksCurrentPeriod(t *testing.T) {
	cases := []struct {
		name              string
		currentOpenPeriod string
		thirdPartyPeriod  string
		want              bool
	}{
		{name: "historical accepted pending", currentOpenPeriod: "101", thirdPartyPeriod: "100", want: false},
		{name: "same target period", currentOpenPeriod: "101", thirdPartyPeriod: "101", want: true},
		{name: "future target period", currentOpenPeriod: "101", thirdPartyPeriod: "102", want: true},
		{name: "missing third party period", currentOpenPeriod: "101", thirdPartyPeriod: "", want: true},
		{name: "missing current period", currentOpenPeriod: "", thirdPartyPeriod: "100", want: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := acceptedPendingBlocksCurrentPeriod(tc.currentOpenPeriod, tc.thirdPartyPeriod); got != tc.want {
				t.Fatalf("got %t want %t", got, tc.want)
			}
		})
	}
}
