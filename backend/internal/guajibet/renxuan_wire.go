package guajibet

import (
	"sort"
	"strconv"
	"strings"
)

var sscPositionRunes = []rune{'万', '千', '百', '十', '个'}

func isSyxwRenxuanMeta(meta RuleMeta) bool {
	switch strings.TrimSpace(meta.PlayTemplate) {
	case "syxw_std":
		switch strings.TrimSpace(meta.TypeID) {
		case "g005", "g006", "renxuan_fs", "renxuan_ds":
			return true
		}
	}
	return false
}

func renxuanUsesZhixuanPositionWire(mode, label string) bool {
	if mode != "fushi" && mode != "danshi" {
		return false
	}
	if strings.Contains(label, "组选") {
		return false
	}
	return strings.Contains(label, "直选") || strings.Contains(label, "复式") || strings.Contains(label, "单式")
}

func renxuanDefaultPositions(k int) []string {
	out := make([]string, len(renxuanDefaultPositionIndices(k)))
	for i, idx := range renxuanDefaultPositionIndices(k) {
		out[i] = strconv.Itoa(idx)
	}
	return out
}

func renxuanDefaultPositionIndices(k int) []int {
	switch k {
	case 4:
		return []int{0, 1, 2, 3}
	case 3:
		return []int{0, 1, 4}
	default:
		return []int{1, 4}
	}
}

func sscPositionIndexFromName(name string) int {
	name = strings.TrimSpace(name)
	for idx, r := range sscPositionRunes {
		if name == string(r) {
			return idx
		}
	}
	return -1
}

func parseSSCPositionNames(raw string) []int {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	if strings.ContainsAny(raw, ",，") {
		var out []int
		for _, tok := range splitPickTokens(raw) {
			if idx := sscPositionIndexFromName(tok); idx >= 0 {
				out = append(out, idx)
			}
		}
		return out
	}
	var out []int
	for _, r := range raw {
		if idx := sscPositionIndexFromName(string(r)); idx >= 0 {
			out = append(out, idx)
		}
	}
	return out
}

func parseRenxuanPositionIndicesFromTokens(tokens []string, k int) []int {
	if k <= 0 {
		k = 2
	}
	if len(tokens) >= k && renxuanPartsLookLikePositions(tokens[:k]) {
		// 数字位序：取全部合法位（最多 5），勿截成恰好 k
		n := len(tokens)
		if n > sscPositionCount {
			n = sscPositionCount
		}
		if !renxuanPartsLookLikePositions(tokens[:n]) {
			n = k
		}
		out := make([]int, 0, n)
		for i := 0; i < n; i++ {
			v, err := strconv.Atoi(strings.TrimSpace(tokens[i]))
			if err != nil || v < 0 || v >= sscPositionCount {
				break
			}
			out = append(out, v)
		}
		if len(out) >= k {
			return dedupeSortedPositions(out)
		}
	}
	var out []int
	for _, tok := range tokens {
		tok = strings.TrimSpace(tok)
		if idx := sscPositionIndexFromName(tok); idx >= 0 {
			out = append(out, idx)
			continue
		}
		if len(tok) == 1 && tok[0] >= '0' && tok[0] <= '4' {
			out = append(out, int(tok[0]-'0'))
		}
	}
	if len(out) >= k {
		return dedupeSortedPositions(out)
	}
	if len(tokens) == 1 {
		if indices := parseSSCPositionNames(tokens[0]); len(indices) >= k {
			return dedupeSortedPositions(indices)
		}
	}
	return nil
}

func dedupeSortedPositions(indices []int) []int {
	seen := make(map[int]bool, len(indices))
	out := make([]int, 0, len(indices))
	for _, idx := range indices {
		if idx < 0 || idx >= sscPositionCount || seen[idx] {
			continue
		}
		seen[idx] = true
		out = append(out, idx)
	}
	sort.Ints(out)
	if len(out) > sscPositionCount {
		out = out[:sscPositionCount]
	}
	return out
}

func renxuanPositionLabel(indices []int, k int) string {
	if k <= 0 {
		k = 2
	}
	if len(indices) < k {
		indices = renxuanDefaultPositionIndices(k)
	} else {
		// 多选位：保留全部（最多 5），供任二/任三单式 C(n,k) 计注
		indices = dedupeSortedPositions(indices)
	}
	var b strings.Builder
	for _, idx := range indices {
		if idx >= 0 && idx < len(sscPositionRunes) {
			b.WriteRune(sscPositionRunes[idx])
		}
	}
	return b.String()
}

func splitRenxuanPosPipeWire(wireContent string) (posLabel, picks string, ok bool) {
	wireContent = strings.TrimSpace(wireContent)
	pipe := strings.Index(wireContent, "|")
	if pipe <= 0 || pipe >= len(wireContent)-1 {
		return "", "", false
	}
	return strings.TrimSpace(wireContent[:pipe]), strings.TrimSpace(wireContent[pipe+1:]), true
}

func formatRenxuanPosPipeWire(groupContent string, k int, picks string) string {
	if k <= 0 {
		k = 2
	}
	positions, parsed := parseRenxuanPositionPick(groupContent, k)
	if strings.TrimSpace(picks) == "" {
		picks = strings.TrimSpace(parsed)
	}
	if strings.TrimSpace(picks) == "" {
		picks = "1"
	}
	return renxuanPositionLabel(positions, k) + "|" + picks
}

func renxuanPartsLookLikePositions(parts []string) bool {
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if len(p) != 1 || p[0] < '0' || p[0] > '4' {
			return false
		}
	}
	return len(parts) > 0
}

