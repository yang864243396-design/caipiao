// Package periodaudit 期号归属自检：持续比对下注链路与开奖链路取到的期号是否同族。
//
// 两条链路各走各的映射——下注按 lottery_catalog.outbound_lottery_code 调第三方 periods，
// 开奖按 guaji_ws_key 收 WS 广播。映射到不同彩种时两边都不报错，只是注单期号永远
// 查不到开奖号，盈亏还是照算。极速彩那个 bug 就是这样潜伏了很久。
//
// 自检是周期性而非只在启动跑一次：进程刚起来时 periods 缓存是空的，
// 那一刻什么都比不出来。
package periodaudit

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"caipiao/backend/internal/db"
	"caipiao/backend/internal/lottery"
)

const defaultInterval = 5 * time.Minute

// alertCooldown 同一彩种的重复告警间隔。映射错了会每轮都命中，
// 不压一下会把日志刷满、反而让人忽略它。
const alertCooldown = time.Hour

type Worker struct {
	pool     *db.Pool
	interval time.Duration

	mu         sync.Mutex
	lastAlert  map[string]time.Time
	lastStatus map[string]lottery.PeriodFamilyStatus
}

func NewWorker(pool *db.Pool) *Worker {
	if pool == nil {
		return nil
	}
	return &Worker{
		pool:       pool,
		interval:   defaultInterval,
		lastAlert:  map[string]time.Time{},
		lastStatus: map[string]lottery.PeriodFamilyStatus{},
	}
}

func (w *Worker) Run(ctx context.Context) {
	if w == nil {
		return
	}
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	slog.Info("period family audit started", "interval", w.interval.String())
	for {
		select {
		case <-ctx.Done():
			slog.Info("period family audit stopped")
			return
		case <-ticker.C:
			w.tick(ctx, time.Now())
		}
	}
}

type latestDraw struct {
	LotteryCode string
	IssueNo     string
	DrawnAt     time.Time
}

// tick 逐个在售彩种比对：下注期号取 periods 缓存，开奖期号取 lottery_draws 最新一期。
func (w *Worker) tick(ctx context.Context, now time.Time) {
	draws, err := w.latestDraws(ctx)
	if err != nil {
		slog.Warn("period family audit query failed", "err", err)
		return
	}
	for _, d := range draws {
		// 开奖同步本身断了的话，比对没有意义——那是 drawsync 的问题，不是映射问题
		if now.Sub(d.DrawnAt) > 30*time.Minute {
			continue
		}
		ps, ok := lottery.PeriodsScheduleFor(d.LotteryCode)
		if !ok {
			continue
		}
		status, note := lottery.ComparePeriodFamily(ps.CurrentPeriod, d.IssueNo)
		w.report(d.LotteryCode, status, note, now)
	}
}

func (w *Worker) report(code string, status lottery.PeriodFamilyStatus, note string, now time.Time) {
	w.mu.Lock()
	defer w.mu.Unlock()

	prev := w.lastStatus[code]
	w.lastStatus[code] = status

	if status != lottery.PeriodFamilyMismatch {
		if prev == lottery.PeriodFamilyMismatch {
			slog.Info("period family recovered", "lotteryCode", code)
			delete(w.lastAlert, code)
		}
		return
	}
	if last, ok := w.lastAlert[code]; ok && now.Sub(last) < alertCooldown {
		return
	}
	w.lastAlert[code] = now
	slog.Error("period family mismatch: 下注链路与开奖链路期号不同族",
		"lotteryCode", code, "detail", note,
		"hint", "核对 lottery_catalog 的 outbound_lottery_code 与 guaji_ws_key 是否指向同一彩种")
}

func (w *Worker) latestDraws(ctx context.Context) ([]latestDraw, error) {
	rows, err := w.pool.Query(ctx, `
SELECT DISTINCT ON (d.lottery_code) d.lottery_code, d.issue_no, d.drawn_at
FROM lottery_draws d
JOIN lottery_catalog c ON c.code = d.lottery_code AND c.sale_status = 'on_sale'
ORDER BY d.lottery_code, d.drawn_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []latestDraw
	for rows.Next() {
		var d latestDraw
		if err := rows.Scan(&d.LotteryCode, &d.IssueNo, &d.DrawnAt); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}
