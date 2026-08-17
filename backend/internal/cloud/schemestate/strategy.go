package schemestate

import (
	"context"

	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/schemerounds"
)

type strategyAfterDrawState struct {
	RoundIndex    int32
	PickIndex     int32
	CurrentPick   string
	LastDirection string
}

// nextStrategyAfterDraw contains only strategy state. It intentionally has no
// pnl, turnover, wallet, ledger, or lookback fields so a websocket draw cannot
// mutate financial settlement.
func nextStrategyAfterDraw(inst sqlcdb.SchemeInstance, kind string, definitionConfig []byte, betContent string, hit bool) strategyAfterDrawState {
	state := strategyAfterDrawState{
		RoundIndex: inst.RoundIndex, PickIndex: inst.PickIndex,
		CurrentPick: inst.CurrentPick, LastDirection: inst.LastDirection,
	}
	rounds := schemerounds.ParseFromDefinitionConfig(definitionConfig)
	state.RoundIndex = int32(schemerounds.NextIndex(rounds, int(inst.RoundIndex), hit))
	if FormalPickAdvancer != nil {
		state.PickIndex, state.CurrentPick, state.LastDirection = FormalPickAdvancer(
			kind, definitionConfig, inst, betContent, hit,
		)
	}
	return state
}

// ProcessStrategyAfterDraw advances only an accepted real bet's strategy
// fields. Financial settlement is performed later by the third-party payout
// worker through ProcessFormalFinancialAfterSettlement.
func ProcessStrategyAfterDraw(
	ctx context.Context,
	q *sqlcdb.Queries,
	inst sqlcdb.SchemeInstance,
	periodNo string,
	hit bool,
	definitionConfig []byte,
) error {
	if q == nil || (inst.Status != "running" && inst.Status != "pending") {
		return nil
	}
	snapshot, err := q.GetCloudBetPeriodSnapshot(ctx, inst.ID, periodNo)
	if err != nil {
		return err
	}
	state := nextStrategyAfterDraw(inst, inst.Kind, definitionConfig, snapshot.BetContent, hit)
	return q.ApplySchemeInstanceStrategyAfterDraw(ctx, inst.ID, state.RoundIndex, state.PickIndex, state.CurrentPick, state.LastDirection)
}
