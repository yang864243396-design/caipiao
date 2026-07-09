package schemes

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/ws"
)

// EnsureDrawForIssue returns lottery_draws row for issue; synthesizes deterministically if missing.
func EnsureDrawForIssue(ctx context.Context, q *sqlcdb.Queries, hub *ws.Hub, lotteryCode, issueNo string) (sqlcdb.LotteryDraw, error) {
	draw, err := q.GetLotteryDrawByIssue(ctx, sqlcdb.GetLotteryDrawByIssueParams{
		LotteryCode: lotteryCode,
		IssueNo:     issueNo,
	})
	if err == nil {
		return sqlcdb.LotteryDrawFromIssueRow(draw), nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return sqlcdb.LotteryDraw{}, err
	}
	balls := synthDrawBalls(lotteryCode, issueNo)
	sum := 0
	for _, b := range balls {
		if n, perr := strconv.Atoi(b); perr == nil {
			sum += n
		}
	}
	raw, err := json.Marshal(balls)
	if err != nil {
		return sqlcdb.LotteryDraw{}, err
	}
	return insertDrawAndNotify(ctx, q, hub, sqlcdb.InsertLotteryDrawParams{
		LotteryCode: lotteryCode,
		IssueNo:     issueNo,
		PeriodShort: issuePeriodShort(issueNo),
		Balls:       raw,
		SumValue:    int32(sum),
		DrawnAt:     pgtype.Timestamptz{Time: time.Now(), Valid: true},
	})
}
