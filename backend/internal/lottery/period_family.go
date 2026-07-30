package lottery

import (
	"fmt"
	"strings"
)

// 期号族比对。
//
// 下注链路的期号来自第三方 periods 接口（按 lottery_catalog.outbound_lottery_code 取），
// 开奖链路的期号来自 WS / 历史同步（按 guaji_ws_key 取）。两者映射到不同彩种时，
// 下注与结算都不会报错，只是注单期号永远查不到开奖号——极速彩那个 bug 潜伏很久，
// 就是因为没人从这个角度比过。
//
// **判据只有位数。** 试过用数值相对差补充「同体系内错配」，但长期号里有意义的位埋得太深：
// 日期族差半年，相对差也只有 5.7e-8，跟相邻期没法区分。定不出有依据的阈值就不定，
// 假装精确的规则只会制造误报。
//
// 因此本函数的盲区是：**两个期号体系相同的彩种互相错配**（如哈希一分彩 ↔ 哈希五分彩）。
// 那种情况位数与数值都接近，只能靠 audit-ws-keys 比对 WS 键，或靠 cmd/scheme-audit
// 统计「注单期号在 lottery_draws 的命中率」发现。三者互补，缺一不可。

type PeriodFamilyStatus string

const (
	PeriodFamilyOK       PeriodFamilyStatus = "ok"
	PeriodFamilyMismatch PeriodFamilyStatus = "mismatch"
	PeriodFamilyUnknown  PeriodFamilyStatus = "unknown"
)

// ComparePeriodFamily 比对下注链路与开奖链路在同一时刻的期号。
// 返回 PeriodFamilyUnknown 表示输入不足以判断，调用方应跳过而非报警。
func ComparePeriodFamily(betPeriod, drawPeriod string) (PeriodFamilyStatus, string) {
	bet := strings.TrimSpace(betPeriod)
	draw := strings.TrimSpace(drawPeriod)
	if bet == "" || draw == "" {
		return PeriodFamilyUnknown, "缺少下注期号或开奖期号"
	}
	if !isAllDigits(bet) || !isAllDigits(draw) {
		return PeriodFamilyUnknown, "期号非纯数字，无法比对"
	}
	if len(bet) != len(draw) {
		return PeriodFamilyMismatch, fmt.Sprintf(
			"下注期号 %s（%d 位）与开奖期号 %s（%d 位）位数不同，两条链路指向不同的期号体系",
			bet, len(bet), draw, len(draw))
	}
	return PeriodFamilyOK, ""
}

func isAllDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
