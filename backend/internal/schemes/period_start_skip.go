package schemes

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/guaji/periodsync"
)

// readStartSkipPeriod 读平台 periods 缓存；缺失/过期时由 Syncer 兜底拉取一次。
func readStartSkipPeriod(ctx context.Context, syncer *periodsync.Syncer, lotteryCode string) (string, bool, error) {
	snap, ok, err := readStartSkipSnapshot(ctx, syncer, lotteryCode)
	if err != nil || !ok {
		return "", false, err
	}
	return snap.Period, true, nil
}

// markSchemeStartPeriodSkipped 跳过最近一期（快照期号 + 封盘时刻写入 DB）。
// 仅读本地 periods 缓存：开启接口绝不同步等待第三方 ForceRefresh（其全局锁会被 worker 拉单拖死）。
func markSchemeStartPeriodSkipped(
	ctx context.Context,
	q *sqlcdb.Queries,
	syncer *periodsync.Syncer,
	instanceID, lotteryCode string,
) (skipPeriod string, ok bool, err error) {
	snap, ok, err := readStartSkipSnapshot(ctx, syncer, lotteryCode)
	if err != nil || !ok || strings.TrimSpace(snap.Period) == "" || snap.CloseAt.IsZero() {
		return "", false, err
	}
	n, err := q.SkipSchemeInstanceStartPeriodEx(ctx, instanceID, snap.Period, pgtype.Timestamptz{
		Time:  snap.CloseAt.UTC(),
		Valid: true,
	})
	if err != nil || n == 0 {
		return "", false, err
	}
	slog.Info("scheme skipped start period from cache",
		"instanceId", instanceID, "period", snap.Period, "closeAt", snap.CloseAt.UTC().Format(time.RFC3339))
	return snap.Period, true, nil
}
