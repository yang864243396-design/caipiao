package schemerounds

import "testing"

func TestSimpleMartingaleHitReturnsRound1(t *testing.T) {
	rounds := []Round{
		{Mult: 1, AfterHit: 0, AfterMiss: 1},
		{Mult: 2, AfterHit: 0, AfterMiss: 2},
		{Mult: 4, AfterHit: 0, AfterMiss: 0},
	}
	if got := NextIndex(rounds, 0, true); got != 0 {
		t.Fatalf("667 hit should stay round1 (idx 0), got %d", got)
	}
}

func TestSimpleMartingale667668Sequence(t *testing.T) {
	rounds := []Round{
		{Mult: 1, AfterHit: 0, AfterMiss: 1},
		{Mult: 2, AfterHit: 0, AfterMiss: 2},
		{Mult: 4, AfterHit: 0, AfterMiss: 0},
	}
	idx := 0
	idx = NextIndex(rounds, idx, true) // 667 hit → round1
	if idx != 0 {
		t.Fatalf("after 667 hit idx=%d want 0", idx)
	}
	idx = NextIndex(rounds, idx, false) // 668 miss at round1 → round2
	if idx != 1 {
		t.Fatalf("after 668 miss idx=%d want 1", idx)
	}
}
