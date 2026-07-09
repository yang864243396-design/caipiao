package schemes

import (
	"context"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/guaji/periodsync"
	"caipiao/backend/internal/lottery"
)

type startSkipSnapshot struct {
	Period  string
	CloseAt time.Time
}

func readStartSkipSnapshot(ctx context.Context, syncer *periodsync.Syncer, lotteryCode string) (startSkipSnapshot, bool, error) {
	if syncer != nil {
		p, ca, ok, err := syncer.StartSkipSnapshot(ctx, lotteryCode)
		if err == nil && ok && strings.TrimSpace(p) != "" && !ca.IsZero() {
			return startSkipSnapshot{Period: strings.TrimSpace(p), CloseAt: ca.UTC()}, true, nil
		}
	}
	p, ca, ok := lottery.StartSkipSnapshotFromCache(lotteryCode)
	if !ok || strings.TrimSpace(p) == "" || ca.IsZero() {
		return startSkipSnapshot{}, false, nil
	}
	return startSkipSnapshot{Period: strings.TrimSpace(p), CloseAt: ca.UTC()}, true, nil
}

func resolveSkipPeriodCloseAt(lotteryCode, skipPeriod string) (time.Time, bool) {
	skipPeriod = strings.TrimSpace(skipPeriod)
	if skipPeriod == "" {
		return time.Time{}, false
	}
	ps, ok := lottery.PeriodsScheduleFor(lotteryCode)
	if !ok {
		return time.Time{}, false
	}
	if strings.TrimSpace(ps.StartSkipPeriod) == skipPeriod && !ps.StartSkipCloseAt.IsZero() {
		return ps.StartSkipCloseAt, true
	}
	if strings.TrimSpace(ps.CurrentPeriod) == skipPeriod && !ps.CloseAt.IsZero() {
		return ps.CloseAt, true
	}
	return time.Time{}, false
}

func ensureSchemeStartSkipSnapshot(
	ctx context.Context,
	q *sqlcdb.Queries,
	syncer *periodsync.Syncer,
	inst sqlcdb.SchemeInstance,
) (bool, error) {
	if inst.StartSkipCloseAt.Valid {
		return false, nil
	}
	if syncer != nil {
		_ = syncer.ForceRefresh(ctx, inst.LotteryCode)
	}
	if inst.LastSettledIssue.Valid {
		skipPeriod := strings.TrimSpace(inst.LastSettledIssue.String)
		if skipPeriod != "" {
			if ca, ok := resolveSkipPeriodCloseAt(inst.LotteryCode, skipPeriod); ok {
				n, err := q.BackfillSchemeInstanceStartSkipCloseAt(ctx, inst.ID, skipPeriod, pgtype.Timestamptz{
					Time: ca.UTC(), Valid: true,
				})
				return n > 0, err
			}
		}
	}
	_, ok, err := markSchemeStartPeriodSkipped(ctx, q, syncer, inst.ID, inst.LotteryCode)
	return ok, err
}

func skipCountdownSec(closeAt time.Time, now time.Time) int {
	if closeAt.IsZero() {
		return 0
	}
	rem := int(closeAt.Sub(now.UTC()).Round(time.Second).Seconds())
	if rem < 0 {
		return 0
	}
	return rem
}

func awaitNextBetCloseAt(inst sqlcdb.SchemeInstance, now time.Time) (time.Time, bool) {
	if inst.StatusReason != StatusReasonAwaitNextBet {
		return time.Time{}, false
	}
	now = now.UTC()
	skipPeriod := ""
	if inst.StartSkipPeriod.Valid {
		skipPeriod = strings.TrimSpace(inst.StartSkipPeriod.String)
	}
	cacheFresh := !lottery.PeriodsScheduleStale(inst.LotteryCode, now)
	if skipPeriod != "" && cacheFresh {
		if ca, ok := resolveSkipPeriodCloseAt(inst.LotteryCode, skipPeriod); ok {
			return ca.UTC(), true
		}
	}
	if inst.StartSkipCloseAt.Valid && !inst.StartSkipCloseAt.Time.IsZero() {
		return inst.StartSkipCloseAt.Time.UTC(), true
	}
	if skipPeriod != "" {
		if ca, ok := resolveSkipPeriodCloseAt(inst.LotteryCode, skipPeriod); ok {
			return ca.UTC(), true
		}
	}
	return time.Time{}, false
}

func awaitNextBetCountdownSec(inst sqlcdb.SchemeInstance, now time.Time) (int, bool) {
	if ca, ok := awaitNextBetCloseAt(inst, now); ok {
		return skipCountdownSec(ca, now), true
	}
	return 0, false
}
