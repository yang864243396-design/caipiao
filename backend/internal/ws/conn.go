package ws

import (
	"encoding/json"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 90 * time.Second
	pingPeriod     = 30 * time.Second
	maxMessageSize = 64 << 10
	sendBuffer     = 256
)

type Conn struct {
	hub           *Hub
	conn          *websocket.Conn
	kind          ConnKind
	authenticated bool
	account       string
	memberID      int64
	topics        map[string]struct{}
	send          chan Envelope
	done          chan struct{}
	mu            sync.Mutex
	closed        bool
	closeOnce     sync.Once
	closeFn       func(code int, reason string)
}

type closeAction struct {
	code   int
	reason string
	done   chan struct{}
	fn     func(code int, reason string)
}

func newConn(hub *Hub, conn *websocket.Conn, kind ConnKind) *Conn {
	c := &Conn{
		hub:    hub,
		conn:   conn,
		kind:   kind,
		topics: make(map[string]struct{}),
		send:   make(chan Envelope, sendBuffer),
		done:   make(chan struct{}),
	}
	c.closeFn = func(code int, reason string) {
		_ = conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(code, reason), time.Now().Add(writeWait))
		_ = conn.Close()
	}
	return c
}

func (c *Conn) Run(authFn func(token string) (identity ClientIdentity, ok bool)) {
	defer func() {
		c.hub.Unregister(c)
		c.Close(websocket.CloseNormalClosure, "")
	}()

	c.hub.Register(c)
	_ = c.TrySend(SystemFrame(NameConnected, map[string]any{
		"connId":     c.conn.RemoteAddr().String(),
		"serverTime": time.Now().UTC().Format(time.RFC3339Nano),
	}))

	if c.kind == KindPublic {
		c.setAuthenticated()
		topics := c.hub.Subscribe(c, []string{TopicPublicMaintenance})
		_ = c.TrySend(SystemFrame(NameSubscribed, map[string]any{"topics": topics}))
	} else if c.kind == KindClient && c.isAuthenticated() {
		c.subscribeClientTopics()
	} else if c.kind == KindAdmin && c.isAuthenticated() {
		c.subscribeAdminTopics()
	}

	c.conn.SetReadLimit(maxMessageSize)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	go c.writePump()

	for {
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			return
		}
		_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
		c.handleMessage(data, authFn)
	}
}

func (c *Conn) handleMessage(data []byte, authFn func(token string) (identity ClientIdentity, ok bool)) {
	var in struct {
		Type    string          `json:"type"`
		Name    string          `json:"name"`
		Payload json.RawMessage `json:"payload"`
	}
	if err := json.Unmarshal(data, &in); err != nil {
		_ = c.TrySend(ErrorFrame(4003, "invalid json"))
		return
	}
	if in.Type != FrameTypeCommand {
		return
	}
	switch in.Name {
	case "auth":
		if c.isAuthenticated() || authFn == nil {
			return
		}
		var body struct {
			AccessToken string `json:"accessToken"`
		}
		if err := json.Unmarshal(in.Payload, &body); err != nil {
			_ = c.TrySend(ErrorFrame(4003, "invalid auth payload"))
			return
		}
		identity, ok := authFn(strings.TrimSpace(body.AccessToken))
		if !ok || !c.hub.BindClientIdentity(c, identity) {
			c.Close(4001, "unauthorized")
			return
		}
		_ = c.TrySend(SystemFrame(NameAuthOK, map[string]any{"account": identity.Account}))
		if c.kind == KindClient {
			c.subscribeClientTopics()
		} else if c.kind == KindAdmin {
			c.subscribeAdminTopics()
		}
	case "subscribe":
		var body struct {
			Topics []string `json:"topics"`
		}
		if err := json.Unmarshal(in.Payload, &body); err != nil {
			_ = c.TrySend(ErrorFrame(4003, "invalid subscribe payload"))
			return
		}
		topics := c.hub.Subscribe(c, body.Topics)
		_ = c.TrySend(SystemFrame(NameSubscribed, map[string]any{"topics": topics}))
	case "unsubscribe":
		var body struct {
			Topics []string `json:"topics"`
		}
		if err := json.Unmarshal(in.Payload, &body); err != nil {
			return
		}
		c.hub.Unsubscribe(c, body.Topics)
	case "ping":
		_ = c.TrySend(SystemFrame(NamePong, map[string]any{}))
	}
}

