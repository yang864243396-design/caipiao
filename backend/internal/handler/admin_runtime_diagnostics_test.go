package handler

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"caipiao/backend/internal/guaji/accountsvc"
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
			if got := schemeRuntimeBlockReason(runtimeBlockInput{PreflightReason: schemeRuntimePreflightBlockReason(tc.status, tc.runtime, tc.needsDraw, tc.drawPresent)}); got != tc.want {
				t.Fatalf("got %q want %q", got, tc.want)
			}
		})
	}
}

func TestSchemeRuntimeBlockReasonPrefersTerminalGap(t *testing.T) {
	input := runtimeBlockInput{
		ChainBlockReason: "missed_contiguous_period",
		DrawWSStale:      true,
		AwaitingTarget:   true,
	}
	if got := schemeRuntimeBlockReason(input); got != "missed_contiguous_period" {
		t.Fatalf("got %q want missed_contiguous_period", got)
	}
}

func TestSchemeRuntimeBlockReasonReportsWaitingBeforeDeadline(t *testing.T) {
	input := runtimeBlockInput{AwaitingTarget: true, DeadlineExpired: false, DrawWSStale: false}
	if got := schemeRuntimeBlockReason(input); got != "next_period_unavailable" {
		t.Fatalf("got %q want next_period_unavailable", got)
	}
}

func TestSchemeRuntimeBlockReasonDoesNotInventTerminalResultFromExpiredWait(t *testing.T) {
	input := runtimeBlockInput{AwaitingTarget: true, DeadlineExpired: true, PreflightReason: "no_local_preflight_block"}
	if got := schemeRuntimeBlockReason(input); got != "no_local_preflight_block" {
		t.Fatalf("got %q want no_local_preflight_block", got)
	}
}

func TestRuntimeDiagnosticSanitizerRedactsProviderBodies(t *testing.T) {
	got := sanitizeDiagnosticString(`provider body={"reason":"too_late","trace":"raw-body"} token=raw-token`)
	for _, leaked := range []string{"too_late", "raw-body", "raw-token", "{"} {
		if strings.Contains(got, leaked) {
			t.Fatalf("provider diagnostic leaked %q in %q", leaked, got)
		}
	}
}

func TestRuntimeDiagnosticSanitizerRedactsMultilineJSONBodiesBeforeSecrets(t *testing.T) {
	input := `{
  "message": "keep this diagnostic",
  "providerResponse": {
    "status": 422,
    "body": "first provider line\npassword=inside-password\nlater-provider-line",
    "token": "inside-token"
  },
  "rawBody": "raw first line\nraw-later-line",
  "password": "outer-password"
}`

	got := sanitizeDiagnosticString(input)
	var decoded map[string]any
	if err := json.Unmarshal([]byte(got), &decoded); err != nil {
		t.Fatalf("sanitized diagnostic is not valid JSON: %v; value=%q", err, got)
	}
	if decoded["message"] != "keep this diagnostic" {
		t.Fatalf("non-sensitive diagnostic lost: %#v", decoded)
	}
	for _, leaked := range []string{
		"first provider line", "inside-password", "later-provider-line", "inside-token",
		"raw first line", "raw-later-line", "outer-password",
	} {
		if strings.Contains(got, leaked) {
			t.Fatalf("multiline JSON diagnostic leaked %q in %s", leaked, got)
		}
	}
}

func TestRuntimeDiagnosticSanitizerRedactsNestedRawPayloadsCaseInsensitively(t *testing.T) {
	const (
		rawSecret      = "raw-first-secret\nraw-later-secret"
		bodySecret     = "body-secret"
		tokenSecret    = "token-secret"
		passwordSecret = "password-secret"
		outerSafe      = "outer-safe"
		nestedSafe     = "nested-safe"
	)

	assertSanitized := func(t *testing.T, sanitized string) {
		t.Helper()
		for _, leaked := range []string{rawSecret, "raw-first-secret", "raw-later-secret", bodySecret, tokenSecret, passwordSecret} {
			if strings.Contains(sanitized, leaked) {
				t.Fatalf("nested diagnostic leaked %q in %s", leaked, sanitized)
			}
		}
		for _, safe := range []string{outerSafe, nestedSafe} {
			if !strings.Contains(sanitized, safe) {
				t.Fatalf("nested diagnostic lost safe value %q in %s", safe, sanitized)
			}
		}
	}

	t.Run("JSON", func(t *testing.T) {
		input := `{
  "safe": "outer-safe",
  "nested": {
    "RaW": "raw-first-secret\nraw-later-secret",
    "body": "body-secret",
    "token": "token-secret",
    "password": "password-secret",
    "safe": "nested-safe"
  }
}`

		got := sanitizeDiagnosticString(input)
		if !json.Valid([]byte(got)) {
			t.Fatalf("sanitized nested diagnostic is not valid JSON: %q", got)
		}
		assertSanitized(t, got)
	})

	t.Run("map", func(t *testing.T) {
		input := map[string]any{
			"safe": outerSafe,
			"nested": map[string]any{
				"rAw":      rawSecret,
				"body":     bodySecret,
				"token":    tokenSecret,
				"password": passwordSecret,
				"safe":     nestedSafe,
			},
		}

		got, err := json.Marshal(sanitizeDiagnosticValue(input))
		if err != nil {
			t.Fatalf("marshal sanitized nested map: %v", err)
		}
		assertSanitized(t, string(got))
	})
}