func parseRenxuanPositionPick(groupContent string, k int) (positions []int, picks string) {
	if k <= 0 {
		k = 2
	}
	groupContent = strings.TrimSpace(groupContent)
	if groupContent == "" {
		return renxuanDefaultPositionIndices(k), ""
	}
	lines := splitPositionLines(groupContent)
	if len(lines) >= 2 {
		if indices := parseRenxuanPositionIndicesFromTokens(splitPickTokens(lines[0]), k); len(indices) >= k {
			return indices, strings.Join(lines[1:], "\n")
		}
		if indices := parseSSCPositionNames(lines[0]); len(indices) >= k {
			return dedupeSortedPositions(indices), strings.Join(lines[1:], "\n")
		}
	}
	parts := splitCommaParts(groupContent)
	// 前若干段为数字位序（0–4），其余为号码；允许多于 k 个位
	if len(parts) >= k+1 {
		posEnd := 0
		for posEnd < len(parts) && posEnd < sscPositionCount && renxuanPartsLookLikePositions(parts[:posEnd+1]) {
			posEnd++
		}
		if posEnd >= k && posEnd < len(parts) {
			for i := 0; i < posEnd; i++ {
				n, _ := strconv.Atoi(parts[i])
				positions = append(positions, n)
			}
			return dedupeSortedPositions(positions), strings.Join(parts[posEnd:], ",")
		}
	}
	if pipe := strings.Index(groupContent, "|"); pipe > 0 {
		if indices := parseSSCPositionNames(groupContent[:pipe]); len(indices) >= k {
			return dedupeSortedPositions(indices), groupContent[pipe+1:]
		}
	}
	return renxuanDefaultPositionIndices(k), groupContent
}

func formatRenxuanBetContent(meta RuleMeta, mode, groupContent string) string {
	label := meta.Label
	k := renxuanSegmentLen(meta)
	switch mode {
	case "hezhi", "weishu":
		return formatRenxuanPosPipeWire(groupContent, k, formatRenxuanHezhiPicks(groupContent, k))
	case "danshi":
		// ForcedBetMode=danshi 时仍可能是组三/组六单式（按 label/ruleId 识别）
		if isZu3DanshiMeta(meta) {
			return formatRenxuanPosPipeWire(groupContent, k, formatRenxuanZu3DanshiPicksOnly(groupContent, k))
		}
		if isZu6DanshiMeta(meta) {
			return formatRenxuanPosPipeWire(groupContent, k, formatRenxuanZu6DanshiPicksOnly(groupContent, k))
		}
		return formatRenxuanPosPipeWire(groupContent, k, formatRenxuanDanshiPicksOnly(groupContent, k))
	case "zuxuan_ds":
		// 组三单式：仅两同+一异；组六单式：三位互异；普通组选单式：整注或单码号池 C(n,k)
		if isZu3DanshiMeta(meta) {
			return formatRenxuanPosPipeWire(groupContent, k, formatRenxuanZu3DanshiPicksOnly(groupContent, k))
		}
		if isZu6DanshiMeta(meta) {
			return formatRenxuanPosPipeWire(groupContent, k, formatRenxuanZu6DanshiPicksOnly(groupContent, k))
		}
		return formatRenxuanPosPipeWire(groupContent, k, formatRenxuanZuxuanDanshiPicksOnly(groupContent, k))
	case "fushi":
		if strings.Contains(label, "直选") {
			// 任二/任三/任四直选复式均须五位逗号定位 wire（任二 flat「0,1」会报投注内容格式错误）
			return formatRenxuanZhixuanPositionWire(groupContent, k)
		}
		if strings.Contains(label, "组选") || strings.Contains(label, "组三") || strings.Contains(label, "组六") {
			return formatRenxuanPosPipeWire(groupContent, k, formatRenxuanZuxuanPicksOnly(groupContent, k, mode))
		}
		return formatRenxuanFushiFlat(groupContent)
	case "zuxuan_fs", "zu3", "zu6":
		return formatRenxuanPosPipeWire(groupContent, k, formatRenxuanZuxuanPicksOnly(groupContent, k, mode))
	case "hunhe":
		// 混合组选：三位整注、排除豹子、形态去重（顺序不限）；勿走号池补码
		return formatRenxuanPosPipeWire(groupContent, k, formatRenxuanHunhePicksOnly(groupContent, k))
	case "zu24":
		return formatRenxuanPosPipeWire(groupContent, k, formatRenxuanZuxuanPicksOnly(groupContent, k, mode))
	case "zu12", "zu4":
		// 组选12/4 的内容是双区号池（12,34 = 二重号池,单号池；1,2 = 三重号,单号），
		// 不是选号列表。走号池补码会把「12,34」补成「12,34,3,4」，
		// 第三方回「投注数字不合规」（2026-07-28 真实下单 tron_ffc_1m 144 实测）。
		picks := groupContent
		if _, p := parseRenxuanPositionPick(groupContent, k); strings.TrimSpace(p) != "" {
			picks = p
		}
		if mode == "zu12" {
			return formatRenxuanPosPipeWire(groupContent, k, formatZu12Wire(picks))
		}
		return formatRenxuanPosPipeWire(groupContent, k, formatZu4Wire(picks))
	default:
		return formatRenxuanPosPipeWire(groupContent, k, formatCommaPickDigits(groupContent))
	}
}

