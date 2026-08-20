package schemes

import (
	"context"
	"log/slog"
	"time"

	"caipiao/backend/internal/db/sqlcdb"
)

const (
	startupFormalTakeoverConcurrency  = 8
	startupFormalTakeoverRetryInitial = 500 * time.Millisecond
	// Keep this below the shortest 6-second lottery window. A 30-second cap
	// phase-locks retries to the same close-window position forever.
	startupFormalTakeoverRetryMax = 2 * time.Second
)

type startupFormalScheme struct {
	instance   sqlcdb.SchemeInstance
	owner      string
	chainState string
}

type startupFormalTakeoverResult struct {
	Eligible    int
	Attempted   int
	Transferred int
	Deferred    int
}

// RunStartupFormalTakeover transfers already-running formal schemes into the
// configured event chain. It retries safely while startup dependencies (fresh
// periods, event consumers, and capacity) are becoming ready.
func (w *Worker) RunStartupFormalTakeover(ctx context.Context) {
	if w == nil || w.q == nil || !w.formalEventModeConfigured() {
		return
	}
	result := runStartupFormalTakeover(ctx, w.takeOverRunningFormalSchemes, waitStartupFormalTakeoverRetry)
	if result.Attempted > 0 || result.Deferred > 0 {
		slog.Info("scheme startup event takeover stopped", "eligible", result.Eligible, "attempted", result.Attempted, "transferred", result.Transferred, "deferred", result.Deferred)
	}
}

func waitStartupFormalTakeoverRetry(ctx context.Context, delay time.Duration) bool {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func runStartupFormalTakeover(
	ctx context.Context,
	attempt func(context.Context) startupFormalTakeoverResult,
	wait func(context.Context, time.Duration) bool,
) startupFormalTakeoverResult {
	delay := startupFormalTakeoverRetryInitial
	for {
		result := attempt(ctx)
		if result.Eligible == 0 || result.Deferred == 0 {
			return result
		}
		if !wait(ctx, delay) {
			return result
		}
		delay *= 2
		if delay > startupFormalTakeoverRetryMax {
			delay = startupFormalTakeoverRetryMax
		}
	}
}

func (w *Worker) takeOverRunningFormalSchemes(ctx context.Context) startupFormalTakeoverResult {
	if w == nil || w.q == nil || !w.formalEventModeConfigured() {
		return startupFormalTakeoverResult{}
	}
	rows, err := w.q.ListRunningSchemeInstances(ctx)
	if err != nil {
		slog.Warn("scheme startup event takeover list failed", "err", err)
		return startupFormalTakeoverResult{Eligible: 1, Deferred: 1}
	}
	candidates := make([]startupFormalScheme, 0, len(rows))
	lookupDeferred := 0
	for _, row := range rows {
		inst := sqlcdb.SchemeInstanceFromRunningRow(row)
		if !requiresGuajiRealBet(inst) || !w.formalEventModeForLottery(inst.LotteryCode) {
			continue
		}
		execution, err := w.q.GetSchemeBettingExecutionState(ctx, inst.ID)
		if err != nil {
			slog.Warn("scheme startup event takeover state lookup failed", "id", inst.ID, "err", err)
			lookupDeferred++
			continue
		}
		candidates = append(candidates, startupFormalScheme{instance: inst, owner: execution.Owner, chainState: execution.ChainState})
	}
	result := w.takeOverStartupFormalSchemes(ctx, candidates)
	result.Eligible += lookupDeferred
	result.Deferred += lookupDeferred
	if result.Deferred > 0 {
		slog.Warn("scheme startup event takeover deferred", "eligible", result.Eligible, "attempted", result.Attempted, "transferred", result.Transferred, "deferred", result.Deferred)
	}
	return result
}

func (w *Worker) formalEventModeConfigured() bool {
	return w != nil && w.strategyProcessor != nil &&
		(w.strategyProcessor.bettingMode == "gray" || w.strategyProcessor.bettingMode == "production") &&
		len(w.strategyProcessor.bettingLotteries) > 0
}

func (w *Worker) takeOverStartupFormalSchemes(ctx context.Context, candidates []startupFormalScheme) startupFormalTakeoverResult {
	eligible := make([]startupFormalScheme, 0, len(candidates))
	for _, candidate := range candidates {
		legacyCandidate := candidate.owner == "legacy"
		blockedEventCandidate := candidate.owner == "event" && candidate.chainState == "blocked_requires_rearm"
		if requiresGuajiRealBet(candidate.instance) && (legacyCandidate || blockedEventCandidate) && w.formalEventModeForLottery(candidate.instance.LotteryCode) {
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
	type attemptResult struct {
		schemeID string
		err      error
	}
	results := make(chan attemptResult, len(eligible))
	for i := 0; i < workers; i++ {
		go func() {
			for candidate := range jobs {
				var err error
				if candidate.owner == "event" {
					enabler := w.formalEventEnabler
					if enabler == nil {
						enabler = w
					}
					err = enabler.RearmEventScheme(ctx, candidate.instance.ID, "system", "automatic recovery of proven unsent event chain")
				} else {
					_, err = w.takeOverLegacyFormalScheme(ctx, candidate.instance, candidate.owner)
				}
				results <- attemptResult{schemeID: candidate.instance.ID, err: err}
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

	result := startupFormalTakeoverResult{Eligible: len(eligible)}
	for range eligible {
		select {
		case attempt := <-results:
			result.Attempted++
			if attempt.err != nil {
				result.Deferred++
				slog.Warn("scheme startup event takeover attempt deferred", "id", attempt.schemeID, "err", attempt.err)
			} else {
				result.Transferred++
			}
		case <-ctx.Done():
			return result
		}
	}
	return result
}
