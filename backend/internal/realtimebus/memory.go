package realtimebus

import (
	"context"
	"errors"
	"sync"
)

var errBusClosed = errors.New("realtime bus is closed")

const memorySubscriptionQueueSize = 64

type Memory struct {
	mu            sync.RWMutex
	nextID        uint64
	subscriptions map[string]map[uint64]*memorySubscription
	callbacks     []func(bool)
	diagnostics   Diagnostics
	closed        bool
}

func NewMemory() *Memory {
	return &Memory{
		subscriptions: make(map[string]map[uint64]*memorySubscription),
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
	subscriptions := make([]*memorySubscription, 0, len(b.subscriptions[subject]))
	for _, subscription := range b.subscriptions[subject] {
		subscriptions = append(subscriptions, subscription)
	}
	b.mu.Unlock()

	for _, subscription := range subscriptions {
		subscription.enqueue(subject, payload)
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
		b.subscriptions[subject] = make(map[uint64]*memorySubscription)
	}
	subscription := &memorySubscription{
		bus:     b,
		subject: subject,
		id:      b.nextID,
		handler: handler,
		queue:   make(chan memoryMessage, memorySubscriptionQueueSize),
		done:    make(chan struct{}),
	}
	b.subscriptions[subject][b.nextID] = subscription
	b.diagnostics.Subscriptions++
	go subscription.run()
	return subscription, nil
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
	subscriptions := make([]*memorySubscription, 0, b.diagnostics.Subscriptions)
	for _, handlers := range b.subscriptions {
		for _, subscription := range handlers {
			subscriptions = append(subscriptions, subscription)
		}
	}
	b.subscriptions = nil
	b.diagnostics.Connected = false
	callbacks := append([]func(bool){}, b.callbacks...)
	b.callbacks = nil
	b.mu.Unlock()
	for _, subscription := range subscriptions {
		subscription.stop()
	}
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
	bus      *Memory
	subject  string
	id       uint64
	handler  Handler
	queue    chan memoryMessage
	done     chan struct{}
	once     sync.Once
	stopOnce sync.Once
}

func (s *memorySubscription) Unsubscribe() error {
	s.once.Do(func() {
		s.bus.unsubscribe(s.subject, s.id)
		s.stop()
	})
	return nil
}

func (s *memorySubscription) enqueue(subject string, payload []byte) {
	message := memoryMessage{subject: subject, payload: append([]byte(nil), payload...)}
	select {
	case s.queue <- message:
	default:
	}
}

func (s *memorySubscription) run() {
	for {
		select {
		case <-s.done:
			return
		case message := <-s.queue:
			s.handler(message.subject, message.payload)
		}
	}
}

func (s *memorySubscription) stop() {
	s.stopOnce.Do(func() { close(s.done) })
}

type memoryMessage struct {
	subject string
	payload []byte
}