func formatRenxuanZhixuanPositionWire(groupContent string, k int) string {
	if k <= 0 {
		k = 2
	}
	// 已是五位逗号定位（如 1,2,3,4,5 / ,0,,,1）直接规范化
	if parts := splitCommaParts(groupContent); len(parts) == sscPositionCount && renxuanPartsLookLikeDigitPools(parts) {
		segs := make([]string, sscPositionCount)
		for i, p := range parts {
			segs[i] = strings.TrimSpace(p)
		}
		return strings.Join(segs, ",")
	}
	// 五行位号（含空行）→ 五位逗号
	lines := splitPositionLines(groupContent)
	if len(lines) == sscPositionCount {
		segs := make([]string, sscPositionCount)
		for i, line := range lines {
			segs[i] = strings.TrimSpace(line)
		}
		return strings.Join(segs, ",")
	}
	tokens := splitPickTokens(groupContent)
	// 去掉误入的位名（万千百十个）
	filtered := make([]string, 0, len(tokens))
	for _, t := range tokens {
		if sscPositionIndexFromName(t) >= 0 || t == "位" {
			continue
		}
		filtered = append(filtered, t)
	}
	tokens = filtered
	if len(tokens) < k {
		for len(tokens) < k {
			tokens = append(tokens, string(rune('1'+len(tokens))))
		}
	}
	// 恰好 5 个号码段：视为五位各一池
	if len(tokens) == sscPositionCount {
		return strings.Join(tokens, ",")
	}
	indices := renxuanDefaultPositionIndices(k)
	segments := make([]string, sscPositionCount)
	for i, idx := range indices {
		if i < len(tokens) {
			segments[idx] = tokens[i]
		}
	}
	return strings.Join(segments, ",")
}

func renxuanPartsLookLikeDigitPools(parts []string) bool {
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		for _, r := range p {
			if r < '0' || r > '9' {
				return false
			}
		}
	}
	return true
}

func formatRenxuanFushiFlat(groupContent string) string {
	tokens := splitPickTokens(groupContent)
	if len(tokens) == 0 {
		return strings.TrimSpace(groupContent)
	}
	return strings.Join(tokens, ",")
}

func formatRenxuanDanshiPairs(groupContent string, k int) string {
	if k <= 0 {
		k = 2
	}
	lines := splitPositionLines(groupContent)
	if len(lines) > 1 {
		parts := make([]string, 0, len(lines))
		for _, line := range lines {
			line = normalizePickDigits(line)
			if len(line) >= k {
				parts = append(parts, line[:k])
			}
		}
		if len(parts) > 0 {
			return strings.Join(parts, ",")
		}
	}
	if parts := splitCommaParts(groupContent); len(parts) > 1 {
		out := make([]string, 0, len(parts))
		for _, p := range parts {
			p = normalizePickDigits(p)
			if len(p) >= k {
				out = append(out, p[:k])
			}
		}
		if len(out) > 0 {
			return strings.Join(out, ",")
		}
	}
	digits := normalizePickDigits(groupContent)
	if len(digits) >= k {
		parts := make([]string, 0)
		for i := 0; i+k <= len(digits); i += k {
			parts = append(parts, digits[i:i+k])
		}
		if len(parts) > 0 {
			return strings.Join(parts, ",")
		}
		return digits[:k]
	}
	return digits
}

func formatRenxuanHezhiPicks(groupContent string, k int) string {
	_, picks := parseRenxuanPositionPick(groupContent, k)
	sumTokens := splitPickTokens(picks)
	if len(sumTokens) == 0 {
		sumTokens = parseIntTokenListAsStrings(picks)
	}
	if len(sumTokens) == 0 {
		sumTokens = parseIntTokenListAsStrings(groupContent)
	}
	if len(sumTokens) == 0 {
		return "6"
	}
	return strings.Join(sumTokens, ",")
}

func formatRenxuanDanshiPicksOnly(groupContent string, k int) string {
	_, picks := parseRenxuanPositionPick(groupContent, k)
	if wire := formatRenxuanDanshiPairs(picks, k); wire != "" {
		return wire
	}
	return formatRenxuanDanshiPairs(groupContent, k)
}

// formatRenxuanZuxuanDanshiPicksOnly 任选组选单式：对齐 formatSSCZuxuanDanshiDigits（号池→组合整注）。
func formatRenxuanZuxuanDanshiPicksOnly(groupContent string, k int) string {
	if k <= 0 {
		k = 2
	}
	_, picks := parseRenxuanPositionPick(groupContent, k)
	src := strings.TrimSpace(picks)
	if src == "" {
		src = strings.TrimSpace(groupContent)
	}
	if wire := formatSSCZuxuanDanshiDigits(k, src); wire != "" {
		return wire
	}
	// 无合法整注/号池时保持空串，由上层 Skip / 校验拦截（勿回落直选切段误吞号池）
	return ""
}

// isZu3DanshiMeta 任三组三单式（rule 84 / 组三单式）。
func isZu3DanshiMeta(meta RuleMeta) bool {
	text := strings.ToLower(meta.Label + " " + meta.FullName + " " + meta.SubID + " " + meta.Group)
	if strings.Contains(text, "zu30") || strings.Contains(text, "组选30") {
		return false
	}
	if strings.TrimSpace(meta.SubID) == "84" || strings.Contains(text, "zu3_ds") || strings.Contains(text, "ren3_zu3_ds") {
		return true
	}
	return strings.Contains(meta.Label+meta.FullName, "组三") && strings.Contains(meta.Label+meta.FullName, "单式")
}

// isZu6DanshiMeta 任三组六单式（rule 86 / 组六单式）。
func isZu6DanshiMeta(meta RuleMeta) bool {
	text := strings.ToLower(meta.Label + " " + meta.FullName + " " + meta.SubID + " " + meta.Group)
	if strings.Contains(text, "组选6") || strings.Contains(text, "组选60") || strings.Contains(text, "组选120") ||
		strings.Contains(text, "zu60") || strings.Contains(text, "zu120") {
		return false
	}
	if strings.TrimSpace(meta.SubID) == "86" || strings.Contains(text, "zu6_ds") || strings.Contains(text, "ren3_zu6_ds") {
		return true
	}
	return strings.Contains(meta.Label+meta.FullName, "组六") && strings.Contains(meta.Label+meta.FullName, "单式")
}

