package schemes

import (
	"strconv"
	"strings"
)

func evaluateSSCByBetMode(rule playRule, balls []string, content string) (betEvaluation, bool) {
	mode := strings.TrimSpace(rule.BetMode)
	if mode == "" {
		return betEvaluation{}, false
	}
	switch mode {
	case "longhu", "longhuhe":
		return evaluateLonghu(rule, balls, content), true
	case "hezhi":
		return evaluateHezhi(rule, balls, content), true
	case "kuadu":
		return evaluateKuadu(rule, balls, content), true
	case "budingwei":
		return evaluateBudingwei(rule, balls, content), true
	case "dxds", "daxiao", "danshuang":
		return evaluateDxds(rule, balls, content), true
	case "zu3":
		return evaluateZu3(rule, balls, content), true
	case "zu6":
		return evaluateZu6(rule, balls, content), true
	case "zuhe":
		return evaluateZuhe(rule, balls, content), true
	case "baodan":
		return evaluateBaodan(rule, balls, content), true
	case "hunhe":
		return evaluateHunhe(rule, balls, content), true
	case "weishu":
		return evaluateWeishu(rule, balls, content), true
	case "teshu":
		return evaluateTeshu(rule, balls, content), true
	case "zu24":
		return evaluateZu24(rule, balls, content), true
	case "zu12":
		return evaluateZu12(rule, balls, content), true
	case "zu60":
		return evaluateZu60(rule, balls, content), true
	case "zu30":
		return evaluateZu30(rule, balls, content), true
	case "zu120":
		return evaluateZu120(rule, balls, content), true
	}
	if rule.PlayTypeID == "renxuan" {
		return evaluateRenxuan(rule, balls, content), true
	}
	return betEvaluation{}, false
}

func evaluateLonghu(rule playRule, balls []string, content string) betEvaluation {
	p1, p2, wantTie := longhuPositions(rule.CatalogSubID)
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
	if wantTie && a != b {
		hit = false
	}
	return betEvaluation{Hit: hit, BetUnits: units, Odds: oddsDingwei}
}

func longhuPositions(subID string) (p1, p2 int, tieOnly bool) {
	s := strings.ToLower(strings.TrimSpace(subID))
	if !strings.HasPrefix(s, "lh_") {
		return -1, -1, false
	}
	s = strings.TrimPrefix(s, "lh_")
	if idx := strings.LastIndex(s, "_"); idx >= 0 {
		tieOnly = s[idx+1:] == "he"
		s = s[:idx]
	}
	return longhuPairIndex(s)
}

func longhuPairIndex(pair string) (int, int, bool) {
	keys := []struct {
		key string
		idx int
	}{
		{"qian", 1},
		{"wan", 0},
		{"bai", 2},
		{"shi", 3},
		{"ge", 4},
	}
	var found []int
	rest := pair
	for len(rest) > 0 {
		matched := false
		for _, k := range keys {
			if strings.HasPrefix(rest, k.key) {
				found = append(found, k.idx)
				rest = strings.TrimPrefix(rest, k.key)
				matched = true
				break
			}
		}
		if !matched {
			return -1, -1, false
		}
	}
	if len(found) >= 2 {
		return found[0], found[1], false
	}
	return -1, -1, false
}

func normalizeLonghuPick(s string) string {
	s = strings.TrimSpace(s)
	switch strings.ToLower(s) {
	case "long", "龙":
		return "龙"
	case "hu", "虎":
		return "虎"
	case "he", "和", "tie":
		return "和"
	default:
		return s
	}
}

func evaluateHezhi(rule playRule, balls []string, content string) betEvaluation {
	seg := drawSegmentForRule(rule, balls)
	if len(seg) == 0 {
		return betEvaluation{BetUnits: 1, Odds: oddsZhixuan(rule.SegmentLen)}
	}
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
	return betEvaluation{Hit: hit, BetUnits: units, Odds: oddsZhixuan(rule.SegmentLen)}
}

