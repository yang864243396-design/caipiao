package schemes

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"

	"caipiao/backend/internal/db/sqlcdb"
)

// 冷热出号依赖「当期之前最近 N 期」。下注时上期开奖常尚未入库，统计窗会少一期，
// 表现为实下内容约等于「应下(N-1)」。上期相邻开奖未到且封盘前仍有余量时，推迟本 tick。

// hotColdPrevDrawWaitMinSec：倒计时低于该值时冷热可降级出号；开某投某仍继续等上期，
// 直至 rem<=0 才 Skip（1 分彩 history 入库常落在最后几秒）。
const hotColdPrevDrawWaitMinSec = 8

func compareIssueNo(a, b string) int {
	a = strings.TrimSpace(a)
	b = strings.TrimSpace(b)
	if a == b {
		return 0
	}
	na, errA := strconv.ParseInt(a, 10, 64)
	nb, errB := strconv.ParseInt(b, 10, 64)
	if errA == nil && errB == nil {
		switch {
		case na < nb:
			return -1
		case na > nb:
			return 1
		default:
			return 0
		}
	}
	return strings.Compare(a, b)
}

func prevIssueNo(issue string) string {
	issue = strings.TrimSpace(issue)
	if issue == "" {
		return ""
	}
	n, err := strconv.ParseInt(issue, 10, 64)
	if err != nil || n <= 0 {
		return issue
	}
	return strconv.FormatInt(n-1, 10)
}

// adjacentPrevSameSeriesMaxGap：同日/同系列内允许的最大期号差。
// 差值在此内且 latest < expected → 视为缺上期（含缺 1 期、缺多期），必须阻塞。
// 更大跳变视为跨日/换号段（numeric-1 可能永不存在），不阻塞，由 GetPrevious 取真实上期。
//
// 实盘：投 0215 期望上期 0214，库止于 0212（差 2）时旧逻辑只拦差=1，误用 0212 后三 719 映射出号。
const adjacentPrevSameSeriesMaxGap int64 = 1000

// hotColdAdjacentPrevMissing 判断「期望上期」是否仍缺库。
//
//   - expectedPrev 空：无法推断，不阻塞
//   - latestBefore == expectedPrev：已就绪
//   - latestBefore 空：缺上期，阻塞
//   - 数值上 0 < expectedPrev-latestBefore ≤ adjacentPrevSameSeriesMaxGap：同系列缺期，阻塞
//   - 差值更大：期号跳号/换日（如 6800040 的期望上期 6800039 永不存在），不阻塞
func hotColdAdjacentPrevMissing(expectedPrev, latestBefore string) bool {
	expectedPrev = strings.TrimSpace(expectedPrev)
	latestBefore = strings.TrimSpace(latestBefore)
	if expectedPrev == "" {
		return false
	}
	if latestBefore == "" {
		return true
	}
	if latestBefore == expectedPrev {
		return false
	}
	ne, errE := strconv.ParseInt(expectedPrev, 10, 64)
	nl, errL := strconv.ParseInt(latestBefore, 10, 64)
	if errE != nil || errL != nil || ne <= 0 || nl <= 0 {
		return compareIssueNo(latestBefore, expectedPrev) < 0
	}
	if ne <= nl {
		return false
	}
	return ne-nl <= adjacentPrevSameSeriesMaxGap
}

// hotColdPreviousDrawReady 冷热统计所需的相邻上期开奖是否已在库。
func (w *Worker) hotColdPreviousDrawReady(ctx context.Context, lotteryCode, currentIssue string) bool {
	if w == nil || w.q == nil {
		return true
	}
	currentIssue = strings.TrimSpace(currentIssue)
	lotteryCode = strings.TrimSpace(lotteryCode)
	if currentIssue == "" || lotteryCode == "" {
		return true
	}
	expected := prevIssueNo(currentIssue)
	if expected == "" || expected == currentIssue {
		return true
	}

	_, err := w.q.GetLotteryDrawByIssue(ctx, sqlcdb.GetLotteryDrawByIssueParams{
		LotteryCode: lotteryCode,
		IssueNo:     expected,
	})
	if err == nil {
		return true
	}
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		// 查询异常时不阻塞下注
		return true
	}

	prev, perr := w.q.GetPreviousLotteryDrawByIssue(ctx, sqlcdb.GetPreviousLotteryDrawByIssueParams{
		LotteryCode: lotteryCode,
		IssueNo:     currentIssue,
	})
	latest := ""
	if perr == nil {
		latest = prev.IssueNo
	}
	return !hotColdAdjacentPrevMissing(expected, latest)
}
