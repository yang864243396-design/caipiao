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
