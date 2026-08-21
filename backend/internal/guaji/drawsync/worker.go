package drawsync

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"log/slog"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"

	"caipiao/backend/internal/db"
	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/guaji"
	"caipiao/backend/internal/lottery"
	"caipiao/backend/internal/schemeeventbus"
	"caipiao/backend/internal/ws"
)

const (
	maxDrawWSReconnectBackoff   = 30 * time.Second
	boundaryHealthCheckInterval = 100 * time.Millisecond
)

type drawSubscriber interface {
	SubscribeDraws(context.Context, func([]guaji.DrawEvent)) error
}

type drawWSHealthSource interface {
	DrawWSHealth() guaji.DrawWSHealthSnapshot
}

// Worker 订阅第三方开奖 WS，过滤忽略彩种，按 outbound_lottery_code 反查入库并广播 WS-5（T3）。
type Worker struct {
	pool                *db.Pool
	q                   *sqlcdb.Queries
	client              drawSubscriber
	hub                 *ws.Hub
	reconnectJitter     func(time.Duration) time.Duration
	waitRetry           func(context.Context, time.Duration) bool
	boundaryHealth      *guaji.BoundaryHealth
	boundaryHealthTicks func() (<-chan time.Time, func())
	strategyNotifier    interface {
		NotifyStrategyDraw(context.Context, string, string)
	}
	store              drawStore
	boundaryPublisher  boundaryPublisher
	periodStateUpdater func(string, string, string, time.Time, int) bool
	recoveryReady      func()
	recoveryReadyCodes map[string]struct{}
	recoveryReadyOnce  sync.Once
}

type drawStore interface {
	ResolveLotteries(context.Context, string) ([]lotteryTarget, error)
	DrawIntervalSec(context.Context, string) int
	PersistDraw(context.Context, string, string, []string, time.Time, lottery.DrawFactMeta) (bool, error)
}

type boundaryPublisher interface {
	PublishPeriodBoundary(context.Context, schemeeventbus.PeriodBoundary) error
}

func (w *Worker) SetStrategyNotifier(notifier interface {
	NotifyStrategyDraw(context.Context, string, string)
}) {
	if w != nil {
		w.strategyNotifier = notifier
	}
}

func (w *Worker) SetPeriodBoundaryPublisher(publisher boundaryPublisher) {
	if w != nil {
		w.boundaryPublisher = publisher
	}
}

// SetContiguousRecoveryReady configures the one-shot readiness edge consumed
// by formal contiguous-target recovery. Non-short lotteries are ignored.
func (w *Worker) SetContiguousRecoveryReady(lotteryCodes []string, ready func()) {
	if w == nil {
		return
	}
	codes := make(map[string]struct{}, len(lotteryCodes))
	for _, lotteryCode := range lotteryCodes {
		lotteryCode = strings.TrimSpace(lotteryCode)
		if lottery.RequiresFreshShortPeriodWSBetTarget(lotteryCode) {
			codes[lotteryCode] = struct{}{}
		}
	}
	w.recoveryReadyCodes = codes
	w.recoveryReady = ready
}

// DrawWSHealth returns a local transport snapshot only; it never probes the
// provider or changes the shared subscription.
func (w *Worker) DrawWSHealth() guaji.DrawWSHealthSnapshot {
	if w == nil {
		return guaji.DrawWSHealthSnapshot{}
	}
	if source, ok := w.client.(drawWSHealthSource); ok {
		return source.DrawWSHealth()
	}
	return guaji.DrawWSHealthSnapshot{}
}

// PeriodBoundaryHealth returns the current read-only boundary receipt state.
func (w *Worker) PeriodBoundaryHealth(lotteryCode string, now time.Time) guaji.LotteryBoundaryHealthSnapshot {
	if w == nil || w.boundaryHealth == nil {
		return guaji.LotteryBoundaryHealthSnapshot{}
	}
	return w.boundaryHealth.SnapshotAt(lotteryCode, now)
}

