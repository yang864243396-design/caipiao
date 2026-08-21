package drawsync

import (
	"context"
	"errors"
	"log/slog"
	"math/rand"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"caipiao/backend/internal/db"
	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/guaji"
	"caipiao/backend/internal/lottery"
	"caipiao/backend/internal/ws"
)

const maxDrawWSReconnectBackoff = 30 * time.Second

type drawSubscriber interface {
	SubscribeDraws(context.Context, func([]guaji.DrawEvent)) error
}

// Worker 订阅第三方开奖 WS，过滤忽略彩种，按 outbound_lottery_code 反查入库并广播 WS-5（T3）。
type Worker struct {
	pool             *db.Pool
	q                *sqlcdb.Queries
	client           drawSubscriber
	hub              *ws.Hub
	reconnectJitter  func(time.Duration) time.Duration
	waitRetry        func(context.Context, time.Duration) bool
	strategyNotifier interface {
		NotifyStrategyDraw(context.Context, string, string)
	}
}

func (w *Worker) SetStrategyNotifier(notifier interface {
	NotifyStrategyDraw(context.Context, string, string)
}) {
	if w != nil {
		w.strategyNotifier = notifier
	}
}

func NewWorker(pool *db.Pool, client *guaji.Client, hub *ws.Hub) *Worker {
	if pool == nil || client == nil || !client.Enabled() {
		return nil
	}
	return &Worker{pool: pool, q: sqlcdb.New(pool), client: client, hub: hub}
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
		err := w.client.SubscribeDraws(ctx, func(events []guaji.DrawEvent) {
			if len(events) > 0 {
				validFrame = true
			}
			for _, ev := range events {
				if ierr := w.Ingest(ctx, ev); ierr != nil {
					slog.Warn("guaji draw ingest failed", "gameKey", ev.GameKey, "periods", ev.Periods, "err", ierr)
				}
			}
		})
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
	if w == nil || w.q == nil {
		return errors.New("drawsync worker unavailable")
	}
	if ev.GameKey == "" || ev.Periods == "" {
		return nil
	}
	targets, err := w.resolveLotteries(ctx, ev.GameKey)
	if err != nil {
		return err
	}
	if len(targets) == 0 {
		return nil // outbound_lottery_code 未配置该键，跳过
	}
	for _, tgt := range targets {
		balls := ev.Balls.BallsFor(tgt.template)
		if len(balls) == 0 {
			continue
		}
		drawnAt := ev.DrawnAt
		if drawnAt.IsZero() {
			drawnAt = time.Now()
		}
		if intervalSec := w.drawIntervalSec(ctx, tgt.code); intervalSec > 0 {
			lottery.UpdatePeriodState(tgt.code, ev.Periods, ev.NextPeriods, drawnAt, intervalSec)
		}
		_, inserted, err := lottery.PersistDrawFactFromBalls(ctx, w.q, w.hub, tgt.code, ev.Periods, balls, drawnAt, lottery.DrawFactMeta{
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
	cat, err := w.q.GetLotteryCatalogByCode(ctx, lotteryCode)
	if err != nil || !cat.DrawInterval.Valid {
		return 0
	}
	return lottery.ParseDrawIntervalSec(cat.DrawInterval.String)
}
