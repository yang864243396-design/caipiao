package schemes

import (
	"sort"
	"strings"
)

// playRule describes which draw segment and sub-play mode to evaluate.
type playRule struct {
	PlayTemplate  string
	PlayTypeID    string
	SubPlayID     string
	SegmentStart  int
	SegmentLen    int
	PositionIdx   int // dingwei only
	BetMode       string
	CatalogSubID  string
	SegmentPos    []int // 非连续位段（如前中后三/前后三）
	NumberPoolMin int
	NumberPoolMax int
}

type betEvaluation struct {
	Hit      bool
	BetUnits int
	Odds     float64
}

func resolvePlayRule(cfg map[string]interface{}, playLabel string) playRule {
	playTypeID, _ := cfg["playTypeId"].(string)
	subPlayID, _ := cfg["subPlayId"].(string)
	rule := playRule{
		PlayTemplate: strings.TrimSpace(stringVal(cfg, "playTemplate")),
		PlayTypeID:   strings.TrimSpace(playTypeID),
		SubPlayID:    strings.TrimSpace(subPlayID),
		BetMode:      playBetModeFromConfig(cfg),
	}
	switch rule.PlayTypeID {
	case "hou4", "sixing":
		rule.SegmentStart, rule.SegmentLen = 1, 4
	case "qian3":
		rule.SegmentStart, rule.SegmentLen = 0, 3
	case "zhong3":
		rule.SegmentStart, rule.SegmentLen = 1, 3
	case "dingwei", "":
		rule.SegmentStart = resolvePositionIndex(cfg, playLabel)
		rule.SegmentLen = 1
		rule.PositionIdx = rule.SegmentStart
	default:
		rule.SegmentStart, rule.SegmentLen = 1, 4
	}
	if rule.PlayTypeID == "dingwei" || rule.SegmentLen == 1 {
		rule.PositionIdx = resolvePositionIndex(cfg, playLabel)
		rule.SegmentStart = rule.PositionIdx
	}
	if rule.SubPlayID == "" && rule.PlayTypeID == "dingwei" {
		rule.SubPlayID = "dingwei"
		if rule.BetMode == "" {
			rule.BetMode = "dingwei"
		}
	}
	return rule
}

func drawSegment(balls []string, start, length int) []string {
	if start < 0 || length <= 0 || start >= len(balls) {
		return nil
	}
	end := start + length
	if end > len(balls) {
		end = len(balls)
	}
	seg := make([]string, end-start)
	copy(seg, balls[start:end])
	return seg
}

func evaluatePlayHit(rule playRule, balls []string, groupContent string, contrary bool, contraryPlan string, positionIndex int) betEvaluation {
	if contrary {
		return evaluateContraryHit(rule, balls, contraryPlan, positionIndex)
	}
	if rule.PlayTemplate == "lhc_std" {
		if ev, ok := evaluateLHCByBetMode(rule, balls, groupContent); ok {
			return ev
		}
	} else if rule.PlayTemplate == "syxw_std" {
		if ev, ok := evaluateSYXWByBetMode(rule, balls, groupContent); ok {
			return ev
		}
	} else if rule.PlayTemplate == "pk10_std" {
		if ev, ok := evaluatePK10ByBetMode(rule, balls, groupContent); ok {
			return ev
		}
	} else if rule.PlayTemplate == "k3_std" {
		if ev, ok := evaluateK3ByBetMode(rule, balls, groupContent); ok {
			return ev
		}
	} else if rule.PlayTemplate == "pc28_std" {
		if ev, ok := evaluatePC28ByBetMode(rule, balls, groupContent); ok {
			return ev
		}
	} else if rule.PlayTemplate == "ssc_std" || rule.PlayTemplate == "fast_ssc_std" || rule.PlayTemplate == "" {
		if ev, ok := evaluateSSCByBetMode(rule, balls, groupContent); ok {
			return ev
		}
	}
	sub := rule.SubPlayID
	if sub == "" && rule.SegmentLen == 1 {
		sub = "dingwei"
	}
	switch sub {
	case "zhixuan_ds":
		return evaluateZhixuanDanshi(rule, balls, groupContent)
	case "zuxuan_fs":
		return evaluateZuxuanFushi(rule, balls, groupContent)
	case "zhixuan_fs", "dingwei", "":
		if rule.SegmentLen == 1 {
			return evaluateDingwei(rule, balls, groupContent)
		}
		return evaluateZhixuanFushi(rule, balls, groupContent)
	default:
		if rule.SegmentLen == 1 {
			return evaluateDingwei(rule, balls, groupContent)
		}
		return evaluateZhixuanFushi(rule, balls, groupContent)
	}
}

