package instances

import "testing"

func TestStartPending(t *testing.T) {
	s := NewStore()
	row, err := s.Start("cloud_demo_2001")
	if err != nil {
		t.Fatal(err)
	}
	if row.Status != StatusRunning {
		t.Fatalf("want running, got %s", row.Status)
	}
}

func TestStartTwiceFails(t *testing.T) {
	s := NewStore()
	_, _ = s.Start("cloud_demo_2001")
	_, err := s.Start("cloud_demo_2001")
	if err != ErrInvalidAction {
		t.Fatalf("want invalid action, got %v", err)
	}
}

func TestPauseRunning(t *testing.T) {
	s := NewStore()
	_, _ = s.Start("inst-s2")
	row, err := s.Pause("inst-s2")
	if err != nil {
		t.Fatal(err)
	}
	if row.Status != StatusPaused {
		t.Fatalf("want paused, got %s", row.Status)
	}
}
