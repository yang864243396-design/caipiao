package accountsvc

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
)

// maxAutoReauthAttempts token 失效时单次事件内自动重新授权的最大次数。
// 任一次成功则恢复可用；全部失败由调用方按原逻辑（停方案等）处理。
const maxAutoReauthAttempts = 3

// EnsureActiveAuth 确保会员启用授权可用：已健康则直接返回；
// 否则自动重新授权最多 maxAutoReauthAttempts 次。
func (s *Service) EnsureActiveAuth(ctx context.Context, memberAccount string) error {
	return s.ensureActiveAuth(ctx, memberAccount, false)
}

// ensureActiveAuth force=true 时即使本地判定 token 仍「健康」也强制走重新授权
//（用于上游已返回令牌无效、但本地尚未标记失效的情形）。
// 同会员并发调用会单飞合并：多方案同时 401 时只打一轮上游，避免 refresh 互踩与失败次数被打满。
func (s *Service) ensureActiveAuth(ctx context.Context, memberAccount string, force bool) error {
	if s == nil {
		return ErrUnavailable
	}
	if s.guaji == nil || !s.guaji.Enabled() {
		return ErrGuajiDisabled
	}
	m, err := s.memberID(ctx, memberAccount)
	if err != nil {
		return err
	}
	row, err := s.getActiveRow(ctx, m)
	if err != nil {
		if isNoRows(err) {
			return ErrNoActiveAccount
		}
		return err
	}
	if !force && s.tokenHealthy(row) {
		return nil
	}

	key := fmt.Sprintf("auto-reauth:%d", m)
	sfErr, shared := s.waitAutoReauthFlight(ctx, key, func() (any, error) {
		return nil, s.doAutoReauthAttempts(ctx, memberAccount, m, force)
	})
	if shared {
		slog.Info("guaji auto reauth singleflight shared", "member", memberAccount, "memberId", m)
	}
	if sfErr == nil {
		return nil
	}
	// 等待方拿到失败时再读一次：可能其它路径（手动重授权）已恢复。
	row, err = s.getActiveRow(ctx, m)
	if err == nil && s.tokenHealthy(row) {
		return nil
	}
	return sfErr
}

func (s *Service) waitAutoReauthFlight(ctx context.Context, key string, fn func() (any, error)) (error, bool) {
	if ctx == nil {
		ctx = context.Background()
	}
	result := s.autoReauthSF.DoChan(key, fn)
	select {
	case <-ctx.Done():
		return ctx.Err(), false
	case call := <-result:
		return call.Err, call.Shared
	}
}

func (s *Service) doAutoReauthAttempts(ctx context.Context, memberAccount string, memberID int64, force bool) error {
	var last error
	for attempt := 1; attempt <= maxAutoReauthAttempts; attempt++ {
		row, err := s.getActiveRow(ctx, memberID)
		if err != nil {
			if isNoRows(err) {
				return ErrNoActiveAccount
			}
			return err
		}
		if !force && s.tokenHealthy(row) {
			return nil
		}
		acct, reauthErr := s.reauthAuto(ctx, memberAccount, row.id)
		if reauthErr == nil && !acct.AuthExpired {
			slog.Info("guaji auto reauth succeeded",
				"member", memberAccount, "accountId", row.id, "attempt", attempt)
			return nil
		}
		last = reauthErr
		if last == nil {
			last = ErrTokenInvalid
		}
		slog.Warn("guaji auto reauth attempt failed",
			"member", memberAccount, "accountId", row.id, "attempt", attempt, "err", last)
		if errors.Is(last, ErrReauthNeedsBind) {
			break
		}
		// 后续轮次继续强制重试（上游已证伪时 force 需保持）
		force = true
	}
	return last
}
