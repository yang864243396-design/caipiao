package schemes

import (
	"context"

	"caipiao/backend/internal/cloudlimits"
	"caipiao/backend/internal/schemeevents"
)

func (w *Worker) pauseAllRunningForCloudLimit(ctx context.Context, memberID int64) bool {
	if w == nil {
		return false
	}
	return w.pauseAllRunningForCloudLimitWithMarker(ctx, memberID, w.realtime)
}

func (w *Worker) pauseAllRunningForCloudLimitWithMarker(ctx context.Context, memberID int64, marker schemeevents.Marker) bool {
	if w == nil || memberID <= 0 {
		return false
	}
	return cloudlimits.PauseAllRunningIfHit(ctx, w.q, w.hub, memberID, marker)
}
