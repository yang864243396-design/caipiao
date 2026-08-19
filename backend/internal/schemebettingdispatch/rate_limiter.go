package schemebettingdispatch

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"caipiao/backend/internal/db"
	"caipiao/backend/internal/schemebetting"
)

type DispatchRateLimiter struct {
	pool *db.Pool
}

func NewDispatchRateLimiter(pool *db.Pool) *DispatchRateLimiter {
	if pool == nil {
		return nil
	}
	return &DispatchRateLimiter{pool: pool}
}

func (limiter *DispatchRateLimiter) Allow(ctx context.Context, command schemebetting.LeasedCommand, now time.Time) (bool, error) {
	if limiter == nil || limiter.pool == nil {
		return false, errors.New("dispatch rate limiter is unavailable")
	}
	var frozen FrozenGuajiRequest
	if err := json.Unmarshal(command.FrozenRequest, &frozen); err != nil {
		return false, err
	}
	lotteryCode := strings.TrimSpace(frozen.Request.LotteryCode)
	account := strings.TrimSpace(frozen.MemberAccount)
	if lotteryCode == "" || account == "" || len(account) > 128 {
		return false, errors.New("dispatch rate identity is incomplete")
	}
	tx, err := limiter.pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx)
	var lotteryLimit, accountLimit, globalLimit int
	err = tx.QueryRow(ctx, `
SELECT max_dispatch_per_second, max_account_dispatch_per_second, max_global_dispatch_per_second
FROM scheme_betting_capacity_limits
WHERE lottery_code = $1 AND enabled
FOR SHARE`, lotteryCode).Scan(&lotteryLimit, &accountLimit, &globalLimit)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, errors.New("capacity_not_configured")
	}
	if err != nil {
		return false, err
	}
	windowStart := now.UTC().Truncate(time.Second)
	scopes := []struct {
		typeName string
		key      string
		limit    int
	}{
		{typeName: "global", key: "global", limit: globalLimit},
		{typeName: "lottery", key: lotteryCode, limit: lotteryLimit},
		{typeName: "account", key: account, limit: accountLimit},
	}
	for _, scope := range scopes {
		if _, err := tx.Exec(ctx, `
INSERT INTO scheme_betting_dispatch_rate_buckets
    (scope_type, scope_key, window_start, dispatch_count, updated_at)
VALUES ($1, $2, $3, 0, $4)
ON CONFLICT DO NOTHING`, scope.typeName, scope.key, windowStart, now); err != nil {
			return false, err
		}
	}
	for _, scope := range scopes {
		var count int
		if err := tx.QueryRow(ctx, `
SELECT dispatch_count
FROM scheme_betting_dispatch_rate_buckets
WHERE scope_type = $1 AND scope_key = $2 AND window_start = $3
FOR UPDATE`, scope.typeName, scope.key, windowStart).Scan(&count); err != nil {
			return false, err
		}
		if count >= scope.limit {
			return false, nil
		}
	}
	for _, scope := range scopes {
		if _, err := tx.Exec(ctx, `
UPDATE scheme_betting_dispatch_rate_buckets
SET dispatch_count = dispatch_count + 1, updated_at = $4
WHERE scope_type = $1 AND scope_key = $2 AND window_start = $3`,
			scope.typeName, scope.key, windowStart, now); err != nil {
			return false, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return false, err
	}
	return true, nil
}
