package schemes

import "strings"

func evaluateZu3(rule playRule, balls []string, content string) betEvaluation {
	seg := drawSegmentForRule(rule, balls)
	pool := parseDigitTokens(content)
	units := zu3PoolUnits(pool)
	// 不足最少选号时注数为 0（勿回落成 1），由 worker 本期 Skip
	hit := units > 0 && len(seg) == 3 && isZu3Pattern(seg) && allDigitsInPool(seg, pool)
	return betEvaluation{Hit: hit, BetUnits: units, Odds: oddsZuxuan(3, rule.OddsBase)}
}

func evaluateZu6(rule playRule, balls []string, content string) betEvaluation {
	seg := drawSegmentForRule(rule, balls)
	pool := parseDigitTokens(content)
	units := zu6PoolUnits(pool)
	hit := units > 0 && len(seg) == 3 && isZu6Pattern(seg) && allDigitsInPool(seg, pool)
	return betEvaluation{Hit: hit, BetUnits: units, Odds: oddsZuxuan(3, rule.OddsBase)}
}

func evaluateZuhe(rule playRule, balls []string, content string) betEvaluation {
	// 直选组合：按位复式选号，注数 = 位积 × 段长（三星=三星+后二+后一）。
	// 开奖按「最长后缀」嵌套派奖：命中 hitLen 时发放 1…hitLen 各档各 1 注奖金，
	// 绝不是整票注数 × 最高档直选赔率（否则满号池会算出百万级虚高返还）。
	segLen := rule.SegmentLen
	if segLen <= 0 {
		segLen = 3
	}
	pools, product := zuhePositionPools(rule, content, segLen)
	units := product * segLen
	if units <= 0 {
		units = segLen
	}
	seg := drawSegmentForRule(rule, balls)
	if len(seg) != segLen || len(pools) != segLen {
		return betEvaluation{BetUnits: units, Odds: oddsZhixuan(segLen, rule.OddsBase)}
	}
	hitLen := 0
	for n := segLen; n >= 1; n-- {
		ok := true
		for i := 0; i < n; i++ {
			pos := segLen - n + i
			if !containsDigit(pools[pos], seg[pos]) {
				ok = false
				break
			}
		}
		if ok {
			hitLen = n
			break
		}
	}
	if hitLen <= 0 {
		return betEvaluation{Hit: false, BetUnits: units, Odds: oddsZhixuan(segLen, rule.OddsBase)}
	}
	prizeNet := 0.0
	for n := 1; n <= hitLen; n++ {
		prizeNet += oddsZuheLevelPrize(n, rule.OddsBase)
	}
	odds := prizeNet / float64(units)
	return betEvaluation{Hit: true, BetUnits: units, Odds: odds, PrizeNet: prizeNet}
}

// oddsZuheLevelPrize 直选组合单档净奖金（1 元单注尺度）：一星/二星/三星…各自一注。
// 实测（tron_ffc 2 元）：三星+后二+后一返还 = 1940+194+19.4 = 2153.4 → 1 元尺度 970+97+9.7。
func oddsZuheLevelPrize(level int, base float64) float64 {
	switch level {
	case 1:
		return 9.7 * oddsScale(base)
	default:
		return oddsZhixuan(level, base)
	}
}

// oddsZuheNestedPrize 兼容旧名：命中最长后缀 hitLen 时的嵌套奖金合计。
func oddsZuheNestedPrize(hitLen int, base float64) float64 {
	if hitLen <= 0 {
		return 0
	}
	sum := 0.0
	for n := 1; n <= hitLen; n++ {
		sum += oddsZuheLevelPrize(n, base)
	}
	return sum
}

// zuhePositionPools 解析直选组合各位号池，并返回位积。
func zuhePositionPools(rule playRule, content string, segLen int) ([][]string, int) {
	lines := splitGroupLines(content)
	pools := make([][]string, segLen)
	if len(lines) >= segLen {
		for i := 0; i < segLen; i++ {
			pools[i] = parsePickTokensForRule(rule, lines[i])
		}
	} else {
		pool := parsePickTokensForRule(rule, content)
		if len(pool) == 0 {
			pool = []string{"0"}
		}
		for i := range pools {
			pools[i] = pool
		}
	}
	product := 1
	for _, p := range pools {
		n := len(p)
		if n <= 0 {
			n = 1
		}
		product *= n
	}
	return pools, product
}

