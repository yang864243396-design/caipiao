package schemes

import (
	"context"
	"errors"
	"testing"
	"time"

	"caipiao/backend/internal/guajibet"
	"caipiao/backend/internal/lottery"
)

func TestThirdPartyOpenPeriodMatchesLastBetSkips(t *testing.T) {
	code := "dedup_same_period_test"
	period := "115202606200999"
	lottery.UpdatePeriodsSchedule(code, period, time.Now().Add(30*time.Second))

	currentOpen, ok := thirdPartyOpenPeriod(code)
	if !ok || currentOpen != period {
		t.Fatalf("currentOpen=%q ok=%v", currentOpen, ok)
	}

	lastBet := period
	if lastBet == currentOpen {
		dedup := betPeriodDedup{
			Skip:        true,
			CurrentOpen: currentOpen,
			LastBet:     lastBet,
			Reason:      "same_third_party_period",
		}
		if !dedup.Skip {
			t.Fatal("same third party period should skip bet")
		}
	}
}

func TestThirdPartyOpenPeriodChangesAfterCountdown(t *testing.T) {
	code := "dedup_new_period_test"
	oldPeriod := "115202606200001"
	newPeriod := "115202606200002"
	lottery.UpdatePeriodsSchedule(code, newPeriod, time.Now().Add(20*time.Second))

	lastBet := oldPeriod
	currentOpen, ok := thirdPartyOpenPeriod(code)
	if !ok || currentOpen != newPeriod {
		t.Fatalf("currentOpen=%q ok=%v", currentOpen, ok)
	}
	if lastBet == currentOpen {
		t.Fatal("after countdown ends, new period should allow next bet")
	}
}

func TestClearDuplicateThirdPartyBetReferenceKeepsReservedAmount(t *testing.T) {
	tid, orderNo, amount := clearDuplicateThirdPartyBetReference("126145087", "BO-old", 4, true)

	if tid != "" || orderNo != "" {
		t.Fatalf("duplicate reference should be cleared, got tid=%q order=%q", tid, orderNo)
	}
	if amount != 4 {
		t.Fatalf("reserved amount must remain positive, got %v", amount)
	}
}

func TestGuajiBetPeriodHasSafeWindowAt(t *testing.T) {
	code := "guaji_place_safety_margin_test"
	now := time.Now().UTC()
	lottery.ClearPeriodsSchedule(code)
	t.Cleanup(func() { lottery.ClearPeriodsSchedule(code) })

	lottery.UpdatePeriodsScheduleFullWithDuration(
		code, "P1", "P1", now, now.Add(guajiPlaceCloseSafety), 3, "", now.Add(-3*time.Second),
	)
	if guajiBetPeriodHasSafeWindowAt(code, now) {
		t.Fatal("period at the close safety boundary must not be sent to third party")
	}

	lottery.UpdatePeriodsScheduleFullWithDuration(
		code, "P2", "P2", now, now.Add(guajiPlaceCloseSafety+time.Nanosecond), 3, "", now.Add(-3*time.Second),
	)
	if !guajiBetPeriodHasSafeWindowAt(code, now) {
		t.Fatal("period beyond the close safety boundary should be eligible for placement")
	}
}

func TestGuajiShortPeriodWSWindowRejectsRequestAtActualDrawBoundary(t *testing.T) {
	code := "guaji_ws_close_guard_3s_test"
	now := time.Now().UTC()
	lottery.ClearPeriodsSchedule(code)
	t.Cleanup(func() { lottery.ClearPeriodsSchedule(code) })

	// The periods endpoint still reports P19 open beyond the actual WS cadence,
	// while the latest P18 draw proves P19 rolls at now+1s.
	lottery.UpdatePeriodsScheduleFullWithDuration(
		code, "P19", "P19", now.Add(4*time.Second), now.Add(4*time.Second), 3, "", now.Add(-2*time.Second),
	)
	lottery.UpdatePeriodState(code, "P18", "P19", now, 3)
	if !guajiShortPeriodWSWindowAllowsAt(code, "P19", now) {
		t.Fatal("fresh WS-derived window with sufficient time should allow the matching next issue")
	}
	lottery.UpdatePeriodState(code, "P18", "P19", now.Add(-2*time.Second), 3)

	if guajiShortPeriodWSWindowAllowsAt(code, "P19", now) {
		t.Fatal("three-second request inside WS-derived close safety must be rejected even when periods API still reports it open")
	}
}

