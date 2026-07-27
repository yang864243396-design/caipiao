package schemes

import (
	"fmt"
	"math"
	"strings"
)

// zhixuanFushiMaxBetUnits 直选复式单组最大注数（对齐第三方 / 前端 betPayload.ts）。
// min(P^n−P, (P−1)·P^(n−1))；SSC 前二=90、前三=900。
func zhixuanFushiMaxBetUnits(rule playRule) int {
	if !isZhixuanFushiRule(rule) {
		return 0
	}
	min, max := ruleNumberPool(rule)
	size := max - min + 1
	n := rule.SegmentLen
	if size <= 1 || n <= 1 {
		return 0
	}
	fullMinusSame := int(math.Pow(float64(size), float64(n))) - size
	oneShort := (size - 1) * int(math.Pow(float64(size), float64(n-1)))
	base := fullMinusSame
	if oneShort < base {
		base = oneShort
	}
	return base
}

func isZhixuanFushiRule(rule playRule) bool {
	bm := strings.TrimSpace(rule.BetMode)
	sub := strings.TrimSpace(rule.SubPlayID)
	if bm == "zuhe" || sub == "zuhe" {
		return false
	}
	if bm == "fushi" || bm == "zhixuan_fs" || sub == "zhixuan_fs" {
		return rule.SegmentLen > 1
	}
	return false
}

func countZhixuanFushiBetUnits(content string, segLen int) int {
	if segLen <= 1 {
		return 0
	}
	lines := splitGroupLinesPad(content, segLen)
	units := 1
	for i := 0; i < segLen; i++ {
		n := len(parseDigitTokens(lines[i]))
		if n == 0 {
			return 0
		}
		units *= n
	}
	return units
}

func errMaxBetUnitsExceeded(max int) error {
	return fmt.Errorf("投注注数超过最大投注注数:%d", max)
}
