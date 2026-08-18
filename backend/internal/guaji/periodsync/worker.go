package periodsync

import (
	"context"
	"fmt"
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
	defaultSyncInterval       = 3 * time.Second
	targetsCacheTTL           = 15 * time.Second
	tokenCacheTTL             = 60 * time.Second
	defaultRefreshConcurrency = 4
	refreshRequestQueueSize   = 128
	refreshRequestTimeout     = 1200 * time.Millisecond
	refreshFailureBackoff     = 2 * time.Second
	maxRefreshPerTick         = 4 // legacy synchronous fallback
	dialFailBackoff           = 20 * time.Second
)

type RefreshDiagnostics struct {
	LotteryCode         string        `json:"lotteryCode"`
	LastAttemptAt       time.Time     `json:"lastAttemptAt,omitempty"`
	LastSuccessAt       time.Time     `json:"lastSuccessAt,omitempty"`
	LastDuration        time.Duration `json:"lastDuration,omitempty"`
	ConsecutiveFailures int           `json:"consecutiveFailures"`
	NextAllowedAt       time.Time     `json:"nextAllowedAt,omitempty"`
	LastError           string        `json:"lastError,omitempty"`
}

var runtimeRefreshDiagnostics sync.Map // lotteryCode -> RefreshDiagnostics

type refreshState struct {
	RefreshDiagnostics
	queued   bool
	inFlight bool
}

