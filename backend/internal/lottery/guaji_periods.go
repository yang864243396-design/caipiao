package lottery

import "strings"

// GuajiPeriodsNotProvided 该彩种第三方不提供 periods 列表（非「未开盘」）。
func GuajiPeriodsNotProvided(lotteryCode string) bool {
	_ = strings.TrimSpace(lotteryCode)
	return false
}

// OpenIssueForGuajiBet 解析下注用期号。
func OpenIssueForGuajiBet(lotteryCode string) (issue string, ok bool) {
	return StrictOpenIssueForGuajiBet(lotteryCode)
}
