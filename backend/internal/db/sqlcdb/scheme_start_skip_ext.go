package sqlcdb

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"
)

// SkipSchemeInstanceStartPeriodEx 写入开启跳过期快照（期号 + 封盘时刻）。
func (q *Queries) SkipSchemeInstanceStartPeriodEx(
	ctx context.Context,
	instanceID, skipPeriod string,
	skipCloseAt pgtype.Timestamptz,
) (int64, error) {
	instanceID = strings.TrimSpace(instanceID)
	skipPeriod = strings.TrimSpace(skipPeriod)
	if instanceID == "" || skipPeriod == "" || !skipCloseAt.Valid {
		return 0, nil
	}
	tag, err := q.db.Exec(ctx, `
UPDATE scheme_instances
SET last_settled_issue = $2,
    start_skip_period = $2,
    start_skip_close_at = $3,
    updated_at = now()
WHERE id = $1
  AND status = 'running'
  AND status_reason = 'await_next_bet'
  AND start_skip_close_at IS NULL`, instanceID, skipPeriod, skipCloseAt)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// BackfillSchemeInstanceStartSkipCloseAt 为已有 last_settled 但缺封盘时刻的实例补快照。
func (q *Queries) BackfillSchemeInstanceStartSkipCloseAt(
	ctx context.Context,
	instanceID, skipPeriod string,
	skipCloseAt pgtype.Timestamptz,
) (int64, error) {
	instanceID = strings.TrimSpace(instanceID)
	skipPeriod = strings.TrimSpace(skipPeriod)
	if instanceID == "" || skipPeriod == "" || !skipCloseAt.Valid {
		return 0, nil
	}
	tag, err := q.db.Exec(ctx, `
UPDATE scheme_instances
SET start_skip_period = $2,
    start_skip_close_at = $3,
    updated_at = now()
WHERE id = $1
  AND status = 'running'
  AND status_reason = 'await_next_bet'
  AND start_skip_close_at IS NULL`, instanceID, skipPeriod, skipCloseAt)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}
