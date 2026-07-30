package schemes

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/lottery"
)

const (
	defaultSchemeWorkerConcurrency      = 32
	maxSchemeWorkerConcurrency          = 256
	defaultSchemeWorkerPlaceConcurrency = 16
	maxSchemeWorkerPlaceConcurrency     = 64
	periodPrefetchConcurrency           = 8
	guajiPlaceSafeRetryAttempts         = 3
	guajiPlaceSafeRetryBackoff          = 250 * time.Millisecond
)

func clampSchemeWorkerConcurrency(n int) int {
	if n <= 0 {
		return defaultSchemeWorkerConcurrency
	}
	if n > maxSchemeWorkerConcurrency {
		return maxSchemeWorkerConcurrency
	}
	return n
}

// SetConcurrency 设置单 tick 内并行处理实例的上限（有界池）。
func (w *Worker) SetConcurrency(n int) {
	if w == nil {
		return
	}
	w.concurrency = int32(clampSchemeWorkerConcurrency(n))
}

func clampSchemeWorkerPlaceConcurrency(n int) int {
	if n < 0 {
		return defaultSchemeWorkerPlaceConcurrency
	}
	if n == 0 {
		return 0 // 0 = 不额外限流
	}
	if n > maxSchemeWorkerPlaceConcurrency {
		return maxSchemeWorkerPlaceConcurrency
	}
	return n
}

// SetPlaceConcurrency 限制同时打第三方 PlaceBet 的路数（与实例 tick 并发解耦）。
func (w *Worker) SetPlaceConcurrency(n int) {
	if w == nil {
		return
	}
	n = clampSchemeWorkerPlaceConcurrency(n)
	if n == 0 {
		w.placeSem = nil
		return
	}
	w.placeSem = make(chan struct{}, n)
}

func (w *Worker) acquirePlaceSlot(ctx context.Context) error {
	if w == nil || w.placeSem == nil {
		return nil
	}
	select {
	case w.placeSem <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (w *Worker) releasePlaceSlot() {
	if w == nil || w.placeSem == nil {
		return
	}
	<-w.placeSem
}

func uniqueLotteryCodes(rows []sqlcdb.ListRunningSchemeInstancesRow) []string {
	if len(rows) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, 16)
	out := make([]string, 0, 16)
	for _, row := range rows {
		code := row.LotteryCode
		if code == "" {
			continue
		}
		if _, ok := seen[code]; ok {
			continue
		}
		seen[code] = struct{}{}
		out = append(out, code)
	}
	return out
}

func uniqueMemberIDs(rows []sqlcdb.ListRunningSchemeInstancesRow) []int64 {
	if len(rows) == 0 {
		return nil
	}
	seen := make(map[int64]struct{}, 16)
	out := make([]int64, 0, 16)
	for _, row := range rows {
		if row.MemberID <= 0 {
			continue
		}
		if _, ok := seen[row.MemberID]; ok {
			continue
		}
		seen[row.MemberID] = struct{}{}
		out = append(out, row.MemberID)
	}
	return out
}

// prioritizeOpenBetWindow 将已开盘彩种的实例排到前面（保留各组内 updated_at 顺序），
// 有界并发打满时优先保证窗口内下注。
func prioritizeOpenBetWindow(rows []sqlcdb.ListRunningSchemeInstancesRow) []sqlcdb.ListRunningSchemeInstancesRow {
	if len(rows) < 2 {
		return rows
	}
	open := make([]sqlcdb.ListRunningSchemeInstancesRow, 0, len(rows))
	closed := make([]sqlcdb.ListRunningSchemeInstancesRow, 0, len(rows)/2)
	for _, row := range rows {
		if _, ok := lottery.StrictOpenIssueForGuajiBet(row.LotteryCode); ok {
			open = append(open, row)
		} else {
			closed = append(closed, row)
		}
	}
	if len(open) == 0 || len(closed) == 0 {
		return rows
	}
	return append(open, closed...)
}

func (w *Worker) prefetchPeriodSync(ctx context.Context, codes []string) {
	if w == nil || w.periodSync == nil || len(codes) == 0 {
		return
	}
	sem := make(chan struct{}, periodPrefetchConcurrency)
	var wg sync.WaitGroup
	for _, code := range codes {
		wg.Add(1)
		sem <- struct{}{}
		go func(lotteryCode string) {
			defer wg.Done()
			defer func() { <-sem }()
			if err := w.periodSync.EnsureFreshIfStale(ctx, lotteryCode); err != nil {
				slog.Warn("scheme worker periods prefetch failed", "lottery", lotteryCode, "err", err)
			}
		}(code)
	}
	wg.Wait()
}

func (w *Worker) preloadPlanMultipliers(ctx context.Context, memberIDs []int64) map[int64]float64 {
	out := make(map[int64]float64, len(memberIDs))
	for _, mid := range memberIDs {
		out[mid] = w.memberPlanMultiplier(ctx, mid)
	}
	return out
}

// betWindowGate 同一 tick 内每个彩种最多 ForceRefresh 一次，避免封盘时上万实例重复拉 periods。
type betWindowGate struct {
	w    *Worker
	once sync.Map // lotteryCode -> *sync.Once
}

func newBetWindowGate(w *Worker) *betWindowGate {
	return &betWindowGate{w: w}
}

func (g *betWindowGate) ensureOpen(ctx context.Context, lotteryCode string) bool {
	if _, ok := lottery.StrictOpenIssueForGuajiBet(lotteryCode); ok {
		return true
	}
	if g == nil || g.w == nil || g.w.periodSync == nil {
		return false
	}
	v, _ := g.once.LoadOrStore(lotteryCode, &sync.Once{})
	v.(*sync.Once).Do(func() {
		refreshCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
		defer cancel()
		if err := g.w.periodSync.ForceRefresh(refreshCtx, lotteryCode); err != nil {
			slog.Debug("scheme worker force refresh before bet failed", "lottery", lotteryCode, "err", err)
		}
	})
	_, ok := lottery.StrictOpenIssueForGuajiBet(lotteryCode)
	return ok
}
