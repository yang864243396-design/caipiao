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

// Current uses the existing lottery-wide periods schedule as the sole current
// target. provider_period_snapshots receives an audit copy but never selects
// or overrides the open period.
func Current(
	ctx context.Context,
	recorder SnapshotRecorder,
	lotteryCode, sourcePeriod string,
	now time.Time,
) (schemebetting.PeriodSnapshot, int64, bool, error) {
	lotteryCode = strings.TrimSpace(lotteryCode)
	sourcePeriod = strings.TrimSpace(sourcePeriod)
	now = now.UTC()
	if recorder == nil || lotteryCode == "" || !lottery.PeriodsScheduleFresh(lotteryCode, lottery.PeriodsFallbackStaleAge, now) {
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
	canonical, err := json.Marshal(map[string]any{
		"lotteryCode": lotteryCode, "periodNo": periodNo,
		"openAt": openAt.Format(time.RFC3339Nano), "closeAt": closeAt.Format(time.RFC3339Nano),
		"observedAt": observedAt.Format(time.RFC3339Nano), "source": "guaji_periods_current",
	})
	if err != nil {
		return schemebetting.PeriodSnapshot{}, 0, false, err
	}
	digest := sha256.Sum256(canonical)
	snapshotHash := hex.EncodeToString(digest[:])
	value, _ := currentSnapshotCache.LoadOrStore(lotteryCode, &snapshotCacheEntry{})
	cache := value.(*snapshotCacheEntry)
	cache.mu.Lock()
	defer cache.mu.Unlock()
	if cache.hash == snapshotHash && cache.id > 0 {
		return target, cache.id, true, nil
	}
	snapshotID, err := recorder.RecordCurrentProviderPeriodSnapshot(ctx, sqlcdb.RecordCurrentProviderPeriodSnapshotParams{
		LotteryCode: lotteryCode, PeriodNo: periodNo, OpenAt: openAt, CloseAt: closeAt,
		ObservedAt: observedAt, Source: "guaji_periods_current",
		SnapshotHash: snapshotHash, RawPayload: canonical,
	})
	if err != nil {
		return schemebetting.PeriodSnapshot{}, 0, false, err
	}
	cache.hash = snapshotHash
	cache.id = snapshotID
	return target, snapshotID, true, nil
}
