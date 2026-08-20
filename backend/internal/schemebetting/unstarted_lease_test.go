package schemebetting

import (
	"testing"
	"time"
)

func TestResolveUnstartedLeaseRecovery(t *testing.T) {
	now := time.Date(2026, 8, 20, 3, 20, 0, 0, time.UTC)
	state, reason, blocks := ResolveUnstartedLeaseRecovery(now.Add(-time.Second), now)
	if state != OutboxExpired || reason != "dispatcher_lost_before_start_deadline_elapsed" || !blocks {
		t.Fatalf("expired recovery = (%q, %q, %v)", state, reason, blocks)
	}

	state, reason, blocks = ResolveUnstartedLeaseRecovery(now.Add(time.Second), now)
	if state != OutboxPending || reason != "dispatcher_lost_before_start" || blocks {
		t.Fatalf("retry recovery = (%q, %q, %v)", state, reason, blocks)
	}
}
