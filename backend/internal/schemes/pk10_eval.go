package schemes

import (
	"strings"
)

func evaluatePK10ByBetMode(rule playRule, balls []string, content string) (betEvaluation, bool) {
	rule.NumberPoolMin = 1
	rule.NumberPoolMax = 10
	mode := strings.TrimSpace(rule.BetMode)
	switch mode {
	case "longhu":
		return evaluatePK10Longhu(rule, balls, content), true
	case "daxiao", "danshuang":
		return evaluatePK10RankDxds(rule, balls, content), true
	case "hezhi":
		return evaluatePK10Hezhi(rule, balls, content), true
	case "dxds":
		return evaluatePK10DxdsCombo(rule, balls, content), true
	}
	return betEvaluation{}, false
}

func evaluatePK10Longhu(rule playRule, balls []string, content string) betEvaluation {
	p1, p2 := pk10LonghuPositions(rule.CatalogSubID)
	if p1 < 0 || p2 < 0 || p1 >= len(balls) || p2 >= len(balls) {
		return betEvaluation{BetUnits: 1, Odds: oddsDingwei}
	}
	a, b := atoiBall(balls[p1]), atoiBall(balls[p2])
	picks := parseTextTokens(content)
	units := len(picks)
	if units <= 0 {
		units = 1
	}
	hit := false
	for _, pick := range picks {
		switch normalizeLonghuPick(pick) {
		case "龙":
			if a > b {
				hit = true
			}
		case "虎":
			if a < b {
				hit = true
			}
		case "和":
			if a == b {
				hit = true
			}
		}
	}
	return betEvaluation{Hit: hit, BetUnits: units, Odds: oddsDingwei}
}

func pk10LonghuPositions(subID string) (int, int) {
	switch strings.TrimSpace(subID) {
	case "lh_1v10":
		return 0, 9
	case "lh_2v9":
		return 1, 8
	case "lh_3v8":
		return 2, 7
	case "lh_4v7":
		return 3, 6
	case "lh_5v6":
		return 4, 5
	default:
		return -1, -1
	}
}

func evaluatePK10RankDxds(rule playRule, balls []string, content string) betEvaluation {
	pos := pk10RankPosition(rule.CatalogSubID)
	if pos < 0 || pos >= len(balls) {
		return betEvaluation{BetUnits: 1, Odds: oddsDingwei}
	}
	n := atoiBall(balls[pos])
	picks := parseTextTokens(content)
	units := len(picks)
	if units <= 0 {
		units = 1
	}
	hit := false
	for _, pick := range picks {
		if pk10RankPickHit(rule.BetMode, pick, n) {
			hit = true
			break
		}
	}
	return betEvaluation{Hit: hit, BetUnits: units, Odds: oddsDingwei}
}

func pk10RankPosition(subID string) int {
	switch strings.TrimSpace(subID) {
	case "dx_gj", "ds_gj":
		return 0
	case "dx_yj", "ds_yj":
		return 1
	case "dx_jj", "ds_jj":
		return 2
	case "dx_ds4", "ds_ds4":
		return 3
	case "dx_ds5", "ds_ds5":
		return 4
	default:
		return 0
	}
}

func pk10RankPickHit(mode, pick string, rank int) bool {
	pick = strings.TrimSpace(pick)
	switch mode {
	case "daxiao":
		switch pick {
		case "大":
			return rank >= 6
		case "小":
			return rank <= 5
		}
	case "danshuang":
		switch pick {
		case "单":
			return rank%2 == 1
		case "双":
			return rank%2 == 0
		}
	}
	return false
}

func evaluatePK10Hezhi(rule playRule, balls []string, content string) betEvaluation {
	seg := pk10SegmentForSub(rule.CatalogSubID, balls)
	sum := 0
	for _, d := range seg {
		sum += atoiBall(d)
	}
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
	return betEvaluation{Hit: hit, BetUnits: units, Odds: oddsZhixuan(len(seg))}
}

func evaluatePK10DxdsCombo(rule playRule, balls []string, content string) betEvaluation {
	seg := pk10SegmentForSub(rule.CatalogSubID, balls)
	picks := parseTextTokens(content)
	units := len(picks)
	if units <= 0 {
		units = 1
	}
	hit := false
	sum := 0
	for _, d := range seg {
		sum += atoiBall(d)
	}
	for _, pick := range picks {
		switch pick {
		case "大":
			if sum >= 12 {
				hit = true
			}
		case "小":
			if sum <= 11 {
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

func pk10SegmentForSub(subID string, balls []string) []string {
	switch strings.TrimSpace(subID) {
	case "hz_guanya", "dxds_guanya":
		return drawSegment(balls, 0, 2)
	case "hz_shouwei":
		if len(balls) >= 10 {
			return []string{balls[0], balls[9]}
		}
		return drawSegment(balls, 0, 2)
	case "hz_qian3", "dxds_qian3":
		return drawSegment(balls, 0, 3)
	case "hz_hou3", "dxds_hou3":
		return drawSegment(balls, 7, 3)
	default:
		return drawSegment(balls, 0, 2)
	}
}
