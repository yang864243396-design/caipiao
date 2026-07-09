package schemes

import (
	"caipiao/backend/internal/cloud/lookback"
	"caipiao/backend/internal/db/sqlcdb"
)

func evaluateLookback(
	settings lookback.Settings,
	simBet bool,
	currentLookbackPnl float64,
	runtime lookback.Runtime,
	periodIssue string,
	pnl float64,
	hit bool,
) lookback.EvalResult {
	return lookback.Evaluate(settings, simBet, currentLookbackPnl, runtime, periodIssue, pnl, hit)
}

func mapLookbackRuntimeRow(row sqlcdb.GetMemberLookbackRuntimeRow) lookback.Runtime {
	return lookback.Runtime{
		SessionPnl:     numericToFloat(row.SessionPnl),
		PeriodIssue:    row.PeriodIssue,
		PeriodPnl:      numericToFloat(row.PeriodPnl),
		PeriodHitCount: int(row.PeriodHitCount),
		TotalHitCount:  int(row.TotalHitCount),
	}
}

func mapLookbackSettingsRow(row sqlcdb.GetMemberLookbackSettingsRow) lookback.Settings {
	s := lookback.Settings{
		ApplyFormal:            row.ApplyFormal,
		ApplySim:               row.ApplySim,
		Judgment:               lookback.Judgment(row.Judgment),
		SingleProfitThreshold:  numericToFloat(row.SingleProfitThreshold),
		SingleLossThreshold:    numericToFloat(row.SingleLossThreshold),
		OverallProfitThreshold: numericToFloat(row.OverallProfitThreshold),
		OverallLossThreshold:   numericToFloat(row.OverallLossThreshold),
		SchemeWinsMin:          numericToFloat(row.SchemeWinsMin),
		SchemeWinsMax:          numericToFloat(row.SchemeWinsMax),
		PeriodProfit:           numericToFloat(row.PeriodProfit),
		PeriodLoss:             numericToFloat(row.PeriodLoss),
	}
	if !row.ApplyFormal && !row.ApplySim {
		s.RunModes = lookback.DecodeRunModes(row.RunMode)
		lookback.SyncApplyFlagsFromRunModes(&s)
	} else {
		lookback.SyncRunModesFromApplyFlags(&s)
	}
	return s
}

func defaultLookbackSettings() lookback.Settings {
	return lookback.Settings{
		Judgment:              lookback.JudgmentIndividual,
		SingleProfitThreshold: 100,
		SingleLossThreshold:   0,
	}
}