// formatRenxuanZu3DanshiPicksOnly 任选组三单式：仅组三形态整注。
func formatRenxuanZu3DanshiPicksOnly(groupContent string, k int) string {
	if k <= 0 {
		k = 3
	}
	_, picks := parseRenxuanPositionPick(groupContent, k)
	src := strings.TrimSpace(picks)
	if src == "" {
		src = strings.TrimSpace(groupContent)
	}
	return formatSSCZu3DanshiDigits(src)
}

// formatRenxuanZu6DanshiPicksOnly 任选组六单式：仅组六形态整注（三位互异）。
func formatRenxuanZu6DanshiPicksOnly(groupContent string, k int) string {
	if k <= 0 {
		k = 3
	}
	_, picks := parseRenxuanPositionPick(groupContent, k)
	src := strings.TrimSpace(picks)
	if src == "" {
		src = strings.TrimSpace(groupContent)
	}
	return formatSSCZu6DanshiDigits(src)
}

// formatRenxuanHunhePicksOnly 任选混合组选：排除豹子，按组选形态去重。
func formatRenxuanHunhePicksOnly(groupContent string, k int) string {
	if k <= 0 {
		k = 3
	}
	_, picks := parseRenxuanPositionPick(groupContent, k)
	src := strings.TrimSpace(picks)
	if src == "" {
		src = strings.TrimSpace(groupContent)
	}
	return formatSSCZuxuanDanshiDigits(k, src)
}

func formatRenxuanZuxuanPicksOnly(groupContent string, k int, mode string) string {
	if k <= 0 {
		k = 2
	}
	_, picks := parseRenxuanPositionPick(groupContent, k)
	tokens := splitPickTokens(picks)
	if len(tokens) == 0 {
		tokens = splitPickTokens(groupContent)
	}
	// 去掉位名，避免「万百|4」只剩位名残留
	filtered := make([]string, 0, len(tokens))
	for _, t := range tokens {
		if sscPositionIndexFromName(t) >= 0 {
			continue
		}
		filtered = append(filtered, t)
	}
	tokens = filtered
	// 已是 n 位单式串（112 / 012）勿按号池补码
	for _, tok := range tokens {
		if len(normalizePickDigits(tok)) >= k && k >= 2 {
			return strings.Join(tokens, ",")
		}
	}
	// 组选复式至少 2 码；组三/组六号池至少 2/3；任四组选6 用 C(n,2) 口径至少 2，勿强补到 4
	minNeed := 2
	switch mode {
	case "zu6":
		minNeed = 3
		if k >= 4 {
			minNeed = 2 // 四星/任四组选6：n=2→1 注
		}
	case "zu24", "zu12", "zu4":
		minNeed = 4
	case "zu3":
		minNeed = 2
	default:
		if k >= 4 {
			minNeed = 4
		}
	}
	if len(tokens) < minNeed {
		for len(tokens) < minNeed {
			tokens = append(tokens, string(rune('1'+len(tokens)%9)))
		}
	}
	if len(tokens) == 0 {
		return sampleZuGroupDigits(minNeed)
	}
	return strings.Join(tokens, ",")
}

func parseIntTokenListAsStrings(raw string) []string {
	vals := parseIntTokenList(raw)
	out := make([]string, 0, len(vals))
	for _, v := range vals {
		out = append(out, strconv.Itoa(v))
	}
	return out
}

func countRenxuanDanshiPairs(wireContent string, k int) int {
	parts := splitCommaParts(wireContent)
	if len(parts) > 1 {
		n := 0
		for _, p := range parts {
			if len(normalizePickDigits(p)) >= k {
				n++
			}
		}
		return n
	}
	if len(normalizePickDigits(wireContent)) >= k {
		return 1
	}
	return 0
}

func renxuanPosComboMultiplier(posLabel string, k int) int {
	if k <= 0 {
		k = 2
	}
	indices := parseSSCPositionNames(posLabel)
	nPos := len(indices)
	if nPos < k {
		return 1
	}
	mul := combin(nPos, k)
	if mul <= 0 {
		return 1
	}
	return mul
}

func countRenxuanHezhiWire(meta RuleMeta, wireContent string, k int) int {
	if posLabel, picks, ok := splitRenxuanPosPipeWire(wireContent); ok {
		inner := countHezhiBetNums(meta, picks, k)
		if inner <= 0 {
			return 0
		}
		return renxuanPosComboMultiplier(posLabel, k) * inner
	}
	parts := splitCommaParts(wireContent)
	if len(parts) == k+1 && renxuanPartsLookLikePositions(parts[:k]) {
		sum, err := strconv.Atoi(strings.TrimSpace(parts[k]))
		if err != nil {
			return 0
		}
		if strings.Contains(meta.Label, "组选") {
			return countZuxuanSumCombinations(sum, k)
		}
		return countOrderedSumCombinations(sum, k)
	}
	return countHezhiBetNums(meta, wireContent, k)
}

