package accountsvc

import (
	"context"
	"math"

	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/guaji"
	"caipiao/backend/internal/member"
)

// MultiCurrencyBalance 启用授权账号的三币种可用余额快照。
type MultiCurrencyBalance struct {
	USDT float64 `json:"usdt"`
	TRX  float64 `json:"trx"`
	CNY  float64 `json:"cny"`
}

// AdminMultiCurrencyBalances 批量读取会员启用授权在库内的三币种余额快照；无启用授权时为 0。
func (s *Service) AdminMultiCurrencyBalances(ctx context.Context, memberIDs []int64) map[int64]MultiCurrencyBalance {
	out := make(map[int64]MultiCurrencyBalance, len(memberIDs))
	for _, id := range memberIDs {
		out[id] = MultiCurrencyBalance{}
	}
	if s == nil || s.pool == nil || len(memberIDs) == 0 {
		return out
	}
	rows, err := sqlcdb.New(s.pool).ListActiveGuajiBalancesByMemberIDs(ctx, memberIDs)
	if err != nil {
		return out
	}
	for _, row := range rows {
		out[row.MemberID] = MultiCurrencyBalance{
			USDT: roundMoney(row.BalanceUsdt),
			TRX:  roundMoney(row.BalanceTrx),
			CNY:  roundMoney(row.BalanceCny),
		}
	}
	return out
}

// SyncGuajiBalancesForMemberID 与 Client GET /client/guaji/balance 同源：拉第三方 users/i/info 写库并返回；
// 无启用授权 / Token 无效 / 第三方不可用时回退库内快照（无快照则为 0）。
func (s *Service) SyncGuajiBalancesForMemberID(ctx context.Context, memberID int64) MultiCurrencyBalance {
	if s == nil || memberID <= 0 {
		return MultiCurrencyBalance{}
	}
	if s.guaji != nil && s.guaji.Enabled() && len(s.credKey) > 0 {
		row, err := s.getActiveRow(ctx, memberID)
		if err == nil && s.tokenHealthy(row) {
			token, err := guaji.DecryptSecret(s.credKey, row.accessTokenEnc.String)
			if err == nil && token != "" {
				if info, err := s.guaji.UserInfo(ctx, token); err == nil {
					bal := multiCurrencyFromInfo(info)
					s.persistGuajiBalances(ctx, row.id, bal)
					return bal
				}
			}
		}
	}
	return s.AdminMultiCurrencyBalances(ctx, []int64{memberID})[memberID]
}

func multiCurrencyFromInfo(info *guaji.UserInfo) MultiCurrencyBalance {
	if info == nil {
		return MultiCurrencyBalance{}
	}
	return MultiCurrencyBalance{
		USDT: roundMoney(info.BalanceByCurrency(guaji.CurrencyUSDT)),
		TRX:  roundMoney(info.BalanceByCurrency(guaji.CurrencyTRX)),
		CNY:  roundMoney(info.BalanceByCurrency(guaji.CurrencyCNY)),
	}
}

func (s *Service) persistGuajiBalances(ctx context.Context, accountID int64, bal MultiCurrencyBalance) {
	if s == nil || s.pool == nil || accountID <= 0 {
		return
	}
	_ = sqlcdb.New(s.pool).UpdateMemberGuajiAccountBalances(ctx, sqlcdb.UpdateMemberGuajiAccountBalancesParams{
		ID:          accountID,
		BalanceUsdt: member.NumericFromFloat(bal.USDT),
		BalanceTrx:  member.NumericFromFloat(bal.TRX),
		BalanceCny:  member.NumericFromFloat(bal.CNY),
	})
}

func roundMoney(v float64) float64 {
	return math.Round(v*100) / 100
}
