package sqlcdb

import (
	"context"
	"errors"
	"time"
)

func (q *Queries) AcquireSchemeBettingShardLease(
	ctx context.Context, leaseKind string, shardNo int32, owner string, now, leaseUntil time.Time,
) (int64, bool, error) {
	var epoch int64
	err := q.db.QueryRow(ctx, `
INSERT INTO scheme_betting_shard_leases
    (lease_kind, shard_no, lease_owner, lease_epoch, lease_until, updated_at)
VALUES ($1, $2, $3, 1, $5, $4)
ON CONFLICT (lease_kind, shard_no) DO UPDATE
SET lease_owner = EXCLUDED.lease_owner,
    lease_epoch = CASE
        WHEN scheme_betting_shard_leases.lease_owner = EXCLUDED.lease_owner
            THEN scheme_betting_shard_leases.lease_epoch
        ELSE scheme_betting_shard_leases.lease_epoch + 1
    END,
    lease_until = EXCLUDED.lease_until,
    updated_at = EXCLUDED.updated_at
WHERE scheme_betting_shard_leases.lease_owner = EXCLUDED.lease_owner
   OR scheme_betting_shard_leases.lease_until <= $4
RETURNING lease_epoch`, leaseKind, shardNo, owner, now, leaseUntil).Scan(&epoch)
	if err != nil {
		if isNoRowsError(err) {
			return 0, false, nil
		}
		return 0, false, err
	}
	return epoch, true, nil
}

func (q *Queries) AssertSchemeBettingShardLease(
	ctx context.Context, leaseKind string, shardNo int32, owner string, epoch int64, now time.Time,
) error {
	var held bool
	if err := q.db.QueryRow(ctx, `
SELECT lease_owner = $3 AND lease_epoch = $4 AND lease_until > $5
FROM scheme_betting_shard_leases
WHERE lease_kind = $1 AND shard_no = $2
FOR SHARE`, leaseKind, shardNo, owner, epoch, now).Scan(&held); err != nil {
		return err
	}
	if !held {
		return errors.New("scheme betting shard lease fence is stale")
	}
	return nil
}

func (q *Queries) AcquireSchemeBettingDrawLease(
	ctx context.Context, lotteryCode, owner string, now, leaseUntil time.Time,
) (int64, bool, error) {
	var epoch int64
	err := q.db.QueryRow(ctx, `
INSERT INTO scheme_betting_draw_leases
    (lottery_code, lease_owner, lease_epoch, lease_until, updated_at)
VALUES ($1, $2, 1, $4, $3)
ON CONFLICT (lottery_code) DO UPDATE
SET lease_owner = EXCLUDED.lease_owner,
    lease_epoch = CASE
        WHEN scheme_betting_draw_leases.lease_owner = EXCLUDED.lease_owner
            THEN scheme_betting_draw_leases.lease_epoch
        ELSE scheme_betting_draw_leases.lease_epoch + 1
    END,
    lease_until = EXCLUDED.lease_until,
    updated_at = EXCLUDED.updated_at
WHERE scheme_betting_draw_leases.lease_owner = EXCLUDED.lease_owner
   OR scheme_betting_draw_leases.lease_until <= $3
RETURNING lease_epoch`, lotteryCode, owner, now, leaseUntil).Scan(&epoch)
	if err != nil {
		if isNoRowsError(err) {
			return 0, false, nil
		}
		return 0, false, err
	}
	return epoch, true, nil
}