func countRenxuanZuxuanPickWire(meta RuleMeta, wireContent string, k int) int {
	if k <= 0 {
		k = 2
	}
	posMul := 1
	fromPipe := false
	if posLabel, picks, ok := splitRenxuanPosPipeWire(wireContent); ok {
		fromPipe = true
		wireContent = picks
		posMul = renxuanPosComboMultiplier(posLabel, k)
	}
	mode := InferBetMode(meta)
	countPool := func(poolContent string, n int) int {
		if n <= 0 {
			return 0
		}
		switch mode {
		case "zu12":
			return countZu12BetNums(poolContent)
		case "zu24":
			return countZuGroupBetNums("zu24", n)
		case "zu4":
			return countZuGroupBetNums("zu4", n)
		case "zu6":
			if k >= 4 {
				return countSixingZu6BetNums(n)
			}
			return zu6PoolUnits(n)
		case "zu3":
			if !strings.Contains(poolContent, ",") && len(normalizePickDigits(poolContent)) == k {
				return 1
			}
			return zu3PoolUnits(n)
		case "zuxuan_fs":
			if k == 2 {
				return countZuxuanFushiBetNums(n, k)
			}
			return combin(n, k)
		default:
			if n < k {
				return 0
			}
			return combin(n, k)
		}
	}
	// pipe 线已剥位：号池里的 0,1 是号码不是位序，勿再当选位截断
	if fromPipe {
		return countPool(wireContent, len(splitPickDigits(wireContent))) * posMul
	}
	parts := splitCommaParts(wireContent)
	if len(parts) <= k || !renxuanPartsLookLikePositions(parts[:k]) {
		return countPool(wireContent, len(splitPickDigits(wireContent))) * posMul
	}
	picks := strings.Join(parts[k:], ",")
	return countPool(picks, len(splitPickDigits(picks))) * posMul
}

func countRenxuanDanshiWire(wireContent string, k int) int {
	if k <= 0 {
		k = 2
	}
	if posLabel, picks, ok := splitRenxuanPosPipeWire(wireContent); ok {
		tickets := countRenxuanDanshiPairs(picks, k)
		if tickets <= 0 {
			return 0
		}
		indices := parseSSCPositionNames(posLabel)
		nPos := len(indices)
		if nPos < k {
			// 位名不足时按单组位计（兼容缺省）
			return tickets
		}
		if nPos == k {
			return tickets
		}
		// 多选位：C(n,k)×单式注数
		return combin(nPos, k) * tickets
	}
	return countRenxuanDanshiPairs(wireContent, k)
}

// countRenxuanZuxuanDanshiWire 组选单式：整注或单码号池展成组合后再按位乘 C(n,k)。
func countRenxuanZuxuanDanshiWire(wireContent string, k int) int {
	return countRenxuanZuxuanDanshiWireFiltered(wireContent, k, "")
}

// countRenxuanZu3DanshiWire 组三单式：仅组三形态整注 × C(选位数,k)。
func countRenxuanZu3DanshiWire(wireContent string, k int) int {
	return countRenxuanZuxuanDanshiWireFiltered(wireContent, k, "zu3")
}

// countRenxuanZu6DanshiWire 组六单式：仅组六形态整注 × C(选位数,k)。
func countRenxuanZu6DanshiWire(wireContent string, k int) int {
	return countRenxuanZuxuanDanshiWireFiltered(wireContent, k, "zu6")
}

func countRenxuanZuxuanDanshiWireFiltered(wireContent string, k int, filter string) int {
	if k <= 0 {
		k = 2
	}
	expand := func(src string) string {
		switch filter {
		case "zu3":
			return formatSSCZu3DanshiDigits(src)
		case "zu6":
			return formatSSCZu6DanshiDigits(src)
		default:
			return formatSSCZuxuanDanshiDigits(k, src)
		}
	}
	if posLabel, picks, ok := splitRenxuanPosPipeWire(wireContent); ok {
		exp := expand(picks)
		if exp == "" {
			return 0
		}
		return countRenxuanDanshiWire(posLabel+"|"+exp, k)
	}
	idxs, picks := parseRenxuanPositionPick(wireContent, k)
	src := strings.TrimSpace(picks)
	if src == "" {
		src = strings.TrimSpace(wireContent)
	}
	exp := expand(src)
	if exp == "" {
		return 0
	}
	tickets := countRenxuanDanshiPairs(exp, k)
	if tickets <= 0 {
		return 0
	}
	nPos := len(idxs)
	if nPos < k {
		return tickets
	}
	if nPos == k {
		return tickets
	}
	return combin(nPos, k) * tickets
}

func countRenxuanPoolBetNumsFlat(meta RuleMeta, wireContent string, k int) int {
	n := len(splitPickDigits(wireContent))
	if n < k {
		return 0
	}
	return combin(n, k)
}

