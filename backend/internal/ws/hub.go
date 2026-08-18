package ws

import (
	"strings"
	"sync"
	"sync/atomic"

	"github.com/gorilla/websocket"
)

type ClientIdentity struct {
	Account  string
	MemberID int64
}

type MemberEventSource interface {
	SubscribeMember(memberID int64, emit func(Envelope)) (cancel func(), err error)
}

type HubDiagnostics struct {
	Connections        int    `json:"connections"`
	SubscribedMembers  int    `json:"subscribedMembers"`
	BackpressureCloses uint64 `json:"backpressureCloses"`
}

type memberRoute struct {
	connections map[*Conn]struct{}
	cancel      func()
	subscribing bool
	canceling   bool
	failed      bool
	generation  uint64
	transition  chan struct{}
}

type memberAcquisition struct {
	memberID   int64
	route      *memberRoute
	source     MemberEventSource
	generation uint64
	wait       <-chan struct{}
	failed     bool
}

type Hub struct {
	mu                sync.RWMutex
	subs              map[string]map[*Conn]struct{}
	conns             map[*Conn]struct{}
	memberSource      MemberEventSource
	members           map[int64]*memberRoute
	memberByConn      map[*Conn]int64
	backpressureClose atomic.Uint64
}

func NewHub() *Hub {
	return &Hub{
		subs:         make(map[string]map[*Conn]struct{}),
		conns:        make(map[*Conn]struct{}),
		members:      make(map[int64]*memberRoute),
		memberByConn: make(map[*Conn]int64),
	}
}

func (h *Hub) SetMemberEventSource(source MemberEventSource) {
	if h == nil {
		return
	}
	var acquisitions []memberAcquisition
	h.mu.Lock()
	h.memberSource = source
	if source != nil {
		for memberID, route := range h.members {
			if len(route.connections) == 0 {
				continue
			}
			acquisition := h.prepareMemberAcquisitionLocked(memberID, route)
			if acquisition.route != nil {
				acquisitions = append(acquisitions, acquisition)
			}
		}
	}
	h.mu.Unlock()
	for _, acquisition := range acquisitions {
		h.finishMemberAcquisition(acquisition)
	}
}

func (h *Hub) Register(c *Conn) bool {
	if h == nil || c == nil {
		return false
	}
	identity, hasIdentity := c.clientIdentity()
	var acquisition memberAcquisition
	h.mu.Lock()
	if _, exists := h.conns[c]; exists {
		h.mu.Unlock()
		return true
	}
	h.conns[c] = struct{}{}
	if hasIdentity {
		acquisition = h.bindRegisteredClientLocked(c, identity.MemberID)
	}
	h.mu.Unlock()
	ready := h.finishMemberAcquisition(acquisition)
	if !ready {
		c.Close(websocket.CloseServiceRestart, "realtime_route_unavailable")
	}
	return ready
}

func (h *Hub) Unregister(c *Conn) {
	if h == nil || c == nil {
		return
	}
	var cancel func()
	var cancelMemberID int64
	var cancelRoute *memberRoute
	var cancelGeneration uint64
	h.mu.Lock()
	if _, exists := h.conns[c]; !exists {
		h.mu.Unlock()
		return
	}
	delete(h.conns, c)
	if memberID, ok := h.memberByConn[c]; ok {
		delete(h.memberByConn, c)
		if route := h.members[memberID]; route != nil {
			delete(route.connections, c)
			if len(route.connections) == 0 {
				switch {
				case route.cancel != nil:
					cancel = route.cancel
					cancelMemberID = memberID
					route.cancel = nil
					route.canceling = true
					route.generation++
					route.transition = make(chan struct{})
					cancelRoute = route
					cancelGeneration = route.generation
				case !route.subscribing && !route.canceling:
					delete(h.members, memberID)
				}
			}
		}
	}
	for topic, set := range h.subs {
		delete(set, c)
		if len(set) == 0 {
			delete(h.subs, topic)
		}
	}
	h.mu.Unlock()
	if cancel != nil {
		cancel()
		h.finishMemberCancellation(cancelMemberID, cancelRoute, cancelGeneration)
	}
}

func (h *Hub) BindClientIdentity(c *Conn, identity ClientIdentity) bool {
	if h == nil || c == nil || !c.bindIdentity(identity) {
		return false
	}
	var acquisition memberAcquisition
	h.mu.Lock()
	if _, registered := h.conns[c]; registered && c.kind == KindClient {
		acquisition = h.bindRegisteredClientLocked(c, identity.MemberID)
	}
	h.mu.Unlock()
	ready := h.finishMemberAcquisition(acquisition)
	if !ready {
		c.Close(websocket.CloseServiceRestart, "realtime_route_unavailable")
	}
	return ready
}

func (h *Hub) bindRegisteredClientLocked(c *Conn, memberID int64) memberAcquisition {
	if memberID <= 0 {
		return memberAcquisition{}
	}
	if current, exists := h.memberByConn[c]; exists {
		if current == memberID {
			return memberAcquisition{}
		}
		return memberAcquisition{}
	}
	route := h.members[memberID]
	if route == nil {
		route = &memberRoute{connections: make(map[*Conn]struct{})}
		h.members[memberID] = route
	}
	route.connections[c] = struct{}{}
	h.memberByConn[c] = memberID
	return h.prepareMemberAcquisitionLocked(memberID, route)
}

