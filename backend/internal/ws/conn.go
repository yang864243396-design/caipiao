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
	topics        map[string]struct{}
	send          chan Envelope
	mu            sync.Mutex
	closed        bool
}

func newConn(hub *Hub, conn *websocket.Conn, kind ConnKind) *Conn {
	return &Conn{
		hub:    hub,
		conn:   conn,
		kind:   kind,
		topics: make(map[string]struct{}),
		send:   make(chan Envelope, sendBuffer),
	}
}

func (c *Conn) Run(authFn func(token string) (account string, ok bool)) {
	defer func() {
		c.hub.Unregister(c)
		_ = c.conn.Close()
	}()

	c.hub.Register(c)
	_ = c.TrySend(SystemFrame(NameConnected, map[string]any{
		"connId":     c.conn.RemoteAddr().String(),
		"serverTime": time.Now().UTC().Format(time.RFC3339Nano),
	}))

	if c.kind == KindPublic {
		c.authenticated = true
		topics := c.hub.Subscribe(c, []string{TopicPublicMaintenance})
		_ = c.TrySend(SystemFrame(NameSubscribed, map[string]any{"topics": topics}))
	} else if c.kind == KindClient && c.authenticated {
		c.subscribeClientTopics()
	} else if c.kind == KindAdmin && c.authenticated {
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

func (c *Conn) handleMessage(data []byte, authFn func(token string) (account string, ok bool)) {
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
		if c.authenticated || authFn == nil {
			return
		}
		var body struct {
			AccessToken string `json:"accessToken"`
		}
		if err := json.Unmarshal(in.Payload, &body); err != nil {
			_ = c.TrySend(ErrorFrame(4003, "invalid auth payload"))
			return
		}
		account, ok := authFn(strings.TrimSpace(body.AccessToken))
		if !ok {
			_ = c.conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(4001, "unauthorized"), time.Now().Add(writeWait))
			return
		}
		c.authenticated = true
		c.setAccount(account)
		_ = c.TrySend(SystemFrame(NameAuthOK, map[string]any{"account": account}))
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
		c.mu.Lock()
		if !c.closed {
			c.closed = true
			close(c.send)
		}
		c.mu.Unlock()
	}()
	for {
		select {
		case env, ok := <-c.send:
			if !ok {
				return
			}
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteJSON(env); err != nil {
				return
			}
		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(writeWait)); err != nil {
				return
			}
		}
	}
}

func (c *Conn) subscribeClientTopics() {
	if c.kind != KindClient || !c.authenticated {
		return
	}
	topics := c.hub.Subscribe(c, []string{
		TopicClientSchemeInstance,
		TopicClientWallet,
	})
	_ = c.TrySend(SystemFrame(NameSubscribed, map[string]any{"topics": topics}))
}

func (c *Conn) subscribeAdminTopics() {
	if c.kind != KindAdmin || !c.authenticated {
		return
	}
	topics := c.hub.Subscribe(c, []string{TopicAdminWithdrawQueue, TopicAdminSchemeMonitor, TopicAdminDashboardKpi})
	_ = c.TrySend(SystemFrame(NameSubscribed, map[string]any{"topics": topics}))
}

// setAccount 在锁内写入会员账号，避免与发布 goroutine 的并发读产生数据竞争。
func (c *Conn) setAccount(account string) {
	c.mu.Lock()
	c.account = account
	c.mu.Unlock()
}

// getAccount 在锁内读取会员账号，供 Hub.PublishToAccount 定向投递使用。
func (c *Conn) getAccount() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.account
}

func (c *Conn) TrySend(env Envelope) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return false
	}
	select {
	case c.send <- env:
		return true
	default:
		slog.Warn("ws outbound buffer full, drop frame", "name", env.Name, "topic", env.Topic)
		return false
	}
}
