package guajibet

import "strings"

const pk10PositionCount = 10

func positionCountForTemplate(template string) int {
	switch strings.TrimSpace(template) {
	case "pk10_std":
		return pk10PositionCount
	default:
		return sscPositionCount
	}
}

func digitPadWidth(template string) int {
	switch strings.TrimSpace(template) {
	case "syxw_std", "pk10_std":
		return 2
	default:
		return 1
	}
}

func usesPaddedDigits(template string) bool {
	return digitPadWidth(template) > 1
}

// IsPositionWireContent 判断 bet_content 是否为 N 段逗号分隔的 position wire。
func IsPositionWireContent(betContent string, positions int) bool {
	betContent = strings.TrimSpace(betContent)
	if betContent == "" || positions <= 0 {
		return false
	}
	return len(strings.Split(betContent, ",")) == positions
}

// CountPositionWireBetsNums 统计各位非空段号码个数之和。
func CountPositionWireBetsNums(betContent string, positions int) int {
	if !IsPositionWireContent(betContent, positions) {
		return 0
	}
	total := 0
	for _, seg := range strings.Split(betContent, ",") {
		seg = strings.TrimSpace(seg)
		if seg != "" {
			total += len([]rune(seg))
		}
	}
	return total
}

// countPaddedDingweiPicks 11选5/PK10 定位胆：每段按 2 位号码计 1 注（如 "07"→1 注）。
func countPaddedDingweiPicks(betContent string, positions int) int {
	if !IsPositionWireContent(betContent, positions) {
		return 0
	}
	width := 2
	total := 0
	for _, seg := range strings.Split(betContent, ",") {
		seg = strings.TrimSpace(seg)
		if seg == "" {
			continue
		}
		n := len(seg) / width
		if n <= 0 {
			n = 1
		}
		total += n
	}
	return total
}

func countPositionProduct(wireContent string, positions int) int {
	if !IsPositionWireContent(wireContent, positions) {
		return 0
	}
	product := 1
	hasAny := false
	for _, seg := range strings.Split(wireContent, ",") {
		seg = strings.TrimSpace(seg)
		if seg == "" {
			continue
		}
		hasAny = true
		product *= len([]rune(seg))
	}
	if !hasAny {
		return 0
	}
	return product
}

func countPositionProductForTemplate(template, wireContent string, positions int) int {
	width := digitPadWidth(template)
	if width <= 1 {
		return countPositionProduct(wireContent, positions)
	}
	if !IsPositionWireContent(wireContent, positions) {
		return 0
	}
	product := 1
	hasAny := false
	for _, seg := range strings.Split(wireContent, ",") {
		seg = strings.TrimSpace(seg)
		if seg == "" {
			continue
		}
		hasAny = true
		n := len(seg) / width
		if n <= 0 {
			n = 1
		}
		product *= n
	}
	if !hasAny {
		return 0
	}
	return product
}

func padNumericToken(token string, width int) string {
	token = strings.TrimSpace(token)
	if token == "" {
		return ""
	}
	if width <= 1 {
		return token
	}
	for len(token) < width {
		token = "0" + token
	}
	return token
}

func normalizeSegmentDigits(template, line string) string {
	tokens := splitPickTokens(line)
	if len(tokens) == 0 {
		return ""
	}
	width := digitPadWidth(template)
	if width <= 1 {
		return normalizePickDigits(line)
	}
	parts := make([]string, 0, len(tokens))
	for _, t := range tokens {
		parts = append(parts, padNumericToken(t, width))
	}
	return strings.Join(parts, "")
}

// isLaidOutDingweiWire 区分「已按位编排」与「一位号池恰好 N 个逗号分隔码」。
// ",,12,," / ",,,,13579" / "39,,,," → true；"1,3,5,7,9"（个位五码）→ false，应压到 positionIdx。
func isLaidOutDingweiWire(groupContent string, positions int) bool {
	if !IsPositionWireContent(groupContent, positions) {
		return false
	}
	hasEmpty := false
	hasConcat := false
	for _, seg := range strings.Split(groupContent, ",") {
		digits := strings.Map(func(r rune) rune {
			if r >= '0' && r <= '9' {
				return r
			}
			return -1
		}, seg)
		if digits == "" {
			hasEmpty = true
			continue
		}
		if len(digits) > 1 {
			hasConcat = true
		}
	}
	return hasEmpty || hasConcat
}

