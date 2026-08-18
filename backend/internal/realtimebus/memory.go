package realtimebus

import (
	"context"
	"errors"
	"sync"
)

var errBusClosed = errors.New("realtime bus is closed")

type Memory struct {
	mu            sync.RWMutex
	nextID        uint64
	subscriptions map[string]map[uint64]Handler
	callbacks     []func(bool)
	diagnostics   Diagnostics
	closed        bool
}

func NewMemory() *Memory {
	return &Memory{
		subscriptions: make(map[string]map[uint64]Handler),
		diagnostics:   Diagnostics{Kind: "memory", Connected: true},
	}
}

func (b *Memory) Publish(ctx context.Context, subject string, payload []byte) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return errBusClosed
	}
	b.diagnostics.Published++
	handlers := make([]Handler, 0, len(b.subscriptions[subject]))
	for _, handler := range b.subscriptions[subject] {
		handlers = append(handlers, handler)
	}
	b.mu.Unlock()

	for _, handler := range handlers {
		handler(subject, append([]byte(nil), payload...))
	}
	return nil
}

func (b *Memory) Subscribe(subject string, handler Handler) (Subscription, error) {
	if handler == nil {
		return nil, errors.New("realtime bus handler is required")
	}

	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return nil, errBusClosed
	}
	b.nextID++
	if b.subscriptions[subject] == nil {
		b.subscriptions[subject] = make(map[uint64]Handler)
	}
	b.subscriptions[subject][b.nextID] = handler
	b.diagnostics.Subscriptions++
	return &memorySubscription{bus: b, subject: subject, id: b.nextID}, nil
}

func (b *Memory) OnConnectionChange(callback func(connected bool)) {
	if callback == nil {
		return
	}
	b.mu.Lock()
	if !b.closed {
		b.callbacks = append(b.callbacks, callback)
	}
	b.mu.Unlock()
}

func (b *Memory) Diagnostics() Diagnostics {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.diagnostics
}

func (b *Memory) Close() error {
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return nil
	}
	b.closed = true
	b.subscriptions = nil
	b.diagnostics.Connected = false
	callbacks := append([]func(bool){}, b.callbacks...)
	b.callbacks = nil
	b.mu.Unlock()
	b.notify(callbacks, false)
	return nil
}

func (b *Memory) unsubscribe(subject string, id uint64) {
	b.mu.Lock()
	defer b.mu.Unlock()
	handlers := b.subscriptions[subject]
	if _, ok := handlers[id]; !ok {
		return
	}
	delete(handlers, id)
	if len(handlers) == 0 {
		delete(b.subscriptions, subject)
	}
	b.diagnostics.Subscriptions--
}

func (b *Memory) notify(callbacks []func(bool), connected bool) {
	for _, callback := range callbacks {
		go callback(connected)
	}
}

type memorySubscription struct {
	bus     *Memory
	subject string
	id      uint64
	once    sync.Once
}

func (s *memorySubscription) Unsubscribe() error {
	s.once.Do(func() { s.bus.unsubscribe(s.subject, s.id) })
	return nil
}
