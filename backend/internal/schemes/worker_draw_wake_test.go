package schemes

import "testing"

func TestEnqueueDrawWakeCoalescesByLottery(t *testing.T) {
	w := &Worker{
		drawWake:        make(chan string, 4),
		drawWakePending: make(map[string]struct{}),
	}
	w.enqueueDrawWake("tron_3s")
	w.enqueueDrawWake("tron_3s")
	w.enqueueDrawWake("other_3s")

	if got := len(w.drawWake); got != 2 {
		t.Fatalf("queued wakes=%d, want 2 distinct lotteries", got)
	}
	if got := w.takeDrawWake(); got != "tron_3s" {
		t.Fatalf("first wake=%q, want tron_3s", got)
	}
	w.enqueueDrawWake("tron_3s")
	if got := len(w.drawWake); got != 2 {
		t.Fatalf("requeued wakes=%d, want 2 after first was consumed", got)
	}
}
