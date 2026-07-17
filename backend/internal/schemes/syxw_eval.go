package schemes

import (
	"regexp"
	"strconv"
	"strings"
)

var syxwRenxuanPattern = regexp.MustCompile(`^rx_(\d+)z(\d+)`)

func evaluateSYXWByBetMode(rule playRule, balls []string, content string) (betEvaluation, bool) {
	rule.NumberPoolMin = 1
	rule.NumberPoolMax = 11
	switch rule.PlayTypeID {
	case "renxuan_fs", "renxuan_ds":
		return evaluateSyxwRenxuan(rule, balls, content), true
	}
	switch strings.TrimSpace(rule.BetMode) {
	case "budingwei":
		return evaluateBudingwei(rule, balls, content), true
	}
	return betEvaluation{}, false
}

func evaluateSyxwRenxuan(rule playRule, balls []string, content string) betEvaluation {
	pickN, matchM, ok := syxwRenxuanNM(rule.CatalogSubID)
	if !ok {
		pickN, matchM = 1, 1
	}
	picks := parsePickTokensForRule(rule, content)
	if rule.BetMode == "danshi" || strings.HasSuffix(rule.CatalogSubID, "_ds") {
		lines := splitGroupLines(content)
		units := 0
		hit := false
		for _, line := range lines {
			linePicks := parsePickTokensForRule(rule, line)
			if len(linePicks) == 0 {
				continue
			}
			units++
			if syxwRenxuanHit(balls, linePicks, matchM) {
				hit = true
			}
		}
		if units <= 0 {
			units = 1
		}
		return betEvaluation{Hit: hit, BetUnits: units, Odds: oddsZuxuan(matchM, rule.OddsBase)}
	}
	units := syxwRenxuanUnits(len(picks), pickN, matchM)
	if units <= 0 {
		units = 1
	}
	hit := syxwRenxuanHit(balls, picks, matchM)
	return betEvaluation{Hit: hit, BetUnits: units, Odds: oddsZuxuan(matchM, rule.OddsBase)}
}

func syxwRenxuanNM(subID string) (pickN, matchM int, ok bool) {
	s := strings.ToLower(strings.TrimSpace(subID))
	s = strings.TrimSuffix(s, "_ds")
	m := syxwRenxuanPattern.FindStringSubmatch(s)
	if len(m) != 3 {
		return 0, 0, false
	}
	pickN, _ = strconv.Atoi(m[1])
	matchM, _ = strconv.Atoi(m[2])
	return pickN, matchM, pickN > 0 && matchM > 0
}

func syxwRenxuanHit(balls, picks []string, need int) bool {
	if need <= 0 {
		need = 1
	}
	drawn := map[string]struct{}{}
	for _, b := range balls {
		drawn[strings.TrimSpace(b)] = struct{}{}
	}
	matched := 0
	for _, p := range picks {
		if _, ok := drawn[p]; ok {
			matched++
		}
	}
	return matched >= need
}

func syxwRenxuanUnits(poolSize, pickN, matchM int) int {
	if poolSize < pickN || pickN <= 0 {
		return 0
	}
	return lhcComboUnits(poolSize, pickN)
}
