package accountsvc

import (
	"context"
	"errors"
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

	var last error
	for attempt := 1; attempt <= maxAutoReauthAttempts; attempt++ {
		row, err = s.getActiveRow(ctx, m)
		if err != nil {
			if isNoRows(err) {
				return ErrNoActiveAccount
			}
			return err
		}
		if !force && s.tokenHealthy(row) {
			return nil
		}
		acct, reauthErr := s.Reauth(ctx, memberAccount, row.id)
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
