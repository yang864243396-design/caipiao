package providerperiodtarget

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/lottery"
)

type fakeSnapshotRecorder struct {
	mu     sync.Mutex
	params sqlcdb.RecordCurrentProviderPeriodSnapshotParams
	id     int64
	err    error
	calls  atomic.Int32
}

var testCodeSequence atomic.Uint64

func uniqueTestLotteryCode(prefix string, now time.Time) string {
	return fmt.Sprintf("%s_%d_%d", prefix, now.UnixNano(), testCodeSequence.Add(1))
}

func (recorder *fakeSnapshotRecorder) RecordCurrentProviderPeriodSnapshot(
	_ context.Context, params sqlcdb.RecordCurrentProviderPeriodSnapshotParams,
) (int64, error) {
	recorder.calls.Add(1)
	recorder.mu.Lock()
	defer recorder.mu.Unlock()
	recorder.params = params
	return recorder.id, recorder.err
}

func TestCurrentUsesExistingLotteryWideOpenPeriod(t *testing.T) {
	now := time.Now().UTC()
	code := uniqueTestLotteryCode("shared_target", now)
	period := "P-current"
	closeAt := now.Add(6 * time.Second)
	lottery.UpdatePeriodsScheduleFullWithDuration(
		code, period, period, closeAt, closeAt, 6, closeAt.Format("2006-01-02 15:04:05"), now,
	)
	recorder := &fakeSnapshotRecorder{id: 91}

	target, snapshotID, ok, err := Current(context.Background(), recorder, code, "P-previous", now)
	if err != nil {
		t.Fatal(err)
	}
	if !ok || snapshotID != 91 || target.PeriodNo != period || !target.CloseAt.Equal(closeAt) {
		t.Fatalf("target=%+v snapshot=%d ok=%v", target, snapshotID, ok)
	}
	if recorder.params.PeriodNo != period || recorder.params.LotteryCode != code || recorder.params.Source != "guaji_periods_current" {
		t.Fatalf("recorded snapshot=%+v", recorder.params)
	}
}

func TestCurrentRecordsSharedSnapshotOnlyOncePerProcess(t *testing.T) {
	now := time.Now().UTC()
	code := uniqueTestLotteryCode("shared_once", now)
	period := "P-shared"
	closeAt := now.Add(6 * time.Second)
	lottery.UpdatePeriodsScheduleFullWithDuration(
		code, period, period, closeAt, closeAt, 6, closeAt.Format("2006-01-02 15:04:05"), now,
	)
	recorder := &fakeSnapshotRecorder{id: 93}

	const callers = 32
	var wait sync.WaitGroup
	wait.Add(callers)
	for range callers {
		go func() {
			defer wait.Done()
			_, snapshotID, ok, err := Current(context.Background(), recorder, code, "P-previous", now)
			if err != nil || !ok || snapshotID != 93 {
				t.Errorf("snapshot=%d ok=%v err=%v", snapshotID, ok, err)
			}
		}()
	}
	wait.Wait()
	if calls := recorder.calls.Load(); calls != 1 {
		t.Fatalf("shared snapshot recorder calls=%d want=1", calls)
	}
}

func TestCurrentUncachedRollbackDoesNotPolluteCommittedSnapshotCache(t *testing.T) {
	now := time.Now().UTC()
	code := uniqueTestLotteryCode("transaction_snapshot", now)
	currentSnapshotCache.Delete(code)
	closeAt := now.Add(6 * time.Second)
	lottery.UpdatePeriodsScheduleFullWithDuration(
		code, "P-101", "P-101", closeAt, closeAt, 6, closeAt.Format("2006-01-02 15:04:05"), now,
	)
	t.Cleanup(func() {
		currentSnapshotCache.Delete(code)
		lottery.ClearPeriodsSchedule(code)
	})

	rolledBack := &fakeSnapshotRecorder{id: 501}
	_, snapshotID, ok, err := CurrentUncached(context.Background(), rolledBack, code, "P-100", now)
	if err != nil || !ok || snapshotID != 501 || rolledBack.calls.Load() != 1 {
		t.Fatalf("rolled-back snapshot=%d ok=%v calls=%d err=%v", snapshotID, ok, rolledBack.calls.Load(), err)
	}

	retry := &fakeSnapshotRecorder{id: 502}
	_, snapshotID, ok, err = Current(context.Background(), retry, code, "P-100", now)
	if err != nil || !ok || snapshotID != 502 || retry.calls.Load() != 1 {
		t.Fatalf("retry snapshot=%d ok=%v calls=%d err=%v; want a fresh committed record", snapshotID, ok, retry.calls.Load(), err)
	}
}