// formatDingweiWire 定位胆内容 → 五位逗号 wire。
// multiPos=true（一星/全位定位胆）：逗号分位，与定码轮换录入一致，「1,2,3」→「1,2,3,,」。
// multiPos=false（锁定万/千/百/十/个）：逗号为同一位号池，「1,3,5,7,9」压到 positionIdx。
func formatDingweiWire(template string, positionIdx int, groupContent string, multiPos bool) string {
	positions := positionCountForTemplate(template)
	if strings.Contains(groupContent, "\n") {
		return formatDingweiMultiline(template, positions, groupContent)
	}
	if multiPos {
		return formatDingweiCommaPositions(template, positions, groupContent)
	}
	// 已按位编排（含空位或多码连写段）：按位保留；勿把 "1,3,5,7,9" 当成五位 wire
	if isLaidOutDingweiWire(groupContent, positions) {
		segments := make([]string, positions)
		for i, seg := range strings.Split(groupContent, ",") {
			if i >= positions {
				break
			}
			segments[i] = normalizeSegmentDigits(template, seg)
		}
		return strings.Join(segments, ",")
	}
	digits := normalizeSegmentDigits(template, groupContent)
	if digits == "" {
		return groupContent
	}
	if positionIdx < 0 {
		positionIdx = 0
	}
	if positionIdx >= positions {
		positionIdx = positions - 1
	}
	segments := make([]string, positions)
	segments[positionIdx] = digits
	return strings.Join(segments, ",")
}

// formatDingweiCommaPositions 全位定位胆：无换行时按逗号分位（不足补空位）；无逗号则整段落在首位。
func formatDingweiCommaPositions(template string, positions int, groupContent string) string {
	segments := make([]string, positions)
	if !strings.Contains(groupContent, ",") {
		segments[0] = normalizeSegmentDigits(template, groupContent)
		return strings.Join(segments, ",")
	}
	for i, seg := range strings.Split(groupContent, ",") {
		if i >= positions {
			break
		}
		segments[i] = normalizeSegmentDigits(template, seg)
	}
	return strings.Join(segments, ",")
}

func formatDingweiMultiline(template string, positions int, groupContent string) string {
	lines := splitPositionLines(groupContent)
	segments := make([]string, positions)
	for i := 0; i < positions; i++ {
		line := ""
		if i < len(lines) {
			line = lines[i]
		}
		segments[i] = normalizeSegmentDigits(template, line)
	}
	return strings.Join(segments, ",")
}

func formatPositionWire(template string, start, length int, groupContent string) string {
	positions := positionCountForTemplate(template)
	if length <= 0 {
		length = 1
	}
	if start < 0 {
		start = 0
	}
	lines := splitPositionLines(groupContent)
	segments := make([]string, positions)
	for i := 0; i < length; i++ {
		line := ""
		if i < len(lines) {
			line = lines[i]
		} else if len(lines) == 1 {
			line = lines[0]
		}
		pos := start + i
		if pos >= 0 && pos < positions {
			segments[pos] = normalizeSegmentDigits(template, line)
		}
	}
	return strings.Join(segments, ",")
}

func formatPaddedPickDigits(template, groupContent string) string {
	tokens := splitPickTokens(groupContent)
	if len(tokens) == 0 {
		return strings.TrimSpace(groupContent)
	}
	width := digitPadWidth(template)
	if width <= 1 {
		return strings.Join(tokens, ",")
	}
	parts := make([]string, 0, len(tokens))
	for _, t := range tokens {
		parts = append(parts, padNumericToken(t, width))
	}
	return strings.Join(parts, ",")
}

// formatSSCZuxuanDanshiDigits 组选单式：N 位一注、逗号分隔；排除对子/豹子，形态去重保序。
func formatSSCZuxuanDanshiDigits(segLen int, groupContent string) string {
	if segLen <= 0 {
		segLen = 2
	}
	parts := splitCommaParts(groupContent)
	if len(parts) == 0 {
		digits := digitsOnly(groupContent)
		if len(digits) == segLen && !isBaoziDigits(digits) {
			return digits
		}
		return ""
	}
	seen := make(map[string]struct{}, len(parts))
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		d := digitsOnly(p)
		if len(d) != segLen || isBaoziDigits(d) {
			continue
		}
		key := sortDigitRunes(d)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, d)
	}
	return strings.Join(out, ",")
}