func TestGuajiShortPeriodWSWindowUsesObservedWSCadence(t *testing.T) {
	code := "guaji_ws_close_guard_periods_duration_mismatch_test"
	now := time.Now().UTC()
	lottery.ClearPeriodsSchedule(code)
	t.Cleanup(func() { lottery.ClearPeriodsSchedule(code) })

	// The periods endpoint can report a generic 60-second duration even though
	// the draw websocket proves this lottery advances every three seconds.
	// The observed WS cadence must still protect the actual close boundary.
	lottery.UpdatePeriodsScheduleFullWithDuration(
		code, "P19", "P19", now.Add(4*time.Second), now.Add(4*time.Second), 60, "", now.Add(-2*time.Second),
	)
	lottery.UpdatePeriodState(code, "P18", "P19", now.Add(-2*time.Second), 3)

	if guajiShortPeriodWSWindowAllowsAt(code, "P19", now) {
		t.Fatal("observed three-second WS cadence must reject a request inside its actual close safety window")
	}
}

func TestGuajiBetSnapshotFreshAtRequiresFresherShortPeriodSnapshot(t *testing.T) {
	now := time.Now().UTC()
	short := lottery.PeriodsSchedule{CurrentPeriod: "P1", CloseAt: now.Add(time.Second), PeriodDurationSec: 3, UpdatedAt: now.Add(-2 * time.Second)}
	if guajiBetSnapshotFreshAt(short, now) {
		t.Fatal("three-second period snapshot older than half a period must be rejected")
	}

	standard := lottery.PeriodsSchedule{CurrentPeriod: "P1", CloseAt: now.Add(time.Second), PeriodDurationSec: 60, UpdatedAt: now.Add(-2 * time.Second)}
	if !guajiBetSnapshotFreshAt(standard, now) {
		t.Fatal("standard-period snapshot should remain usable within the five-second safety age")
	}
}

func TestGuajiPrePlaceVerificationRejectsAdvancedUpstreamPeriod(t *testing.T) {
	w := &Worker{periodVerifier: stubGuajiPeriodVerifier{
		period: "85428957",
		close:  time.Now().Add(5 * time.Second),
	}}
	err := w.verifyGuajiPeriodBeforePlace(context.Background(), "tron_ffc_3s", "vs8888", "85428956")
	if !errors.Is(err, guajibet.ErrPeriodClosed) {
		t.Fatalf("err=%v, want ErrPeriodClosed", err)
	}
}

func TestGuajiPrePlaceVerificationRejectsInsufficientConfirmedWindow(t *testing.T) {
	w := &Worker{periodVerifier: stubGuajiPeriodVerifier{
		period: "P1",
		close:  time.Now().Add(guajiVerifiedPlaceSafety - 10*time.Millisecond),
	}}
	err := w.verifyGuajiPeriodBeforePlace(context.Background(), "tron_ffc_3s", "vs8888", "P1")
	if !errors.Is(err, guajibet.ErrPeriodClosed) {
		t.Fatalf("err=%v, want ErrPeriodClosed", err)
	}
}

