package historysync

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"caipiao/backend/internal/db"
	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/guaji"
	"caipiao/backend/internal/lottery"
	"caipiao/backend/internal/ws"
)

const (
	defaultSyncInterval = 15 * time.Second
	defaultPageLimit    = 30
)

type lotteryTarget struct {
	code     string
	template string
}

// Worker 周期性拉取第三方历史开奖 REST（文档 §5），写入 lottery_draws。
type Worker struct {
	pool     *db.Pool
	q        *sqlcdb.Queries
	client   *guaji.Client
	hub      *ws.Hub
	interval time.Duration
	pageSize int
}

func NewWorker(pool *db.Pool, client *guaji.Client, hub *ws.Hub) *Worker {
	if pool == nil || client == nil || !client.Enabled() {
		return nil
	}
	return &Worker{
		pool:     pool,
		q:        sqlcdb.New(pool),
		client:   client,
		hub:      hub,
		interval: defaultSyncInterval,
		pageSize: defaultPageLimit,
	}
}

func (w *Worker) Run(ctx context.Context) {
	if w == nil {
		return
	}
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	slog.Info("guaji history draw sync started", "interval", w.interval.String(), "pageSize", w.pageSize)
	w.tick(ctx)
	for {
		select {
		case <-ctx.Done():
			slog.Info("guaji history draw sync stopped")
			return
		case <-ticker.C:
			w.tick(ctx)
		}
	}
}

func (w *Worker) tick(ctx context.Context) {
	targets, err := w.listTargets(ctx)
	if err != nil {
		slog.Warn("guaji history draw sync list targets failed", "err", err)
		return
	}
	w.syncTargets(ctx, targets)
}

// SyncLottery 按需拉取单彩种最近历史开奖（玩法详情展示前补齐 lottery_draws 缺口）。
func (w *Worker) SyncLottery(ctx context.Context, lotteryCode string) error {
	if w == nil || w.q == nil || w.client == nil {
		return nil
	}
	lotteryCode = strings.TrimSpace(lotteryCode)
	if lotteryCode == "" {
		return nil
	}
	template, err := w.lotteryTemplate(ctx, lotteryCode)
	if err != nil {
		return err
	}
	w.syncTargets(ctx, []lotteryTarget{{code: lotteryCode, template: template}})
	return nil
}

func (w *Worker) lotteryTemplate(ctx context.Context, lotteryCode string) (string, error) {
	row, err := w.q.GetLotteryCatalogByCode(ctx, lotteryCode)
	if err != nil {
		return "", err
	}
	if row.PlayTemplate.Valid {
		return strings.TrimSpace(row.PlayTemplate.String), nil
	}
	return "", nil
}

func (w *Worker) syncTargets(ctx context.Context, targets []lotteryTarget) {
	if len(targets) == 0 {
		return
	}
	byPath := map[string][]lotteryTarget{}
	for _, tgt := range targets {
		path := HistoryAPIPathForCode(tgt.code)
		if path == "" {
			continue
		}
		byPath[path] = append(byPath[path], tgt)
	}
	dialFails := 0
	for apiPath, lotteries := range byPath {
		if dialFails >= 2 {
			slog.Warn("guaji history draw sync abort remaining paths after dial failures", "fails", dialFails)
			return
		}
		logs, err := w.client.FetchHistoryDrawLogs(ctx, apiPath, 1, w.pageSize)
		if err != nil {
			slog.Warn("guaji history draw sync fetch failed", "path", apiPath, "err", err)
			s := strings.ToLower(err.Error())
			if strings.Contains(s, "dial ") || strings.Contains(s, "i/o timeout") || strings.Contains(s, "all ips failed") {
				dialFails++
			}
			continue
		}
		dialFails = 0
		inserted := 0
		for _, row := range logs {
			for _, tgt := range lotteries {
				if n, ierr := w.ingestRow(ctx, tgt, row); ierr != nil {
					slog.Debug("guaji history draw ingest skip", "lottery", tgt.code, "period", row.Periods, "err", ierr)
				} else {
					inserted += n
				}
			}
		}
		if inserted > 0 {
			slog.Debug("guaji history draw sync ok", "path", apiPath, "fetched", len(logs), "inserted", inserted)
		}
	}
}

func (w *Worker) listTargets(ctx context.Context) ([]lotteryTarget, error) {
	rows, err := w.pool.Query(ctx, `
SELECT code, COALESCE(play_template, '')
FROM lottery_catalog
WHERE sale_status = 'on_sale' AND on_sale = true
ORDER BY code`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []lotteryTarget
	for rows.Next() {
		var t lotteryTarget
		if err := rows.Scan(&t.code, &t.template); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (w *Worker) ingestRow(ctx context.Context, tgt lotteryTarget, row guaji.HistoryDrawLog) (int, error) {
	balls := row.Balls.BallsFor(tgt.template)
	if len(balls) == 0 {
		return 0, nil
	}
	drawnAt := row.DrawnAt
	if drawnAt.IsZero() {
		drawnAt = time.Now().UTC()
	}
	_, inserted, err := lottery.PersistDrawFromBalls(ctx, w.q, w.hub, tgt.code, row.Periods, balls, drawnAt)
	if err != nil {
		return 0, err
	}
	if inserted {
		lottery.LogDrawCloseToIngestLatency(tgt.code, row.Periods, "history_rest", drawnAt)
		return 1, nil
	}
	return 0, nil
}
