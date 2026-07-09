package schemes

import (
	"sort"
	"strings"
)

func evaluateK3ByBetMode(rule playRule, balls []string, content string) (betEvaluation, bool) {
	mode := strings.TrimSpace(rule.BetMode)
	switch mode {
	case "hezhi":
		return evaluateK3Hezhi(balls, content), true
	case "danshi":
		return evaluateK3ErTongDan(balls, content), true
	case "fushi":
		if rule.PlayTypeID == "tonghao" && strings.Contains(rule.CatalogSubID, "ertong") {
			return evaluateK3ErTongFu(balls, content), true
		}
		return evaluateK3Biaozhun(balls, content), true
	case "tonghao":
		return evaluateK3SanTong(balls, content), true
	case "butong":
		return evaluateK3ErBuTong(balls, content), true
	case "shoudong":
		return evaluateK3ShouDong(balls, content), true
	case "lianhao":
		return evaluateK3SanLian(balls, content), true
	case "dantiao":
		return evaluateK3DanTiao(balls, content), true
	}
	return betEvaluation{}, false
}

func evaluateK3Hezhi(balls []string, content string) betEvaluation {
	sum := k3DiceSum(balls)
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

func evaluateK3ErTongDan(balls []string, content string) betEvaluation {
	parts := strings.Split(content, "|")
	if len(parts) < 2 {
		parts = strings.FieldsFunc(content, func(r rune) bool { return r == ',' || r == '，' || r == '+' })
	}
	units := 1
	if len(parts) >= 2 {
		pair := atoiBall(strings.TrimSpace(parts[0]))
		other := atoiBall(strings.TrimSpace(parts[1]))
		hit := k3CountFace(balls, pair) == 2 && k3CountFace(balls, other) == 1
		return betEvaluation{Hit: hit, BetUnits: units, Odds: oddsZuxuan(2)}
	}
	return betEvaluation{BetUnits: units, Odds: oddsZuxuan(2)}
}

func evaluateK3ErTongFu(balls []string, content string) betEvaluation {
	picks := parseIntTokens(content)
	units := len(picks)
	if units <= 0 {
		units = 1
	}
	hit := false
	for _, p := range picks {
		if p >= 1 && p <= 6 && k3CountFace(balls, p) >= 2 {
			hit = true
			break
		}
	}
	return betEvaluation{Hit: hit, BetUnits: units, Odds: oddsZuxuan(2)}
}

func evaluateK3SanTong(balls []string, content string) betEvaluation {
	picks := parseIntTokens(content)
	units := len(picks)
	if units <= 0 {
		units = 1
	}
	hit := false
	for _, p := range picks {
		if p >= 1 && p <= 6 && k3CountFace(balls, p) == 3 {
			hit = true
			break
		}
	}
	return betEvaluation{Hit: hit, BetUnits: units, Odds: oddsZhixuan(1)}
}

func evaluateK3ErBuTong(balls []string, content string) betEvaluation {
	picks := parseIntTokens(content)
	units := lhcComboUnits(len(picks), 2)
	if units <= 0 {
		units = 1
	}
	vals := k3DiceValues(balls)
	hit := false
	if len(picks) >= 2 {
		for i := 0; i < len(picks); i++ {
			for j := i + 1; j < len(picks); j++ {
				a, b := picks[i], picks[j]
				if containsInt(vals, a) && containsInt(vals, b) && a != b {
					hit = true
					break
				}
			}
		}
	}
	return betEvaluation{Hit: hit, BetUnits: units, Odds: oddsZuxuan(2)}
}

func evaluateK3Biaozhun(balls []string, content string) betEvaluation {
	picks := parseIntTokens(content)
	units := len(picks)
	if units <= 0 {
		units = 1
	}
	vals := k3DiceValues(balls)
	hit := false
	matched := 0
	for _, p := range picks {
		if containsInt(vals, p) {
			matched++
		}
	}
	if matched >= 2 {
		hit = true
	}
	return betEvaluation{Hit: hit, BetUnits: units, Odds: oddsZuxuan(2)}
}

func evaluateK3ShouDong(balls []string, content string) betEvaluation {
	return evaluateK3Biaozhun(balls, content)
}

func evaluateK3SanLian(balls []string, content string) betEvaluation {
	vals := k3DiceValues(balls)
	sort.Ints(vals)
	hit := len(vals) == 3 && vals[1] == vals[0]+1 && vals[2] == vals[1]+1
	units := 1
	if strings.TrimSpace(content) != "" {
		units = len(parseTextTokens(content))
		if units <= 0 {
			units = 1
		}
	}
	return betEvaluation{Hit: hit, BetUnits: units, Odds: oddsZhixuan(1)}
}

func evaluateK3DanTiao(balls []string, content string) betEvaluation {
	picks := parseIntTokens(content)
	units := len(picks)
	if units <= 0 {
		units = 1
	}
	hit := false
	for _, p := range picks {
		if p >= 1 && p <= 6 && k3CountFace(balls, p) >= 1 {
			hit = true
			break
		}
	}
	return betEvaluation{Hit: hit, BetUnits: units, Odds: oddsDingwei}
}

func k3DiceSum(balls []string) int {
	sum := 0
	for _, b := range balls {
		sum += atoiBall(b)
	}
	return sum
}

func k3DiceValues(balls []string) []int {
	out := make([]int, 0, len(balls))
	for _, b := range balls {
		out = append(out, atoiBall(b))
	}
	return out
}

func k3CountFace(balls []string, face int) int {
	n := 0
	for _, b := range balls {
		if atoiBall(b) == face {
			n++
		}
	}
	return n
}

func containsInt(vals []int, target int) bool {
	for _, v := range vals {
		if v == target {
			return true
		}
	}
	return false
}

