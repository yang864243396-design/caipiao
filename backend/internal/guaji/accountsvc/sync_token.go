package accountsvc

import (
	"context"
	"fmt"
	"strings"

	"caipiao/backend/internal/guaji"
	"caipiao/backend/internal/guajibet"
)

// SyncAccessToken 返回任一健康挂机 token，供 periods 等只读第三方接口同步。
func (s *Service) SyncAccessToken(ctx context.Context) (string, error) {
	if s == nil || s.pool == nil || s.guaji == nil || !s.guaji.Enabled() {
		return "", fmt.Errorf("guaji sync token: unavailable")
	}
	rows, err := s.pool.Query(ctx, `
SELECT `+accountCols+`
FROM member_guaji_accounts
WHERE is_active = true
ORDER BY last_sync_at DESC NULLS LAST, bound_at DESC
LIMIT 8`)
	if err != nil {
		return "", err
	}
	defer rows.Close()
	for rows.Next() {
		acc, err := scanRow(rows)
		if err != nil {
			return "", err
		}
		if !s.tokenHealthy(acc) {
			continue
		}
		token, err := guaji.DecryptSecret(s.credKey, acc.accessTokenEnc.String)
		if err != nil {
			continue
		}
		if token != "" {
			return token, nil
		}
	}
	return "", fmt.Errorf("guaji sync token: no healthy active account")
}

// MemberAccessToken 返回指定会员的启用授权 token（矩阵/探测等场景）。
func (s *Service) MemberAccessToken(ctx context.Context, memberAccount string) (string, error) {
	if s == nil || s.pool == nil {
		return "", fmt.Errorf("guaji member token: unavailable")
	}
	m, err := s.memberID(ctx, memberAccount)
	if err != nil {
		return "", err
	}
	row, err := s.getActiveRow(ctx, m)
	if err != nil {
		if isNoRows(err) {
			return "", guajibet.ErrNoActiveAuth
		}
		return "", err
	}
	token, err := guaji.DecryptSecret(s.credKey, row.accessTokenEnc.String)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(token) == "" {
		return "", fmt.Errorf("guaji member token: empty")
	}
	return token, nil
}
