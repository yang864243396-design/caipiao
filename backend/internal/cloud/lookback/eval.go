package lookback

import "math"

// Runtime tracks member-level counters for overall judgment.
type Runtime struct {
	SessionPnl     float64
	PeriodIssue    string
	PeriodPnl      float64
	PeriodHitCount int
	TotalHitCount  int
}

// EvalResult 回头评估结果。
type EvalResult struct {
	ResetIndividual bool
	ResetOverall    bool
	OverallRT       Runtime
	TrackOverall    bool
	LookbackAfter   float64
}

func Evaluate(settings Settings, simBet bool, currentLookbackPnl float64, runtime Runtime, periodIssue string, pnl float64, hit bool) EvalResult {
	out := EvalResult{LookbackAfter: currentLookbackPnl + pnl}
	if !AppliesTo(settings, simBet) {
		return out
	}
	switch settings.Judgment {
	case JudgmentIndividual:
		out.ResetIndividual = ShouldResetIndividual(settings, out.LookbackAfter)
	case JudgmentOverall:
		out.OverallRT = AdvanceOverallRuntime(runtime, periodIssue, pnl, hit)
		out.ResetOverall = ShouldResetOverall(settings, out.OverallRT)
		out.TrackOverall = true
	}
	return out
}

func ShouldResetIndividual(settings Settings, lookbackPnlAfter float64) bool {
	if settings.SingleProfitThreshold > 0 && lookbackPnlAfter >= settings.SingleProfitThreshold {
		return true
	}
	if settings.SingleLossThreshold > 0 && lookbackPnlAfter <= -settings.SingleLossThreshold {
		return true
	}
	return false
}

func AdvanceOverallRuntime(rt Runtime, periodIssue string, pnl float64, hit bool) Runtime {
	out := rt
	if rt.PeriodIssue != periodIssue {
		out.PeriodIssue = periodIssue
		out.PeriodPnl = 0
		out.PeriodHitCount = 0
	}
	out.SessionPnl = round2(out.SessionPnl + pnl)
	out.PeriodPnl = round2(out.PeriodPnl + pnl)
	if hit {
		out.PeriodHitCount++
		out.TotalHitCount++
	}
	return out
}

func ShouldResetOverall(settings Settings, rt Runtime) bool {
	if settings.OverallProfitThreshold > 0 && rt.SessionPnl >= settings.OverallProfitThreshold {
		return true
	}
	if settings.OverallLossThreshold > 0 && rt.SessionPnl <= -settings.OverallLossThreshold {
		return true
	}
	if settings.PeriodProfit > 0 && rt.PeriodPnl >= settings.PeriodProfit {
		return true
	}
	if settings.PeriodLoss > 0 && rt.PeriodPnl <= -settings.PeriodLoss {
		return true
	}
	min := int(settings.SchemeWinsMin)
	max := int(settings.SchemeWinsMax)
	if min > 0 && max >= min && rt.TotalHitCount >= min && rt.TotalHitCount <= max {
		return true
	}
	return false
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}
