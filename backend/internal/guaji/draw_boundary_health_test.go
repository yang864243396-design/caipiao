package guaji

import (
	"reflect"
	"testing"
	"time"
)

func TestBoundaryHealthMarksOnlySilentLotteryStale(t *testing.T) {
	h := NewBoundaryHealth([]string{"tron_ffc_3s", "tron_ffc_6s"})
	base := time.Unix(100, 0)
	h.Observe("tron_ffc_3s", "10", "11", base, 3*time.Second)
	h.Observe("tron_ffc_6s", "20", "21", base.Add(5*time.Second), 6*time.Second)

	stale := h.Stale(base.Add(3501 * time.Millisecond))
	if got, want := lotteryCodes(stale), []string{"tron_ffc_3s"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("stale lotteries = %v, want %v", got, want)
	}
}

func TestBoundaryHealthEmitsOneReconnectPerStaleGeneration(t *testing.T) {
	h := NewBoundaryHealth([]string{"tron_ffc_6s"})
	base := time.Unix(100, 0)
	h.Observe("tron_ffc_6s", "20", "21", base, 6*time.Second)

	if got := h.Stale(base.Add(7 * time.Second)); len(got) != 1 {
		t.Fatalf("first stale signal = %v, want one", got)
	}
	if got := h.Stale(base.Add(8 * time.Second)); len(got) != 0 {
		t.Fatalf("duplicate stale signal = %v, want none", got)
	}

	h.Observe("tron_ffc_6s", "21", "22", base.Add(8*time.Second), 6*time.Second)
	if got := h.Stale(base.Add(15 * time.Second)); len(got) != 1 {
		t.Fatalf("next stale generation = %v, want one", got)
	}
}

func TestBoundaryHealthDuplicateBoundaryDoesNotClearStaleGeneration(t *testing.T) {
	h := NewBoundaryHealth([]string{"tron_ffc_3s"})
	base := time.Unix(100, 0)
	h.Observe("tron_ffc_3s", "10", "11", base, 3*time.Second)
	_ = h.Stale(base.Add(4 * time.Second))

	h.Observe("tron_ffc_3s", "10", "11", base.Add(4*time.Second), 3*time.Second)
	snapshot := h.Snapshot("tron_ffc_3s")
	if !snapshot.ReconnectRequested {
		t.Fatal("duplicate boundary cleared stale generation")
	}
	if !snapshot.LastReceivedMono.Equal(base) {
		t.Fatalf("duplicate boundary receipt = %s, want %s", snapshot.LastReceivedMono, base)
	}
}
