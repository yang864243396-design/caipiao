package games

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/lottery"
)

type periodDisplay struct {
	CurrentIssue       string
	NextIssue          string
	CountdownPeriod    string
	CountdownSec       int
	CountdownEndTime   string
	CountdownCloseAt   string
	CountdownWindowSec int
	CountdownLabel     string
	DrawPhase          string
	DrawnNumbers       []string
}

func (s *Service) resolvePeriodDisplay(ctx context.Context, code string, lhc bool) (periodDisplay, error) {
	now := time.Now().UTC()
	out := periodDisplay{
		DrawPhase: "drawing",
	}

	currentIssue, drawnNumbers, drawPhase, err := s.resolveLatestDrawnDisplay(ctx, code)
	if err != nil {
		return periodDisplay{}, err
	}
	out.CurrentIssue = currentIssue
	out.DrawnNumbers = drawnNumbers
	out.DrawPhase = drawPhase

	if out.CurrentIssue == "" {
		out.CurrentIssue = defaultDemoCurrentIssue(lhc)
	}

	out.NextIssue = bumpIssueNo(out.CurrentIssue)
	applyGameDetailCountdown(&out, code, now)

	return out, nil
}

// resolveLatestDrawnDisplay 合并 DB / 开奖 WS / periods，与云端中心可投期对齐。
// latestDrawIssueFromDB 取 lottery_draws 中期号最大的一期（避免 drawn_at 乱序）。
func latestDrawIssueFromDB(ctx context.Context, q *sqlcdb.Queries, lotteryCode string) (string, error) {
	if q == nil {
		return "", nil
	}
	draws, err := q.ListLotteryDraws(ctx, sqlcdb.ListLotteryDrawsParams{
		LotteryCode: lotteryCode,
		RowLimit:    30,
	})
	if err != nil {
		return "", err
	}
	return maxIssueNo(drawIssues(draws)...), nil
}

func drawIssues(draws []sqlcdb.ListLotteryDrawsRow) []string {
	out := make([]string, 0, len(draws))
	for _, d := range draws {
		if issue := strings.TrimSpace(d.IssueNo); issue != "" {
			out = append(out, issue)
		}
	}
	return out
}

// openBettingPeriod 当前可投期（periods 缓存优先）；开盘中则返回该期号。
func openBettingPeriod(code string, now time.Time) string {
	if ps, ok := lottery.PeriodsScheduleFor(code); ok {
		return strings.TrimSpace(ps.CurrentPeriod)
	}
	_ = now
	return ""
}

// clampDrawDisplayIssue 开奖区「当前期」不得等于/超过正在倒计时的可投期。
func clampDrawDisplayIssue(issue, openPeriod string) string {
	issue = strings.TrimSpace(issue)
	openPeriod = strings.TrimSpace(openPeriod)
	if openPeriod == "" {
		return issue
	}
	maxDisplay := prevIssueNo(openPeriod)
	if maxDisplay == "" {
		return issue
	}
	if issue == "" {
		return maxDisplay
	}
	if compareIssueNo(issue, maxDisplay) > 0 {
		return maxDisplay
	}
	return issue
}

func (s *Service) resolveLatestDrawnDisplay(ctx context.Context, code string) (issue string, balls []string, drawPhase string, err error) {
	drawPhase = "drawing"
	now := time.Now().UTC()
	openPeriod := openBettingPeriod(code, now)
	maxDisplayIssue := ""
	if openPeriod != "" {
		maxDisplayIssue = prevIssueNo(openPeriod)
	}

	candidates := make([]string, 0, 4)
	if maxDisplayIssue != "" {
		candidates = append(candidates, maxDisplayIssue)
	}

	if dbIssue, dbErr := latestDrawIssueFromDB(ctx, s.q, code); dbErr != nil {
		return "", nil, "", dbErr
	} else if dbIssue != "" {
		candidates = append(candidates, dbIssue)
	}

	if st, ok := lottery.PeriodStateFor(code); ok {
		if wsIssue := strings.TrimSpace(st.CurrentIssue); wsIssue != "" {
			candidates = append(candidates, wsIssue)
		}
	}

	issue = clampDrawDisplayIssue(maxIssueNo(candidates...), openPeriod)
	if issue == "" {
		return "", nil, drawPhase, nil
	}

	row, rowErr := s.q.GetLotteryDrawByIssue(ctx, sqlcdb.GetLotteryDrawByIssueParams{
		LotteryCode: code,
		IssueNo:     issue,
	})
	if rowErr == nil {
		balls = parseBalls(row.Balls)
		if len(balls) > 0 {
			drawPhase = "drawn"
		}
		return issue, balls, drawPhase, nil
	}
	if !errors.Is(rowErr, pgx.ErrNoRows) {
		return "", nil, "", rowErr
	}

	if cached, ok := lottery.DrawResultForIssue(code, issue); ok {
		cp := append([]string(nil), cached.Balls...)
		return issue, cp, "drawn", nil
	}

	// periods/WS 已推进，DB 尚未入库：展示上一期期号并等待开奖结果（前端在倒计时中加速刷新）
	return issue, nil, drawPhase, nil
}

// applyGameDetailCountdown 系统统一倒计时：与方案启停无关，仅认 periods 第三方 end_time。
func applyGameDetailCountdown(out *periodDisplay, lotteryCode string, now time.Time) {
	if out == nil {
		return
	}
	cd := lottery.BuildPeriodsDisplayCountdown(lotteryCode, now)
	if cd.Period != "" {
		out.NextIssue = cd.Period
		out.CountdownPeriod = cd.Period
	}
	out.CountdownSec = cd.Sec
	out.CountdownEndTime = cd.EndTimeRaw
	out.CountdownCloseAt = cd.CloseAtRFC3339
	out.CountdownWindowSec = cd.WindowSec
	out.CountdownLabel = cd.WaitingLabel
}
