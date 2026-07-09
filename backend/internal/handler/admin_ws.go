package handler

import "caipiao/backend/internal/ws"

func (h *Handler) publishWithdrawQueueWS(orderNo, status, action string) {
	if h.wsHub == nil || orderNo == "" {
		return
	}
	ws.PublishWithdrawQueue(h.wsHub, ws.WithdrawQueueChangedPayload{
		OrderNo: orderNo,
		Status:  status,
		Action:  action,
	})
}

func (h *Handler) publishSchemeMonitorWS(instanceID, status, action string) {
	if h.wsHub == nil || instanceID == "" {
		return
	}
	ws.PublishSchemeMonitor(h.wsHub, ws.AdminSchemeMonitorPayload{
		InstanceID: instanceID,
		Status:     status,
		Action:     action,
	})
}

func (h *Handler) publishDashboardKpiWS(metric, orderNo, action string, amount float64) {
	if h.wsHub == nil || metric == "" {
		return
	}
	ws.PublishDashboardKpi(h.wsHub, ws.DashboardKpiChangedPayload{
		Metric:  metric,
		OrderNo: orderNo,
		Amount:  amount,
		Action:  action,
	})
}
