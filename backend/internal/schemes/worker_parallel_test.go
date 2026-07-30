package schemes

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/lottery"
)

func TestClampSchemeWorkerConcurrency(t *testing.T) {
	if got := clampSchemeWorkerConcurrency(0); got != defaultSchemeWorkerConcurrency {
		t.Fatalf("0 -> %d, want default %d", got, defaultSchemeWorkerConcurrency)
	}
	if got := clampSchemeWorkerConcurrency(-1); got != defaultSchemeWorkerConcurrency {
		t.Fatalf("-1 -> %d", got)
	}
	if got := clampSchemeWorkerConcurrency(64); got != 64 {
		t.Fatalf("64 -> %d", got)
	}
	if got := clampSchemeWorkerConcurrency(maxSchemeWorkerConcurrency + 10); got != maxSchemeWorkerConcurrency {
		t.Fatalf("over max -> %d", got)
	}
}

func TestUniqueLotteryCodesAndMembers(t *testing.T) {
	rows := []sqlcdb.ListRunningSchemeInstancesRow{
		{LotteryCode: "a", MemberID: 1},
		{LotteryCode: "b", MemberID: 1},
		{LotteryCode: "a", MemberID: 2},
		{LotteryCode: "", MemberID: 3},
		{LotteryCode: "c", MemberID: 0},
	}
	codes := uniqueLotteryCodes(rows)
	if len(codes) != 3 || codes[0] != "a" || codes[1] != "b" || codes[2] != "c" {
		t.Fatalf("codes=%v", codes)
	}
	mids := uniqueMemberIDs(rows)
	if len(mids) != 3 || mids[0] != 1 || mids[1] != 2 || mids[2] != 3 {
		t.Fatalf("members=%v", mids)
	}
}

func TestPrioritizeOpenBetWindow(t *testing.T) {
	openCode := "prio_open_" + t.Name()
	closedCode := "prio_closed_" + t.Name()
	t.Cleanup(func() {
		lottery.ClearPeriodsSchedule(openCode)
		lottery.ClearPeriodsSchedule(closedCode)
	})
	lottery.UpdatePeriodsSchedule(openCode, "P-OPEN", time.Now().UTC().Add(2*time.Minute))
	lottery.UpdatePeriodsSchedule(closedCode, "P-CLOSED", time.Now().UTC().Add(-time.Minute))

	rows := []sqlcdb.ListRunningSchemeInstancesRow{
		{ID: "c1", LotteryCode: closedCode},
		{ID: "o1", LotteryCode: openCode},
		{ID: "c2", LotteryCode: closedCode},
		{ID: "o2", LotteryCode: openCode},
	}
	got := prioritizeOpenBetWindow(rows)
	if len(got) != 4 {
		t.Fatalf("len=%d", len(got))
	}
	if got[0].ID != "o1" || got[1].ID != "o2" || got[2].ID != "c1" || got[3].ID != "c2" {
		t.Fatalf("order=%v %v %v %v", got[0].ID, got[1].ID, got[2].ID, got[3].ID)
	}
}

func TestBetWindowGateEnsureOpenUsesCacheWithoutRefresh(t *testing.T) {
	code := "gate_open_" + t.Name()
	t.Cleanup(func() { lottery.ClearPeriodsSchedule(code) })
	lottery.UpdatePeriodsSchedule(code, "P1", time.Now().UTC().Add(time.Minute))

	gate := newBetWindowGate(&Worker{})
	if !gate.ensureOpen(context.Background(), code) {
		t.Fatal("expected open from cache")
	}
}

func TestBetWindowGateNilSyncerClosed(t *testing.T) {
	code := "gate_closed_" + t.Name()
	t.Cleanup(func() { lottery.ClearPeriodsSchedule(code) })
	lottery.ClearPeriodsSchedule(code)

	gate := newBetWindowGate(&Worker{})
	if gate.ensureOpen(context.Background(), code) {
		t.Fatal("expected closed without schedule/syncer")
	}
}

func TestSetConcurrency(t *testing.T) {
	w := &Worker{}
	w.SetConcurrency(128)
	if w.concurrency != 128 {
		t.Fatalf("concurrency=%d", w.concurrency)
	}
	w.SetConcurrency(0)
	if w.concurrency != int32(defaultSchemeWorkerConcurrency) {
		t.Fatalf("default concurrency=%d", w.concurrency)
	}
}

func TestPrioritizeOpenBetWindow_allClosedUnchanged(t *testing.T) {
	rows := []sqlcdb.ListRunningSchemeInstancesRow{
		{ID: "a", LotteryCode: "no_sched_a_" + t.Name()},
		{ID: "b", LotteryCode: "no_sched_b_" + t.Name()},
	}
	got := prioritizeOpenBetWindow(rows)
	if got[0].ID != "a" || got[1].ID != "b" {
		t.Fatalf("should keep order when none open: %v %v", got[0].ID, got[1].ID)
	}
}

func TestBoundedConcurrencyPoolShape(t *testing.T) {
	const n, conc = 20, 4
	var inflight, peak atomic.Int32
	sem := make(chan struct{}, conc)
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			cur := inflight.Add(1)
			for {
				old := peak.Load()
				if cur <= old || peak.CompareAndSwap(old, cur) {
					break
				}
			}
			time.Sleep(5 * time.Millisecond)
			inflight.Add(-1)
		}()
	}
	wg.Wait()
	if p := peak.Load(); p < 1 || p > conc {
		t.Fatalf("peak inflight=%d want 1..%d", p, conc)
	}
}
