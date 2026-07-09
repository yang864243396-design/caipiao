package accountsvc

import (
	"context"
	"log/slog"
	"time"
)

// HealthStats 第三方授权健康汇总（T6 监控）。
type HealthStats struct {
	Enabled        bool `json:"enabled"`
	TotalBindings  int  `json:"totalBindings"`
	ActiveAccounts int  `json:"activeAccounts"`
	ErroredTokens  int  `json:"erroredTokens"`
	ExpiringSoon   int  `json:"expiringSoon"` // 启用中且 1 小时内过期
}

// Health 汇总授权账号健康度（Admin 监控只读）。
func (s *Service) Health(ctx context.Context) (HealthStats, error) {
	out := HealthStats{Enabled: s != nil && s.guaji != nil && s.guaji.Enabled()}
	if s == nil || s.pool == nil {
		return out, nil
	}
	err := s.pool.QueryRow(ctx, `
SELECT
    COUNT(*)::int,
    COUNT(*) FILTER (WHERE is_active)::int,
    COUNT(*) FILTER (WHERE last_token_error IS NOT NULL)::int,
    COUNT(*) FILTER (WHERE is_active AND token_expires_at IS NOT NULL AND token_expires_at < now() + interval '1 hour')::int
FROM member_guaji_accounts`).
		Scan(&out.TotalBindings, &out.ActiveAccounts, &out.ErroredTokens, &out.ExpiringSoon)
	if err != nil {
		return out, err
	}
	return out, nil
}

// RunTokenMonitor 周期巡检授权 Token 健康，对临期/失效告警（T6）。
// 仅记录日志告警（接入告警系统时替换 slog）；不自动改库。
func (s *Service) RunTokenMonitor(ctx context.Context, interval time.Duration) {
	if s == nil || s.pool == nil {
		return
	}
	if interval <= 0 {
		interval = 5 * time.Minute
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			stats, err := s.Health(ctx)
			if err != nil {
				slog.Warn("guaji token monitor query failed", "err", err)
				continue
			}
			if stats.ErroredTokens > 0 || stats.ExpiringSoon > 0 {
				slog.Warn("guaji token health alert",
					"erroredTokens", stats.ErroredTokens,
					"expiringSoon", stats.ExpiringSoon,
					"activeAccounts", stats.ActiveAccounts,
					"totalBindings", stats.TotalBindings,
				)
			}
		}
	}
}
