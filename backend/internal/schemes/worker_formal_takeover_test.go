package schemes

import (
	"context"
	"errors"
	"testing"
	"time"

	"caipiao/backend/internal/db/sqlcdb"
)

type fakeFormalEventEnabler struct {
	calls  int
	scheme string
	actor  string
	reason string
	err    error
}

func (f *fakeFormalEventEnabler) EnableEventScheme(_ context.Context, schemeID, actor, reason string) error {
	f.calls++
	f.scheme = schemeID
	f.actor = actor
	f.reason = reason
	return f.err
}

func TestWorkerFormalEventModeRecognizesOnlyConfiguredLottery(t *testing.T) {
	w := &Worker{strategyProcessor: &StrategyProcessor{}}
	w.SetSchemeBettingMode("gray", []string{"tron_ffc_6s"})

	if !w.formalEventModeForLottery("tron_ffc_6s") {
		t.Fatal("configured gray lottery must use the formal event chain")
	}
	if w.formalEventModeForLottery("tron_ffc_3s") {
		t.Fatal("lottery outside the gray allowlist must not use the formal event chain")
	}
}

func TestLegacyFormalBetIsNotAllowedWhenEventModeIsEnabled(t *testing.T) {
	if legacyFormalBetAllowed("legacy", true) {
		t.Fatal("legacy owner must not place a formal bet while event mode is enabled")
	}
	if !legacyFormalBetAllowed("legacy", false) {
		t.Fatal("legacy owner should retain legacy placement when event mode is disabled")
	}
	if legacyFormalBetAllowed("event", false) {
		t.Fatal("event-owned scheme must never be placed by the legacy worker")
	}
}

func TestWorkerTakesOverLegacyFormalSchemeWhenEventModeIsEnabled(t *testing.T) {
	enabler := &fakeFormalEventEnabler{}
	w := &Worker{strategyProcessor: &StrategyProcessor{}, formalEventEnabler: enabler}
	w.SetSchemeBettingMode("gray", []string{"tron_ffc_6s"})
	inst := sqlcdb.SchemeInstance{ID: "scheme-1", LotteryCode: "tron_ffc_6s", SimBet: false}

	taken, err := w.takeOverLegacyFormalScheme(context.Background(), inst, "legacy")
	if err != nil || !taken {
		t.Fatalf("takeover taken=%v err=%v", taken, err)
	}
	if enabler.calls != 1 || enabler.scheme != inst.ID || enabler.actor != "system" || enabler.reason == "" {
		t.Fatalf("unexpected takeover call: %+v", enabler)
	}
}

func TestWorkerDefersFailedFormalTakeoverWithoutLegacyFallback(t *testing.T) {
	enabler := &fakeFormalEventEnabler{err: errors.New("no_fresh_provider_target")}
	w := &Worker{strategyProcessor: &StrategyProcessor{}, formalEventEnabler: enabler}
	w.SetSchemeBettingMode("gray", []string{"tron_ffc_6s"})

	taken, err := w.takeOverLegacyFormalScheme(context.Background(), sqlcdb.SchemeInstance{
		ID: "scheme-1", LotteryCode: "tron_ffc_6s", SimBet: false,
	}, "legacy")
	if err == nil || !taken {
		t.Fatalf("failed takeover taken=%v err=%v", taken, err)
	}
	if legacyFormalBetAllowed("legacy", w.formalEventModeForLottery("tron_ffc_6s")) {
		t.Fatal("failed takeover must not permit a legacy fallback bet")
	}
}

func TestWorkerStartupTakeoverOnlyIncludesEligibleLegacyFormalSchemes(t *testing.T) {
	enabler := &fakeFormalEventEnabler{}
	w := &Worker{strategyProcessor: &StrategyProcessor{}, formalEventEnabler: enabler}
	w.SetSchemeBettingMode("gray", []string{"tron_ffc_6s"})

	result := w.takeOverStartupFormalSchemes(context.Background(), []startupFormalScheme{
		{instance: sqlcdb.SchemeInstance{ID: "eligible", LotteryCode: "tron_ffc_6s"}, owner: "legacy"},
		{instance: sqlcdb.SchemeInstance{ID: "simulated", LotteryCode: "tron_ffc_6s", SimBet: true}, owner: "legacy"},
		{instance: sqlcdb.SchemeInstance{ID: "other-lottery", LotteryCode: "tron_ffc_3s"}, owner: "legacy"},
		{instance: sqlcdb.SchemeInstance{ID: "event-owned", LotteryCode: "tron_ffc_6s"}, owner: "event"},
	})

	if result.Attempted != 1 || result.Transferred != 1 || result.Deferred != 0 {
		t.Fatalf("unexpected startup takeover result: %+v", result)
	}
	if enabler.calls != 1 || enabler.scheme != "eligible" {
		t.Fatalf("unexpected startup takeover call: %+v", enabler)
	}
}

func TestRunStartupFormalTakeoverRetriesDeferredSchemes(t *testing.T) {
	attempts := 0
	waits := 0
	result := runStartupFormalTakeover(
		context.Background(),
		func(context.Context) startupFormalTakeoverResult {
			attempts++
			if attempts == 1 {
				return startupFormalTakeoverResult{Eligible: 1, Attempted: 1, Deferred: 1}
			}
			return startupFormalTakeoverResult{Eligible: 1, Attempted: 1, Transferred: 1}
		},
		func(_ context.Context, delay time.Duration) bool {
			waits++
			if delay <= 0 {
				t.Fatalf("retry delay must be positive: %s", delay)
			}
			return true
		},
	)

	if attempts != 2 || waits != 1 {
		t.Fatalf("attempts=%d waits=%d", attempts, waits)
	}
	if result.Transferred != 1 || result.Deferred != 0 {
		t.Fatalf("unexpected final result: %+v", result)
	}
}