func evaluateKuadu(rule playRule, balls []string, content string) betEvaluation {
	seg := drawSegmentForRule(rule, balls)
	if len(seg) == 0 {
		return betEvaluation{BetUnits: 1, Odds: oddsZhixuan(rule.SegmentLen)}
	}
	vals := make([]int, len(seg))
	for i, d := range seg {
		vals[i] = atoiBall(d)
	}
	span := maxInt(vals) - minInt(vals)
	picks := parseIntTokens(content)
	units := len(picks)
	if units <= 0 {
		units = 1
	}
	hit := false
	for _, p := range picks {
		if p == span {
			hit = true
			break
		}
	}
	return betEvaluation{Hit: hit, BetUnits: units, Odds: oddsZhixuan(rule.SegmentLen)}
}

func evaluateBudingwei(rule playRule, balls []string, content string) betEvaluation {
	seg := drawSegmentForRule(rule, balls)
	need := budingweiNeedCount(rule.CatalogSubID)
	picks := parsePickTokensForRule(rule, content)
	units := len(picks)
	if units <= 0 {
		units = 1
	}
	if need <= 0 {
		need = 1
	}
	matched := 0
	for _, p := range picks {
		if containsDigit(seg, p) {
			matched++
		}
	}
	hit := matched >= need
	return betEvaluation{Hit: hit, BetUnits: units, Odds: oddsZuxuan(rule.SegmentLen)}
}

func budingweiNeedCount(subID string) int {
	s := strings.ToLower(subID)
	switch {
	case strings.Contains(s, "_3ma"):
		return 3
	case strings.Contains(s, "_2ma"):
		return 2
	default:
		return 1
	}
}

func evaluateDxds(rule playRule, balls []string, content string) betEvaluation {
	seg := drawSegmentForRule(rule, balls)
	if len(seg) == 0 {
		return betEvaluation{BetUnits: 1, Odds: oddsDingwei}
	}
	picks := parseTextTokens(content)
	units := len(picks)
	if units <= 0 {
		units = 1
	}
	hit := false
	for _, pick := range picks {
		if dxdsPickHit(rule, pick, seg) {
			hit = true
			break
		}
	}
	return betEvaluation{Hit: hit, BetUnits: units, Odds: oddsDingwei}
}

func dxdsPickHit(rule playRule, pick string, seg []string) bool {
	pick = strings.TrimSpace(pick)
	if strings.Contains(strings.ToLower(rule.CatalogSubID), "hz") {
		sum := 0
		for _, d := range seg {
			sum += atoiBall(d)
		}
		switch pick {
		case "大":
			return sum >= 23
		case "小":
			return sum <= 22
		case "单":
			return sum%2 == 1
		case "双":
			return sum%2 == 0
		}
	}
	for _, d := range seg {
		n := atoiBall(d)
		switch pick {
		case "大":
			if n >= 5 {
				return true
			}
		case "小":
			if n <= 4 {
				return true
			}
		case "单":
			if n%2 == 1 {
				return true
			}
		case "双":
			if n%2 == 0 {
				return true
			}
		}
	}
	return false
}

func evaluateRenxuan(rule playRule, balls []string, content string) betEvaluation {
	n := renPickCount(rule.CatalogSubID)
	if n <= 0 || n > 5 {
		n = 2
	}
	lines := splitGroupLines(content)
	for len(lines) < 5 {
		lines = append(lines, "")
	}
	pools := make([][]string, 5)
	for i := 0; i < 5; i++ {
		pools[i] = parseDigitTokens(lines[i])
	}
	sub := strings.ToLower(rule.SubPlayID)
	catalogSub := strings.ToLower(rule.CatalogSubID)
	switch {
	case strings.Contains(sub, "zu24") || strings.Contains(catalogSub, "zu24"):
		return evaluateRenxuanZuN(balls, pools, n, isZu24Pattern)
	case strings.Contains(sub, "zu12") || strings.Contains(catalogSub, "zu12"):
		return evaluateRenxuanZuN(balls, pools, n, isZu12Pattern)
	case strings.Contains(sub, "zu60") || strings.Contains(catalogSub, "zu60"):
		return evaluateRenxuanZuN(balls, pools, n, isZu60Pattern)
	case strings.Contains(sub, "zu30") || strings.Contains(catalogSub, "zu30"):
		return evaluateRenxuanZuN(balls, pools, n, isZu30Pattern)
	case strings.Contains(sub, "zu120") || strings.Contains(catalogSub, "zu120"):
		return evaluateRenxuanZuN(balls, pools, n, isZu120Pattern)
	case strings.Contains(sub, "zuxuan") || strings.Contains(sub, "zu3") || strings.Contains(sub, "zu6"):
		return evaluateRenxuanZuxuan(balls, pools, n)
	}
	return evaluateRenxuanZhixuan(balls, pools, n)
}