func renxuanNeedsSoloTrue(meta RuleMeta, mode string, betsNums int) bool {
	if !isRenxuanMeta(meta) || meta.PlayTemplate == "syxw_std" {
		return false
	}
	switch mode {
	case "zu3":
		return betsNums == 1 || betsNums == 2
	case "zu6":
		// 任四组选6：与四星/前后四组选6 一致，任意注数 solo=false。
		// 单注 solo=true →「单挑参数错误」（g013/g014 probe；def-1-1785898019073）。
		// 任三组六仍：仅 1 注 solo=true。
		if renxuanSegmentLen(meta) >= 4 {
			return false
		}
		return betsNums == 1
	case "hunhe":
		// 任三混合组选：与单区混合组选同阈 ≤hunheSoloMaxBets(3)。
		// def-1-1785836661744：万千个|183,838 → 2 注 solo=false → 单挑参数错误；
		// 同内容 solo=true 可过（≤3）；≥4 注须 solo=false。
		return betsNums > 0 && betsNums <= hunheSoloMaxBets
	case "zuxuan_ds":
		// 任三组三单式：与三星组三同阈 zu3SoloMaxBets(6)。
		// def-1-1785829274113：万千百个|223 → 4 注 solo=false 拒；
		//   万千百十个|445 → 10 注 solo=true 拒（五位全选 C(5,3)）；
		//   90 注 solo=false 可过。
		if isZu3DanshiMeta(meta) {
			return betsNums > 0 && betsNums <= zu3SoloMaxBets
		}
		// 任三组六单式：与三星组六同阈 ≤zu6SoloMaxBets(3)
		if isZu6DanshiMeta(meta) {
			return betsNums > 0 && betsNums <= zu6SoloMaxBets
		}
		// 其它组选单式：有注即须 solo=true，再由 ResolveSolo 按 guajiSoloMaxBets 钳制。
		return betsNums > 0
	case "danshi":
		if isZu3DanshiMeta(meta) {
			return betsNums > 0 && betsNums <= zu3SoloMaxBets
		}
		if isZu6DanshiMeta(meta) {
			return betsNums > 0 && betsNums <= zu6SoloMaxBets
		}
		// 与同组直选复式一致：有注即须 solo=true，再由 ResolveSolo 按 guajiSoloMaxBets 钳制。
		// 单注实测 rule75/81（2026-07-28）；多选位扩注同理：
		// def-1-1785825380160 万千百个|345 → C(4,3)=4 注 solo=false → 单挑参数错误，
		// 同方案 35 注 solo=false 可过（>guajiSoloMaxBets）。
		return betsNums > 0
	case "fushi":
		// 任二/任三/任四直选复式：有注即须 solo=true，再由 ResolveSolo 按 guajiSoloMaxBets 钳制。
		// 实测 rule80 content=1,1,1,1,2 bets=10 solo=false → 单挑参数错误（def-1-1785814730739）；
		// 同实例 56 注 solo=false 可过（>guajiSoloMaxBets）。
		return betsNums > 0
	case "hezhi", "weishu":
		// 组选和值（方案常只存数字 subId，须 isZuxuanHezhiMeta）：
		// - 任二 rule79：任意注数 solo=false（同前二组选和值）
		// - 任三 rule88 / 任四 rule96：与中三组选和值同阈 ≤hunheSoloMaxBets(3)
		//   实测 def-1-1785839664877：万千个|1（1注）/ 万千个|3（2注）solo=false → 单挑参数错误，须 solo=true
		if isZuxuanHezhiMeta(meta) {
			if renxuanErxingZuxuanHezhi(meta) {
				return false
			}
			return betsNums > 0 && betsNums <= hunheSoloMaxBets
		}
		// 直选和值：有注即须 solo=true，再由 ResolveSolo 按 guajiSoloMaxBets 钳制。
		// def-1-1785827226818：万千百个|1 → 12 注 solo=false → 单挑参数错误；
		// 同方案满选和值 1000 注 solo=false 可过（>guajiSoloMaxBets）。
		return betsNums > 0
	default:
		return false
	}
}

// renxuanErxingZuxuanHezhi 任二组选和值（rule79）：任意注数须 solo=false。
func renxuanErxingZuxuanHezhi(meta RuleMeta) bool {
	for _, id := range []string{meta.RuleID, meta.SubID} {
		if strings.TrimSpace(id) == "79" {
			return true
		}
	}
	text := meta.Label + meta.FullName + meta.TeamLabel
	if strings.Contains(text, "任二") && strings.Contains(text, "组选和值") {
		return true
	}
	if strings.Contains(text, "任选二") && strings.Contains(text, "组选和值") {
		return true
	}
	return false
}

func isSingleTokenTextDxds(meta RuleMeta) bool {
	if meta.PlayTemplate == "pk10_std" {
		switch strings.TrimSpace(meta.TypeID) {
		case "g008", "g009":
			return true
		}
	}
	if isWuxingSumDxds(meta) {
		return true
	}
	label := meta.Label
	return strings.Contains(label, "尾数单双") || strings.Contains(label, "尾数大小")
}

func isWuxingSumDxds(meta RuleMeta) bool {
	sub := strings.ToLower(strings.TrimSpace(meta.SubID))
	label := meta.Label
	// rules/v2：268=五星和值单双、269=五星和值大小
	switch strings.TrimSpace(meta.SubID) {
	case "268", "269":
		return true
	}
	if strings.Contains(sub, "wuxing_hz") || strings.Contains(sub, "hz_ds") || strings.Contains(sub, "hz_dx") {
		return true
	}
	return strings.Contains(label, "和值单双") || strings.Contains(label, "和值大小")
}

func isPositionDxds(meta RuleMeta) bool {
	mode := InferBetMode(meta)
	if mode != "dxds" {
		return false
	}
	return !isWuxingSumDxds(meta)
}

func formatDxdsBetContent(meta RuleMeta, groupContent string) string {
	if isSingleTokenTextDxds(meta) {
		return formatTextTokens(groupContent)
	}
	_, length := segmentRange(meta)
	if length <= 0 {
		length = 1
	}
	lines := splitPositionLines(groupContent)
	tokens := splitPickTokens(groupContent)
	parts := make([]string, 0, length)
	for i := 0; i < length; i++ {
		tok := ""
		switch {
		case i < len(lines) && strings.TrimSpace(lines[i]) != "":
			// 第三方按位 wire 每位仅 1 个选项（十,个 → 大,小）；行内多选时取首个
			if lineToks := splitPickTokens(lines[i]); len(lineToks) > 0 {
				tok = lineToks[0]
			} else {
				tok = strings.TrimSpace(lines[i])
			}
		case len(lines) == 1 && strings.TrimSpace(lines[0]) != "":
			if lineToks := splitPickTokens(lines[0]); len(lineToks) > 0 {
				tok = lineToks[0]
			} else {
				tok = strings.TrimSpace(lines[0])
			}
		case len(tokens) == 1 && length == 1:
			tok = tokens[0]
		case i < len(tokens):
			tok = tokens[i]
		default:
			tok = "大"
		}
		parts = append(parts, tok)
	}
	return strings.Join(parts, ",")
}

