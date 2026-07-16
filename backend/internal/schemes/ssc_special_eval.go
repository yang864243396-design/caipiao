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
	// 任选须走专用评估（勿被 fushi→直选复式抢走）
	// catalog 常存 typeId=g011，须与语义 id renxuan 同等对待
	if isRenxuanPlayType(rule.PlayTypeID) {
		return evaluateMultiZone(rule, balls, content, evaluateRenxuan), true
	}
	evalOne, ok := sscBetModeEvaluator(mode, rule)
	if !ok {
		return betEvaluation{}, false
	}
	return evaluateMultiZone(rule, balls, content, evalOne), true
}

func sscBetModeEvaluator(mode string, rule playRule) (func(playRule, []string, string) betEvaluation, bool) {
	switch mode {
	case "longhu", "longhuhe":
		return evaluateLonghu, true
	case "danshi", "zhixuan_ds":
		return evaluateZhixuanDanshi, true
	case "fushi", "zhixuan_fs":
		if rule.SegmentLen == 1 {
			return evaluateDingwei, true
		}
		return evaluateZhixuanFushi, true
	case "zuxuan_fs", "zuxuan_ds":
		return evaluateZuxuanFushi, true
	case "hezhi":
		return evaluateHezhi, true
	case "kuadu":
		return evaluateKuadu, true
	case "budingwei":
		return evaluateBudingwei, true
	case "dxds", "daxiao", "danshuang":
		return evaluateDxds, true
	case "zu3":
		return evaluateZu3, true
	case "zu6":
		return evaluateZu6, true
	case "zuhe":
		return evaluateZuhe, true
	case "baodan":
		return evaluateBaodan, true
	case "hunhe":
		return evaluateHunhe, true
	case "weishu":
		return evaluateWeishu, true
	case "teshu":
		return evaluateTeshu, true
	case "zu24":
		return evaluateZu24, true
	case "zu12":
		return evaluateZu12, true
	case "zu60":
		return evaluateZu60, true
	case "zu30":
		return evaluateZu30, true
	case "zu120":
		return evaluateZu120, true
	default:
		return nil, false
	}
}

// evaluateMultiZone 对前中后三/前后三/前后二等多区位：逐区判中，注数×区数，
// 部分区位中奖时把赔率折成等价 odds（使 amount*odds ≈ 分区独立结算净盈亏之和）。
func evaluateMultiZone(
	rule playRule,
	balls []string,
	content string,
	fn func(playRule, []string, string) betEvaluation,
) betEvaluation {
	starts := multiZoneSegmentStarts(rule)
	if len(starts) <= 1 {
		return fn(rule, balls, content)
	}
	n := len(starts)
	sumNet := 0.0
	prizeOnly := 0.0
	anyHit := false
	var base betEvaluation
	for i, start := range starts {
		zr := rule
		zr.SegmentStart = start
		zr.SegmentPos = nil
		ev := fn(zr, balls, content)
		if i == 0 {
			base = ev
		}
		if ev.Hit {
			anyHit = true
			zoneNet := ev.Odds * float64(ev.BetUnits)
			if ev.PrizeNet > 0 {
				zoneNet = ev.PrizeNet
				prizeOnly += ev.PrizeNet
			}
			sumNet += zoneNet
		} else {
			sumNet -= float64(ev.BetUnits)
		}
	}
	units := base.BetUnits * n
	if units <= 0 {
		units = n
	}
	if !anyHit {
		return betEvaluation{Hit: false, BetUnits: units, Odds: base.Odds}
	}
	// 嵌套小奖：整单净额被其它区位亏损打成 ≤0（或明显小于小奖本身）时，
	// 按第三方口径记小奖净奖金，避免派奖对比出现「平台=0 / 第三方≈9.65」。
	if prizeOnly > 0 && sumNet < prizeOnly {
		sumNet = prizeOnly
	}
	oddsEff := sumNet / float64(units)
	return betEvaluation{Hit: true, BetUnits: units, Odds: oddsEff}
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
	raw := strings.TrimSpace(subID)
	s := strings.ToLower(raw)
	tieOnly = strings.Contains(raw, "和") || strings.Contains(s, "_he") || strings.HasSuffix(s, "he")

	// 中文位名：万千、百十、万个…
	if p1, p2, ok := longhuPairIndexChinese(raw); ok {
		return p1, p2, tieOnly
	}

	if !strings.HasPrefix(s, "lh_") {
		// 数字 guaji subId：依赖 CatalogSubID 已合并 playMethod
		if p1, p2, ok := longhuPairIndexChinese(raw); ok {
			return p1, p2, tieOnly
		}
		return -1, -1, false
	}
	s = strings.TrimPrefix(s, "lh_")
	if idx := strings.LastIndex(s, "_"); idx >= 0 {
		tieOnly = s[idx+1:] == "he" || tieOnly
		s = s[:idx]
	}
	a, b, _ := longhuPairIndex(s)
	return a, b, tieOnly
}

