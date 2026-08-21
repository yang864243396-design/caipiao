package guaji

import (
	"strings"
	"sync"
	"time"

	"caipiao/backend/internal/periodissue"
)

// StaleLottery identifies one configured lottery whose most recent draw
// boundary has not arrived by its expected local receipt time.
type StaleLottery struct {
	LotteryCode  string
	CurrentIssue string
	NextIssue    string
	StaleAt      time.Time
}

// LotteryBoundaryHealthSnapshot is the local receipt state for one lottery.
// LastReceivedMono intentionally records when this process received the
// boundary, rather than the provider's wall-clock draw time.
type LotteryBoundaryHealthSnapshot struct {
	LotteryCode        string
	CurrentIssue       string
	NextIssue          string
	LastReceivedMono   time.Time
	Interval           time.Duration
	StaleAt            time.Time
	ReconnectRequested bool
}

// BoundaryHealth tracks independently configured draw boundaries. It is safe
// for the draw callback and its supervisor ticker to use concurrently.
type BoundaryHealth struct {
	mu      sync.RWMutex
	order   []string
	entries map[string]LotteryBoundaryHealthSnapshot
}

func NewBoundaryHealth(monitored []string) *BoundaryHealth {
	h := &BoundaryHealth{entries: make(map[string]LotteryBoundaryHealthSnapshot)}
	for _, lotteryCode := range monitored {
		lotteryCode = strings.TrimSpace(lotteryCode)
		if lotteryCode == "" {
			continue
		}
		if _, exists := h.entries[lotteryCode]; exists {
			continue
		}
		h.order = append(h.order, lotteryCode)
		h.entries[lotteryCode] = LotteryBoundaryHealthSnapshot{LotteryCode: lotteryCode}
	}
	return h
}

// Observe records a new valid boundary for a monitored lottery. Repeated or
// older boundaries deliberately do not refresh local receipt time or clear a
// reconnect request for the current stale generation.
func (h *BoundaryHealth) Observe(lotteryCode, currentIssue, nextIssue string, receivedMono time.Time, interval time.Duration) {
	if h == nil || interval <= 0 || receivedMono.IsZero() {
		return
	}
	lotteryCode = strings.TrimSpace(lotteryCode)
	currentIssue = strings.TrimSpace(currentIssue)
	nextIssue = strings.TrimSpace(nextIssue)
	if lotteryCode == "" || currentIssue == "" || nextIssue == "" || currentIssue == nextIssue {
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	previous, monitored := h.entries[lotteryCode]
	if !monitored || !isNewerBoundary(previous, currentIssue) {
		return
	}
	grace := interval / 6
	if grace > 500*time.Millisecond {
		grace = 500 * time.Millisecond
	}
	h.entries[lotteryCode] = LotteryBoundaryHealthSnapshot{
		LotteryCode:      lotteryCode,
		CurrentIssue:     currentIssue,
		NextIssue:        nextIssue,
		LastReceivedMono: receivedMono,
		Interval:         interval,
		StaleAt:          receivedMono.Add(interval).Add(grace),
	}
}

// Stale emits each lottery at most once for a stale generation. A later,
// newer valid boundary is the only event that clears that generation.
func (h *BoundaryHealth) Stale(now time.Time) []StaleLottery {
	if h == nil || now.IsZero() {
		return nil
	}
	h.mu.Lock()
	defer h.mu.Unlock()

	var stale []StaleLottery
	for _, lotteryCode := range h.order {
		snapshot := h.entries[lotteryCode]
		if snapshot.LastReceivedMono.IsZero() || snapshot.StaleAt.IsZero() || now.Before(snapshot.StaleAt) || snapshot.ReconnectRequested {
			continue
		}
		snapshot.ReconnectRequested = true
		h.entries[lotteryCode] = snapshot
		stale = append(stale, StaleLottery{
			LotteryCode:  snapshot.LotteryCode,
			CurrentIssue: snapshot.CurrentIssue,
			NextIssue:    snapshot.NextIssue,
			StaleAt:      snapshot.StaleAt,
		})
	}
	return stale
}

func (h *BoundaryHealth) Snapshot(lotteryCode string) LotteryBoundaryHealthSnapshot {
	if h == nil {
		return LotteryBoundaryHealthSnapshot{}
	}
	lotteryCode = strings.TrimSpace(lotteryCode)
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.entries[lotteryCode]
}

func lotteryCodes(stale []StaleLottery) []string {
	codes := make([]string, 0, len(stale))
	for _, lottery := range stale {
		codes = append(codes, lottery.LotteryCode)
	}
	return codes
}

func isNewerBoundary(previous LotteryBoundaryHealthSnapshot, currentIssue string) bool {
	if previous.LastReceivedMono.IsZero() {
		return true
	}
	return periodissue.Advances(previous.CurrentIssue, currentIssue)
}
