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
