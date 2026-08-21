package providerperiodtarget

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"sync"
	"time"

	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/lottery"
	"caipiao/backend/internal/schemebetting"
)

type SnapshotRecorder interface {
	RecordCurrentProviderPeriodSnapshot(context.Context, sqlcdb.RecordCurrentProviderPeriodSnapshotParams) (int64, error)
}

type snapshotCacheEntry struct {
	mu   sync.Mutex
	hash string
	id   int64
}

var currentSnapshotCache sync.Map // lotteryCode -> *snapshotCacheEntry

// Current requires a fresh provider websocket boundary for supported short
// lotteries. Other lotteries may fall back to the lottery-wide periods
// schedule. The selected target is copied into provider_period_snapshots.
func Current(
	ctx context.Context,
	recorder SnapshotRecorder,
	lotteryCode, sourcePeriod string,
	now time.Time,
) (schemebetting.PeriodSnapshot, int64, bool, error) {
	return current(ctx, recorder, lotteryCode, sourcePeriod, now, recordCurrentSnapshot)
}

// CurrentUncached records a provider snapshot without consulting or updating
// the process-wide committed snapshot cache. Callers whose recorder is scoped
// to a database transaction must use this path so rollback cannot publish an
// unusable snapshot ID to later transactions.
func CurrentUncached(
	ctx context.Context,
	recorder SnapshotRecorder,
	lotteryCode, sourcePeriod string,
	now time.Time,
) (schemebetting.PeriodSnapshot, int64, bool, error) {
	return current(ctx, recorder, lotteryCode, sourcePeriod, now, recordCurrentSnapshotUncached)
}

type currentSnapshotRecorder func(
	context.Context,
	SnapshotRecorder,
	string,
	schemebetting.PeriodSnapshot,
	string,
) (schemebetting.PeriodSnapshot, int64, bool, error)

func current(
	ctx context.Context,
	recorder SnapshotRecorder,
	lotteryCode, sourcePeriod string,
	now time.Time,
	record currentSnapshotRecorder,
) (schemebetting.PeriodSnapshot, int64, bool, error) {
	lotteryCode = strings.TrimSpace(lotteryCode)
	sourcePeriod = strings.TrimSpace(sourcePeriod)
	now = now.UTC()
	if recorder == nil || lotteryCode == "" {
		return schemebetting.PeriodSnapshot{}, 0, false, nil
	}
	if target, ok := freshShortPeriodWSNext(lotteryCode, sourcePeriod, now); ok {
		return record(ctx, recorder, lotteryCode, target, "guaji_draw_ws_next")
	}
	if lottery.RequiresFreshShortPeriodWSBetTarget(lotteryCode) {
		return schemebetting.PeriodSnapshot{}, 0, false, nil
	}
	if !lottery.PeriodsScheduleFresh(lotteryCode, lottery.PeriodsFallbackStaleAge, now) {
		return schemebetting.PeriodSnapshot{}, 0, false, nil
	}
	schedule, ok := lottery.PeriodsScheduleFor(lotteryCode)
	if !ok {
		return schemebetting.PeriodSnapshot{}, 0, false, nil
	}
	periodNo := strings.TrimSpace(schedule.CurrentPeriod)
	if periodNo == "" || periodNo == sourcePeriod {
		return schemebetting.PeriodSnapshot{}, 0, false, nil
	}
	closeAt := schedule.CloseAt.UTC()
	if schedule.ProvisionalClose && schedule.RealCloseAt.UTC().After(closeAt) {
		closeAt = schedule.RealCloseAt.UTC()
	}
	if closeAt.IsZero() || !now.Before(closeAt) {
		return schemebetting.PeriodSnapshot{}, 0, false, nil
	}
	observedAt := schedule.UpdatedAt.UTC()
	openAt := schedule.OpenStartAt.UTC()
	if openAt.IsZero() || openAt.After(now) || !openAt.Before(closeAt) {
		openAt = observedAt
	}
	if openAt.IsZero() || !openAt.Before(closeAt) {
		openAt = now
	}
	target := schemebetting.PeriodSnapshot{
		PeriodNo: periodNo, OpenAt: openAt, CloseAt: closeAt, ObservedAt: observedAt,
	}
	return record(ctx, recorder, lotteryCode, target, "guaji_periods_current")
}

