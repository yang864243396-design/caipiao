package games

import (
	"strconv"
	"strings"

	"caipiao/backend/internal/db/sqlcdb"
)

// compareIssueNo 比较第三方期号；数值可比时按 int64，否则按字符串。
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

// prevIssueNo 返回上一期期号（数值减 1）。
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

// maxIssueNo 返回候选期号中数值最大者。
func maxIssueNo(candidates ...string) string {
	best := ""
	for _, c := range candidates {
		c = strings.TrimSpace(c)
		if c == "" {
			continue
		}
		if best == "" || compareIssueNo(c, best) > 0 {
			best = c
		}
	}
	return best
}

// filterDrawsBeforeOpenPeriod 仅保留已封盘/已开奖期（issue < 当前可投期）。
func filterDrawsBeforeOpenPeriod(
	draws []sqlcdb.ListLotteryDrawsRow,
	openPeriod, latestDrawn string,
) []sqlcdb.ListLotteryDrawsRow {
	cutoff := strings.TrimSpace(openPeriod)
	if cutoff == "" {
		cutoff = strings.TrimSpace(bumpIssueNo(latestDrawn))
	}
	if cutoff == "" {
		return draws
	}
	out := make([]sqlcdb.ListLotteryDrawsRow, 0, len(draws))
	for _, d := range draws {
		if compareIssueNo(d.IssueNo, cutoff) < 0 {
			out = append(out, d)
		}
	}
	return out
}
