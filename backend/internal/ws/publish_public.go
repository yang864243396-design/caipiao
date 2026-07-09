package ws

func PublishDraw(h *Hub, lotteryCode string, payload DrawResultPayload) {
	if h == nil || lotteryCode == "" {
		return
	}
	if payload.Hint == "" {
		payload.Hint = "refresh_detail"
	}
	topic := TopicPublicDraw(lotteryCode)
	h.Publish(topic, NewEvent(NameDrawResult, topic, payload))
}
