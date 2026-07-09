package schemes

import (
	"context"
	"log/slog"

	"caipiao/backend/internal/cloud/lookback"
	"caipiao/backend/internal/db/sqlcdb"
)

func (w *Worker) lookbackEngine() *lookback.Engine {
	return lookback.NewEngine(w.q)
}

func (w *Worker) loadLookbackSettings(ctx context.Context, memberID int64) lookback.Settings {
	return w.lookbackEngine().LoadSettings(ctx, memberID)
}

func (w *Worker) loadLookbackRuntime(ctx context.Context, memberID int64, simBet bool) lookback.Runtime {
	return w.lookbackEngine().LoadRuntime(ctx, memberID, simBet)
}

func (w *Worker) saveLookbackRuntime(
	ctx context.Context,
	qtx *sqlcdb.Queries,
	memberID int64,
	simBet bool,
	rt lookback.Runtime,
	reset bool,
) error {
	return lookback.NewEngine(qtx).SaveRuntime(ctx, memberID, simBet, rt, reset)
}

func (w *Worker) applyLookbackResets(
	ctx context.Context,
	qtx *sqlcdb.Queries,
	inst sqlcdb.SchemeInstance,
	periodNo string,
	resetIndividual, resetOverall bool,
) error {
	engine := lookback.NewEngine(qtx)
	ids, err := engine.ApplyInstanceResets(ctx, inst, resetIndividual, resetOverall)
	if err != nil {
		return err
	}
	if len(ids) == 0 {
		return nil
	}
	individual := resetIndividual && !resetOverall
	overall := resetOverall
	count := len(ids)
	if individual {
		count = 0
	}
	if err := appendLookbackResetAudit(ctx, qtx, inst, periodNo, individual, overall, count); err != nil {
		return err
	}
	if overall {
		slog.Info("lookback reset overall", "memberId", inst.MemberID, "simBet", inst.SimBet, "instances", len(ids))
	}
	return nil
}
