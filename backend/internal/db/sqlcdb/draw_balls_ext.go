package sqlcdb

import (
	"encoding/json"

	"github.com/jackc/pgx/v5/pgtype"
)

// ParseDrawBalls 解析 lottery_draws.balls JSON 为字符串数组。
func ParseDrawBalls(raw []byte) []string {
	if len(raw) == 0 {
		return nil
	}
	var balls []string
	if err := json.Unmarshal(raw, &balls); err != nil {
		return nil
	}
	return balls
}

func lotteryDrawFromCore(id int64, lotteryCode, issueNo, periodShort string, balls []byte, sumValue int32, drawnAt pgtype.Timestamptz) LotteryDraw {
	return LotteryDraw{
		ID:          id,
		LotteryCode: lotteryCode,
		IssueNo:     issueNo,
		PeriodShort: periodShort,
		Balls:       balls,
		SumValue:    sumValue,
		DrawnAt:     drawnAt,
	}
}

// LotteryDrawFromIssueRow 将 sqlc 查询行转为 LotteryDraw。
func LotteryDrawFromIssueRow(r GetLotteryDrawByIssueRow) LotteryDraw {
	return lotteryDrawFromCore(r.ID, r.LotteryCode, r.IssueNo, r.PeriodShort, r.Balls, r.SumValue, r.DrawnAt)
}

// LotteryDrawFromInsertRow 将插入返回行转为 LotteryDraw。
func LotteryDrawFromInsertRow(r InsertLotteryDrawRow) LotteryDraw {
	return lotteryDrawFromCore(r.ID, r.LotteryCode, r.IssueNo, r.PeriodShort, r.Balls, r.SumValue, r.DrawnAt)
}