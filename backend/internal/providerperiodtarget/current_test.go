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
