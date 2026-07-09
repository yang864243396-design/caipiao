package ws

func PublishWithdrawQueue(h *Hub, payload WithdrawQueueChangedPayload) {
	if h == nil {
		return
	}
	if payload.Hint == "" {
		payload.Hint = "refresh_list"
	}
	h.Publish(TopicAdminWithdrawQueue, NewEvent(NameWithdrawQueueChanged, TopicAdminWithdrawQueue, payload))
}

func PublishSchemeMonitor(h *Hub, payload AdminSchemeMonitorPayload) {
	if h == nil {
		return
	}
	if payload.Hint == "" {
		payload.Hint = "refresh_list"
	}
	h.Publish(TopicAdminSchemeMonitor, NewEvent(NameSchemeMonitorChanged, TopicAdminSchemeMonitor, payload))
}

func PublishDashboardKpi(h *Hub, payload DashboardKpiChangedPayload) {
	if h == nil {
		return
	}
	if payload.Hint == "" {
		payload.Hint = "refresh_kpi"
	}
	h.Publish(TopicAdminDashboardKpi, NewEvent(NameDashboardKpiChanged, TopicAdminDashboardKpi, payload))
}