func (c *Conn) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Close(websocket.CloseNormalClosure, "")
	}()
	for {
		select {
		case env := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteJSON(env); err != nil {
				return
			}
		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(writeWait)); err != nil {
				return
			}
		case <-c.done:
			return
		}
	}
}

func (c *Conn) subscribeClientTopics() {
	if c.kind != KindClient || !c.isAuthenticated() {
		return
	}
	topics := c.hub.Subscribe(c, []string{
		TopicClientSchemeInstance,
		TopicClientCloudStats,
		TopicClientWallet,
	})
	_ = c.TrySend(SystemFrame(NameSubscribed, map[string]any{"topics": topics}))
}

func (c *Conn) subscribeAdminTopics() {
	if c.kind != KindAdmin || !c.isAuthenticated() {
		return
	}
	topics := c.hub.Subscribe(c, []string{TopicAdminWithdrawQueue, TopicAdminSchemeMonitor, TopicAdminDashboardKpi})
	_ = c.TrySend(SystemFrame(NameSubscribed, map[string]any{"topics": topics}))
}

func (c *Conn) bindIdentity(identity ClientIdentity) bool {
	identity.Account = strings.TrimSpace(identity.Account)
	if identity.Account == "" || (c.kind == KindClient && identity.MemberID <= 0) {
		return false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.authenticated {
		return c.account == identity.Account && c.memberID == identity.MemberID
	}
	c.authenticated = true
	c.account = identity.Account
	c.memberID = identity.MemberID
	return true
}

func (c *Conn) setAuthenticated() {
	c.mu.Lock()
	c.authenticated = true
	c.mu.Unlock()
}

func (c *Conn) isAuthenticated() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.authenticated
}

func (c *Conn) clientIdentity() (ClientIdentity, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	identity := ClientIdentity{Account: c.account, MemberID: c.memberID}
	return identity, c.kind == KindClient && c.authenticated && c.memberID > 0
}

func (c *Conn) getAccount() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.account
}

func (c *Conn) Close(code int, reason string) {
	c.close(code, reason, false)
}

func (c *Conn) closeAsync(code int, reason string) {
	c.close(code, reason, true)
}

func (c *Conn) close(code int, reason string, async bool) {
	if c == nil {
		return
	}
	c.mu.Lock()
	action, reserved := c.reserveCloseLocked(code, reason)
	c.mu.Unlock()
	if reserved {
		executeClose(action, async)
	}
}

func (c *Conn) reserveCloseLocked(code int, reason string) (closeAction, bool) {
	var action closeAction
	reserved := false
	c.closeOnce.Do(func() {
		reserved = true
		c.closed = true
		action = closeAction{code: code, reason: reason, done: c.done, fn: c.closeFn}
	})
	return action, reserved
}

func executeClose(action closeAction, async bool) {
	if action.done != nil {
		close(action.done)
	}
	if action.fn == nil {
		return
	}
	if async {
		go action.fn(action.code, action.reason)
		return
	}
	action.fn(action.code, action.reason)
}

func (c *Conn) TrySend(env Envelope) bool {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return false
	}
	select {
	case c.send <- env:
		c.mu.Unlock()
		return true
	default:
		action, reserved := c.reserveCloseLocked(4010, "realtime_backpressure")
		c.mu.Unlock()
		if !reserved {
			return false
		}
		executeClose(action, true)
	}
	if c.hub != nil {
		c.hub.recordBackpressureClose()
	}
	slog.Warn("ws outbound buffer full, closing connection", "name", env.Name, "topic", env.Topic)
	return false
}