func evaluateBaodan(rule playRule, balls []string, content string) betEvaluation {
	seg := drawSegmentForRule(rule, balls)
	picks := parseDigitTokens(content)
	pickCount := len(picks)
	if pickCount <= 0 {
		pickCount = 1
	}
	// 与 guajibet 一致：每胆覆盖组六+组三全部组合（三码 54 注）
	units := pickCount * baodanUnitsPerDanLocal(rule.SegmentLen)
	hit := false
	for _, dan := range picks {
		if containsDigit(seg, dan) {
			hit = true
			break
		}
	}
	// calcPnLWithOdds(amount,hit,odds)=amount*odds 记的是净盈亏；
	// 包胆只中 1 注：net≈(amount/units)*prizeOdds - amount ⇒ odds = prizeOdds/units - 1
	odds := oddsZuxuan(rule.SegmentLen, rule.OddsBase)
	if hit && rule.SegmentLen == 3 && units > 1 {
		prize := oddsBaodanZu6 * oddsScale(rule.OddsBase)
		if isZu3Pattern(seg) {
			prize = oddsBaodanZu3 * oddsScale(rule.OddsBase)
		}
		odds = prize/float64(units) - 1
	}
	return betEvaluation{Hit: hit, BetUnits: units, Odds: odds}
}

func baodanUnitsPerDanLocal(segLen int) int {
	if segLen < 2 {
		return 1
	}
	if segLen == 2 {
		return 9
	}
	if segLen == 4 {
		return 84 // C(9,3)
	}
	// 三码：C(9,2)+9*2 = 36+18 = 54
	zu6 := 1
	n, k := 9, segLen-1
	for i := 0; i < k; i++ {
		zu6 = zu6 * (n - i) / (i + 1)
	}
	zu3 := 9 * (segLen - 1)
	return zu6 + zu3
}

func evaluateHunhe(rule playRule, balls []string, content string) betEvaluation {
	odds := oddsZuxuan(rule.SegmentLen, rule.OddsBase)
	seg := drawSegmentForRule(rule, balls)
	tokens := parseNumberTokens(content, rule.SegmentLen)
	units := countUniqueHunheTokens(tokens)
	// 排除豹子后无有效注 → 0 注（本期不投），勿回落成 1
	if units <= 0 {
		return betEvaluation{BetUnits: 0, Odds: odds}
	}
	if len(seg) != rule.SegmentLen {
		return betEvaluation{BetUnits: units, Odds: odds}
	}
	drawnSorted := sortDigits(seg)
	hit := false
	for _, t := range tokens {
		if isBaoziToken(t) {
			continue
		}
		if sortStringDigits(t) == drawnSorted {
			hit = true
			break
		}
	}
	return betEvaluation{Hit: hit, BetUnits: units, Odds: odds}
}

// countUniqueHunheTokens 排除豹子并按组选形态去重。
func countUniqueHunheTokens(tokens []string) int {
	seen := make(map[string]struct{}, len(tokens))
	n := 0
	for _, t := range tokens {
		if isBaoziToken(t) {
			continue
		}
		key := sortStringDigits(t)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		n++
	}
	return n
}

func isBaoziToken(s string) bool {
	if s == "" {
		return false
	}
	for i := 1; i < len(s); i++ {
		if s[i] != s[0] {
			return false
		}
	}
	return true
}

func evaluateWeishu(rule playRule, balls []string, content string) betEvaluation {
	seg := drawSegmentForRule(rule, balls)
	sum := 0
	for _, d := range seg {
		sum += atoiBall(d)
	}
	tail := sum % 10
	picks := parseIntTokens(content)
	units := len(picks)
	if units <= 0 {
		units = 1
	}
	hit := false
	for _, p := range picks {
		if p == tail {
			hit = true
			break
		}
	}
	return betEvaluation{Hit: hit, BetUnits: units, Odds: oddsZuxuan(rule.SegmentLen, rule.OddsBase)}
}

