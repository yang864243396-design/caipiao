package sqlcdb

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

const getPreviousLotteryDrawByIssue = `
SELECT id, lottery_code, issue_no, period_short, balls, sum_value, drawn_at
FROM lottery_draws
WHERE lottery_code = $1
  AND issue_no < $2
ORDER BY issue_no DESC
LIMIT 1
`

type GetPreviousLotteryDrawByIssueParams struct {
	LotteryCode string `json:"lottery_code"`
	IssueNo     string `json:"issue_no"`
}

type GetPreviousLotteryDrawByIssueRow struct {
	ID          int64              `json:"id"`
	LotteryCode string             `json:"lottery_code"`
	IssueNo     string             `json:"issue_no"`
	PeriodShort string             `json:"period_short"`
	Balls       []byte             `json:"balls"`
	SumValue    int32              `json:"sum_value"`
	DrawnAt     pgtype.Timestamptz `json:"drawn_at"`
}

// GetPreviousLotteryDrawByIssue 取严格小于当前期号的最近一期开奖。
func (q *Queries) GetPreviousLotteryDrawByIssue(ctx context.Context, arg GetPreviousLotteryDrawByIssueParams) (GetPreviousLotteryDrawByIssueRow, error) {
	row := q.db.QueryRow(ctx, getPreviousLotteryDrawByIssue, arg.LotteryCode, arg.IssueNo)
	var i GetPreviousLotteryDrawByIssueRow
	err := row.Scan(
		&i.ID,
		&i.LotteryCode,
		&i.IssueNo,
		&i.PeriodShort,
		&i.Balls,
		&i.SumValue,
		&i.DrawnAt,
	)
	return i, err
}
