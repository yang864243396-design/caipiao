package sqlcdb

import (
	"context"
	"time"
)

type RecordCurrentProviderPeriodSnapshotParams struct {
	LotteryCode  string
	PeriodNo     string
	OpenAt       time.Time
	CloseAt      time.Time
	ObservedAt   time.Time
	Source       string
	SnapshotHash string
	RawPayload   []byte
}

// RecordCurrentProviderPeriodSnapshot preserves the already-resolved,
// lottery-wide open period as an immutable audit fact. It does not select or
// infer a period from historical rows.
func (q *Queries) RecordCurrentProviderPeriodSnapshot(
	ctx context.Context, arg RecordCurrentProviderPeriodSnapshotParams,
) (int64, error) {
	var id int64
	err := q.db.QueryRow(ctx, `
INSERT INTO provider_period_snapshots
    (lottery_code, period_no, open_at, close_at, observed_at, source, snapshot_hash, raw_payload)
VALUES ($1, $2, NULLIF($3::timestamptz, '-infinity'::timestamptz), $4, $5, $6, $7, $8)
ON CONFLICT (lottery_code, period_no, snapshot_hash) DO UPDATE
SET source = EXCLUDED.source
RETURNING id`, arg.LotteryCode, arg.PeriodNo, nullableCurrentPeriodOpenAt(arg.OpenAt), arg.CloseAt,
		arg.ObservedAt, arg.Source, arg.SnapshotHash, arg.RawPayload).Scan(&id)
	return id, err
}

func nullableCurrentPeriodOpenAt(value time.Time) any {
	if value.IsZero() {
		return "-infinity"
	}
	return value.UTC()
}