func NewWorker(pool *db.Pool, client *guaji.Client, hub *ws.Hub) *Worker {
	if pool == nil || client == nil || !client.Enabled() {
		return nil
	}
	return &Worker{
		pool:           pool,
		q:              sqlcdb.New(pool),
		client:         client,
		hub:            hub,
		boundaryHealth: guaji.NewBoundaryHealth(lottery.FormalShortPeriodLotteryCodes()),
	}
}

// Run 持续订阅；断开后退避重连，直至 ctx 取消。
func (w *Worker) Run(ctx context.Context) {
	if w == nil {
		return
	}
	backoff := newDrawWSReconnectBackoff(w.jitterReconnectDelay)
	for {
		if ctx.Err() != nil {
			return
		}
		validFrame := false
		connectionCtx, cancelConnection := context.WithCancel(ctx)
		supervisorDone := w.superviseBoundaryHealth(connectionCtx, cancelConnection)
		err := w.client.SubscribeDraws(connectionCtx, func(events []guaji.DrawEvent) {
			if len(events) > 0 {
				validFrame = true
			}
			for _, ev := range events {
				if ierr := w.Ingest(ctx, ev); ierr != nil {
					slog.Warn("guaji draw ingest failed", "gameKey", ev.GameKey, "periods", ev.Periods, "err", ierr)
				}
			}
		})
		cancelConnection()
		if supervisorDone != nil {
			<-supervisorDone
		}
		if ctx.Err() != nil {
			return
		}
		if validFrame {
			backoff.Reset()
		}
		delay := backoff.Next()
		slog.Warn("guaji draw ws disconnected, retrying", "err", err, "backoff", delay.String())
		if !w.waitForReconnect(ctx, delay) {
			return
		}
	}
}

// superviseBoundaryHealth only requests cancellation of the current shared
// subscription. Run owns the one serial reconnect loop, while SubscribeDraws
// and its Task 2 liveness goroutine close the current connection.
func (w *Worker) superviseBoundaryHealth(ctx context.Context, cancelConnection context.CancelFunc) <-chan struct{} {
	if w == nil || w.boundaryHealth == nil {
		return nil
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		ticks, stopTicks := w.newBoundaryHealthTicks()
		defer stopTicks()
		for {
			select {
			case <-ctx.Done():
				return
			case now := <-ticks:
				stale := w.boundaryHealth.Stale(now)
				if len(stale) == 0 {
					continue
				}
				slog.Warn("guaji draw boundary stale, reconnecting shared websocket", "staleLotteryCount", len(stale))
				cancelConnection()
				return
			}
		}
	}()
	return done
}

func (w *Worker) newBoundaryHealthTicks() (<-chan time.Time, func()) {
	if w != nil && w.boundaryHealthTicks != nil {
		return w.boundaryHealthTicks()
	}
	ticker := time.NewTicker(boundaryHealthCheckInterval)
	return ticker.C, ticker.Stop
}

type drawWSReconnectBackoff struct {
	next   time.Duration
	jitter func(time.Duration) time.Duration
}

func newDrawWSReconnectBackoff(jitter func(time.Duration) time.Duration) *drawWSReconnectBackoff {
	if jitter == nil {
		jitter = jitterDrawWSReconnectDelay
	}
	return &drawWSReconnectBackoff{next: time.Second, jitter: jitter}
}

func (b *drawWSReconnectBackoff) Next() time.Duration {
	base := b.next
	if b.next < maxDrawWSReconnectBackoff {
		b.next *= 2
		if b.next > maxDrawWSReconnectBackoff {
			b.next = maxDrawWSReconnectBackoff
		}
	}
	return b.jitter(base)
}

func (b *drawWSReconnectBackoff) Reset() {
	b.next = time.Second
}

func (w *Worker) jitterReconnectDelay(delay time.Duration) time.Duration {
	if w != nil && w.reconnectJitter != nil {
		return w.reconnectJitter(delay)
	}
	return jitterDrawWSReconnectDelay(delay)
}

