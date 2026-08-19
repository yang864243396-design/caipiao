package schemes

import (
	"context"
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

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func uniqueInt64s(values []int64) []int64 {
	seen := make(map[int64]struct{}, len(values))
	out := make([]int64, 0, len(values))
	for _, value := range values {
		if value <= 0 {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
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

func prioritizeOpenBetWindowInstances(instances []sqlcdb.SchemeInstance) []sqlcdb.SchemeInstance {
	if len(instances) < 2 {
		return instances
	}
	open := make([]sqlcdb.SchemeInstance, 0, len(instances))
	closed := make([]sqlcdb.SchemeInstance, 0, len(instances)/2)
	for _, inst := range instances {
		if _, ok := lottery.StrictOpenIssueForGuajiBet(inst.LotteryCode); ok {
			open = append(open, inst)
		} else {
			closed = append(closed, inst)
		}
	}
	if len(open) == 0 || len(closed) == 0 {
		return instances
	}
	return append(open, closed...)
}

func (w *Worker) prefetchPeriodSync(ctx context.Context, codes []string) {
	_ = ctx
	if w == nil || w.periodRefresh == nil || len(codes) == 0 {
		return
	}
	seen := make(map[string]struct{}, len(codes))
	for _, code := range codes {
		if _, ok := seen[code]; ok {
			continue
		}
		seen[code] = struct{}{}
		w.requestPeriodRefresh(code)
	}
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
	if g == nil || g.w == nil {
		return false
	}
	v, _ := g.once.LoadOrStore(lotteryCode, &sync.Once{})
	v.(*sync.Once).Do(func() {
		g.w.requestPeriodRefresh(lotteryCode)
	})
	_, ok := lottery.StrictOpenIssueForGuajiBet(lotteryCode)
	return ok
}
