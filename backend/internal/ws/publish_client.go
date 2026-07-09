package ws

func PublishSchemeInstance(h *Hub, account string, payload SchemeInstancePayload) {
	if h == nil || account == "" {
		return
	}
	h.PublishToAccount(account, TopicClientSchemeInstance, NewEvent(NameSchemeInstanceUpdated, TopicClientSchemeInstance, payload))
}

func PublishWallet(h *Hub, account string, payload WalletUpdatedPayload) {
	if h == nil || account == "" {
		return
	}
	h.PublishToAccount(account, TopicClientWallet, NewEvent(NameWalletUpdated, TopicClientWallet, payload))
}

func PublishMaintenance(h *Hub, payload MaintenanceChangedPayload) {
	if h == nil {
		return
	}
	h.Publish(TopicPublicMaintenance, NewEvent(NameMaintenanceChanged, TopicPublicMaintenance, payload))
}
