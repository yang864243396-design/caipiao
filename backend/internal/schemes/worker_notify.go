package schemes

import (
	"context"
	"strings"

	"caipiao/backend/internal/schemeevents"
	"caipiao/backend/internal/ws"
)

type workerOperationMarker struct {
	marker schemeevents.Marker
	seen   map[RealtimeInstanceRef]struct{}
}

func newWorkerOperationMarker(marker schemeevents.Marker) *workerOperationMarker {
	return &workerOperationMarker{marker: marker, seen: make(map[RealtimeInstanceRef]struct{})}
}

func (m *workerOperationMarker) MarkScheme(memberID int64, instanceID string) {
	if m == nil || m.marker == nil || memberID <= 0 || strings.TrimSpace(instanceID) == "" {
		return
	}
	ref := RealtimeInstanceRef{MemberID: memberID, InstanceID: instanceID}
	if _, ok := m.seen[ref]; ok {
		return
	}
	m.seen[ref] = struct{}{}
	m.marker.MarkScheme(memberID, instanceID)
}

type workerOperationMarkerContextKey struct{}

func (w *Worker) withCommittedSchemeMarker(
	ctx context.Context,
	ref RealtimeInstanceRef,
	afterCommit func(context.Context, schemeevents.Marker),
) {
	if w == nil {
		return
	}
	marker := newWorkerOperationMarker(w.realtime)
	operationCtx := context.WithValue(ctx, workerOperationMarkerContextKey{}, schemeevents.Marker(marker))
	if afterCommit != nil {
		afterCommit(operationCtx, marker)
	}
	marker.MarkScheme(ref.MemberID, ref.InstanceID)
}

func (w *Worker) markerForContext(ctx context.Context) schemeevents.Marker {
	if ctx != nil {
		if marker, ok := ctx.Value(workerOperationMarkerContextKey{}).(schemeevents.Marker); ok && marker != nil {
			return marker
		}
	}
	if w == nil {
		return nil
	}
	return w.realtime
}

func (w *Worker) memberAccount(ctx context.Context, memberID int64) string {
	if w == nil || w.q == nil {
		return ""
	}
	account, err := w.q.GetMemberAccountByID(ctx, memberID)
	if err != nil {
		return ""
	}
	return account
}

func (w *Worker) notifySchemeInstance(ctx context.Context, memberID int64, instanceID, _ string, status, _ string) {
	if w == nil {
		return
	}
	if marker := w.markerForContext(ctx); marker != nil && memberID > 0 && strings.TrimSpace(instanceID) != "" {
		marker.MarkScheme(memberID, instanceID)
	}
	if w.hub == nil {
		return
	}
	ws.PublishSchemeMonitor(w.hub, ws.AdminSchemeMonitorPayload{
		InstanceID: instanceID,
		Status:     status,
		Action:     "status_changed",
	})
}

func (w *Worker) notifyWallet(ctx context.Context, memberID int64, available, frozen float64, reason string) {
	if w == nil || w.hub == nil {
		return
	}
	account := w.memberAccount(ctx, memberID)
	if account == "" {
		return
	}
	ws.PublishWallet(w.hub, account, ws.WalletUpdatedPayload{
		Available: available,
		Frozen:    frozen,
		Currency:  "CNY",
		Reason:    reason,
	})
}