func sampleZuGroupDigits(count int) string {
	if count <= 0 {
		count = 5
	}
	parts := make([]string, count)
	for i := 0; i < count; i++ {
		parts[i] = string(rune('1' + i))
	}
	return strings.Join(parts, ",")
}

func sampleZu120Digits() string {
	return "0,1,2,3,4"
}

func sampleRenxuanFushiContent(meta RuleMeta) string {
	k := renxuanSegmentLen(meta)
	if k < 2 {
		k = 2
	}
	switch k {
	case 2:
		return "1,2,3,4,5"
	case 4:
		return "1,2,3,4"
	default:
		parts := make([]string, k)
		for i := range parts {
			parts[i] = string(rune('1' + i))
		}
		return strings.Join(parts, ",")
	}
}

func sampleRenxuanPositionLine(k int) string {
	indices := renxuanDefaultPositionIndices(k)
	var b strings.Builder
	for i, idx := range indices {
		if i > 0 {
			b.WriteByte(',')
		}
		if idx >= 0 && idx < len(sscPositionRunes) {
			b.WriteRune(sscPositionRunes[idx])
		}
	}
	return b.String()
}

func renxuanSequentialDigits(k int) string {
	if k <= 0 {
		k = 2
	}
	digits := make([]byte, k)
	for i := range digits {
		digits[i] = byte('1' + i)
	}
	return string(digits)
}

func sampleRenxuanPosPipeContent(meta RuleMeta, picks string) string {
	k := renxuanSegmentLen(meta)
	if k <= 0 {
		k = 2
	}
	return sampleRenxuanPositionLine(k) + "\n" + picks
}

func sampleRenxuanDanshiContent(meta RuleMeta) string {
	k := renxuanSegmentLen(meta)
	if k <= 0 {
		k = 2
	}
	picks := renxuanSequentialDigits(k)
	label := meta.Label
	switch {
	case strings.Contains(label, "组三"):
		picks = "112"
	case strings.Contains(label, "组六") && k == 3:
		picks = "123"
	case strings.Contains(label, "混合"):
		picks = "112"
	}
	return sampleRenxuanPosPipeContent(meta, picks)
}

func sampleRenxuanHezhiContent(meta RuleMeta) string {
	if strings.Contains(meta.Label, "组选") {
		return sampleRenxuanPosPipeContent(meta, "1")
	}
	// 直选和值 0 在任 k 中均为 1 注（全 0 组合）。
	return sampleRenxuanPosPipeContent(meta, "0")
}

func sampleRenxuanZuxuanFsContent(meta RuleMeta) string {
	k := renxuanSegmentLen(meta)
	switch InferBetMode(meta) {
	case "zu3":
		return sampleRenxuanPosPipeContent(meta, "1,2")
	case "zu6":
		parts := make([]string, k)
		for i := range parts {
			parts[i] = string(rune('1' + i))
		}
		return sampleRenxuanPosPipeContent(meta, strings.Join(parts, ","))
	case "hunhe":
		return sampleRenxuanPosPipeContent(meta, "112")
	default:
		return sampleRenxuanPosPipeContent(meta, "1,2")
	}
}

func sampleRenxuanContent(meta RuleMeta) string {
	mode := InferBetMode(meta)
	switch mode {
	case "danshi", "zuxuan_ds":
		return sampleRenxuanDanshiContent(meta)
	case "hezhi", "weishu":
		return sampleRenxuanHezhiContent(meta)
	case "zuxuan_fs", "zu3", "zu6", "zu24", "zu12", "zu4", "hunhe":
		return sampleRenxuanZuxuanFsContent(meta)
	default:
		return sampleRenxuanFushiContent(meta)
	}
}

func sampleDxdsContent(meta RuleMeta) string {
	if isSingleTokenTextDxds(meta) {
		if meta.PlayTemplate == "pk10_std" && meta.TypeID == "g009" {
			return "单"
		}
		if strings.Contains(meta.Label, "单") {
			return "单"
		}
		if strings.Contains(meta.Label, "小") {
			return "小"
		}
		return "大"
	}
	_, segLen := segmentRange(meta)
	if segLen <= 0 {
		segLen = 1
	}
	lines := make([]string, segLen)
	for i := range lines {
		lines[i] = "大"
	}
	return strings.Join(lines, "\n")
}

func syxwRenxuanPickN(meta RuleMeta) int {
	text := strings.ToLower(meta.Label + meta.SubID + meta.TeamLabel)
	switch {
	case strings.Contains(text, "八中五"), strings.Contains(text, "rx_8z5"):
		return 8
	case strings.Contains(text, "七中五"), strings.Contains(text, "rx_7z5"):
		return 7
	case strings.Contains(text, "六中五"), strings.Contains(text, "rx_6z5"):
		return 6
	case strings.Contains(text, "五中五"), strings.Contains(text, "rx_5z5"):
		return 5
	case strings.Contains(text, "四中四"), strings.Contains(text, "rx_4z4"):
		return 4
	case strings.Contains(text, "三中三"), strings.Contains(text, "rx_3z3"):
		return 3
	case strings.Contains(text, "二中二"), strings.Contains(text, "rx_2z2"):
		return 2
	default:
		return 1
	}
}