func evaluateTeshu(rule playRule, balls []string, content string) betEvaluation {
	seg := drawSegmentForRule(rule, balls)
	if len(seg) == 0 {
		seg = balls
	}
	sub := strings.ToLower(rule.CatalogSubID)
	picks := parseTextTokens(content)
	units := len(picks)
	if units <= 0 {
		units = 1
	}
	hit := false
	if len(picks) == 0 {
		hit = teshuPatternHit(sub, seg)
	} else {
		for _, pick := range picks {
			if teshuPickHit(sub, seg, pick) {
				hit = true
				break
			}
		}
	}
	return betEvaluation{Hit: hit, BetUnits: units, Odds: oddsZuxuan(len(seg), rule.OddsBase)}
}

func evaluateZu24(rule playRule, balls []string, content string) betEvaluation {
	return evaluateZuNPattern(rule, balls, content, isZu24Pattern)
}

func evaluateZu12(rule playRule, balls []string, content string) betEvaluation {
	return evaluateZuNPattern(rule, balls, content, isZu12Pattern)
}

func evaluateZu60(rule playRule, balls []string, content string) betEvaluation {
	return evaluateZuNPattern(rule, balls, content, isZu60Pattern)
}

func evaluateZu30(rule playRule, balls []string, content string) betEvaluation {
	return evaluateZuNPattern(rule, balls, content, isZu30Pattern)
}

func evaluateZu120(rule playRule, balls []string, content string) betEvaluation {
	return evaluateZuNPattern(rule, balls, content, isZu120Pattern)
}

func evaluateZuNPattern(rule playRule, balls []string, content string, patternFn func([]string) bool) betEvaluation {
	seg := drawSegmentForRule(rule, balls)
	pool := parseDigitTokens(content)
	units := len(pool)
	if units <= 0 {
		units = 1
	}
	hit := len(seg) > 0 && patternFn(seg) && allDigitsInPool(seg, pool)
	return betEvaluation{Hit: hit, BetUnits: units, Odds: oddsZuxuan(len(seg), rule.OddsBase)}
}

func evaluateRenxuanZuN(balls []string, pools [][]string, pickCount int, patternFn func([]string) bool) betEvaluation {
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
			poolFlat := make([]string, 0)
			for _, pos := range combo {
				if pos < len(balls) {
					seg = append(seg, balls[pos])
				}
				poolFlat = append(poolFlat, pools[pos]...)
			}
			if patternFn(seg) && allDigitsInPool(seg, poolFlat) {
				hit = true
			}
		}
	}
	if units <= 0 {
		units = 1
	}
	return betEvaluation{Hit: hit, BetUnits: units, Odds: oddsZuxuan(pickCount, 0)}
}

func isZu3Pattern(seg []string) bool {
	counts := digitCounts(seg)
	if len(counts) != 2 {
		return false
	}
	for _, c := range counts {
		if c != 1 && c != 2 {
			return false
		}
	}
	return true
}

func isZu6Pattern(seg []string) bool {
	return len(seg) == 3 && len(digitCounts(seg)) == 3
}

func isZu24Pattern(seg []string) bool {
	return len(seg) == 4 && len(digitCounts(seg)) == 4
}

func isZu12Pattern(seg []string) bool {
	counts := digitCounts(seg)
	if len(seg) != 4 || len(counts) != 3 {
		return false
	}
	two, ones := false, 0
	for _, c := range counts {
		switch c {
		case 2:
			two = true
		case 1:
			ones++
		}
	}
	return two && ones == 2
}

func isZu60Pattern(seg []string) bool {
	counts := digitCounts(seg)
	if len(seg) != 5 || len(counts) != 4 {
		return false
	}
	two, ones := false, 0
	for _, c := range counts {
		switch c {
		case 2:
			two = true
		case 1:
			ones++
		}
	}
	return two && ones == 3
}

func isZu30Pattern(seg []string) bool {
	counts := digitCounts(seg)
	if len(seg) != 5 || len(counts) != 3 {
		return false
	}
	pairs, ones := 0, 0
	for _, c := range counts {
		switch c {
		case 2:
			pairs++
		case 1:
			ones++
		}
	}
	return pairs == 2 && ones == 1
}