func TestGuajiPrePlaceVerificationIsOnlyNeededForAtRiskSnapshot(t *testing.T) {
	code := "guaji_preplace_verify_at_risk_test"
	now := time.Now().UTC()
	lottery.ClearPeriodsSchedule(code)
	t.Cleanup(func() { lottery.ClearPeriodsSchedule(code) })

	lottery.UpdatePeriodsScheduleFullWithDuration(
		code, "P1", "P1", now, now.Add(5*time.Second), 3, "", now,
	)
	if guajiPrePlaceVerificationNeeded(code, "P1", now) {
		t.Fatal("fresh period with ample time should use the centralized snapshot")
	}

	lottery.UpdatePeriodsScheduleFullWithDuration(
		code, "P1", "P1", now, now.Add(guajiPrePlaceVerificationLead), 3, "", now,
	)
	if !guajiPrePlaceVerificationNeeded(code, "P1", now) {
		t.Fatal("near-close period must synchronously confirm with upstream")
	}
}

func TestGuajiPrePlaceVerificationSkipsMemberProbeWhenShortPeriodSnapshotsAgree(t *testing.T) {
	code := "guaji_preplace_short_period_phase_lock_test"
	now := time.Now().UTC()
	lottery.ClearPeriodsSchedule(code)
	t.Cleanup(func() { lottery.ClearPeriodsSchedule(code) })

	closeAt := now.Add(1800 * time.Millisecond)
	lottery.UpdatePeriodsScheduleFullWithDuration(code, "P2", "P2", now, closeAt, 3, "", now)
	lottery.UpdatePeriodState(code, "P1", "P2", closeAt.Add(-3*time.Second), 3)

	if guajiPrePlaceVerificationNeeded(code, "P2", now) {
		t.Fatal("matching fresh short-period snapshots must not spend the remaining window on member verification")
	}
}

func TestGuajiPrePlaceVerificationKeepsSafetyFallbacks(t *testing.T) {
	now := time.Now().UTC()

	t.Run("short period WS mismatch", func(t *testing.T) {
		code := "guaji_preplace_short_period_ws_mismatch_test"
		lottery.ClearPeriodsSchedule(code)
		t.Cleanup(func() { lottery.ClearPeriodsSchedule(code) })
		closeAt := now.Add(1800 * time.Millisecond)
		lottery.UpdatePeriodsScheduleFullWithDuration(code, "P2", "P2", now, closeAt, 3, "", now)
		lottery.UpdatePeriodState(code, "P1", "P3", closeAt.Add(-3*time.Second), 3)

		if !guajiPrePlaceVerificationNeeded(code, "P2", now) {
			t.Fatal("mismatched WS target must retain member verification")
		}
	})

	t.Run("short period exact safety boundary", func(t *testing.T) {
		code := "guaji_preplace_short_period_safety_boundary_test"
		lottery.ClearPeriodsSchedule(code)
		t.Cleanup(func() { lottery.ClearPeriodsSchedule(code) })
		closeAt := now.Add(guajiPlaceCloseSafety)
		lottery.UpdatePeriodsScheduleFullWithDuration(code, "P2", "P2", now, closeAt, 3, "", now)
		lottery.UpdatePeriodState(code, "P1", "P2", closeAt.Add(-3*time.Second), 3)

		if !guajiPrePlaceVerificationNeeded(code, "P2", now) {
			t.Fatal("short period at the safety boundary must retain member verification")
		}
	})

	t.Run("ordinary period near close", func(t *testing.T) {
		code := "guaji_preplace_standard_period_near_close_test"
		lottery.ClearPeriodsSchedule(code)
		t.Cleanup(func() { lottery.ClearPeriodsSchedule(code) })
		closeAt := now.Add(guajiPrePlaceVerificationLead)
		lottery.UpdatePeriodsScheduleFullWithDuration(code, "P2", "P2", now, closeAt, 60, "", now)
		lottery.UpdatePeriodState(code, "P1", "P2", closeAt.Add(-60*time.Second), 60)

		if !guajiPrePlaceVerificationNeeded(code, "P2", now) {
			t.Fatal("ordinary near-close period must retain member verification")
		}
	})
}

type stubGuajiPeriodVerifier struct {
	period string
	close  time.Time
	err    error
}

func (s stubGuajiPeriodVerifier) VerifyOpenPeriodForMember(context.Context, string, string) (string, time.Time, error) {
	return s.period, s.close, s.err
}
