package realtimebus

import (
	"context"
	"time"
)

type Handler func(subject string, payload []byte)

type Subscription interface {
	Unsubscribe() error
}

type Diagnostics struct {
	Kind               string    `json:"kind"`
	Connected          bool      `json:"connected"`
	LastConnectedAt    time.Time `json:"lastConnectedAt,omitempty"`
	LastDisconnectedAt time.Time `json:"lastDisconnectedAt,omitempty"`
	Reconnects         uint64    `json:"reconnects"`
	Published          uint64    `json:"published"`
	PublishErrors      uint64    `json:"publishErrors"`
	Subscriptions      int64     `json:"subscriptions"`
	LastError          string    `json:"lastError,omitempty"`
}

type Bus interface {
	Publish(ctx context.Context, subject string, payload []byte) error
	Subscribe(subject string, handler Handler) (Subscription, error)
	OnConnectionChange(func(connected bool))
	Diagnostics() Diagnostics
	Close() error
}