func evaluateDingwei(rule playRule, balls []string, groupContent string) betEvaluation {
	if strings.Contains(groupContent, "\n") {
		return evaluateDingweiMultiline(rule, balls, groupContent)
	}
	picks := parsePickTokensForRule(rule, groupContent)
	pos := rule.PositionIdx
	if pos < 0 {
		pos = 0
	}
	hit := evaluatePositionHit(balls, pos, picks)
	units := len(picks)
	if units <= 0 {
		units = 1
	}
	return betEvaluation{Hit: hit, BetUnits: units, Odds: oddsDingwei}
}

func evaluateDingweiMultiline(rule playRule, balls []string, groupContent string) betEvaluation {
	lines := splitDingweiPositionLines(groupContent)
	units := 0
	hit := false
	for i := 0; i < 5; i++ {
		line := ""
		if i < len(lines) {
			line = lines[i]
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		picks := parsePickTokensForRule(rule, line)
		if len(picks) == 0 {
			continue
		}
		units += len(picks)
		if evaluatePositionHit(balls, i, picks) {
			hit = true
		}
	}
	if units <= 0 {
		units = 1
	}
	return betEvaluation{Hit: hit, BetUnits: units, Odds: oddsDingwei}
}

func splitDingweiPositionLines(content string) []string {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	return strings.Split(content, "\n")
}

func evaluateZhixuanFushi(rule playRule, balls []string, groupContent string) betEvaluation {
	seg := drawSegment(balls, rule.SegmentStart, rule.SegmentLen)
	if len(seg) != rule.SegmentLen {
		return betEvaluation{BetUnits: 1, Odds: oddsZhixuan(rule.SegmentLen)}
	}
	lines := splitGroupLines(groupContent)
	var pools [][]string
	if len(lines) >= rule.SegmentLen {
		pools = make([][]string, rule.SegmentLen)
		for i := 0; i < rule.SegmentLen; i++ {
			pools[i] = parsePickTokensForRule(rule, lines[i])
		}
	} else {
		pool := parsePickTokensForRule(rule, groupContent)
		if len(pool) == 0 {
			pool = []string{"0"}
		}
		pools = make([][]string, rule.SegmentLen)
		for i := range pools {
			pools[i] = pool
		}
	}
	units := 1
	for _, p := range pools {
		n := len(p)
		if n <= 0 {
			n = 1
		}
		units *= n
	}
	hit := true
	for i, digit := range seg {
		if !containsDigit(pools[i], digit) {
			hit = false
			break
		}
	}
	return betEvaluation{Hit: hit, BetUnits: units, Odds: oddsZhixuan(rule.SegmentLen)}
}

func evaluateZhixuanDanshi(rule playRule, balls []string, groupContent string) betEvaluation {
	seg := drawSegment(balls, rule.SegmentStart, rule.SegmentLen)
	if len(seg) != rule.SegmentLen {
		return betEvaluation{BetUnits: 1, Odds: oddsZhixuan(rule.SegmentLen)}
	}
	tokens := parseSegmentTokensForRule(rule, groupContent, rule.SegmentLen)
	if len(tokens) == 0 {
		tokens = parseNumberTokens(groupContent, rule.SegmentLen)
	}
	units := len(tokens)
	if units <= 0 {
		units = 1
	}
	hit := false
	for _, t := range tokens {
		if ballsMatchToken(seg, t) {
			hit = true
			break
		}
	}
	return betEvaluation{Hit: hit, BetUnits: units, Odds: oddsZhixuan(rule.SegmentLen)}
}

func evaluateZuxuanFushi(rule playRule, balls []string, groupContent string) betEvaluation {
	seg := drawSegment(balls, rule.SegmentStart, rule.SegmentLen)
	if len(seg) != rule.SegmentLen {
		return betEvaluation{BetUnits: 1, Odds: oddsZuxuan(rule.SegmentLen)}
	}
	tokens := parseNumberTokens(groupContent, rule.SegmentLen)
	if len(tokens) == 0 {
		pool := parsePickTokensForRule(rule, groupContent)
		if len(pool) == 0 {
			pool = parseDigitTokens(groupContent)
		}
		hit := zuxuanPoolHit(seg, pool)
		units := zuxuanPoolUnits(pool, rule.SegmentLen)
		if units <= 0 {
			units = 1
		}
		return betEvaluation{Hit: hit, BetUnits: units, Odds: oddsZuxuan(rule.SegmentLen)}
	}
	drawnSorted := sortDigits(seg)
	hit := false
	for _, t := range tokens {
		if sortStringDigits(t) == drawnSorted {
			hit = true
			break
		}
	}
	return betEvaluation{Hit: hit, BetUnits: len(tokens), Odds: oddsZuxuan(rule.SegmentLen)}
}

func evaluateContraryHit(rule playRule, balls []string, planInverse string, positionIndex int) betEvaluation {
	picks := parseContraryPicks(planInverse, positionIndex)
	if rule.SegmentLen == 1 {
		hit := evaluatePositionHit(balls, positionIndex, picks)
		return betEvaluation{Hit: hit, BetUnits: len(picks), Odds: oddsDingwei}
	}
	// 反买多码：逐位取补集后按直选复式判定
	lines := splitGroupLines(planInverse)
	if len(lines) < rule.SegmentLen {
		lines = strings.Split(planInverse, ",")
	}
	pools := make([][]string, rule.SegmentLen)
	for i := 0; i < rule.SegmentLen; i++ {
		forbidden := []string{}
		if i < len(lines) {
			forbidden = parseDigitTokens(lines[i])
		}
		pools[i] = allDigitsExcept(forbidden)
	}
	seg := drawSegment(balls, rule.SegmentStart, rule.SegmentLen)
	units := 1
	for _, p := range pools {
		n := len(p)
		if n <= 0 {
			n = 1
		}
		units *= n
	}
	hit := len(seg) == rule.SegmentLen
	if hit {
		for i, digit := range seg {
			if !containsDigit(pools[i], digit) {
				hit = false
				break
			}
		}
	} else {
		hit = false
	}
	return betEvaluation{Hit: hit, BetUnits: units, Odds: oddsZhixuan(rule.SegmentLen)}
}

func splitGroupLines(content string) []string {
	raw := strings.Split(content, "\n")
	out := make([]string, 0, len(raw))
	for _, line := range raw {
		line = strings.TrimSpace(line)
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}

func parseNumberTokens(raw string, expectLen int) []string {
	raw = strings.NewReplacer("\n", ",", "，", ",", " ", ",").Replace(raw)
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" || !isAllDigits(p) {
			continue
		}
		if expectLen > 0 && len(p) != expectLen {
			continue
		}
		out = append(out, p)
	}
	return out
}