// expandPositionPoolToDanshiTickets 将按位号池（每位一行单码）展开为直选单式票。
// 例：segLen=3, "4,5\n3,5\n2,5" → "432,435,452,455,532,535,552,555"。
func expandPositionPoolToDanshiTickets(segLen int, groupContent string) (string, bool) {
	if segLen <= 1 {
		return "", false
	}
	raw := strings.ReplaceAll(groupContent, "\r\n", "\n")
	raw = strings.ReplaceAll(raw, "\r", "\n")
	if !strings.Contains(raw, "\n") {
		return "", false
	}
	lines := splitPositionLines(raw)
	for len(lines) < segLen {
		lines = append(lines, "")
	}
	pools := make([][]string, segLen)
	for i := 0; i < segLen; i++ {
		toks := splitPickTokens(lines[i])
		seen := make(map[string]struct{}, len(toks))
		out := make([]string, 0, len(toks))
		for _, t := range toks {
			d := digitsOnly(t)
			if len(d) != 1 {
				return "", false
			}
			if _, ok := seen[d]; ok {
				continue
			}
			seen[d] = struct{}{}
			out = append(out, d)
		}
		if len(out) == 0 {
			return "", false
		}
		pools[i] = out
	}
	cur := []string{""}
	for _, pool := range pools {
		next := make([]string, 0, len(cur)*len(pool))
		for _, prefix := range cur {
			for _, d := range pool {
				next = append(next, prefix+d)
			}
		}
		cur = next
	}
	if len(cur) == 0 {
		return "", false
	}
	return strings.Join(uniqueStringsPreserve(cur), ","), true
}

// formatSSCDanshiDigits 直选单式：保留「N 位一注、逗号分隔」。
// 勿把 "012,345" 压成 "012345"（第三方会报投注数字格式不正确）。
// 冷热出号等按位号池（"4,5\n3,5\n2,5"）先展开为笛卡尔积单式票。
func formatSSCDanshiDigits(segLen int, groupContent string) string {
	if segLen <= 0 {
		segLen = 1
	}
	if expanded, ok := expandPositionPoolToDanshiTickets(segLen, groupContent); ok {
		return expanded
	}
	tokens := splitPickTokens(groupContent)
	if len(tokens) == 0 {
		return strings.TrimSpace(groupContent)
	}
	complete := make([]string, 0, len(tokens))
	allComplete := true
	for _, t := range tokens {
		d := digitsOnly(t)
		if len(d) != segLen {
			allComplete = false
			break
		}
		complete = append(complete, d)
	}
	if allComplete {
		return strings.Join(uniqueStringsPreserve(complete), ",")
	}
	raw := digitsOnly(strings.Join(tokens, ""))
	if len(raw) < segLen {
		return raw
	}
	if len(raw)%segLen == 0 {
		parts := make([]string, 0, len(raw)/segLen)
		for i := 0; i+segLen <= len(raw); i += segLen {
			parts = append(parts, raw[i:i+segLen])
		}
		return strings.Join(parts, ",")
	}
	// 无法按位宽切分时保持逗号分隔（避免 silent 丢分隔符）
	return formatCommaPickDigits(groupContent)
}

func digitsOnly(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func formatPaddedDanshiDigits(template string, segLen int, groupContent string) string {
	if segLen <= 0 {
		segLen = 1
	}
	width := digitPadWidth(template)
	if width <= 1 {
		return formatSSCDanshiDigits(segLen, groupContent)
	}
	raw := normalizePickDigits(groupContent)
	if len(raw) >= segLen {
		var b strings.Builder
		for i := 0; i < segLen; i++ {
			b.WriteString(padNumericToken(string(raw[i]), width))
		}
		return b.String()
	}
	tokens := splitPickTokens(groupContent)
	if len(tokens) >= segLen {
		var b strings.Builder
		for i := 0; i < segLen; i++ {
			b.WriteString(padNumericToken(tokens[i], width))
		}
		return b.String()
	}
	var b strings.Builder
	for i := 0; i < segLen; i++ {
		b.WriteString(padNumericToken(string(rune('1'+i)), width))
	}
	return b.String()
}
