package schemebetting

import "time"

// ResolveUnstartedLeaseRecovery decides how to recover a lease which expired
// before any outbound attempt was durably started. No provider request can
// have been sent on this path, so a still-safe command may retry; an unsafe
// command must become terminal and block its strict chain.
func ResolveUnstartedLeaseRecovery(safeDeadline, now time.Time) (OutboxState, string, bool) {
	if !now.Before(safeDeadline.UTC()) {
		return OutboxExpired, "dispatcher_lost_before_start_deadline_elapsed", true
	}
	return OutboxPending, "dispatcher_lost_before_start", false
}
