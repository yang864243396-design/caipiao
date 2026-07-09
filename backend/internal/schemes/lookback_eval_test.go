package schemes

import (
	"testing"

	"caipiao/backend/internal/cloud/lookback"
)

func TestShouldResetIndividual(t *testing.T) {
	s := lookback.Settings{SingleProfitThreshold: 100, SingleLossThreshold: 50}
	if !lookback.ShouldResetIndividual(s, 100) {
		t.Fatal("profit threshold")
	}
	if !lookback.ShouldResetIndividual(s, -50) {
		t.Fatal("loss threshold")
	}
	if lookback.ShouldResetIndividual(s, 99) {
		t.Fatal("below profit")
	}
}

func TestShouldResetOverall(t *testing.T) {
	s := lookback.Settings{
		OverallProfitThreshold: 200,
		SchemeWinsMin:          2,
		SchemeWinsMax:          3,
	}
	rt := lookback.Runtime{SessionPnl: 200}
	if !lookback.ShouldResetOverall(s, rt) {
		t.Fatal("session profit")
	}
	rt2 := lookback.Runtime{TotalHitCount: 2}
	if !lookback.ShouldResetOverall(s, rt2) {
		t.Fatal("scheme wins in range")
	}
}

func TestAdvanceOverallRuntimeNewPeriod(t *testing.T) {
	rt := lookback.Runtime{PeriodIssue: "100", PeriodPnl: 10, PeriodHitCount: 1}
	next := lookback.AdvanceOverallRuntime(rt, "101", 5, true)
	if next.PeriodIssue != "101" || next.PeriodPnl != 5 || next.PeriodHitCount != 1 {
		t.Fatalf("got %+v", next)
	}
	if next.SessionPnl != 5 {
		t.Fatalf("session=%v", next.SessionPnl)
	}
}
