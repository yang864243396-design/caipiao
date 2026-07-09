package ws

import "strings"

type ConnKind string

const (
	KindPublic ConnKind = "public"
	KindClient ConnKind = "client"
	KindAdmin  ConnKind = "admin"
)

func CanSubscribe(kind ConnKind, authenticated bool, topic string) bool {
	topic = strings.TrimSpace(topic)
	if topic == "" {
		return false
	}
	switch kind {
	case KindPublic:
		return strings.HasPrefix(topic, "public.")
	case KindClient:
		if !authenticated {
			return false
		}
		return strings.HasPrefix(topic, "client.") || strings.HasPrefix(topic, "public.")
	case KindAdmin:
		if !authenticated {
			return false
		}
		return strings.HasPrefix(topic, "admin.") || strings.HasPrefix(topic, "public.")
	default:
		return false
	}
}
