package bets

import (
	"context"
	"fmt"
	"log/slog"

	"caipiao/backend/internal/db"
	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/member"
	"caipiao/backend/internal/schemes"
	"caipiao/backend/internal/ws"
)

const settlementBatchSize = 50

func settlePendingOrder(ctx context.Context, qtx *sqlcdb.Queries, hub *ws.Hub, row sqlcdb.ListPendingBetOrdersForSettlementRow) error {
	// T5/C8：real 第三方订单（已记 guaji_account_id）以第三方派奖为准，
	// 不走本地开奖结算（否则会用本地开奖错误派奖到本地钱包）。由第三方派奖同步处理。
	if row.GuajiAccountID.Valid {
		return nil
	}

	draw, err := schemes.EnsureDrawForIssue(ctx, qtx, hub, row.LotteryCode, row.IssueNo)
	if err != nil {
		return fmt.Errorf("draw %s/%s: %w", row.LotteryCode, row.IssueNo, err)
	}

	balls := sqlcdb.ParseDrawBalls(draw.Balls)
	payload := schemes.EnsureBetPayload(row.BetPayload, row.PlayMethod, row.OrderNo)
	hit, odds := schemes.EvaluateBetPayload(payload, balls)
	amount := member.RoundMoney(row.Amount)
	pnl := schemes.CalcOrderPnL(amount, hit, odds)

	status := "lose"
	if hit {
		status = "win"
	}

	// 先用 status='pending' 守卫抢占结算权（行锁 + 条件更新）。并发/多实例下只有
	// 一个事务能把订单从 pending 翻转为 win/lose；其余拿到 n=0 直接返回，绝不重复派奖。
	n, err := qtx.SettleBetOrder(ctx, sqlcdb.SettleBetOrderParams{
		ID:     row.ID,
		Status: status,
		Pnl:    member.NumericFromFloat(pnl),
	})
	if err != nil {
		return err
	}
	if n == 0 {
		return nil
	}

	if hit {
		payout := member.PayoutGross(amount, pnl)
		if err := member.CreditWalletPayout(ctx, qtx, row.MemberID, row.OrderNo, payout); err != nil {
			return err
		}
	}
	return nil
}

// SettlementWorker settles pending bet_orders against lottery_draws.
type SettlementWorker struct {
	pool *db.Pool
	q    *sqlcdb.Queries
	hub  *ws.Hub
}

func NewSettlementWorker(pool *db.Pool, hub *ws.Hub) *SettlementWorker {
	if pool == nil {
		return nil
	}
	return &SettlementWorker{pool: pool, q: sqlcdb.New(pool), hub: hub}
}

func (w *SettlementWorker) Tick(ctx context.Context) {
	if w == nil {
		return
	}
	rows, err := w.q.ListPendingBetOrdersForSettlement(ctx, settlementBatchSize)
	if err != nil {
		slog.Warn("bet settlement list failed", "err", err)
		return
	}
	for _, row := range rows {
		w.settleOne(ctx, row)
	}
}

func (w *SettlementWorker) settleOne(ctx context.Context, row sqlcdb.ListPendingBetOrdersForSettlementRow) {
	// real 第三方挂机单：派奖由第三方同步，不走本地开奖结算；绝不能空跑 commit+日志，否则会每 tick 刷同一批单拖死 API。
	if row.GuajiAccountID.Valid {
		return
	}
	tx, err := w.pool.Begin(ctx)
	if err != nil {
		slog.Warn("bet settlement tx begin failed", "orderNo", row.OrderNo, "err", err)
		return
	}
	defer tx.Rollback(ctx)

	if err := settlePendingOrder(ctx, w.q.WithTx(tx), w.hub, row); err != nil {
		slog.Warn("bet settlement failed", "orderNo", row.OrderNo, "err", err)
		return
	}
	if err := tx.Commit(ctx); err != nil {
		slog.Warn("bet settlement commit failed", "orderNo", row.OrderNo, "err", err)
		return
	}
	slog.Info("bet order settled", "orderNo", row.OrderNo, "issue", row.IssueNo)
}