// CurrentForInitialDispatch resolves callers that do not have a persisted
// source period. Formal short lotteries first derive a non-empty source from a
// fresh in-memory websocket boundary; other lotteries retain REST fallback.
func CurrentForInitialDispatch(
	ctx context.Context,
	recorder SnapshotRecorder,
	lotteryCode string,
	now time.Time,
) (schemebetting.PeriodSnapshot, int64, bool, error) {
	sourcePeriod := ""
	if lottery.RequiresFreshShortPeriodWSBetTarget(lotteryCode) {
		var ok bool
		sourcePeriod, ok = lottery.FreshShortPeriodWSCurrentIssue(lotteryCode, now)
		if !ok {
			return schemebetting.PeriodSnapshot{}, 0, false, nil
		}
	}
	return Current(ctx, recorder, lotteryCode, sourcePeriod, now)
}

func freshShortPeriodWSNext(lotteryCode, sourcePeriod string, now time.Time) (schemebetting.PeriodSnapshot, bool) {
	state, ok := lottery.FreshShortPeriodWSBetTarget(lotteryCode, sourcePeriod, now)
	if !ok {
		return schemebetting.PeriodSnapshot{}, false
	}
	openAt := state.CloseAt.UTC().Add(-time.Duration(state.IntervalSec) * time.Second)
	return schemebetting.PeriodSnapshot{
		PeriodNo: state.NextIssue, OpenAt: openAt, CloseAt: state.CloseAt.UTC(), ObservedAt: state.UpdatedAt.UTC(),
	}, true
}

func recordCurrentSnapshot(
	ctx context.Context,
	recorder SnapshotRecorder,
	lotteryCode string,
	target schemebetting.PeriodSnapshot,
	source string,
) (schemebetting.PeriodSnapshot, int64, bool, error) {
	return recordCurrentSnapshotWithCache(ctx, recorder, lotteryCode, target, source, true)
}

func recordCurrentSnapshotUncached(
	ctx context.Context,
	recorder SnapshotRecorder,
	lotteryCode string,
	target schemebetting.PeriodSnapshot,
	source string,
) (schemebetting.PeriodSnapshot, int64, bool, error) {
	return recordCurrentSnapshotWithCache(ctx, recorder, lotteryCode, target, source, false)
}

func recordCurrentSnapshotWithCache(
	ctx context.Context,
	recorder SnapshotRecorder,
	lotteryCode string,
	target schemebetting.PeriodSnapshot,
	source string,
	useCache bool,
) (schemebetting.PeriodSnapshot, int64, bool, error) {
	periodNo := strings.TrimSpace(target.PeriodNo)
	openAt := target.OpenAt.UTC()
	closeAt := target.CloseAt.UTC()
	observedAt := target.ObservedAt.UTC()
	canonical, err := json.Marshal(map[string]any{
		"lotteryCode": lotteryCode, "periodNo": periodNo,
		"openAt": openAt.Format(time.RFC3339Nano), "closeAt": closeAt.Format(time.RFC3339Nano),
		"observedAt": observedAt.Format(time.RFC3339Nano), "source": source,
	})
	if err != nil {
		return schemebetting.PeriodSnapshot{}, 0, false, err
	}
	digest := sha256.Sum256(canonical)
	snapshotHash := hex.EncodeToString(digest[:])
	params := sqlcdb.RecordCurrentProviderPeriodSnapshotParams{
		LotteryCode: lotteryCode, PeriodNo: periodNo, OpenAt: openAt, CloseAt: closeAt,
		ObservedAt: observedAt, Source: source,
		SnapshotHash: snapshotHash, RawPayload: canonical,
	}
	if !useCache {
		snapshotID, err := recorder.RecordCurrentProviderPeriodSnapshot(ctx, params)
		if err != nil {
			return schemebetting.PeriodSnapshot{}, 0, false, err
		}
		return target, snapshotID, true, nil
	}
	value, _ := currentSnapshotCache.LoadOrStore(lotteryCode, &snapshotCacheEntry{})
	cache := value.(*snapshotCacheEntry)
	cache.mu.Lock()
	defer cache.mu.Unlock()
	if cache.hash == snapshotHash && cache.id > 0 {
		return target, cache.id, true, nil
	}
	snapshotID, err := recorder.RecordCurrentProviderPeriodSnapshot(ctx, params)
	if err != nil {
		return schemebetting.PeriodSnapshot{}, 0, false, err
	}
	cache.hash = snapshotHash
	cache.id = snapshotID
	return target, snapshotID, true, nil
}
