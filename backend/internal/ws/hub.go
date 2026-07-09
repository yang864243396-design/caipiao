package ws

import (
	"strings"
	"sync"
)

type Hub struct {
	mu    sync.RWMutex
	subs  map[string]map[*Conn]struct{}
	conns map[*Conn]struct{}
}

func NewHub() *Hub {
	return &Hub{
		subs:  make(map[string]map[*Conn]struct{}),
		conns: make(map[*Conn]struct{}),
	}
}

func (h *Hub) Register(c *Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.conns[c] = struct{}{}
}

func (h *Hub) Unregister(c *Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.conns, c)
	for topic, set := range h.subs {
		delete(set, c)
		if len(set) == 0 {
			delete(h.subs, topic)
		}
	}
}

func (h *Hub) Subscribe(c *Conn, topics []string) []string {
	accepted := make([]string, 0, len(topics))
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, topic := range topics {
		if !CanSubscribe(c.kind, c.authenticated, topic) {
			continue
		}
		set, ok := h.subs[topic]
		if !ok {
			set = make(map[*Conn]struct{})
			h.subs[topic] = set
		}
		set[c] = struct{}{}
		c.topics[topic] = struct{}{}
		accepted = append(accepted, topic)
	}
	return accepted
}

func (h *Hub) Unsubscribe(c *Conn, topics []string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, topic := range topics {
		delete(c.topics, topic)
		if set, ok := h.subs[topic]; ok {
			delete(set, c)
			if len(set) == 0 {
				delete(h.subs, topic)
			}
		}
	}
}

func (h *Hub) Publish(topic string, env Envelope) {
	h.mu.RLock()
	set := h.subs[topic]
	targets := make([]*Conn, 0, len(set))
	for c := range set {
		targets = append(targets, c)
	}
	h.mu.RUnlock()
	for _, c := range targets {
		c.TrySend(env)
	}
}

func (h *Hub) PublishToAccount(account, topic string, env Envelope) {
	account = strings.TrimSpace(account)
	if account == "" {
		return
	}
	h.mu.RLock()
	set := h.subs[topic]
	targets := make([]*Conn, 0, len(set))
	for c := range set {
		if c.getAccount() == account {
			targets = append(targets, c)
		}
	}
	h.mu.RUnlock()
	for _, c := range targets {
		c.TrySend(env)
	}
}

func (h *Hub) ConnCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.conns)
}