func isAllDigits(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

func containsDigit(pool []string, digit string) bool {
	for _, p := range pool {
		if p == digit {
			return true
		}
	}
	return false
}

func zuxuanPoolHit(seg, pool []string) bool {
	if len(seg) == 0 {
		return false
	}
	for _, d := range seg {
		if !containsDigit(pool, d) {
			return false
		}
	}
	if len(seg) == 3 {
		return zuxuan3Pattern(seg)
	}
	if len(seg) == 4 {
		return zuxuan4Pattern(seg)
	}
	// 其它长度：组内数字均在池中即可
	return true
}

func zuxuan3Pattern(seg []string) bool {
	counts := digitCounts(seg)
	switch len(counts) {
	case 1:
		return true // 豹子
	case 2:
		return true // 组三
	case 3:
		return true // 组六
	default:
		return false
	}
}

func zuxuan4Pattern(seg []string) bool {
	counts := digitCounts(seg)
	switch len(counts) {
	case 1, 2, 3, 4:
		return true
	default:
		return false
	}
}

func digitCounts(seg []string) map[string]int {
	m := map[string]int{}
	for _, d := range seg {
		m[d]++
	}
	return m
}

func zuxuanPoolUnits(pool []string, segLen int) int {
	n := len(pool)
	if n <= 0 {
		return 1
	}
	if segLen == 3 {
		// 组六 C(n,3) + 组三 n*(n-1) 近似
		if n < 3 {
			return n
		}
		return n*(n-1) + n*(n-1)*(n-2)/6
	}
	if segLen == 4 && n >= 4 {
		return n * (n - 1) / 2
	}
	return n
}

func sortDigits(seg []string) string {
	cp := append([]string(nil), seg...)
	sort.Strings(cp)
	return strings.Join(cp, "")
}

func sortStringDigits(s string) string {
	runes := []rune(s)
	sort.Slice(runes, func(i, j int) bool { return runes[i] < runes[j] })
	return string(runes)
}

const oddsDingwei = 9.0

func oddsZhixuan(segLen int) float64 {
	switch segLen {
	case 4:
		return 99.0
	case 3:
		return 49.0
	default:
		return 9.0
	}
}

func oddsZuxuan(segLen int) float64 {
	switch segLen {
	case 4:
		return 24.0
	case 3:
		return 16.0
	default:
		return 9.0
	}
}

func calcPnLWithOdds(amount float64, hit bool, odds float64) float64 {
	if hit {
		return round2(amount * odds)
	}
	return round2(-amount)
}
