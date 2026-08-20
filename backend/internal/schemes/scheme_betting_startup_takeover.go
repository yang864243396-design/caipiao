package schemes

import (
	"context"
	"log/slog"

	"caipiao/backend/internal/db/sqlcdb"
)

const startupFormalTakeoverConcurrency = 8

type startupFormalScheme struct {
	instance sqlcdb.SchemeInstance
	owner    string
}

type startupFormalTakeoverResult struct {
	Attempted   int
	Transferred int
	Deferred    int
}

// TakeOverRunningFormalSchemes transfers already-running formal schemes into
// the configured event chain once per backend start. The regular worker also
// retains its per-bet handoff as a safety net for schemes started later.
func (w *Worker) TakeOverRunningFormalSchemes(ctx context.Context) {
	if w == nil || w.q == nil || !w.formalEventModeConfigured() {
		return
	}
	rows, err := w.q.ListRunningSchemeInstances(ctx)
	if err != nil {
		slog.Warn("scheme startup event takeover list failed", "err", err)
		return
	}
	candidates := make([]startupFormalScheme, 0, len(rows))
	for _, row := range rows {
		inst := sqlcdb.SchemeInstanceFromRunningRow(row)
		if !requiresGuajiRealBet(inst) || !w.formalEventModeForLottery(inst.LotteryCode) {
			continue
		}
		execution, err := w.q.GetSchemeBettingExecutionState(ctx, inst.ID)
		if err != nil {
			slog.Warn("scheme startup event takeover state lookup failed", "id", inst.ID, "err", err)
			continue
		}
		candidates = append(candidates, startupFormalScheme{instance: inst, owner: execution.Owner})
	}
	result := w.takeOverStartupFormalSchemes(ctx, candidates)
	if result.Attempted > 0 || result.Deferred > 0 {
		slog.Info("scheme startup event takeover completed", "attempted", result.Attempted, "transferred", result.Transferred, "deferred", result.Deferred)
	}
}

func (w *Worker) formalEventModeConfigured() bool {
	return w != nil && w.strategyProcessor != nil &&
		(w.strategyProcessor.bettingMode == "gray" || w.strategyProcessor.bettingMode == "production") &&
		len(w.strategyProcessor.bettingLotteries) > 0
}

func (w *Worker) takeOverStartupFormalSchemes(ctx context.Context, candidates []startupFormalScheme) startupFormalTakeoverResult {
	eligible := make([]startupFormalScheme, 0, len(candidates))
	for _, candidate := range candidates {
		if requiresGuajiRealBet(candidate.instance) && candidate.owner == "legacy" && w.formalEventModeForLottery(candidate.instance.LotteryCode) {
			eligible = append(eligible, candidate)
		}
	}
	if len(eligible) == 0 {
		return startupFormalTakeoverResult{}
	}

	workers := startupFormalTakeoverConcurrency
	if workers > len(eligible) {
		workers = len(eligible)
	}
	jobs := make(chan startupFormalScheme)
	results := make(chan error, len(eligible))
	for i := 0; i < workers; i++ {
		go func() {
			for candidate := range jobs {
				_, err := w.takeOverLegacyFormalScheme(ctx, candidate.instance, candidate.owner)
				results <- err
			}
		}()
	}
	go func() {
		defer close(jobs)
		for _, candidate := range eligible {
			select {
			case <-ctx.Done():
				return
			case jobs <- candidate:
			}
		}
	}()

	result := startupFormalTakeoverResult{}
	for range eligible {
		select {
		case err := <-results:
			result.Attempted++
			if err != nil {
				result.Deferred++
			} else {
				result.Transferred++
			}
		case <-ctx.Done():
			return result
		}
	}
	return result
}
