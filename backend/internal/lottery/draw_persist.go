package lottery

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/guaji"
	"caipiao/backend/internal/ws"
)

// PersistDrawFromBalls 写入 lottery_draws；新插入时可选广播 WS-5。
// 始终更新内存缓存，便于 DB 尚未可见时玩法详情先展示球号。
func PersistDrawFromBalls(
	ctx context.Context,
	q *sqlcdb.Queries,
	hub *ws.Hub,
	lotteryCode, issueNo string,
	balls []string,
	drawnAt time.Time,
) (sqlcdb.LotteryDraw, bool, error) {
	lotteryCode = strings.TrimSpace(lotteryCode)
	issueNo = strings.TrimSpace(issueNo)
	if q == nil || lotteryCode == "" || issueNo == "" || len(balls) == 0 {
		return sqlcdb.LotteryDraw{}, false, nil
	}
	if drawnAt.IsZero() {
		drawnAt = time.Now().UTC()
	} else {
		drawnAt = drawnAt.UTC()
	}
	StoreDrawResult(lotteryCode, issueNo, balls, drawnAt)

	ballsJSON, err := json.Marshal(balls)
	if err != nil {
		return sqlcdb.LotteryDraw{}, false, err
	}
	draw, err := q.InsertLotteryDraw(ctx, sqlcdb.InsertLotteryDrawParams{
		LotteryCode: lotteryCode,
		IssueNo:     issueNo,
		PeriodShort: drawPeriodShort(issueNo),
		Balls:       ballsJSON,
		SumValue:    int32(guaji.SumBalls(balls)),
		DrawnAt:     pgtype.Timestamptz{Time: drawnAt, Valid: true},
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			row, gerr := q.GetLotteryDrawByIssue(ctx, sqlcdb.GetLotteryDrawByIssueParams{
				LotteryCode: lotteryCode,
				IssueNo:     issueNo,
			})
			if gerr != nil {
				return sqlcdb.LotteryDraw{}, false, gerr
			}
			return sqlcdb.LotteryDrawFromIssueRow(row), false, nil
		}
		return sqlcdb.LotteryDraw{}, false, err
	}
	out := sqlcdb.LotteryDrawFromInsertRow(draw)
	publishDrawResult(hub, out)
	return out, true, nil
}

func publishDrawResult(hub *ws.Hub, draw sqlcdb.LotteryDraw) {
	if hub == nil {
		return
	}
	drawnAt := time.Now().UTC().Format(time.RFC3339Nano)
	if draw.DrawnAt.Valid {
		drawnAt = draw.DrawnAt.Time.UTC().Format(time.RFC3339Nano)
	}
	ws.PublishDraw(hub, draw.LotteryCode, ws.DrawResultPayload{
		LotteryCode: draw.LotteryCode,
		IssueNo:     draw.IssueNo,
		PeriodShort: draw.PeriodShort,
		Balls:       sqlcdb.ParseDrawBalls(draw.Balls),
		SumValue:    int(draw.SumValue),
		DrawnAt:     drawnAt,
	})
}

func drawPeriodShort(issueNo string) string {
	if len(issueNo) <= 6 {
		return issueNo
	}
	return issueNo[len(issueNo)-6:]
}

