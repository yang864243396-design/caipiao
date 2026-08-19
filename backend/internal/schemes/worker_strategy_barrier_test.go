package schemes

import (
	"context"
	"testing"

	"caipiao/backend/internal/db/sqlcdb"
)

type fakeLegacyStrategyBarrierStore struct {
	barrier sqlcdb.LegacyStrategyBarrier
	found   bool
	inst    sqlcdb.SchemeInstance
	version int64
}

func (f *fakeLegacyStrategyBarrierStore) GetLegacyStrategyBarrier(context.Context, string) (sqlcdb.LegacyStrategyBarrier, bool, error) {
	return f.barrier, f.found, nil
}

func (f *fakeLegacyStrategyBarrierStore) GetSchemeInstanceFull(context.Context, string) (sqlcdb.SchemeInstance, error) {
	return f.inst, nil
}

func (f *fakeLegacyStrategyBarrierStore) GetSchemeStateVersion(context.Context, string) (int64, error) {
	return f.version, nil
}

type changingLegacyPreparedStateStore struct {
	version   int64
	instances []sqlcdb.SchemeInstance
}

func (f *changingLegacyPreparedStateStore) GetLegacyStrategyBarrier(context.Context, string) (sqlcdb.LegacyStrategyBarrier, bool, error) {
	return sqlcdb.LegacyStrategyBarrier{}, false, nil
}

func (f *changingLegacyPreparedStateStore) GetSchemeInstanceFull(context.Context, string) (sqlcdb.SchemeInstance, error) {
	inst := f.instances[0]
	f.instances = f.instances[1:]
	if len(f.instances) == 1 {
		f.version++
	}
	return inst, nil
}

func (f *changingLegacyPreparedStateStore) GetSchemeStateVersion(context.Context, string) (int64, error) {
	return f.version, nil
}

func TestLoadLegacyPreparedBetStateRetriesAcrossVersionChange(t *testing.T) {
	store := &changingLegacyPreparedStateStore{
		version: 7,
		instances: []sqlcdb.SchemeInstance{
			{ID: "scheme-1", RoundIndex: 0},
			{ID: "scheme-1", RoundIndex: 1},
		},
	}

	got, err := loadLegacyPreparedBetState(context.Background(), store, "scheme-1")
	if err != nil {
		t.Fatal(err)
	}
	if got.StateVersion != 8 || got.Instance.RoundIndex != 1 {
		t.Fatalf("prepared state = version %d round %d, want coherent version 8 round 1",
			got.StateVersion, got.Instance.RoundIndex)
	}
}

type unstableLegacyPreparedStateStore struct {
	version int64
}

func (f *unstableLegacyPreparedStateStore) GetLegacyStrategyBarrier(context.Context, string) (sqlcdb.LegacyStrategyBarrier, bool, error) {
	return sqlcdb.LegacyStrategyBarrier{}, false, nil
}

func (f *unstableLegacyPreparedStateStore) GetSchemeInstanceFull(context.Context, string) (sqlcdb.SchemeInstance, error) {
	f.version++
	return sqlcdb.SchemeInstance{ID: "scheme-1"}, nil
}

func (f *unstableLegacyPreparedStateStore) GetSchemeStateVersion(context.Context, string) (int64, error) {
	return f.version, nil
}

func TestConvergeLegacyStrategyDefersUnstablePreparedStateWithoutError(t *testing.T) {
	store := &unstableLegacyPreparedStateStore{version: 1}
	_, ready, err := convergeLegacyStrategyBeforeBet(
		context.Background(), store, nil, sqlcdb.SchemeInstance{ID: "scheme-1"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if ready {
		t.Fatal("continuously changing state must defer the current worker tick")
	}
}

func TestConvergeLegacyStrategyBeforeBetProcessesPreviousDrawInSameCall(t *testing.T) {
	store := &fakeLegacyStrategyBarrierStore{
		barrier: sqlcdb.LegacyStrategyBarrier{
			RecordID: 91, LotteryCode: "tron_ffc_6s", PeriodNo: "1001",
			HasDraw: true,
		},
		found:   true,
		inst:    sqlcdb.SchemeInstance{ID: "scheme-1", RoundIndex: 0},
		version: 7,
	}

	processStrategy := func(
		_ context.Context,
		recordID int64,
		schemeID, lotteryCode, periodNo string,
		expectedVersion int64,
	) error {
		if recordID != 91 || schemeID != "scheme-1" ||
			lotteryCode != "tron_ffc_6s" || periodNo != "1001" || expectedVersion != 7 {
			t.Fatalf("unexpected strategy target record=%d scheme=%s draw=%s/%s version=%d",
				recordID, schemeID, lotteryCode, periodNo, expectedVersion)
		}
		store.barrier.StrategyEvaluated = true
		store.version = 8
		store.inst.RoundIndex = 1
		return nil
	}

	got, ready, err := convergeLegacyStrategyBeforeBet(
		context.Background(), store, processStrategy, store.inst,
	)
	if err != nil {
		t.Fatal(err)
	}
	if !ready {
		t.Fatal("previous draw was available, strategy must converge in the current call")
	}
	if got.StateVersion != 8 || got.Instance.RoundIndex != 1 {
		t.Fatalf("instance = version %d round %d, want refreshed version 8 round 1", got.StateVersion, got.Instance.RoundIndex)
	}
}

func TestLegacyPreparedStateStillCurrentRejectsStaleOrUnevaluatedState(t *testing.T) {
	tests := []struct {
		name      string
		prepared  int64
		current   int64
		evaluated bool
		want      bool
	}{
		{name: "current evaluated state", prepared: 8, current: 8, evaluated: true, want: true},
		{name: "strategy advanced after calculation", prepared: 7, current: 8, evaluated: true},
		{name: "previous strategy still pending", prepared: 8, current: 8, evaluated: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &fakeLegacyStrategyBarrierStore{
				barrier: sqlcdb.LegacyStrategyBarrier{StrategyEvaluated: tt.evaluated},
				found:   true,
				version: tt.current,
			}
			got, err := legacyPreparedStateStillCurrent(
				context.Background(), store, "scheme-1", tt.prepared,
			)
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Fatalf("ready = %v, want %v", got, tt.want)
			}
		})
	}
}
