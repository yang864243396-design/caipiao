package schemes

import (
	"strconv"
	"strings"
)

// advTriggerPC28Subs PC28 支持高级开某投某的子玩法。
var advTriggerPC28Subs = map[string]bool{
	"hezhi":      true,
	"dxds":       true,
	"longhubao":  true,
}

// SupportsAdvTriggerBet 高级开某投某玩法关联矩阵（定位胆/龙虎 + PC28 和值/大小单双/龙虎豹）。
// 请优先传入 guajiGroup/subLabel（rules/v2 同步后）；旧 playTypeID/subPlayID 仍兼容。
func SupportsAdvTriggerBetLegacy(playTypeID, subPlayID string) bool {
	return SupportsAdvTriggerBet(playTypeID, subPlayID, "", "")
}

// triggerOpenMatches 上期开奖是否命中映射行的「开出」条件。
// watchPositions 非空时：任一选定投注位开出等于 open 即命中；空则回退 rule.PositionIdx。
func triggerOpenMatches(rule playRule, balls []string, open string, watchPositions ...[]int) bool {
	open = normalizeTriggerToken(open)
	if open == "" || len(balls) == 0 {
		return false
	}
	if isLonghuPlay(rule) {
		return normalizeTriggerToken(longhuResult(rule, balls)) == open
	}
	if rule.PlayTemplate == "pc28_std" {
		switch strings.TrimSpace(rule.BetMode) {
		case "hezhi":
			return strconv.Itoa(pc28Sum(balls)) == open
		case "dxds":
			return pc28DxdsOpenMatches(balls, open)
		case "longhubao":
			return normalizeTriggerToken(pc28LonghubaoResult(balls)) == open
		}
	}
	// SSC/哈希等：和值/跨度/尾数/特殊号的「开出」是区位派生形态，不是某一球号
	switch strings.ToLower(strings.TrimSpace(rule.BetMode)) {
	case "hezhi":
		return strconv.Itoa(triggerSegmentSum(rule, balls)) == open
	case "kuadu":
		return strconv.Itoa(triggerSegmentKuadu(rule, balls)) == open
	case "weishu":
		return strconv.Itoa(triggerSegmentSum(rule, balls)%10) == open
	case "teshu":
		// 中三特殊号：开出=豹子/对子/顺子（def-1-1785499025837 曾因落入球号比较而永不命中）
		seg := drawSegmentForRule(rule, balls)
		sub := strings.TrimSpace(rule.CatalogSubID)
		if sub == "" {
			sub = strings.TrimSpace(rule.SubPlayID)
		}
		return teshuPickHit(sub, seg, open)
	}
	positions := []int{rule.PositionIdx}
	if len(watchPositions) > 0 && len(watchPositions[0]) > 0 {
		positions = watchPositions[0]
	}
	for _, pos := range positions {
		if pos >= 0 && pos < len(balls) &&
			normalizeTriggerToken(strings.TrimSpace(balls[pos])) == open {
			return true
		}
	}
	return false
}

func triggerSegmentSum(rule playRule, balls []string) int {
	seg := drawSegmentForRule(rule, balls)
	sum := 0
	for _, d := range seg {
		sum += atoiBall(d)
	}
	return sum
}

func triggerSegmentKuadu(rule playRule, balls []string) int {
	seg := drawSegmentForRule(rule, balls)
	if len(seg) == 0 {
		return 0
	}
	vals := make([]int, len(seg))
	for i, d := range seg {
		vals[i] = atoiBall(d)
	}
	return maxInt(vals) - minInt(vals)
}

func pc28DxdsOpenMatches(balls []string, pick string) bool {
	sum := pc28Sum(balls)
	switch pick {
	case "大":
		return sum >= 14
	case "小":
		return sum <= 13
	case "单":
		return sum%2 == 1
	case "双":
		return sum%2 == 0
	default:
		return false
	}
}

func pc28LonghubaoResult(balls []string) string {
	if len(balls) < 3 {
		return ""
	}
	a, c := atoiBall(balls[0]), atoiBall(balls[2])
	switch {
	case a > c:
		return "龙"
	case a < c:
		return "虎"
	default:
		return "豹"
	}
}