// Worker 周期性拉取第三方 /api/web_bets/lott/periods，更新封盘倒计时缓存。
type Worker struct {
	pool     *db.Pool
	client   *guaji.Client
	accounts *accountsvc.Service
	interval time.Duration

	mu           sync.Mutex
	cachedToken  string
	tokenUntil   time.Time
	targetsCache []syncTarget
	targetsUntil time.Time
	backoffUntil time.Time

	refreshMu          sync.Mutex
	refreshStates      map[string]*refreshState
	highRefreshQueue   chan string
	normalRefreshQueue chan string
	refreshStart       sync.Once
	refreshWait        sync.WaitGroup
	refreshConcurrency int
	refreshTimeout     time.Duration
	refreshFn          func(context.Context, string) error
	refreshFunc        func(context.Context, string) error
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
	w.startRefreshWorkers(ctx)
	defer w.refreshWait.Wait()
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

func (w *Worker) legacyTick(ctx context.Context) {
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

func (w *Worker) tick(ctx context.Context) {
	targets, err := w.syncTargets(ctx)
	if err != nil {
		slog.Warn("guaji period sync list targets failed", "err", err)
		return
	}
	for _, target := range targets {
		if lottery.PeriodsScheduleNeedsRefresh(target.lotteryCode, time.Now()) {
			w.RequestRefresh(target.lotteryCode)
		}
	}
}

func (w *Worker) ensureRefreshQueues() {
	if w == nil {
		return
	}
	w.refreshMu.Lock()
	defer w.refreshMu.Unlock()
	if w.refreshStates == nil {
		w.refreshStates = make(map[string]*refreshState)
	}
	if w.highRefreshQueue == nil {
		w.highRefreshQueue = make(chan string, refreshRequestQueueSize)
	}
	if w.normalRefreshQueue == nil {
		w.normalRefreshQueue = make(chan string, refreshRequestQueueSize)
	}
}

func (w *Worker) startRefreshWorkers(ctx context.Context) {
	if w == nil {
		return
	}
	w.ensureRefreshQueues()
	w.refreshStart.Do(func() {
		n := w.refreshConcurrency
		if n <= 0 {
			n = defaultRefreshConcurrency
		}
		for range n {
			w.refreshWait.Add(1)
			go func() {
				defer w.refreshWait.Done()
				w.refreshLoop(ctx)
			}()
		}
	})
}

func (w *Worker) RequestRefresh(lotteryCode string) {
	if w == nil {
		return
	}
	lotteryCode = strings.TrimSpace(lotteryCode)
	if lotteryCode == "" {
		return
	}
	w.ensureRefreshQueues()
	now := time.Now().UTC()
	w.refreshMu.Lock()
	state := w.refreshStates[lotteryCode]
	if state == nil {
		state = &refreshState{RefreshDiagnostics: RefreshDiagnostics{LotteryCode: lotteryCode}}
		w.refreshStates[lotteryCode] = state
	}
	if state.queued || state.inFlight || now.Before(state.NextAllowedAt) {
		w.refreshMu.Unlock()
		return
	}
	state.queued = true
	queue := w.normalRefreshQueue
	if isShortPeriodLottery(lotteryCode) {
		queue = w.highRefreshQueue
	}
	w.refreshMu.Unlock()
	select {
	case queue <- lotteryCode:
	default:
		w.refreshMu.Lock()
		state.queued = false
		w.refreshMu.Unlock()
	}
}

func isShortPeriodLottery(lotteryCode string) bool {
	if ps, ok := lottery.PeriodsScheduleFor(lotteryCode); ok && ps.PeriodDurationSec > 0 {
		return ps.PeriodDurationSec <= 15
	}
	code := strings.ToLower(strings.TrimSpace(lotteryCode))
	return strings.Contains(code, "_3s") || strings.Contains(code, "_15s")
}

func (w *Worker) refreshLoop(ctx context.Context) {
	for {
		var code string
		select {
		case code = <-w.highRefreshQueue:
		default:
			select {
			case <-ctx.Done():
				return
			case code = <-w.highRefreshQueue:
			case code = <-w.normalRefreshQueue:
			}
		}
		w.runRefresh(ctx, code)
	}
}

func (w *Worker) runRefresh(parent context.Context, lotteryCode string) {
	started := time.Now().UTC()
	w.refreshMu.Lock()
	state := w.refreshStates[lotteryCode]
	if state == nil {
		w.refreshMu.Unlock()
		return
	}
	state.queued = false
	state.inFlight = true
	state.LastAttemptAt = started
	w.refreshMu.Unlock()

	timeout := w.refreshTimeout
	if timeout <= 0 {
		timeout = refreshRequestTimeout
	}
	ctx, cancel := context.WithTimeout(parent, timeout)
	err := w.refreshLottery(ctx, lotteryCode)
	cancel()
	finished := time.Now().UTC()

	w.refreshMu.Lock()
	state.inFlight = false
	state.LastDuration = finished.Sub(started)
	if err == nil {
		state.LastSuccessAt = finished
		state.ConsecutiveFailures = 0
		state.NextAllowedAt = time.Time{}
		state.LastError = ""
	} else {
		state.ConsecutiveFailures++
		state.NextAllowedAt = finished.Add(refreshFailureBackoff)
		state.LastError = err.Error()
	}
	snapshot := state.RefreshDiagnostics
	w.refreshMu.Unlock()
	runtimeRefreshDiagnostics.Store(lotteryCode, snapshot)
	if err != nil {
		if guaji.ClassifyUpstreamError(err).IsTokenInvalid {
			w.invalidateToken()
		}
		slog.Warn("guaji period sync failed", "lottery", lotteryCode, "err", err)
	}
}

func (w *Worker) refreshLottery(ctx context.Context, lotteryCode string) error {
	if w.refreshFunc != nil {
		return w.refreshFunc(ctx, lotteryCode)
	}
	if w.refreshFn != nil {
		return w.refreshFn(ctx, lotteryCode)
	}
	targets, err := w.syncTargets(ctx)
	if err != nil {
		return err
	}
	for _, target := range targets {
		if target.lotteryCode != lotteryCode {
			continue
		}
		token, err := w.syncToken(ctx)
		if err != nil {
			return err
		}
		return w.syncOne(ctx, token, target, time.Now())
	}
	return fmt.Errorf("period sync target not found: %s", lotteryCode)
}

func (w *Worker) Diagnostics(lotteryCode string) (RefreshDiagnostics, bool) {
	if w == nil {
		return RefreshDiagnostics{}, false
	}
	lotteryCode = strings.TrimSpace(lotteryCode)
	w.refreshMu.Lock()
	defer w.refreshMu.Unlock()
	state := w.refreshStates[lotteryCode]
	if state == nil {
		return RefreshDiagnostics{}, false
	}
	return state.RefreshDiagnostics, true
}

func DiagnosticsForLottery(lotteryCode string) (RefreshDiagnostics, bool) {
	lotteryCode = strings.TrimSpace(lotteryCode)
	v, ok := runtimeRefreshDiagnostics.Load(lotteryCode)
	if !ok {
		return RefreshDiagnostics{}, false
	}
	diag, ok := v.(RefreshDiagnostics)
	return diag, ok
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
