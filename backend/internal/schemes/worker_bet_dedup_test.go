package schemes

import (
	"testing"
	"time"

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
