package lookback

import (
	"context"
	"errors"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"caipiao/backend/internal/db/sqlcdb"
)

// Engine 执行回头 runtime 读写与复位。
type Engine struct {
	q *sqlcdb.Queries
}

func NewEngine(q *sqlcdb.Queries) *Engine {
	return &Engine{q: q}
}

func (e *Engine) LoadSettings(ctx context.Context, memberID int64) Settings {
	if e == nil || e.q == nil {
		return defaultSettings()
	}
	row, err := e.q.GetMemberLookbackSettings(ctx, memberID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return defaultSettings()
		}
		slog.Warn("lookback settings load failed", "memberId", memberID, "err", err)
		return defaultSettings()
	}
	return mapSettingsRow(row)
}

func (e *Engine) LoadRuntime(ctx context.Context, memberID int64, simBet bool) Runtime {
	if e == nil || e.q == nil {
		return Runtime{}
	}
	row, err := e.q.GetMemberLookbackRuntime(ctx, sqlcdb.GetMemberLookbackRuntimeParams{
		MemberID: memberID,
		SimBet:   simBet,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Runtime{}
		}
		slog.Warn("lookback runtime load failed", "memberId", memberID, "simBet", simBet, "err", err)
		return Runtime{}
	}
	return mapRuntimeRow(row)
}

func (e *Engine) SaveRuntime(ctx context.Context, memberID int64, simBet bool, rt Runtime, reset bool) error {
	if e == nil || e.q == nil {
		return nil
	}
	if reset {
		_, err := e.q.ResetMemberLookbackRuntime(ctx, sqlcdb.ResetMemberLookbackRuntimeParams{
			MemberID: memberID,
			SimBet:   simBet,
		})
		return err
	}
	_, err := e.q.UpsertMemberLookbackRuntime(ctx, sqlcdb.UpsertMemberLookbackRuntimeParams{
		MemberID:       memberID,
		SimBet:         simBet,
		SessionPnl:     floatToNumeric(rt.SessionPnl),
		PeriodIssue:    rt.PeriodIssue,
		PeriodPnl:      floatToNumeric(rt.PeriodPnl),
		PeriodHitCount: int32(rt.PeriodHitCount),
		TotalHitCount:  int32(rt.TotalHitCount),
	})
	return err
}

func mapRuntimeRow(row sqlcdb.GetMemberLookbackRuntimeRow) Runtime {
	return Runtime{
		SessionPnl:     numericToFloat(row.SessionPnl),
		PeriodIssue:    row.PeriodIssue,
		PeriodPnl:      numericToFloat(row.PeriodPnl),
		PeriodHitCount: int(row.PeriodHitCount),
		TotalHitCount:  int(row.TotalHitCount),
	}
}

// ApplyInstanceResets 执行个别/整体回头复位。
func (e *Engine) ApplyInstanceResets(
	ctx context.Context,
	inst sqlcdb.SchemeInstance,
	resetIndividual, resetOverall bool,
) ([]string, error) {
	if e == nil || e.q == nil {
		return nil, nil
	}
	if resetIndividual && !resetOverall {
		if err := e.q.ResetSchemeInstanceLookbackRoundEx(ctx, inst.ID); err != nil {
			return nil, err
		}
		return []string{inst.ID}, nil
	}
	if !resetOverall {
		return nil, nil
	}
	ids, err := e.q.ListRunningSchemeInstanceIDsByMemberSimBet(ctx, sqlcdb.ListRunningSchemeInstanceIDsByMemberSimBetParams{
		MemberID: inst.MemberID,
		SimBet:   inst.SimBet,
	})
	if err != nil {
		return nil, err
	}
	for _, id := range ids {
		if err := e.q.ResetSchemeInstanceLookbackRoundEx(ctx, id); err != nil {
			return nil, err
		}
	}
	if err := e.SaveRuntime(ctx, inst.MemberID, inst.SimBet, Runtime{}, true); err != nil {
		return nil, err
	}
	return ids, nil
}

// ProcessFormalAfterSettlement 正式盘派奖后更新 lookback_pnl 并评估复位。
func (e *Engine) ProcessFormalAfterSettlement(
	ctx context.Context,
	inst sqlcdb.SchemeInstance,
	periodNo string,
	pnl float64,
	hit bool,
	numericFromFloat func(float64) pgtype.Numeric,
) error {
	if e == nil || e.q == nil || inst.Status != "running" || inst.SimBet {
		return nil
	}
	settings := e.LoadSettings(ctx, inst.MemberID)
	currentLookback := numericToFloat(inst.LookbackPnl)
	var overallRT Runtime
	if AppliesTo(settings, false) && settings.Judgment == JudgmentOverall {
		overallRT = e.LoadRuntime(ctx, inst.MemberID, false)
	}
	lbEval := Evaluate(settings, false, currentLookback, overallRT, periodNo, pnl, hit)
	lookbackDelta := lbEval.LookbackAfter - currentLookback

	if _, err := e.q.ApplySchemeInstanceBet(ctx, sqlcdb.ApplySchemeInstanceBetParams{
		ID:               inst.ID,
		CountdownSec:     inst.CountdownSec,
		Turnover:         numericFromFloat(0), // 流水已在下单时累加，派奖后仅更新 lookback
		Pnl:              numericFromFloat(0),
		Multiplier:       inst.Multiplier,
		RoundIndex:       inst.RoundIndex,
		LastSettledIssue: inst.LastSettledIssue,
		LookbackPnl:      numericFromFloat(lookbackDelta),
		PickIndex:        inst.PickIndex,
		CurrentPick:      inst.CurrentPick,
		LastDirection:    inst.LastDirection,
	}); err != nil {
		return err
	}
	if lbEval.TrackOverall {
		if err := e.SaveRuntime(ctx, inst.MemberID, false, lbEval.OverallRT, lbEval.ResetOverall); err != nil {
			return err
		}
	}
	if lbEval.ResetIndividual || lbEval.ResetOverall {
		if _, err := e.ApplyInstanceResets(ctx, inst, lbEval.ResetIndividual, lbEval.ResetOverall); err != nil {
			return err
		}
		if lbEval.ResetIndividual && !lbEval.ResetOverall {
			slog.Info("lookback reset individual (formal)", "instanceId", inst.ID, "memberId", inst.MemberID)
		}
	}
	return nil
}