func longhuPairIndexChinese(label string) (int, int, bool) {
	order := []struct {
		name string
		idx  int
	}{
		{"万", 0}, {"千", 1}, {"百", 2}, {"十", 3}, {"个", 4},
	}
	var found []int
	for _, o := range order {
		if strings.Contains(label, o.name) {
			found = append(found, o.idx)
		}
	}
	if len(found) >= 2 {
		return found[0], found[1], true
	}
	return -1, -1, false
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
	n := len(picks)
	if need <= 0 {
		need = 1
	}
	// 一码：选几个号几注；二码/三码：C(n,k)（与 guajibet countBudingweiBetNums / 第三方一致）
	units := budingweiBetUnits(n, need)
	if units <= 0 {
		units = 1
	}
	hitN := budingweiHitComboCount(seg, picks, need)
	unitNet := oddsBudingweiUnitNet(need, rule.SegmentLen)
	if hitN <= 0 {
		return betEvaluation{Hit: false, BetUnits: units, Odds: unitNet}
	}
	// 按「中几注算几注」：中奖组合得 unitNet，未中组合 −1（对齐第三方净额，避免整单×组选赔率高估）
	net := float64(hitN)*unitNet - float64(units-hitN)
	odds := net / float64(units)
	return betEvaluation{Hit: true, BetUnits: units, Odds: odds, PrizeNet: net}
}

func budingweiBetUnits(pickCount, need int) int {
	if need <= 1 {
		if pickCount <= 0 {
			return 0
		}
		// 一码第三方最多 2 个号
		if pickCount > 2 {
			return 2
		}
		return pickCount
	}
	if pickCount < need {
		return 0
	}
	return combinInt(pickCount, need)
}

func budingweiHit(seg, picks []string, need int) bool {
	return budingweiHitComboCount(seg, picks, need) > 0
}

func budingweiHitComboCount(seg, picks []string, need int) int {
	if need <= 1 {
		n := 0
		for _, p := range picks {
			if containsDigit(seg, p) {
				n++
			}
		}
		if n > 2 {
			n = 2
		}
		return n
	}
	if len(picks) < need {
		return 0
	}
	hit := 0
	idxs := combinations(len(picks), need)
	for _, combo := range idxs {
		ok := true
		for _, i := range combo {
			if !containsDigit(seg, picks[i]) {
				ok = false
				break
			}
		}
		if ok {
			hit++
		}
	}
	return hit
}

// oddsBudingweiUnitNet 不定位单注净赔率（1 元尺度）。五星二码实测净额≈「1 注中×unitNet − 其余挂」。
func oddsBudingweiUnitNet(need, segLen int) float64 {
	if need <= 1 {
		switch {
		case segLen >= 5:
			return 2.2
		case segLen >= 4:
			return 2.5
		default:
			return 3.5
		}
	}
	if need >= 3 {
		switch {
		case segLen >= 5:
			return 35.0
		case segLen >= 4:
			return 24.0
		default:
			return 16.0
		}
	}
	// 二码
	switch {
	case segLen >= 5:
		return 10.95 // 对齐 E2E：6 注中 1 → net≈5.95
	case segLen >= 4:
		return 12.0
	default:
		return 16.0
	}
}

func combinInt(n, k int) int {
	if k < 0 || n < k {
		return 0
	}
	if k == 0 || k == n {
		return 1
	}
	if k > n-k {
		k = n - k
	}
	out := 1
	for i := 0; i < k; i++ {
		out = out * (n - i) / (i + 1)
	}
	return out
}