func TestCurrentRejectsSourcePeriod(t *testing.T) {
	now := time.Now().UTC()
	code := uniqueTestLotteryCode("shared_source", now)
	period := "P-same"
	closeAt := now.Add(6 * time.Second)
	lottery.UpdatePeriodsSchedule(code, period, closeAt)
	recorder := &fakeSnapshotRecorder{id: 92}

	if _, _, ok, err := Current(context.Background(), recorder, code, period, now); err != nil || ok {
		t.Fatalf("source-period target ok=%v err=%v", ok, err)
	}
	if !recorder.params.ObservedAt.IsZero() {
		t.Fatal("source period must not be recorded as a new target")
	}
}

func TestCurrentUsesFreshWSNextPeriodForTronSixSecondFormalTarget(t *testing.T) {
	now := time.Now().UTC()
	code := "tron_ffc_6s"
	currentSnapshotCache.Delete(code)
	lottery.ClearPeriodsSchedule(code)

	// The periods feed can expose a future period as its first candidate. The
	// provider still accepts the fresh draw websocket's next_periods value.
	lottery.UpdatePeriodsScheduleFullWithDuration(
		code, "10114255902823", "10114255902823", now.Add(12*time.Second), now.Add(12*time.Second),
		6, now.Add(12*time.Second).Format("2006-01-02 15:04:05"), now.Add(6*time.Second),
	)
	lottery.UpdatePeriodState(code, "10114255902821", "10114255902822", now, 6)
	lottery.ClearPeriodsSchedule(code)
	recorder := &fakeSnapshotRecorder{id: 94}

	target, snapshotID, ok, err := Current(context.Background(), recorder, code, "10114255902821", now.Add(100*time.Millisecond))
	if err != nil {
		t.Fatal(err)
	}
	if !ok || snapshotID != 94 {
		t.Fatalf("snapshot=%d ok=%v", snapshotID, ok)
	}
	if target.PeriodNo != "10114255902822" {
		t.Fatalf("target period=%s want fresh WS next period", target.PeriodNo)
	}
	if wantClose := now.Add(6 * time.Second); !target.CloseAt.Equal(wantClose) {
		t.Fatalf("target close=%s want=%s", target.CloseAt, wantClose)
	}
	if recorder.params.Source != "guaji_draw_ws_next" {
		t.Fatalf("snapshot source=%q want guaji_draw_ws_next", recorder.params.Source)
	}
}

func TestCurrentDoesNotFallbackToRESTForTronSixSecondFormalTarget(t *testing.T) {
	now := time.Now().UTC()
	code := "tron_ffc_6s"
	currentSnapshotCache.Delete(code)
	lottery.ClearPeriodsSchedule(code)
	t.Cleanup(func() { lottery.ClearPeriodsSchedule(code) })

	// A stale/contradictory websocket boundary must stop formal dispatch. The
	// REST periods feed can be one issue ahead of the provider's bet endpoint.
	lottery.UpdatePeriodState(code, "10114255902823", "10114255902824", now, 6)
	lottery.UpdatePeriodsScheduleFullWithDuration(
		code, "P-rest-future", "P-rest-future", now.Add(6*time.Second), now.Add(6*time.Second),
		6, "", now,
	)
	recorder := &fakeSnapshotRecorder{id: 95}

	_, _, ok, err := Current(context.Background(), recorder, code, "P-different-source", now.Add(100*time.Millisecond))
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("formal short-period target fell back to the REST periods schedule")
	}
	if calls := recorder.calls.Load(); calls != 0 {
		t.Fatalf("snapshot recorder calls=%d want=0", calls)
	}
}

