package schemes

import (
	"strings"
)

func evaluatePC28ByBetMode(rule playRule, balls []string, content string) (betEvaluation, bool) {
	mode := strings.TrimSpace(rule.BetMode)
	switch mode {
	case "hezhi":
		return evaluatePC28Hezhi(balls, content), true
	case "dxds":
		return evaluatePC28Dxds(balls, content), true
	case "teshu":
		return evaluatePC28Teshu(balls, content), true
	case "longhubao":
		return evaluatePC28Longhubao(balls, content), true
	}
	return betEvaluation{}, false
}

func evaluatePC28Hezhi(balls []string, content string) betEvaluation {
	sum := pc28Sum(balls)
	picks := parseIntTokens(content)
	units := len(picks)
	if units <= 0 {
		units = 1
	}
	hit := false
	for _, p := range picks {
		if p == sum {
			hit = true
			break
		}
	}
	return betEvaluation{Hit: hit, BetUnits: units, Odds: oddsZhixuan(1)}
}

func evaluatePC28Dxds(balls []string, content string) betEvaluation {
	sum := pc28Sum(balls)
	picks := parseTextTokens(content)
	units := len(picks)
	if units <= 0 {
		units = 1
	}
	hit := false
	for _, pick := range picks {
		switch pick {
		case "大":
			if sum >= 14 {
				hit = true
			}
		case "小":
			if sum <= 13 {
				hit = true
			}
		case "单":
			if sum%2 == 1 {
				hit = true
			}
		case "双":
			if sum%2 == 0 {
				hit = true
			}
		}
	}
	return betEvaluation{Hit: hit, BetUnits: units, Odds: oddsDingwei}
}

func evaluatePC28Teshu(balls []string, content string) betEvaluation {
	picks := parseTextTokens(content)
	units := len(picks)
	if units <= 0 {
		units = 1
	}
	hit := false
	for _, pick := range picks {
		if pc28TeshuPickHit(balls, pick) {
			hit = true
			break
		}
	}
	return betEvaluation{Hit: hit, BetUnits: units, Odds: oddsZuxuan(3)}
}

func evaluatePC28Longhubao(balls []string, content string) betEvaluation {
	if len(balls) < 3 {
		return betEvaluation{BetUnits: 1, Odds: oddsDingwei}
	}
	a, c := atoiBall(balls[0]), atoiBall(balls[2])
	picks := parseTextTokens(content)
	units := len(picks)
	if units <= 0 {
		units = 1
	}
	hit := false
	for _, pick := range picks {
		switch pick {
		case "龙":
			if a > c {
				hit = true
			}
		case "虎":
			if a < c {
				hit = true
			}
		case "豹":
			if a == c {
				hit = true
			}
		}
	}
	return betEvaluation{Hit: hit, BetUnits: units, Odds: oddsDingwei}
}

func pc28Sum(balls []string) int {
	sum := 0
	for _, b := range balls {
		sum += atoiBall(b)
	}
	return sum
}

func pc28TeshuPickHit(balls []string, pick string) bool {
	if len(balls) < 3 {
		return false
	}
	vals := []int{atoiBall(balls[0]), atoiBall(balls[1]), atoiBall(balls[2])}
	sum := vals[0] + vals[1] + vals[2]
	switch strings.TrimSpace(pick) {
	case "豹子":
		return vals[0] == vals[1] && vals[1] == vals[2]
	case "对子":
		return vals[0] == vals[1] || vals[1] == vals[2] || vals[0] == vals[2]
	case "顺子":
		return pc28IsStraight(vals)
	case "极大":
		return sum >= 22
	case "极小":
		return sum <= 5
	default:
		return false
	}
}

func pc28IsStraight(vals []int) bool {
	if len(vals) != 3 {
		return false
	}
	sorted := append([]int(nil), vals...)
	for i := 0; i < len(sorted)-1; i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[j] < sorted[i] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}
	return sorted[1] == sorted[0]+1 && sorted[2] == sorted[1]+1
}