func isZu120Pattern(seg []string) bool {
	return len(seg) == 5 && len(digitCounts(seg)) == 5
}

func allDigitsInPool(seg, pool []string) bool {
	if len(pool) == 0 {
		return false
	}
	for _, d := range seg {
		if !containsDigit(pool, d) {
			return false
		}
	}
	return true
}

func zu3PoolUnits(pool []string) int {
	n := len(pool)
	if n < 2 {
		return 0
	}
	return n * (n - 1)
}

func zu6PoolUnits(pool []string) int {
	n := len(pool)
	if n < 3 {
		return 0
	}
	return n * (n - 1) * (n - 2) / 6
}

func combo2Hit(seg []string, a, b string) bool {
	hasA, hasB := false, false
	for _, d := range seg {
		if d == a {
			hasA = true
		}
		if d == b {
			hasB = true
		}
	}
	return hasA && hasB
}

func parseZuhePicks(content string) [][2]string {
	raw := strings.NewReplacer("\n", ",", "，", ",", " ", ",").Replace(content)
	parts := strings.Split(raw, ",")
	var out [][2]string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if len(p) == 2 && isAllDigits(p) {
			out = append(out, [2]string{string(p[0]), string(p[1])})
			continue
		}
		digits := parseDigitTokens(p)
		if len(digits) == 2 {
			out = append(out, [2]string{digits[0], digits[1]})
		}
	}
	return out
}

func teshuPatternHit(subID string, seg []string) bool {
	switch {
	case strings.Contains(subID, "yifan"):
		return len(digitCounts(seg)) == 1 && len(seg) >= 3
	case strings.Contains(subID, "haoshi"):
		for _, c := range digitCounts(seg) {
			if c >= 2 {
				return true
			}
		}
	case strings.Contains(subID, "sanxing"):
		for _, c := range digitCounts(seg) {
			if c >= 3 {
				return true
			}
		}
	case strings.Contains(subID, "siji"):
		for _, c := range digitCounts(seg) {
			if c >= 4 {
				return true
			}
		}
	default:
		return len(seg) > 0
	}
	return false
}

func teshuPickHit(subID string, seg []string, pick string) bool {
	pick = strings.TrimSpace(pick)
	switch pick {
	case "豹子":
		return len(seg) >= 3 && len(digitCounts(seg)) == 1
	case "对子":
		return isZu3Pattern(seg)
	case "顺子":
		return sscIsStraight(seg)
	case "极大":
		if len(seg) < 3 {
			return false
		}
		return atoiBall(seg[0])+atoiBall(seg[1])+atoiBall(seg[2]) >= 22
	case "极小":
		if len(seg) < 3 {
			return false
		}
		return atoiBall(seg[0])+atoiBall(seg[1])+atoiBall(seg[2]) <= 5
	case "一帆风顺", "yifan":
		return teshuPatternHit("yifan", seg)
	case "好事成双", "haoshi":
		return teshuPatternHit("haoshi", seg)
	case "三星报喜", "sanxing":
		return teshuPatternHit("sanxing", seg)
	case "四季发财", "siji":
		return teshuPatternHit("siji", seg)
	}
	if isAllDigits(pick) && len(pick) == 1 {
		return containsDigit(seg, pick)
	}
	return teshuPatternHit(subID, seg)
}

// sscIsStraight 三星顺子：排序后连号，或 089 / 019 环绕顺子。
func sscIsStraight(seg []string) bool {
	if len(seg) != 3 || len(digitCounts(seg)) != 3 {
		return false
	}
	vals := []int{atoiBall(seg[0]), atoiBall(seg[1]), atoiBall(seg[2])}
	for i := 0; i < 2; i++ {
		for j := i + 1; j < 3; j++ {
			if vals[j] < vals[i] {
				vals[i], vals[j] = vals[j], vals[i]
			}
		}
	}
	if vals[1] == vals[0]+1 && vals[2] == vals[1]+1 {
		return true
	}
	// 089、019
	return vals[0] == 0 && vals[1] == 8 && vals[2] == 9 ||
		vals[0] == 0 && vals[1] == 1 && vals[2] == 9
}