func budingweiNeedCount(subID string) int {
	s := strings.ToLower(subID)
	switch {
	case strings.Contains(subID, "三码") || strings.Contains(s, "_3ma") || strings.Contains(s, "3ma"):
		return 3
	case strings.Contains(subID, "二码") || strings.Contains(s, "_2ma") || strings.Contains(s, "2ma"):
		return 2
	default:
		return 1
	}
}

func evaluateDxds(rule playRule, balls []string, content string) betEvaluation {
	seg := drawSegmentForRule(rule, balls)
	odds := oddsDingwei
	if isWuxingSumDxdsRule(rule) {
		odds = 1.9 // 五星和值大小/单双：V6 实测净额约 1.9
	}
	if len(seg) == 0 {
		return betEvaluation{BetUnits: 1, Odds: odds}
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
	return betEvaluation{Hit: hit, BetUnits: units, Odds: odds}
}

func isWuxingSumDxdsRule(rule playRule) bool {
	text := strings.ToLower(rule.CatalogSubID + " " + rule.SubPlayID + " " + rule.BetMode + " " + rule.PlayTypeID)
	label := rule.CatalogSubID + rule.SubPlayID
	if strings.Contains(label, "和值大小") || strings.Contains(label, "和值单双") ||
		strings.Contains(label, "五星和值") {
		return true
	}
	if strings.Contains(text, "hz") || strings.Contains(text, "hezhi") {
		return strings.Contains(text, "dx") || strings.Contains(text, "ds") ||
			strings.Contains(text, "daxiao") || strings.Contains(text, "danshuang") ||
			strings.Contains(text, "大小") || strings.Contains(text, "单双")
	}
	return false
}

func dxdsPickHit(rule playRule, pick string, seg []string) bool {
	pick = strings.TrimSpace(pick)
	useSum := isWuxingSumDxdsRule(rule) || strings.Contains(strings.ToLower(rule.CatalogSubID), "hz")
	if useSum {
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
	mode := strings.ToLower(strings.TrimSpace(rule.BetMode))
	sub := strings.ToLower(rule.SubPlayID + " " + rule.CatalogSubID)
	if mode == "hezhi" || strings.Contains(sub, "hezhi") || strings.Contains(sub, "和值") {
		return evaluateRenxuanHezhi(balls, content, n)
	}
	if mode == "weishu" || strings.Contains(sub, "weishu") || strings.Contains(sub, "尾数") {
		return evaluateRenxuanWeishu(balls, content, n)
	}

	lines := splitGroupLines(content)
	// pipe：千个|12 → 当作和值（若 BetMode 未标 hezhi 但内容是和值 wire）
	if posLabel, picks, ok := splitPipeContent(content); ok {
		if mode == "hezhi" || looksLikeRenxuanHezhiPicks(picks) {
			return evaluateRenxuanHezhi(balls, content, n)
		}
		_ = posLabel
	}
	for len(lines) < 5 {
		lines = append(lines, "")
	}
	pools := make([][]string, 5)
	for i := 0; i < 5; i++ {
		pools[i] = parseDigitTokens(lines[i])
	}
	switch {
	case strings.Contains(sub, "zu24"):
		return evaluateRenxuanZuN(balls, pools, n, isZu24Pattern)
	case strings.Contains(sub, "zu12"):
		return evaluateRenxuanZuN(balls, pools, n, isZu12Pattern)
	case strings.Contains(sub, "zu60"):
		return evaluateRenxuanZuN(balls, pools, n, isZu60Pattern)
	case strings.Contains(sub, "zu30"):
		return evaluateRenxuanZuN(balls, pools, n, isZu30Pattern)
	case strings.Contains(sub, "zu120"):
		return evaluateRenxuanZuN(balls, pools, n, isZu120Pattern)
	case strings.Contains(sub, "zuxuan") || strings.Contains(sub, "zu3") || strings.Contains(sub, "zu6"):
		return evaluateRenxuanZuxuan(balls, pools, n)
	}
	// 五位逗号直选复式 content
	if parts := strings.Split(content, ","); len(parts) == 5 {
		for i := 0; i < 5; i++ {
			pools[i] = parseDigitTokens(parts[i])
		}
		return evaluateRenxuanZhixuan(balls, pools, n)
	}
	return evaluateRenxuanZhixuan(balls, pools, n)
}

func splitPipeContent(content string) (posLabel, picks string, ok bool) {
	content = strings.TrimSpace(content)
	pipe := strings.Index(content, "|")
	if pipe <= 0 || pipe >= len(content)-1 {
		return "", "", false
	}
	return strings.TrimSpace(content[:pipe]), strings.TrimSpace(content[pipe+1:]), true
}

func looksLikeRenxuanHezhiPicks(picks string) bool {
	toks := parseIntTokens(picks)
	if len(toks) == 0 {
		return false
	}
	for _, t := range toks {
		if t > 9 {
			return true
		}
	}
	return false
}

func renxuanPositionsFromLabel(posLabel string, n int) []int {
	order := []struct {
		name string
		idx  int
	}{
		{"万", 0}, {"千", 1}, {"百", 2}, {"十", 3}, {"个", 4},
	}
	var found []int
	for _, o := range order {
		if strings.Contains(posLabel, o.name) {
			found = append(found, o.idx)
		}
	}
	if len(found) >= n {
		return found[:n]
	}
	switch n {
	case 4:
		return []int{0, 1, 2, 3}
	case 3:
		return []int{0, 1, 4}
	default:
		return []int{1, 4}
	}
}

func evaluateRenxuanHezhi(balls []string, content string, pickCount int) betEvaluation {
	posLabel, picks, ok := splitPipeContent(content)
	if !ok {
		lines := splitGroupLines(content)
		if len(lines) >= 2 {
			posLabel = lines[0]
			picks = strings.Join(lines[1:], ",")
		} else {
			picks = content
			posLabel = ""
		}
	}
	positions := renxuanPositionsFromLabel(posLabel, pickCount)
	sum := 0
	for _, p := range positions {
		if p >= 0 && p < len(balls) {
			sum += atoiBall(balls[p])
		}
	}
	vals := parseIntTokens(picks)
	units := 0
	hit := false
	for _, v := range vals {
		units += countOrderedSumCombos(v, pickCount)
		if v == sum {
			hit = true
		}
	}
	if units <= 0 {
		units = len(vals)
	}
	if units <= 0 {
		units = 1
	}
	return betEvaluation{Hit: hit, BetUnits: units, Odds: oddsZhixuan(pickCount)}
}

func evaluateRenxuanWeishu(balls []string, content string, pickCount int) betEvaluation {
	posLabel, picks, ok := splitPipeContent(content)
	if !ok {
		lines := splitGroupLines(content)
		if len(lines) >= 2 {
			posLabel = lines[0]
			picks = strings.Join(lines[1:], ",")
		} else {
			picks = content
		}
	}
	positions := renxuanPositionsFromLabel(posLabel, pickCount)
	sum := 0
	for _, p := range positions {
		if p >= 0 && p < len(balls) {
			sum += atoiBall(balls[p])
		}
	}
	tail := sum % 10
	vals := parseIntTokens(picks)
	hit := false
	for _, v := range vals {
		if v == tail {
			hit = true
			break
		}
	}
	units := len(vals)
	if units <= 0 {
		units = 1
	}
	return betEvaluation{Hit: hit, BetUnits: units, Odds: oddsZhixuan(pickCount)}
}

func countOrderedSumCombos(targetSum, positions int) int {
	if positions <= 0 {
		return 0
	}
	if positions == 1 {
		if targetSum >= 0 && targetSum <= 9 {
			return 1
		}
		return 0
	}
	n := 0
	var walk func(left, rem int)
	walk = func(left, rem int) {
		if left == 1 {
			if rem >= 0 && rem <= 9 {
				n++
			}
			return
		}
		for d := 0; d <= 9; d++ {
			if rem-d < 0 {
				break
			}
			walk(left-1, rem-d)
		}
	}
	walk(positions, targetSum)
	return n
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

func isRenxuanPlayType(typeID string) bool {
	t := strings.TrimSpace(strings.ToLower(typeID))
	return t == "renxuan" || t == "g011"
}

func renPickCount(subID string) int {
	s := strings.ToLower(subID)
	raw := subID
	switch {
	case strings.Contains(raw, "任选四"), strings.Contains(raw, "任四"),
		strings.HasPrefix(s, "ren4"):
		return 4
	case strings.Contains(raw, "任选三"), strings.Contains(raw, "任三"),
		strings.HasPrefix(s, "ren3"):
		return 3
	case strings.Contains(raw, "任选二"), strings.Contains(raw, "任二"),
		strings.HasPrefix(s, "ren2"):
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