func (h *Hub) prepareMemberAcquisitionLocked(memberID int64, route *memberRoute) memberAcquisition {
	if h.members[memberID] != route {
		return memberAcquisition{memberID: memberID, route: route, failed: true}
	}
	if route.failed {
		return memberAcquisition{memberID: memberID, route: route, failed: true}
	}
	if route.cancel != nil {
		return memberAcquisition{}
	}
	if route.canceling || route.subscribing {
		return memberAcquisition{memberID: memberID, route: route, wait: route.transition}
	}
	if h.memberSource == nil {
		return memberAcquisition{}
	}
	route.subscribing = true
	route.generation++
	route.transition = make(chan struct{})
	return memberAcquisition{
		memberID:   memberID,
		route:      route,
		source:     h.memberSource,
		generation: route.generation,
	}
}

func (h *Hub) finishMemberAcquisition(acquisition memberAcquisition) bool {
	for {
		if acquisition.failed {
			return false
		}
		if acquisition.route == nil {
			return true
		}
		if acquisition.wait != nil {
			<-acquisition.wait
			h.mu.Lock()
			acquisition = h.prepareMemberAcquisitionLocked(acquisition.memberID, acquisition.route)
			h.mu.Unlock()
			continue
		}
		if acquisition.source == nil {
			return true
		}
		cancel, err := acquisition.source.SubscribeMember(acquisition.memberID, func(env Envelope) {
			h.publishToMemberGeneration(acquisition.memberID, acquisition.route, acquisition.generation, env)
		})
		var closeTargets []*Conn
		var closeTransition chan struct{}
		var cancelGeneration uint64
		keep := false
		valid := false
		h.mu.Lock()
		current := h.members[acquisition.memberID]
		if current == acquisition.route && current.generation == acquisition.generation {
			valid = true
			current.subscribing = false
			if err == nil && cancel != nil && h.memberSource == acquisition.source && len(current.connections) > 0 {
				current.cancel = cancel
				keep = true
				closeTransition = current.transition
				current.transition = nil
			} else if cancel != nil {
				current.failed = err != nil || h.memberSource != acquisition.source
				if current.failed {
					closeTargets = memberRouteConnections(current)
				}
				current.canceling = true
				current.generation++
				cancelGeneration = current.generation
			} else {
				current.failed = true
				closeTargets = memberRouteConnections(current)
				closeTransition = current.transition
				current.transition = nil
				if len(current.connections) == 0 {
					delete(h.members, acquisition.memberID)
				}
			}
		}
		h.mu.Unlock()
		if keep {
			if closeTransition != nil {
				close(closeTransition)
			}
			return true
		}
		for _, c := range closeTargets {
			c.Close(websocket.CloseServiceRestart, "realtime_route_unavailable")
		}
		if cancel != nil {
			cancel()
			if valid {
				h.finishMemberCancellation(acquisition.memberID, acquisition.route, cancelGeneration)
			}
		}
		if closeTransition != nil {
			close(closeTransition)
		}
		return false
	}
}

func memberRouteConnections(route *memberRoute) []*Conn {
	targets := make([]*Conn, 0, len(route.connections))
	for c := range route.connections {
		targets = append(targets, c)
	}
	return targets
}

func (h *Hub) finishMemberCancellation(memberID int64, route *memberRoute, generation uint64) {
	if route == nil {
		return
	}
	h.mu.Lock()
	if h.members[memberID] == route && route.canceling && route.generation == generation {
		route.canceling = false
		transition := route.transition
		route.transition = nil
		if len(route.connections) == 0 {
			delete(h.members, memberID)
		}
		if transition != nil {
			close(transition)
		}
	}
	h.mu.Unlock()
}

func (h *Hub) Subscribe(c *Conn, topics []string) []string {
	accepted := make([]string, 0, len(topics))
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, topic := range topics {
		if !CanSubscribe(c.kind, c.isAuthenticated(), topic) {
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

func (h *Hub) publishToMemberGeneration(memberID int64, route *memberRoute, generation uint64, env Envelope) {
	if h == nil || memberID <= 0 || strings.TrimSpace(env.Topic) == "" {
		return
	}
	h.mu.RLock()
	current := h.members[memberID]
	topicSubscribers := h.subs[env.Topic]
	targets := make([]*Conn, 0)
	if current == route && current.generation == generation {
		targets = make([]*Conn, 0, len(route.connections))
		for c := range route.connections {
			if _, subscribed := topicSubscribers[c]; subscribed {
				targets = append(targets, c)
			}
		}
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

func (h *Hub) CloseClientConnections(code int, reason string) {
	if h == nil {
		return
	}
	h.mu.RLock()
	targets := make([]*Conn, 0, len(h.conns))
	for c := range h.conns {
		if c.kind == KindClient {
			targets = append(targets, c)
		}
	}
	h.mu.RUnlock()
	for _, c := range targets {
		c.Close(code, reason)
	}
}

func (h *Hub) Diagnostics() HubDiagnostics {
	if h == nil {
		return HubDiagnostics{}
	}
	h.mu.RLock()
	subscribedMembers := 0
	for _, route := range h.members {
		if route.cancel != nil {
			subscribedMembers++
		}
	}
	diagnostics := HubDiagnostics{
		Connections:       len(h.conns),
		SubscribedMembers: subscribedMembers,
	}
	h.mu.RUnlock()
	diagnostics.BackpressureCloses = h.backpressureClose.Load()
	return diagnostics
}

func (h *Hub) recordBackpressureClose() {
	if h != nil {
		h.backpressureClose.Add(1)
	}
}
