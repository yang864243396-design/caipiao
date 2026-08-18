package realtimebus

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
)

const reconnectBufferSize = 8 * 1024 * 1024

type NATSConfig struct {
	URL             string
	Name            string
	User            string
	Password        string
	Token           string
	CredentialsFile string
	ReconnectWait   time.Duration
}

type NATS struct {
	nc *nats.Conn

	mu            sync.RWMutex
	diagnostics   Diagnostics
	callbacks     []func(bool)
	subscriptions int64
	closed        bool
}

func NewNATS(config NATSConfig) (*NATS, error) {
	bus := &NATS{diagnostics: Diagnostics{Kind: "nats"}}
	options := []nats.Option{
		nats.RetryOnFailedConnect(true),
		nats.MaxReconnects(-1),
		nats.ReconnectBufSize(reconnectBufferSize),
		nats.DisconnectErrHandler(func(_ *nats.Conn, err error) { bus.disconnected(err) }),
		nats.ReconnectHandler(func(_ *nats.Conn) { bus.reconnected() }),
		nats.ClosedHandler(func(_ *nats.Conn) { bus.connectionClosed() }),
	}
	if config.Name != "" {
		options = append(options, nats.Name(config.Name))
	}
	if config.ReconnectWait > 0 {
		options = append(options, nats.ReconnectWait(config.ReconnectWait))
	}
	switch {
	case config.CredentialsFile != "":
		options = append(options, nats.UserCredentials(config.CredentialsFile))
	case config.Token != "":
		options = append(options, nats.Token(config.Token))
	case config.User != "" || config.Password != "":
		options = append(options, nats.UserInfo(config.User, config.Password))
	}

	nc, err := nats.Connect(config.URL, options...)
	if err != nil {
		return nil, err
	}
	bus.nc = nc
	if nc.IsConnected() {
		bus.mu.Lock()
		bus.diagnostics.Connected = true
		bus.diagnostics.LastConnectedAt = time.Now().UTC()
		bus.mu.Unlock()
	}
	return bus, nil
}

func (b *NATS) Publish(ctx context.Context, subject string, payload []byte) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	b.mu.RLock()
	closed := b.closed
	b.mu.RUnlock()
	if closed {
		return errBusClosed
	}
	if err := b.nc.Publish(subject, payload); err != nil {
		b.recordPublishError(err)
		return err
	}
	b.mu.Lock()
	b.diagnostics.Published++
	b.mu.Unlock()
	return nil
}

func (b *NATS) Subscribe(subject string, handler Handler) (Subscription, error) {
	if handler == nil {
		return nil, errors.New("realtime bus handler is required")
	}
	b.mu.RLock()
	closed := b.closed
	b.mu.RUnlock()
	if closed {
		return nil, errBusClosed
	}
	sub, err := b.nc.Subscribe(subject, func(message *nats.Msg) {
		handler(message.Subject, append([]byte(nil), message.Data...))
	})
	if err != nil {
		return nil, err
	}
	b.mu.Lock()
	b.subscriptions++
	b.diagnostics.Subscriptions = b.subscriptions
	b.mu.Unlock()
	return &natsSubscription{bus: b, subscription: sub}, nil
}

func (b *NATS) OnConnectionChange(callback func(connected bool)) {
	if callback == nil {
		return
	}
	b.mu.Lock()
	if !b.closed {
		b.callbacks = append(b.callbacks, callback)
	}
	b.mu.Unlock()
}

func (b *NATS) Diagnostics() Diagnostics {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.diagnostics
}

func (b *NATS) Close() error {
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return nil
	}
	b.closed = true
	b.mu.Unlock()
	b.nc.Close()
	return nil
}

func (b *NATS) disconnected(err error) {
	b.mu.Lock()
	b.diagnostics.Connected = false
	b.diagnostics.LastDisconnectedAt = time.Now().UTC()
	b.diagnostics.LastError = safeNATSError(err)
	callbacks := append([]func(bool){}, b.callbacks...)
	b.mu.Unlock()
	b.notify(callbacks, false)
}

func (b *NATS) reconnected() {
	b.mu.Lock()
	b.diagnostics.Connected = true
	b.diagnostics.LastConnectedAt = time.Now().UTC()
	b.diagnostics.Reconnects++
	callbacks := append([]func(bool){}, b.callbacks...)
	b.mu.Unlock()
	b.notify(callbacks, true)
}

func (b *NATS) connectionClosed() {
	b.mu.Lock()
	b.diagnostics.Connected = false
	b.diagnostics.LastDisconnectedAt = time.Now().UTC()
	callbacks := append([]func(bool){}, b.callbacks...)
	b.mu.Unlock()
	b.notify(callbacks, false)
}

func (b *NATS) recordPublishError(err error) {
	b.mu.Lock()
	b.diagnostics.PublishErrors++
	b.diagnostics.LastError = safeNATSError(err)
	b.mu.Unlock()
}

func (b *NATS) unsubscribe() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.subscriptions == 0 {
		return
	}
	b.subscriptions--
	b.diagnostics.Subscriptions = b.subscriptions
}

func (b *NATS) notify(callbacks []func(bool), connected bool) {
	for _, callback := range callbacks {
		go callback(connected)
	}
}

func safeNATSError(err error) string {
	if err == nil {
		return ""
	}
	if errors.Is(err, nats.ErrAuthorization) {
		return "nats authentication error"
	}
	return "nats transport error"
}

type natsSubscription struct {
	bus          *NATS
	subscription *nats.Subscription
	once         sync.Once
	err          error
}

func (s *natsSubscription) Unsubscribe() error {
	s.once.Do(func() {
		s.err = s.subscription.Unsubscribe()
		s.bus.unsubscribe()
	})
	return s.err
}
