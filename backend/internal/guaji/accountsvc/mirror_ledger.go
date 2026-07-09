package accountsvc

import (
	"context"

	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/guaji"
	"caipiao/backend/internal/member"
)

// MirrorBetDebitLedger 第三方接单成功后写 bet_debit 镜像流水（T5，与派奖 payout 对称）。
func (s *Service) MirrorBetDebitLedger(
	ctx context.Context,
	qtx *sqlcdb.Queries,
	memberID int64,
	orderNo string,
	stake float64,
	guajiAccountID int64,
	currency string,
) error {
	if s == nil || qtx == nil || memberID <= 0 || guajiAccountID <= 0 || stake <= 0 {
		return nil
	}
	balanceSnapshot := 0.0
	if s.guaji != nil && s.guaji.Enabled() && len(s.credKey) > 0 {
		acc, err := s.getRowByIDAny(ctx, guajiAccountID)
		if err == nil && s.tokenHealthy(acc) {
			token, err := guaji.DecryptSecret(s.credKey, acc.accessTokenEnc.String)
			if err == nil && token != "" {
				if info, err := s.guaji.UserInfo(ctx, token); err == nil {
					balanceSnapshot = info.BalanceByCurrency(currency)
				}
			}
		}
	}
	return member.MirrorRealLedger(ctx, qtx, memberID, orderNo, "bet_debit", -stake, balanceSnapshot, guajiAccountID, currency)
}
