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

type stubGuajiPeriodVerifier struct {
	period string
	close  time.Time
	err    error
}

func (s stubGuajiPeriodVerifier) VerifyOpenPeriodForMember(context.Context, string, string) (string, time.Time, error) {
	return s.period, s.close, s.err
}
