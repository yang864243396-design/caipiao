package schemes

import (
	"context"

	"caipiao/backend/internal/ws"
)

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

func (w *Worker) notifySchemeInstance(ctx context.Context, memberID int64, instanceID, runMode, status, reason string) {
	if w == nil || w.hub == nil {
		return
	}
	account := w.memberAccount(ctx, memberID)
	if account == "" {
		return
	}
	ws.PublishSchemeInstance(w.hub, account, ws.SchemeInstancePayload{
		InstanceID: instanceID,
		RunMode:    runMode,
		Status:     status,
		Reason:     reason,
		Hint:       "refresh_running_list",
	})
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