func jitterDrawWSReconnectDelay(delay time.Duration) time.Duration {
	spread := delay / 5
	if spread <= 0 {
		return delay
	}
	jittered := delay - spread + time.Duration(rand.Int63n(int64(2*spread)+1))
	if jittered > maxDrawWSReconnectBackoff {
		return maxDrawWSReconnectBackoff
	}
	return jittered
}

func (w *Worker) waitForReconnect(ctx context.Context, delay time.Duration) bool {
	if w != nil && w.waitRetry != nil {
		return w.waitRetry(ctx, delay)
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

type lotteryTarget struct {
	code     string
	template string
}

// Ingest 将一条彩种线开奖映射为内部彩种并入库 + 广播（可单测）。
// 一个 lottery_logXXX 键可能对应多个本平台彩种（不同 play_template，如同线下的
// 极速SSC/11选5/六合彩共享同区块）；逐个按各自 template 选号入库。
func (w *Worker) Ingest(ctx context.Context, ev guaji.DrawEvent) error {
	if w == nil || (w.q == nil && w.store == nil) {
		return errors.New("drawsync worker unavailable")
	}
	if ev.GameKey == "" || ev.Periods == "" {
		return nil
	}
	store := w.drawStore()
	targets, err := store.ResolveLotteries(ctx, ev.GameKey)
	if err != nil {
		return err
	}
	if len(targets) == 0 {
		return nil // outbound_lottery_code 未配置该键，跳过
	}
	receivedMono := time.Now()
	for _, tgt := range targets {
		drawnAt := ev.DrawnAt
		if drawnAt.IsZero() {
			drawnAt = time.Now()
		}
		if intervalSec := store.DrawIntervalSec(ctx, tgt.code); intervalSec > 0 {
			accepted := w.updatePeriodState(tgt.code, ev.Periods, ev.NextPeriods, drawnAt, intervalSec)
			w.observeAcceptedBoundary(accepted, tgt.code, ev.Periods, ev.NextPeriods, receivedMono, intervalSec)
			if accepted {
				w.publishPeriodBoundary(ctx, tgt.code, ev.Periods, ev.NextPeriods, receivedMono)
			}
		}
		balls := ev.Balls.BallsFor(tgt.template)
		if len(balls) == 0 {
			continue
		}
		inserted, err := store.PersistDraw(ctx, tgt.code, ev.Periods, balls, drawnAt, lottery.DrawFactMeta{
			Source:          "draw_ws",
			ProviderEventID: strings.TrimSpace(ev.GameKey) + ":" + strings.TrimSpace(ev.Periods),
			ReceivedAt:      time.Now().UTC(),
			ConfirmedAt:     drawnAt,
		})
		if err != nil {
			return err
		}
		if inserted {
			lottery.LogDrawCloseToIngestLatency(tgt.code, ev.Periods, "draw_ws", drawnAt)
			if w.strategyNotifier != nil {
				w.strategyNotifier.NotifyStrategyDraw(ctx, tgt.code, ev.Periods)
			}
		}
	}
	return nil
}

func (w *Worker) drawStore() drawStore {
	if w.store != nil {
		return w.store
	}
	return workerDrawStore{worker: w}
}

type workerDrawStore struct{ worker *Worker }

func (store workerDrawStore) ResolveLotteries(ctx context.Context, gameKey string) ([]lotteryTarget, error) {
	return store.worker.resolveLotteries(ctx, gameKey)
}

func (store workerDrawStore) DrawIntervalSec(ctx context.Context, lotteryCode string) int {
	return store.worker.drawIntervalSec(ctx, lotteryCode)
}

func (store workerDrawStore) PersistDraw(ctx context.Context, lotteryCode, issueNo string, balls []string, drawnAt time.Time, meta lottery.DrawFactMeta) (bool, error) {
	_, inserted, err := lottery.PersistDrawFactFromBalls(ctx, store.worker.q, store.worker.hub, lotteryCode, issueNo, balls, drawnAt, meta)
	return inserted, err
}

func (w *Worker) publishPeriodBoundary(ctx context.Context, lotteryCode, currentIssue, nextIssue string, receivedAt time.Time) {
	generation := periodBoundaryToken(lotteryCode, currentIssue, nextIssue)
	if w.boundaryPublisher == nil {
		return
	}
	event := schemeeventbus.PeriodBoundary{
		LotteryCode: lotteryCode, CurrentIssue: currentIssue, NextIssue: nextIssue,
		ReceivedAt: receivedAt.UTC(), Generation: generation,
	}
	if err := w.boundaryPublisher.PublishPeriodBoundary(ctx, event); err != nil {
		slog.Error("publish period boundary failed; draw persistence continues", "lottery", lotteryCode, "currentIssue", currentIssue, "nextIssue", nextIssue, "generation", generation, "err", err)
	}
}

func (w *Worker) updatePeriodState(lotteryCode, currentIssue, nextIssue string, drawnAt time.Time, intervalSec int) bool {
	if w.periodStateUpdater != nil {
		return w.periodStateUpdater(lotteryCode, currentIssue, nextIssue, drawnAt, intervalSec)
	}
	return lottery.UpdatePeriodState(lotteryCode, currentIssue, nextIssue, drawnAt, intervalSec)
}

// periodBoundaryToken is an idempotency token, not an ordering sequence.
// Canonical boundary identity is stable across worker and process restarts.
func periodBoundaryToken(lotteryCode, currentIssue, nextIssue string) uint64 {
	canonical := strings.Join([]string{
		strings.TrimSpace(lotteryCode),
		strings.TrimSpace(currentIssue),
		strings.TrimSpace(nextIssue),
	}, "\x00")
	digest := sha256.Sum256([]byte(canonical))
	token := binary.BigEndian.Uint64(digest[:8])
	if token == 0 {
		return 1
	}
	return token
}

// observeAcceptedBoundary refreshes boundary health only after the formal
// period-state update has accepted the same websocket boundary.
func (w *Worker) observeAcceptedBoundary(accepted bool, lotteryCode, currentIssue, nextIssue string, receivedMono time.Time, intervalSec int) {
	if w == nil || !accepted || w.boundaryHealth == nil || intervalSec <= 0 {
		return
	}
	w.boundaryHealth.Observe(lotteryCode, currentIssue, nextIssue, receivedMono, time.Duration(intervalSec)*time.Second)
	if w.recoveryReady == nil {
		return
	}
	if _, configured := w.recoveryReadyCodes[strings.TrimSpace(lotteryCode)]; !configured {
		return
	}
	snapshot := w.boundaryHealth.Snapshot(lotteryCode)
	if snapshot.CurrentIssue != strings.TrimSpace(currentIssue) || snapshot.NextIssue != strings.TrimSpace(nextIssue) {
		return
	}
	w.recoveryReadyOnce.Do(w.recoveryReady)
}

func (w *Worker) resolveLotteries(ctx context.Context, gameKey string) ([]lotteryTarget, error) {
	gameKey = strings.TrimSpace(gameKey)
	if gameKey == "" {
		return nil, nil
	}
	rows, err := w.pool.Query(ctx, `
SELECT code, COALESCE(play_template, '') FROM lottery_catalog
WHERE sale_status = 'on_sale'
  AND (guaji_ws_key = $1 OR outbound_lottery_code = $1 OR code = $1)`, gameKey)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	defer rows.Close()
	var out []lotteryTarget
	for rows.Next() {
		var t lotteryTarget
		if scanErr := rows.Scan(&t.code, &t.template); scanErr != nil {
			return nil, scanErr
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (w *Worker) drawIntervalSec(ctx context.Context, lotteryCode string) int {
	if w == nil || w.q == nil {
		return 0
	}
	return lottery.DrawIntervalSecForLottery(ctx, w.q, lotteryCode)
}
