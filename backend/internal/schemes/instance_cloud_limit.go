package schemes

import (
	"context"

	"caipiao/backend/internal/cloudlimits"
)

func (w *Worker) pauseAllRunningForCloudLimit(ctx context.Context, memberID int64) bool {
	if w == nil || memberID <= 0 {
		return false
	}
	return cloudlimits.PauseAllRunningIfHit(ctx, w.q, w.hub, memberID)
}
