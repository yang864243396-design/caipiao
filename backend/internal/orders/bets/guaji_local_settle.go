package bets

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"caipiao/backend/internal/db"
	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/guaji/accountsvc"
	"caipiao/backend/internal/member"
	"caipiao/backend/internal/schemes"
)

// LocalGuajiDrawFallback 构造第三方注单缺失时的本地开奖派奖评估。
func LocalGuajiDrawFallback(pool *db.Pool) accountsvc.LocalDrawFallback {
	q := sqlcdb.New(pool)
	return func(ctx context.Context, orderID int64, orderNo string) (accountsvc.LocalDrawSettlement, bool, error) {
		var lotteryCode, issueNo, playMethod string
		var payload []byte
		var amount float64
		err := pool.QueryRow(ctx, `
SELECT lottery_code, issue_no, play_method, bet_payload, amount::float8
FROM bet_orders WHERE id = $1 AND status = 'pending'`, orderID).Scan(&lotteryCode, &issueNo, &playMethod, &payload, &amount)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return accountsvc.LocalDrawSettlement{}, false, nil
			}
			return accountsvc.LocalDrawSettlement{}, false, err
		}
		drawRow, err := q.GetLotteryDrawByIssue(ctx, sqlcdb.GetLotteryDrawByIssueParams{
			LotteryCode: lotteryCode,
			IssueNo:     issueNo,
		})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return accountsvc.LocalDrawSettlement{}, false, nil
			}
			return accountsvc.LocalDrawSettlement{}, false, err
		}
		balls := sqlcdb.ParseDrawBalls(drawRow.Balls)
		betPayload := schemes.EnsureBetPayload(payload, playMethod, orderNo)
		hit, odds := schemes.EvaluateBetPayload(betPayload, balls)
		pnl := schemes.CalcOrderPnL(amount, hit, odds)
		out := accountsvc.LocalDrawSettlement{Status: "lose", Pnl: pnl}
		if hit {
			out.Status = "win"
			out.Payout = member.PayoutGross(amount, pnl)
		}
		return out, true, nil
	}
}
