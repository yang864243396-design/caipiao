package schemestate

import (
	"context"
	"log/slog"

	"github.com/jackc/pgx/v5/pgtype"

	"caipiao/backend/internal/cloud/lookback"
	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/schemerounds"
)

// ProcessFormalAfterSettlement 正式盘派奖后：回头盈亏 + 按实际中/未中推进倍投轮次。
// 轮次不在下单时推进（第三方待开奖），派奖后在此更新。
func ProcessFormalAfterSettlement(
	ctx context.Context,
	q *sqlcdb.Queries,
	inst sqlcdb.SchemeInstance,
	periodNo string,
	pnl float64,
	hit bool,
	definitionConfig []byte,
	numericFromFloat func(float64) pgtype.Numeric,
) error {
	if q == nil || inst.Status != "running" || inst.SimBet {
		return nil
	}

	engine := lookback.NewEngine(q)
	settings := engine.LoadSettings(ctx, inst.MemberID)
	currentLookback := numericToFloat(inst.LookbackPnl)
	var overallRT lookback.Runtime
	if lookback.AppliesTo(settings, false) && settings.Judgment == lookback.JudgmentOverall {
		overallRT = engine.LoadRuntime(ctx, inst.MemberID, false)
	}
	lbEval := lookback.Evaluate(settings, false, currentLookback, overallRT, periodNo, pnl, hit)
	lookbackDelta := lbEval.LookbackAfter - currentLookback

	applyRoundIndex := inst.RoundIndex
	if lbEval.ResetIndividual || lbEval.ResetOverall {
		applyRoundIndex = 0
	} else {
		rounds := schemerounds.ParseFromDefinitionConfig(definitionConfig)
		applyRoundIndex = int32(schemerounds.NextIndex(rounds, int(inst.RoundIndex), hit))
	}

	if _, err := q.ApplySchemeInstanceBet(ctx, sqlcdb.ApplySchemeInstanceBetParams{
		ID:               inst.ID,
		CountdownSec:     inst.CountdownSec,
		Turnover:         numericFromFloat(0),
		Pnl:              numericFromFloat(0),
		Multiplier:       inst.Multiplier,
		RoundIndex:       applyRoundIndex,
		LastSettledIssue: inst.LastSettledIssue,
		LookbackPnl:      numericFromFloat(lookbackDelta),
		PickIndex:        inst.PickIndex,
		CurrentPick:      inst.CurrentPick,
		LastDirection:    inst.LastDirection,
	}); err != nil {
		return err
	}

	if lbEval.TrackOverall {
		if err := engine.SaveRuntime(ctx, inst.MemberID, false, lbEval.OverallRT, lbEval.ResetOverall); err != nil {
			return err
		}
	}
	if lbEval.ResetIndividual || lbEval.ResetOverall {
		if _, err := engine.ApplyInstanceResets(ctx, inst, lbEval.ResetIndividual, lbEval.ResetOverall); err != nil {
			return err
		}
		if lbEval.ResetIndividual && !lbEval.ResetOverall {
			slog.Info("lookback reset individual (formal)", "instanceId", inst.ID, "memberId", inst.MemberID)
		}
	}
	return nil
}

func numericToFloat(n pgtype.Numeric) float64 {
	if !n.Valid {
		return 0
	}
	f, err := n.Float64Value()
	if err != nil || !f.Valid {
		return 0
	}
	return f.Float64
}
