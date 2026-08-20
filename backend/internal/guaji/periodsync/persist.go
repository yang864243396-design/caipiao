package periodsync

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"time"

	"caipiao/backend/internal/db"
	"caipiao/backend/internal/guaji"
)

type providerPeriodSnapshot struct {
	LotteryCode  string
	PeriodNo     string
	OpenAt       time.Time
	CloseAt      time.Time
	ObservedAt   time.Time
	SnapshotHash string
	RawPayload   []byte
}

func buildProviderPeriodSnapshots(lotteryCode string, periods []guaji.LottPeriod, observedAt time.Time) []providerPeriodSnapshot {
	lotteryCode = strings.TrimSpace(lotteryCode)
	if lotteryCode == "" {
		return nil
	}
	if observedAt.IsZero() {
		observedAt = time.Now().UTC()
	} else {
		observedAt = observedAt.UTC()
	}
	selectedPeriod, selectedCloseAt, selectedOK := guaji.PickOpenLottPeriod(periods, lotteryCode, observedAt)
	selectedPeriodNo := ""
	if selectedOK {
		selectedPeriodNo = strings.TrimSpace(selectedPeriod.Period)
	}
	out := make([]providerPeriodSnapshot, 0, len(periods))
	for _, period := range periods {
		periodNo := strings.TrimSpace(period.Period)
		if periodNo == "" {
			continue
		}
		openAt, openErr := guaji.ParseGuajiPeriodTimeForLottery(lotteryCode, period.StartTime)
		closeAt, closeErr := guaji.ParseGuajiPeriodTimeForLottery(lotteryCode, period.EndTime)
		if closeErr != nil || closeAt.IsZero() || (openErr == nil && !openAt.Before(closeAt)) {
			continue
		}
		if openErr != nil {
			openAt = time.Time{}
		}
		// The provider can omit the currently open period and return the next
		// period instead. Only the period selected by the shared provider-period
		// resolver may represent that current window. Later future periods must
		// keep their real open_at so the dispatcher cannot select them early.
		if periodNo == selectedPeriodNo && !openAt.IsZero() && openAt.After(observedAt) {
			periodDuration := closeAt.Sub(openAt)
			closeAt = selectedCloseAt
			openAt = closeAt.Add(-periodDuration)
			if openAt.After(observedAt) {
				openAt = observedAt
			}
		}
		canonical := struct {
			LotteryCode string `json:"lotteryCode"`
			PeriodNo    string `json:"periodNo"`
			OpenAt      string `json:"openAt,omitempty"`
			CloseAt     string `json:"closeAt"`
		}{LotteryCode: lotteryCode, PeriodNo: periodNo, CloseAt: closeAt.UTC().Format(time.RFC3339Nano)}
		if !openAt.IsZero() {
			canonical.OpenAt = openAt.UTC().Format(time.RFC3339Nano)
		}
		raw, err := json.Marshal(canonical)
		if err != nil {
			continue
		}
		sum := sha256.Sum256(raw)
		out = append(out, providerPeriodSnapshot{
			LotteryCode: lotteryCode, PeriodNo: periodNo, OpenAt: openAt.UTC(), CloseAt: closeAt.UTC(),
			ObservedAt: observedAt, SnapshotHash: hex.EncodeToString(sum[:]), RawPayload: raw,
		})
	}
	return out
}

func persistProviderPeriodSnapshots(ctx context.Context, pool *db.Pool, lotteryCode string, periods []guaji.LottPeriod, observedAt time.Time) error {
	if pool == nil {
		return nil
	}
	for _, snapshot := range buildProviderPeriodSnapshots(lotteryCode, periods, observedAt) {
		_, err := pool.Exec(ctx, `
INSERT INTO provider_period_snapshots
    (lottery_code, period_no, open_at, close_at, observed_at, source, snapshot_hash, raw_payload)
VALUES ($1, $2, $3, $4, $5, 'guaji_periods', $6, $7)
ON CONFLICT (lottery_code, period_no, snapshot_hash) DO NOTHING`,
			snapshot.LotteryCode, snapshot.PeriodNo, nullableSnapshotTime(snapshot.OpenAt), snapshot.CloseAt,
			snapshot.ObservedAt, snapshot.SnapshotHash, snapshot.RawPayload)
		if err != nil {
			return err
		}
	}
	return nil
}

func nullableSnapshotTime(value time.Time) any {
	if value.IsZero() {
		return nil
	}
	return value.UTC()
}
