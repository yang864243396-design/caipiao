package schemes

import (
	"context"

	"caipiao/backend/internal/db/sqlcdb"
)

func legacyOwnsFormalBet(owner string) bool {
	return owner == "legacy"
}

// formalEventModeForLottery reports whether this worker has an enabled formal
// event chain for the lottery. Once enabled, the legacy poller must never send
// a real bet for a legacy-owned instance: it has to hand ownership to the
// event chain first.
func (w *Worker) formalEventModeForLottery(lotteryCode string) bool {
	return w != nil && w.strategyProcessor != nil && w.strategyProcessor.formalModeForLottery(lotteryCode) != ""
}

func legacyFormalBetAllowed(owner string, formalEventEnabled bool) bool {
	return legacyOwnsFormalBet(owner) && !formalEventEnabled
}

// takeOverLegacyFormalScheme moves a configured formal scheme into the event
// chain before the legacy poller can submit a real bet. A failed handoff is
// deliberately retried by the next worker tick; it never re-enables a legacy
// fallback order for the same scheme.
func (w *Worker) takeOverLegacyFormalScheme(
	ctx context.Context,
	inst sqlcdb.SchemeInstance,
	owner string,
) (bool, error) {
	if w == nil || !requiresGuajiRealBet(inst) || owner != "legacy" || !w.formalEventModeForLottery(inst.LotteryCode) {
		return false, nil
	}
	enabler := w.formalEventEnabler
	if enabler == nil {
		enabler = w
	}
	return true, enabler.EnableEventScheme(ctx, inst.ID, "system", "automatic formal event takeover")
}
