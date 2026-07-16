package periodsync

import (
	"context"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"

	"caipiao/backend/internal/db"
	"caipiao/backend/internal/guaji"
	"caipiao/backend/internal/guaji/accountsvc"
	"caipiao/backend/internal/lottery"
)

const (
	defaultSyncInterval = 3 * time.Second
	targetsCacheTTL     = 15 * time.Second
	tokenCacheTTL       = 60 * time.Second
	maxRefreshPerTick   = 4
	dialFailBackoff     = 20 * time.Second
)

// Worker 周期性拉取第三方 /api/web_bets/lott/periods，更新封盘倒计时缓存。
type Worker struct {
	pool     *db.Pool
	client   *guaji.Client
	accounts *accountsvc.Service
	interval time.Duration

	mu            sync.Mutex
	cachedToken   string
	tokenUntil    time.Time
	targetsCache  []syncTarget
	targetsUntil  time.Time
	backoffUntil  time.Time
}

func NewWorker(pool *db.Pool, client *guaji.Client, accounts *accountsvc.Service) *Worker {
	if pool == nil || client == nil || !client.Enabled() || accounts == nil {
		return nil
	}
	return &Worker{
		pool:     pool,
		client:   client,
		accounts: accounts,
		interval: defaultSyncInterval,
	}
}

func (w *Worker) Run(ctx context.Context) {
	if w == nil {
		return
	}
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	slog.Info("guaji period sync started", "interval", w.interval.String())
	for {
		select {
		case <-ctx.Done():
			slog.Info("guaji period sync stopped")
			return
		case <-ticker.C:
			w.tick(ctx)
		}
	}
}

func (w *Worker) tick(ctx context.Context) {
	now := time.Now()
	w.mu.Lock()
	if now.Before(w.backoffUntil) {
		w.mu.Unlock()
		return
	}
	w.mu.Unlock()

	targets, err := w.syncTargets(ctx)
	if err != nil {
		slog.Warn("guaji period sync list targets failed", "err", err)
		return
	}
	if len(targets) == 0 {
		return
	}

	token, err := w.syncToken(ctx)
	if err != nil {
		slog.Debug("guaji period sync: no token", "err", err)
		return
	}

	refreshed := 0
	dialFails := 0
	for _, tgt := range targets {
		if refreshed >= maxRefreshPerTick {
			break
		}
		if !lottery.PeriodsScheduleNeedsRefresh(tgt.lotteryCode, now) {
			continue
		}
		if err := w.syncOne(ctx, token, tgt, now); err != nil {
			if guaji.ClassifyUpstreamError(err).IsTokenInvalid {
				w.invalidateToken()
			}
			if isDialOrTimeoutErr(err) {
				dialFails++
			}
			slog.Warn("guaji period sync failed", "lottery", tgt.lotteryCode, "gameId", tgt.gameID, "err", err)
			continue
		}
		refreshed++
		dialFails = 0
	}
	// 连续拨号失败：全局退避，避免 30+ 彩种轮询堵死可用 CDN IP。
	if dialFails >= 2 {
		until := time.Now().Add(dialFailBackoff)
		w.mu.Lock()
		w.backoffUntil = until
		w.mu.Unlock()
		slog.Warn("guaji period sync dial backoff", "until", until.Format(time.RFC3339), "fails", dialFails)
	}
}

func isDialOrTimeoutErr(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "dial ") ||
		strings.Contains(s, "i/o timeout") ||
		strings.Contains(s, "connection refused") ||
		strings.Contains(s, "no such host") ||
		strings.Contains(s, "all ips failed")
}

type syncTarget struct {
	lotteryCode string
	gameID      int
}

func (w *Worker) syncTargets(ctx context.Context) ([]syncTarget, error) {
	now := time.Now()
	w.mu.Lock()
	if len(w.targetsCache) > 0 && now.Before(w.targetsUntil) {
		out := append([]syncTarget(nil), w.targetsCache...)
		w.mu.Unlock()
		return out, nil
	}
	w.mu.Unlock()

	targets, err := w.listSyncTargets(ctx)
	if err != nil {
		return nil, err
	}

	w.mu.Lock()
	w.targetsCache = append([]syncTarget(nil), targets...)
	w.targetsUntil = now.Add(targetsCacheTTL)
	w.mu.Unlock()
	return targets, nil
}

func (w *Worker) syncToken(ctx context.Context) (string, error) {
	now := time.Now()
	w.mu.Lock()
	if w.cachedToken != "" && now.Before(w.tokenUntil) {
		token := w.cachedToken
		w.mu.Unlock()
		return token, nil
	}
	w.mu.Unlock()

	token, err := w.accounts.SyncAccessToken(ctx)
	if err != nil {
		return "", err
	}

	w.mu.Lock()
	w.cachedToken = token
	w.tokenUntil = now.Add(tokenCacheTTL)
	w.mu.Unlock()
	return token, nil
}

func (w *Worker) invalidateToken() {
	w.mu.Lock()
	w.cachedToken = ""
	w.tokenUntil = time.Time{}
	w.mu.Unlock()
}

func (w *Worker) listSyncTargets(ctx context.Context) ([]syncTarget, error) {
	// 运行中方案的彩种优先，避免全量目录同步时饿死挂机下注所需的 periods。
	rows, err := w.pool.Query(ctx, `
SELECT c.code,
       COALESCE(NULLIF(TRIM(c.outbound_lottery_code), ''), c.code) AS game_key
FROM lottery_catalog c
LEFT JOIN (
  SELECT DISTINCT lottery_code
  FROM scheme_instances
  WHERE status = 'running'
) r ON r.lottery_code = c.code
WHERE c.sale_status = 'on_sale'
  AND c.on_sale = true
ORDER BY (r.lottery_code IS NOT NULL) DESC, c.code`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	seen := map[string]bool{}
	var out []syncTarget
	for rows.Next() {
		var lotteryCode, gameKey string
		if err := rows.Scan(&lotteryCode, &gameKey); err != nil {
			return nil, err
		}
		if tgt, ok := mergeSyncTarget(seen, lotteryCode, gameKey); ok {
			out = append(out, tgt)
		}
	}
	return out, rows.Err()
}

// mergeSyncTarget 解析一行 DISTINCT lottery_code + game_key，去重并过滤非法 game_id。
func mergeSyncTarget(seen map[string]bool, lotteryCode, gameKey string) (syncTarget, bool) {
	lotteryCode = strings.TrimSpace(lotteryCode)
	if lotteryCode == "" || seen[lotteryCode] {
		return syncTarget{}, false
	}
	gameID, err := strconv.Atoi(strings.TrimSpace(gameKey))
	if err != nil || gameID <= 0 {
		return syncTarget{}, false
	}
	seen[lotteryCode] = true
	return syncTarget{lotteryCode: lotteryCode, gameID: gameID}, true
}

func (w *Worker) syncOne(ctx context.Context, token string, tgt syncTarget, now time.Time) error {
	periods, _, err := w.client.FetchLottPeriods(ctx, token, tgt.gameID, workerNumPeriods)
	if err != nil {
		return err
	}
	applyPeriodsListToCache(tgt.lotteryCode, periods, now)
	return nil
}