func TestCurrentRejectsEmptyOrMismatchedSourceForEveryFormalShortLottery(t *testing.T) {
	now := time.Now().UTC()
	for code, interval := range map[string]int{
		"tron_ffc_3s":  3,
		"tron_ffc_6s":  6,
		"tron_ffc_15s": 15,
	} {
		t.Run(code, func(t *testing.T) {
			lottery.ClearPeriodsSchedule(code)
			t.Cleanup(func() { lottery.ClearPeriodsSchedule(code) })
			current := "900000000000000000000000000001"
			lottery.UpdatePeriodState(code, current, "900000000000000000000000000002", now, interval)
			lottery.UpdatePeriodsScheduleFullWithDuration(
				code, "rest-candidate", "rest-candidate", now.Add(time.Duration(interval)*time.Second),
				now.Add(time.Duration(interval)*time.Second), interval, "", now,
			)

			for _, source := range []string{"", "mismatched-source"} {
				currentSnapshotCache.Delete(code)
				recorder := &fakeSnapshotRecorder{id: 96}
				_, _, ok, err := Current(context.Background(), recorder, code, source, now.Add(100*time.Millisecond))
				if err != nil {
					t.Fatal(err)
				}
				if ok {
					t.Fatalf("source %q unexpectedly authorized a formal target", source)
				}
				if calls := recorder.calls.Load(); calls != 0 {
					t.Fatalf("source %q recorder calls=%d want=0", source, calls)
				}
			}
		})
	}
}

func TestCurrentForInitialDispatchUsesFreshBoundaryForEveryFormalShortLottery(t *testing.T) {
	now := time.Now().UTC()
	for code, interval := range map[string]int{
		"tron_ffc_3s":  3,
		"tron_ffc_6s":  6,
		"tron_ffc_15s": 15,
	} {
		t.Run(code, func(t *testing.T) {
			currentSnapshotCache.Delete(code)
			current := "900000000000000000000000000002"
			next := "900000000000000000000000000003"
			lottery.UpdatePeriodState(code, current, next, now, interval)
			recorder := &fakeSnapshotRecorder{id: 97}

			target, snapshotID, ok, err := CurrentForInitialDispatch(context.Background(), recorder, code, now.Add(100*time.Millisecond))
			if err != nil {
				t.Fatal(err)
			}
			if !ok || snapshotID != 97 || target.PeriodNo != next {
				t.Fatalf("target=%+v snapshot=%d ok=%v, want fresh WS next issue", target, snapshotID, ok)
			}
			if recorder.params.Source != "guaji_draw_ws_next" {
				t.Fatalf("snapshot source=%q want guaji_draw_ws_next", recorder.params.Source)
			}
		})
	}
}

func TestCurrentForInitialDispatchRejectsStaleFormalBoundary(t *testing.T) {
	now := time.Now().UTC()
	for code, interval := range map[string]int{
		"tron_ffc_3s":  3,
		"tron_ffc_6s":  6,
		"tron_ffc_15s": 15,
	} {
		t.Run(code, func(t *testing.T) {
			currentSnapshotCache.Delete(code)
			lottery.UpdatePeriodState(code, "900000000000000000000000000003", "900000000000000000000000000004", now.Add(-time.Minute), interval)
			recorder := &fakeSnapshotRecorder{id: 98}

			_, _, ok, err := CurrentForInitialDispatch(context.Background(), recorder, code, now)
			if err != nil {
				t.Fatal(err)
			}
			if ok {
				t.Fatal("stale formal boundary authorized an initial target")
			}
			if calls := recorder.calls.Load(); calls != 0 {
				t.Fatalf("recorder calls=%d want=0", calls)
			}
		})
	}
}

func TestCurrentForInitialDispatchPreservesRESTTargetForOtherLotteries(t *testing.T) {
	now := time.Now().UTC()
	code := uniqueTestLotteryCode("initial_rest", now)
	lottery.UpdatePeriodsScheduleFullWithDuration(
		code, "rest-current", "rest-current", now.Add(time.Minute), now.Add(time.Minute),
		60, "", now,
	)
	t.Cleanup(func() { lottery.ClearPeriodsSchedule(code) })
	recorder := &fakeSnapshotRecorder{id: 99}

	target, snapshotID, ok, err := CurrentForInitialDispatch(context.Background(), recorder, code, now)
	if err != nil {
		t.Fatal(err)
	}
	if !ok || snapshotID != 99 || target.PeriodNo != "rest-current" {
		t.Fatalf("target=%+v snapshot=%d ok=%v, want REST target", target, snapshotID, ok)
	}
}