func TestRuntimeDiagnosticSanitizerRedactsArbitraryMultilineBodyThroughEnd(t *testing.T) {
	input := "request failed token=prefix-token\nprovider response:\nstatus=500\npassword=inside-password\nbody:\nfirst body line\nlate-body-secret\ntrailer=must-not-leak"

	got := sanitizeDiagnosticString(input)
	if !strings.Contains(got, "request failed") {
		t.Fatalf("non-sensitive prefix lost: %q", got)
	}
	for _, leaked := range []string{
		"prefix-token", "status=500", "inside-password", "first body line", "late-body-secret", "must-not-leak",
	} {
		if strings.Contains(got, leaked) {
			t.Fatalf("arbitrary multiline diagnostic leaked %q in %q", leaked, got)
		}
	}
}

func TestRuntimeDrawHealthSnapshotsKeepUnavailableFieldsNullable(t *testing.T) {
	drawWS, boundary := runtimeDrawHealthSnapshots(nil, "tron_ffc_3s", time.Now())
	if drawWS == nil || boundary == nil {
		t.Fatalf("unavailable snapshots must keep stable objects: drawWS=%#v boundary=%#v", drawWS, boundary)
	}
	if drawWS.Connected != nil || drawWS.LastFrameAt != nil || drawWS.LastPongAt != nil || drawWS.Reconnects != nil || drawWS.LastError != nil {
		t.Fatalf("draw WS unavailable fields must be null: %#v", drawWS)
	}
	if boundary.CurrentIssue != nil || boundary.NextIssue != nil || boundary.ReceivedAt != nil || boundary.WSRestLagPeriods != nil {
		t.Fatalf("boundary unavailable fields must be null: %#v", boundary)
	}
}

func TestSchemeRuntimeBlockReasonUsesExactPrecedence(t *testing.T) {
	cases := []struct {
		name  string
		input runtimeBlockInput
		want  string
	}{
		{name: "provider wrong period", input: runtimeBlockInput{ProviderAcceptedWrongPeriod: true, ProviderAcceptanceUnknown: true, ChainBlockReason: "missed_contiguous_period"}, want: "provider_accepted_wrong_period"},
		{name: "provider acceptance unknown", input: runtimeBlockInput{ProviderAcceptanceUnknown: true, ChainBlockReason: "missed_contiguous_period"}, want: "provider_acceptance_unknown"},
		{name: "terminal contiguous gap", input: runtimeBlockInput{DecisionStatus: "missed_contiguous_period", StrategyEvaluationFailed: true}, want: "missed_contiguous_period"},
		{name: "strategy failure", input: runtimeBlockInput{StrategyEvaluationFailed: true, DrawMissing: true}, want: "strategy_evaluation_failed"},
		{name: "draw missing", input: runtimeBlockInput{DrawMissing: true, DrawWSStale: true}, want: "draw_missing"},
		{name: "draw websocket stale", input: runtimeBlockInput{DrawWSStale: true, AwaitingTarget: true}, want: "draw_ws_stale"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := schemeRuntimeBlockReason(tc.input); got != tc.want {
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

func TestRuntimePayoutDiagnosticsUsesResolvedAccountSnapshot(t *testing.T) {
	want := accountsvc.PayoutSyncDiagnostics{
		AccountID:              17,
		PendingCount:           3,
		ProviderUnsettledCount: 3,
	}
	got := runtimePayoutDiagnostics(17, func(id int64) (accountsvc.PayoutSyncDiagnostics, bool) {
		return want, id == 17
	})
	if got == nil || got.AccountID != 17 || got.PendingCount != 3 || got.ProviderUnsettledCount != 3 {
		t.Fatalf("got=%+v", got)
	}
}

func TestRuntimePayoutDiagnosticsMissingAccountDoesNotProbe(t *testing.T) {
	calls := 0
	got := runtimePayoutDiagnostics(0, func(int64) (accountsvc.PayoutSyncDiagnostics, bool) {
		calls++
		return accountsvc.PayoutSyncDiagnostics{}, false
	})
	if got != nil || calls != 0 {
		t.Fatalf("got=%+v calls=%d", got, calls)
	}
}
