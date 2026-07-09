package schemes

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/ws"
)

func notifyDrawPublished(h *ws.Hub, draw sqlcdb.LotteryDraw) {
	if h == nil {
		return
	}
	drawnAt := time.Now().UTC().Format(time.RFC3339Nano)
	if draw.DrawnAt.Valid {
		drawnAt = draw.DrawnAt.Time.UTC().Format(time.RFC3339Nano)
	}
	ws.PublishDraw(h, draw.LotteryCode, ws.DrawResultPayload{
		LotteryCode: draw.LotteryCode,
		IssueNo:     draw.IssueNo,
		PeriodShort: draw.PeriodShort,
		Balls:       sqlcdb.ParseDrawBalls(draw.Balls),
		SumValue:    int(draw.SumValue),
		DrawnAt:     drawnAt,
	})
}

func insertDrawAndNotify(
	ctx context.Context,
	q *sqlcdb.Queries,
	h *ws.Hub,
	params sqlcdb.InsertLotteryDrawParams,
) (sqlcdb.LotteryDraw, error) {
	draw, err := q.InsertLotteryDraw(ctx, params)
	if err != nil {
		// 并发下另一 worker 已插入同期开奖：ON CONFLICT DO NOTHING 不返回行（pgx.ErrNoRows）
		// 且不会中止事务，此时回查既有记录并跳过重复 WS 广播。
		if errors.Is(err, pgx.ErrNoRows) {
			row, gerr := q.GetLotteryDrawByIssue(ctx, sqlcdb.GetLotteryDrawByIssueParams{
				LotteryCode: params.LotteryCode,
				IssueNo:     params.IssueNo,
			})
			return sqlcdb.LotteryDrawFromIssueRow(row), gerr
		}
		return sqlcdb.LotteryDraw{}, err
	}
	notifyDrawPublished(h, sqlcdb.LotteryDrawFromInsertRow(draw))
	return sqlcdb.LotteryDrawFromInsertRow(draw), nil
}
