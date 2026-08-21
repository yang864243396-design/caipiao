package guaji

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	drawWSReadIdleTimeout = 45 * time.Second
	drawWSPingInterval    = 15 * time.Second
	drawWSWriteWait       = 5 * time.Second
)

var errDrawWSReadIdleTimeout = errors.New("guaji draw websocket read idle timeout")

// drawWSConn is the narrow portion of a Gorilla websocket connection that the
// draw transport needs. The application keeps exactly one ReadMessage caller:
// SubscribeDraws.
type drawWSConn interface {
	ReadMessage() (int, []byte, error)
	WriteControl(int, []byte, time.Time) error
	SetReadDeadline(time.Time) error
	SetPongHandler(func(string) error)
	Close() error
}

// DrawWSHealthSnapshot is the state of one shared draw websocket connection.
type DrawWSHealthSnapshot struct {
	ConnectedAt time.Time
	LastFrameAt time.Time
	LastPongAt  time.Time
	Reconnects  uint64
	LastError   string
}

// drawWSDeadlineWaiter lets deterministic test connections wait for a fake
// clock deadline. Real websocket connections use a standard timer instead.
type drawWSDeadlineWaiter interface {
	waitForReadDeadline(time.Time) <-chan time.Time
}

type drawWSLiveness struct {
	conn drawWSConn
	now  func() time.Time

	mu       sync.RWMutex
	health   DrawWSHealthSnapshot
	deadline time.Time
	refresh  chan struct{}
}

func newDrawWSLiveness(conn drawWSConn, now func() time.Time) *drawWSLiveness {
	if now == nil {
		now = time.Now
	}
	connectedAt := now()
	l := &drawWSLiveness{
		conn: conn,
		now:  now,
		health: DrawWSHealthSnapshot{
			ConnectedAt: connectedAt,
		},
		refresh: make(chan struct{}, 1),
	}
	l.conn.SetPongHandler(func(string) error {
		l.markPong()
		return nil
	})
	l.refreshReadDeadline(connectedAt, false, false)
	return l
}

// MarkFrame records a successfully read transport frame and extends the idle
// deadline. It is called by the sole ReadMessage loop after every frame.
func (l *drawWSLiveness) MarkFrame() {
	l.refreshReadDeadline(l.now(), true, false)
}

func (l *drawWSLiveness) markPong() {
	l.refreshReadDeadline(l.now(), false, true)
}

func (l *drawWSLiveness) refreshReadDeadline(at time.Time, frame, pong bool) {
	deadline := at.Add(drawWSReadIdleTimeout)
	err := l.conn.SetReadDeadline(deadline)

	l.mu.Lock()
	l.deadline = deadline
	if frame {
		l.health.LastFrameAt = at
	}
	if pong {
		l.health.LastPongAt = at
	}
	if err != nil {
		l.health.LastError = err.Error()
	}
	l.mu.Unlock()

	select {
	case l.refresh <- struct{}{}:
	default:
	}
}

// Run supervises only liveness writes and deadlines. It deliberately never
// reads frames; SubscribeDraws remains the one reader for this connection.
func (l *drawWSLiveness) Run(ctx context.Context) error {
	if l == nil || l.conn == nil {
		return nil
	}
	pinger := time.NewTicker(drawWSPingInterval)
	defer pinger.Stop()

	for {
		deadline := l.currentDeadline()
		deadlineC, stopDeadline := l.waitForDeadline(deadline)
		select {
		case <-ctx.Done():
			stopDeadline()
			_ = l.conn.Close()
			return nil
		case <-l.refresh:
			stopDeadline()
			continue
		case <-pinger.C:
			stopDeadline()
			if err := l.conn.WriteControl(websocket.PingMessage, nil, l.now().Add(drawWSWriteWait)); err != nil {
				l.recordError(err)
				_ = l.conn.Close()
				return err
			}
		case <-deadlineC:
			stopDeadline()
			if l.now().Before(deadline) || !l.isCurrentDeadline(deadline) {
				continue
			}
			l.recordReconnect(errDrawWSReadIdleTimeout)
			_ = l.conn.Close()
			return errDrawWSReadIdleTimeout
		}
	}
}

func (l *drawWSLiveness) Snapshot() DrawWSHealthSnapshot {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.health
}

func (l *drawWSLiveness) currentDeadline() time.Time {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.deadline
}

func (l *drawWSLiveness) isCurrentDeadline(deadline time.Time) bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.deadline.Equal(deadline)
}

func (l *drawWSLiveness) waitForDeadline(deadline time.Time) (<-chan time.Time, func()) {
	if waiter, ok := l.conn.(drawWSDeadlineWaiter); ok {
		return waiter.waitForReadDeadline(deadline), func() {}
	}
	delay := time.Until(deadline)
	if delay <= 0 {
		ready := make(chan time.Time, 1)
		ready <- l.now()
		return ready, func() {}
	}
	timer := time.NewTimer(delay)
	return timer.C, func() {
		if !timer.Stop() {
			select {
			case <-timer.C:
			default:
			}
		}
	}
}

func (l *drawWSLiveness) recordError(err error) {
	l.mu.Lock()
	l.health.LastError = err.Error()
	l.mu.Unlock()
}

func (l *drawWSLiveness) recordReconnect(err error) {
	l.mu.Lock()
	l.health.Reconnects++
	l.health.LastError = err.Error()
	l.mu.Unlock()
}