func countSyxwRenxuanBetNums(meta RuleMeta, wireContent string) int {
	k := syxwRenxuanPickN(meta)
	if k <= 0 {
		return 0
	}
	if syxwRenxuanUsesDanshiWire(meta) {
		if tokens := splitPaddedWireTokens(wireContent, digitPadWidth(meta.PlayTemplate)); len(tokens) > 0 {
			if len(tokens) < k {
				return 0
			}
			if len(tokens) == k {
				return 1
			}
			return comboCount(len(tokens), k)
		}
	}
	tokens := splitPickTokens(wireContent)
	if len(tokens) < k {
		return 0
	}
	if len(tokens) == k {
		return 1
	}
	return comboCount(len(tokens), k)
}

func syxwRenxuanUsesDanshiWire(meta RuleMeta) bool {
	if strings.TrimSpace(meta.TypeID) == "g006" {
		return true
	}
	return InferBetMode(meta) == "danshi"
}

// syxwRenxuanNeedsSoloTrue 11选5 任选单注 solo：k∈[4,6] 须 solo=true，其余 solo=false（bet-probe g005/g006）。
func syxwRenxuanNeedsSoloTrue(meta RuleMeta, betsNums int) bool {
	if betsNums != 1 {
		return false
	}
	k := syxwRenxuanPickN(meta)
	return k >= 4 && k <= 6
}

func splitPaddedWireTokens(wireContent string, width int) []string {
	wireContent = strings.TrimSpace(wireContent)
	if wireContent == "" || width <= 1 {
		return splitPickTokens(wireContent)
	}
	if strings.ContainsAny(wireContent, ",，") {
		tokens := splitPickTokens(wireContent)
		out := make([]string, 0, len(tokens))
		for _, t := range tokens {
			out = append(out, padNumericToken(t, width))
		}
		return out
	}
	if len(wireContent)%width != 0 {
		return nil
	}
	n := len(wireContent) / width
	out := make([]string, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, wireContent[i*width:(i+1)*width])
	}
	return out
}

func comboCount(n, k int) int {
	if n < k || k <= 0 {
		return 0
	}
	out := 1
	for i := 0; i < k; i++ {
		out = out * (n - i) / (i + 1)
	}
	return out
}

func sampleSyxwRenxuanContent(meta RuleMeta) string {
	mode := InferBetMode(meta)
	if syxwRenxuanUsesDanshiWire(meta) || mode == "danshi" {
		k := syxwRenxuanPickN(meta)
		if k <= 0 {
			k = 1
		}
		digits := make([]byte, k)
		for i := range digits {
			digits[i] = byte('1' + i)
		}
		return string(digits)
	}
	k := syxwRenxuanPickN(meta)
	if k <= 0 {
		k = 1
	}
	parts := make([]string, k)
	for i := 0; i < k; i++ {
		parts[i] = padNumericToken(string(rune('1'+i)), 2)
	}
	return strings.Join(parts, ",")
}

func wuxingFushiNeedsSolo(meta RuleMeta, mode string, betsNums int) bool {
	if betsNums != 1 {
		return false
	}
	if meta.TypeID != "g015" && meta.Group != "五星" {
		return false
	}
	return mode == "fushi" || strings.Contains(meta.Label, "直选复式")
}

func paddedFushiUsesFlatPick(meta RuleMeta) bool {
	if !usesPaddedDigits(meta.PlayTemplate) {
		return false
	}
	switch meta.PlayTemplate {
	case "syxw_std", "pk10_std":
		return !isSyxwRenxuanMeta(meta)
	default:
		return false
	}
}

func paddedFushiNeedsSolo(meta RuleMeta, mode string, betsNums int) bool {
	if betsNums != 1 {
		return false
	}
	if mode != "fushi" && mode != "zuxuan_fs" {
		return false
	}
	if !usesPaddedDigits(meta.PlayTemplate) {
		return false
	}
	if isSyxwRenxuanMeta(meta) {
		return false
	}
	// PK10 前一复式单注须 solo=false（bet-probe rule=192）；前二及以上须 solo=true。
	if meta.PlayTemplate == "pk10_std" {
		_, segLen := segmentRange(meta)
		return segLen > 1
	}
	return true
}

func paddedDanshiNeedsSolo(meta RuleMeta, mode string, betsNums int) bool {
	if betsNums != 1 {
		return false
	}
	if !usesPaddedDigits(meta.PlayTemplate) {
		return false
	}
	if isSyxwRenxuanMeta(meta) {
		return false
	}
	return mode == "danshi" || mode == "zuxuan_ds"
}

func samplePaddedFlatFushiContent(segLen int) string {
	if segLen <= 0 {
		segLen = 1
	}
	parts := make([]string, segLen)
	for i := 0; i < segLen; i++ {
		parts[i] = string(rune('1' + i))
	}
	return strings.Join(parts, ",")
}

func countK3HezhiBetNums(wireContent string) int {
	picks := parseIntTokenList(wireContent)
	if len(picks) == 0 {
		return countK3SumCombinations(6)
	}
	total := 0
	for _, sum := range picks {
		total += countK3SumCombinations(sum)
	}
	if total <= 0 {
		return 1
	}
	return total
}

func countK3SumCombinations(targetSum int) int {
	if targetSum < 3 || targetSum > 18 {
		return 0
	}
	count := 0
	for a := 1; a <= 6; a++ {
		for b := 1; b <= 6; b++ {
			for c := 1; c <= 6; c++ {
				if a+b+c == targetSum {
					count++
				}
			}
		}
	}
	return count
}

func zuGroupPickCount(mode string) int {
	switch mode {
	case "zu120", "zu60", "zu30", "zu20", "zu10", "zu5":
		return 5
	default:
		return 5
	}
}