func evaluateRenxuanZhixuan(balls []string, pools [][]string, pickCount int) betEvaluation {
	combos := combinations(5, pickCount)
	units := 0
	hit := false
	for _, combo := range combos {
		u := 1
		ok := true
		for _, pos := range combo {
			if len(pools[pos]) == 0 {
				u = 0
				ok = false
				break
			}
			u *= len(pools[pos])
		}
		if u > 0 {
			units += u
		}
		if ok && !hit {
			match := true
			for _, pos := range combo {
				if pos >= len(balls) || !containsDigit(pools[pos], balls[pos]) {
					match = false
					break
				}
			}
			if match {
				hit = true
			}
		}
	}
	if units <= 0 {
		units = 1
	}
	return betEvaluation{Hit: hit, BetUnits: units, Odds: oddsZhixuan(pickCount)}
}

func evaluateRenxuanZuxuan(balls []string, pools [][]string, pickCount int) betEvaluation {
	combos := combinations(5, pickCount)
	units := 0
	hit := false
	for _, combo := range combos {
		u := 1
		for _, pos := range combo {
			if len(pools[pos]) == 0 {
				u = 0
				break
			}
			u *= len(pools[pos])
		}
		if u > 0 {
			units += u
		}
		if !hit {
			seg := make([]string, 0, pickCount)
			for _, pos := range combo {
				if pos < len(balls) {
					seg = append(seg, balls[pos])
				}
			}
			poolFlat := []string{}
			for _, pos := range combo {
				poolFlat = append(poolFlat, pools[pos]...)
			}
			if zuxuanPoolHit(seg, poolFlat) {
				hit = true
			}
		}
	}
	if units <= 0 {
		units = 1
	}
	return betEvaluation{Hit: hit, BetUnits: units, Odds: oddsZuxuan(pickCount)}
}

func renPickCount(subID string) int {
	s := strings.ToLower(subID)
	switch {
	case strings.HasPrefix(s, "ren4"):
		return 4
	case strings.HasPrefix(s, "ren3"):
		return 3
	case strings.HasPrefix(s, "ren2"):
		return 2
	default:
		return 2
	}
}

func drawSegmentForRule(rule playRule, balls []string) []string {
	if len(rule.SegmentPos) > 0 {
		out := make([]string, 0, len(rule.SegmentPos))
		for _, idx := range rule.SegmentPos {
			if idx >= 0 && idx < len(balls) {
				out = append(out, balls[idx])
			}
		}
		return out
	}
	return drawSegment(balls, rule.SegmentStart, rule.SegmentLen)
}

func parseIntTokens(raw string) []int {
	raw = strings.NewReplacer("\n", ",", "，", ",", " ", ",").Replace(raw)
	parts := strings.Split(raw, ",")
	out := make([]int, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if n, err := strconv.Atoi(p); err == nil {
			out = append(out, n)
		}
	}
	return out
}

func parseTextTokens(raw string) []string {
	raw = strings.NewReplacer("\n", ",", "，", ",", " ", ",").Replace(raw)
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func atoiBall(s string) int {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	n, _ := strconv.Atoi(s)
	return n
}

func minInt(vals []int) int {
	if len(vals) == 0 {
		return 0
	}
	m := vals[0]
	for _, v := range vals[1:] {
		if v < m {
			m = v
		}
	}
	return m
}

func maxInt(vals []int) int {
	if len(vals) == 0 {
		return 0
	}
	m := vals[0]
	for _, v := range vals[1:] {
		if v > m {
			m = v
		}
	}
	return m
}

func combinations(n, k int) [][]int {
	if k <= 0 || k > n {
		return nil
	}
	var out [][]int
	var buf []int
	var dfs func(start int)
	dfs = func(start int) {
		if len(buf) == k {
			c := make([]int, k)
			copy(c, buf)
			out = append(out, c)
			return
		}
		for i := start; i < n; i++ {
			buf = append(buf, i)
			dfs(i + 1)
			buf = buf[:len(buf)-1]
		}
	}
	dfs(0)
	return out
}
