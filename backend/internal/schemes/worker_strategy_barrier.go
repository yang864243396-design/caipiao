package schemes

import (
	"context"
	"errors"
	"fmt"

	"caipiao/backend/internal/db/sqlcdb"
)

const legacyPreparedStateReadAttempts = 4

var errLegacyPreparedStateUnstable = errors.New("legacy scheme prepared state is unstable")

type legacyStrategyBarrierStore interface {
	GetLegacyStrategyBarrier(context.Context, string) (sqlcdb.LegacyStrategyBarrier, bool, error)
	GetSchemeInstanceFull(context.Context, string) (sqlcdb.SchemeInstance, error)
	GetSchemeStateVersion(context.Context, string) (int64, error)
}

type legacyPreparedBetState struct {
	Instance     sqlcdb.SchemeInstance
	StateVersion int64
}

type legacyStrategyProcessor func(
	context.Context,
	int64,
	string,
	string,
	string,
	int64,
) error

func loadLegacyPreparedBetState(
	ctx context.Context,
	store legacyStrategyBarrierStore,
	schemeID string,
) (legacyPreparedBetState, error) {
	for attempt := 0; attempt < legacyPreparedStateReadAttempts; attempt++ {
		versionBefore, err := store.GetSchemeStateVersion(ctx, schemeID)
		if err != nil {
			return legacyPreparedBetState{}, err
		}
		inst, err := store.GetSchemeInstanceFull(ctx, schemeID)
		if err != nil {
			return legacyPreparedBetState{}, err
		}
		versionAfter, err := store.GetSchemeStateVersion(ctx, schemeID)
		if err != nil {
			return legacyPreparedBetState{}, err
		}
		if versionBefore == versionAfter {
			return legacyPreparedBetState{Instance: inst, StateVersion: versionAfter}, nil
		}
	}
	return legacyPreparedBetState{}, fmt.Errorf(
		"%w: scheme %s changed during %d reads",
		errLegacyPreparedStateUnstable,
		schemeID,
		legacyPreparedStateReadAttempts,
	)
}

func convergeLegacyStrategyBeforeBet(
	ctx context.Context,
	store legacyStrategyBarrierStore,
	processStrategy legacyStrategyProcessor,
	inst sqlcdb.SchemeInstance,
) (legacyPreparedBetState, bool, error) {
	barrier, found, err := store.GetLegacyStrategyBarrier(ctx, inst.ID)
	if err != nil {
		return legacyPreparedBetState{}, false, err
	}
	if found && !barrier.StrategyEvaluated {
		if !barrier.HasDraw {
			state, loadErr := loadLegacyPreparedBetState(ctx, store, inst.ID)
			if errors.Is(loadErr, errLegacyPreparedStateUnstable) {
				return state, false, nil
			}
			return state, false, loadErr
		}
		expectedVersion, err := store.GetSchemeStateVersion(ctx, inst.ID)
		if err != nil {
			return legacyPreparedBetState{}, false, err
		}
		if err := processStrategy(
			ctx,
			barrier.RecordID,
			inst.ID,
			barrier.LotteryCode,
			barrier.PeriodNo,
			expectedVersion,
		); err != nil {
			return legacyPreparedBetState{}, false, err
		}
		barrier, found, err = store.GetLegacyStrategyBarrier(ctx, inst.ID)
		if err != nil {
			return legacyPreparedBetState{}, false, err
		}
		if found && !barrier.StrategyEvaluated {
			state, loadErr := loadLegacyPreparedBetState(ctx, store, inst.ID)
			if errors.Is(loadErr, errLegacyPreparedStateUnstable) {
				return state, false, nil
			}
			return state, false, loadErr
		}
	}
	state, err := loadLegacyPreparedBetState(ctx, store, inst.ID)
	if errors.Is(err, errLegacyPreparedStateUnstable) {
		return state, false, nil
	}
	return state, err == nil, err
}

func legacyPreparedStateStillCurrent(
	ctx context.Context,
	store legacyStrategyBarrierStore,
	schemeID string,
	preparedVersion int64,
) (bool, error) {
	currentVersion, err := store.GetSchemeStateVersion(ctx, schemeID)
	if err != nil {
		return false, err
	}
	if currentVersion != preparedVersion {
		return false, nil
	}
	barrier, found, err := store.GetLegacyStrategyBarrier(ctx, schemeID)
	if err != nil {
		return false, err
	}
	return !found || barrier.StrategyEvaluated, nil
}
