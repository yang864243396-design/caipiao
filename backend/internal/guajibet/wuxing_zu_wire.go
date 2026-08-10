package guajibet

import "strings"

// 五星组选 wire（第三方 game19 实测）：
//   - zu60:  二重号池,单号池   如 1,234 / 12,3456  → Σ_d C(|单号\{d}|,3)
//   - zu30:  二重号池,单号池   如 123,1 / 123,45   → 二重≥3、单号≥1 → Σ_{d1<d2} |单号\{d1,d2}|
//   - zu20:  三重号,单号       如 12,34 / 123,456 → 两区个数须相同且各≥2 → Σ_t C(|单号\{t}|,2)
//   - zu10:  三重号,二重号     如 1,2 / 12,34      → Σ_t |二重\{t}|
//   - zu5:   四重号,单号       如 1,2 / 12,34      → Σ_q |单号\{q}|

func sampleWuxingZu60Content() string {
	return "1,234"
}

func sampleWuxingZu30Content() string {
	return "123,1"
}

func sampleWuxingZu20Content() string {
	return "12,34"
}

func sampleWuxingZu10Content() string {
	return "1,2"
}

func sampleWuxingZu5Content() string {
	return "1,2"
}

func formatWuxingZuWire(mode, groupContent string) string {
	switch mode {
	case "zu60":
		return formatWuxingZu60Wire(groupContent)
	case "zu30":
		return formatWuxingZu30Wire(groupContent)
	case "zu20":
		return formatWuxingZu20Wire(groupContent)
	case "zu10":
		return formatWuxingZu10Wire(groupContent)
	case "zu5":
		return formatWuxingZu5Wire(groupContent)
	default:
		return formatCommaPickDigits(groupContent)
	}
}

// formatZu12Wire 四星/前后四组选12：双区「二重号池,单号池」，如 12,34 / 1,234 / 23,123。
// 二重 ≥1、单号区 ≥2（各位 0–9 连写；区内去重保序；跨区重叠原样出站，勿剔除）。
func formatZu12Wire(groupContent string) string {
	groupContent = strings.TrimSpace(groupContent)
	if groupContent == "" {
		return "12,34"
	}
	parts := splitCommaParts(groupContent)
	if len(parts) == 2 {
		a := uniqueDigitRun(normalizePickDigits(parts[0]))
		b := uniqueDigitRun(normalizePickDigits(parts[1]))
		if len(a) >= 1 && len(b) >= 2 {
			return a + "," + b
		}
	}
	digits := splitPickDigits(groupContent)
	if len(digits) >= 4 {
		// 扁选兼容：前 2 码二重、后 2 码单号（历史口径 1,2,3,4 → 12,34）
		a := uniqueDigitRun(strings.Join(digits[:2], ""))
		b := uniqueDigitRun(strings.Join(digits[2:4], ""))
		if len(a) >= 1 && len(b) >= 2 {
			return a + "," + b
		}
	}
	if len(digits) >= 3 {
		// 扁选：首码作二重，其余作单号
		a := uniqueDigitRun(digits[0])
		b := uniqueDigitRun(strings.Join(digits[1:], ""))
		if len(a) >= 1 && len(b) >= 2 {
			return a + "," + b
		}
	}
	return "12,34"
}

// uniqueDigitRun 从数字串提取 0–9，去重保序。
func uniqueDigitRun(s string) string {
	seen := make(map[byte]bool, 10)
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c < '0' || c > '9' || seen[c] {
			continue
		}
		seen[c] = true
		b.WriteByte(c)
	}
	return b.String()
}

// formatZu4Wire 四星/前后四组选4：双区「三重号池,单号池」，如 12,34 / 1,234 / 1,2。
// 三重 ≥1、单号区 ≥1（各位 0–9 连写；区内去重保序；跨区重叠原样出站，勿剔除）。
func formatZu4Wire(groupContent string) string {
	groupContent = strings.TrimSpace(groupContent)
	if groupContent == "" {
		return "1,2"
	}
	parts := splitCommaParts(groupContent)
	if len(parts) == 2 {
		a := uniqueDigitRun(normalizePickDigits(parts[0]))
		b := uniqueDigitRun(normalizePickDigits(parts[1]))
		if len(a) >= 1 && len(b) >= 1 {
			return a + "," + b
		}
	}
	digits := splitPickDigits(groupContent)
	if len(digits) >= 4 {
		// 扁选兼容：首码三重、后 3 码单号（1,2,3,4 → 1,234）
		a := uniqueDigitRun(digits[0])
		b := uniqueDigitRun(strings.Join(digits[1:4], ""))
		if len(a) >= 1 && len(b) >= 1 {
			return a + "," + b
		}
	}
	if len(digits) >= 2 {
		a := uniqueDigitRun(digits[0])
		b := uniqueDigitRun(strings.Join(digits[1:], ""))
		if len(a) >= 1 && len(b) >= 1 {
			return a + "," + b
		}
	}
	return "1,2"
}

func formatWuxingZu60Wire(groupContent string) string {
	groupContent = strings.TrimSpace(groupContent)
	if groupContent == "" {
		return sampleWuxingZu60Content()
	}
	if wire, ok := normalizeWuxingZu60Wire(groupContent); ok {
		return wire
	}
	// 扁选「0,1,2,3,4」→「0,1234」
	if wire, ok := coerceFlatDigitsToDoubleSingle(groupContent, 1, 3); ok {
		if nwire, ok2 := normalizeWuxingZu60Wire(wire); ok2 {
			return nwire
		}
	}
	return sampleWuxingZu60Content()
}

// coerceFlatDigitsToDoubleSingle 把扁选号池压成「前 headLen 码作头段, 其余拼成尾段」。
func coerceFlatDigitsToDoubleSingle(groupContent string, headLen, minTail int) (string, bool) {
	digits := splitPickDigits(groupContent)
	if headLen <= 0 || len(digits) < headLen+minTail {
		return "", false
	}
	// 已是两段且头段长度匹配时勿再压
	if parts := splitCommaParts(groupContent); len(parts) == 2 {
		if len(normalizePickDigits(parts[0])) == headLen && len(normalizePickDigits(parts[1])) >= minTail {
			return "", false
		}
	}
	head := strings.Join(digits[:headLen], "")
	tail := strings.Join(digits[headLen:], "")
	if head == "" || tail == "" {
		return "", false
	}
	return head + "," + tail, true
}

func formatWuxingZu30Wire(groupContent string) string {
	groupContent = strings.TrimSpace(groupContent)
	if groupContent == "" {
		return sampleWuxingZu30Content()
	}
	if wire, ok := normalizeWuxingZu30Wire(groupContent); ok {
		return wire
	}
	// 扁选 ≥4 码 →「前3码作二重, 其余作单号」
	if wire, ok := coerceFlatDigitsToDoubleSingle(groupContent, 3, 1); ok {
		if nwire, ok2 := normalizeWuxingZu30Wire(wire); ok2 {
			return nwire
		}
	}
	return sampleWuxingZu30Content()
}

func formatWuxingZu20Wire(groupContent string) string {
	groupContent = strings.TrimSpace(groupContent)
	if groupContent == "" {
		return sampleWuxingZu20Content()
	}
	if wire, ok := normalizeWuxingZu20Wire(groupContent); ok {
		return wire
	}
	// 扁选偶数码 ≥4 → 对半拆「三重,单号」（保两区个数相同）
	digits := splitPickDigits(groupContent)
	if n := len(digits); n >= 4 && n%2 == 0 {
		half := n / 2
		wire := strings.Join(digits[:half], "") + "," + strings.Join(digits[half:], "")
		if nwire, ok := normalizeWuxingZu20Wire(wire); ok {
			return nwire
		}
	}
	return sampleWuxingZu20Content()
}

func formatWuxingZu10Wire(groupContent string) string {
	groupContent = strings.TrimSpace(groupContent)
	if groupContent == "" {
		return sampleWuxingZu10Content()
	}
	if wire, ok := normalizeWuxingZuPairWire(groupContent, 1, 1); ok {
		return wire
	}
	// 兼容旧「0,五码池」
	if wire, ok := normalizeWuxingZuZeroPoolWire(groupContent); ok {
		return wire
	}
	digits := splitPickDigits(groupContent)
	if len(digits) >= 2 {
		a := uniqueDigitRun(digits[0])
		b := uniqueDigitRun(strings.Join(digits[1:], ""))
		if wire, ok := normalizeWuxingZuPairWire(a+","+b, 1, 1); ok {
			return wire
		}
	}
	return sampleWuxingZu10Content()
}

func formatWuxingZu5Wire(groupContent string) string {
	groupContent = strings.TrimSpace(groupContent)
	if groupContent == "" {
		return sampleWuxingZu5Content()
	}
	if wire, ok := normalizeWuxingZuPairWire(groupContent, 1, 1); ok {
		return wire
	}
	if wire, ok := normalizeWuxingZuZeroPoolWire(groupContent); ok {
		return wire
	}
	digits := splitPickDigits(groupContent)
	if len(digits) >= 2 {
		a := uniqueDigitRun(digits[0])
		b := uniqueDigitRun(strings.Join(digits[1:], ""))
		if wire, ok := normalizeWuxingZuPairWire(a+","+b, 1, 1); ok {
			return wire
		}
	}
	return sampleWuxingZu5Content()
}

// normalizeWuxingZu60Wire 二重 ≥1、单号 ≥3。
func normalizeWuxingZu60Wire(wire string) (string, bool) {
	parts := splitCommaParts(strings.TrimSpace(wire))
	if len(parts) != 2 {
		return "", false
	}
	a := uniqueDigitRun(normalizePickDigits(parts[0]))
	b := uniqueDigitRun(normalizePickDigits(parts[1]))
	if len(a) < 1 || len(b) < 3 {
		return "", false
	}
	return a + "," + b, true
}

// normalizeWuxingZu30Wire 二重 ≥3、单号 ≥1（区内去重保序；跨区重叠保留）。
func normalizeWuxingZu30Wire(wire string) (string, bool) {
	parts := splitCommaParts(strings.TrimSpace(wire))
	if len(parts) != 2 {
		return "", false
	}
	a := uniqueDigitRun(normalizePickDigits(parts[0]))
	b := uniqueDigitRun(normalizePickDigits(parts[1]))
	if len(a) < 3 || len(b) < 1 {
		return "", false
	}
	return a + "," + b, true
}

// normalizeWuxingZu20Wire 三重号与单号个数须相同，且各≥2（C(n,2) 形态）。
func normalizeWuxingZu20Wire(wire string) (string, bool) {
	parts := splitCommaParts(strings.TrimSpace(wire))
	if len(parts) != 2 {
		return "", false
	}
	a := uniqueDigitRun(normalizePickDigits(parts[0]))
	b := uniqueDigitRun(normalizePickDigits(parts[1]))
	if len(a) < 2 || len(b) < 2 || len(a) != len(b) {
		return "", false
	}
	return a + "," + b, true
}

// normalizeWuxingZuPairWire 双区各至少 minHead/minTail 码（组选10/5）。
func normalizeWuxingZuPairWire(wire string, minHead, minTail int) (string, bool) {
	parts := splitCommaParts(strings.TrimSpace(wire))
	if len(parts) != 2 {
		return "", false
	}
	a := uniqueDigitRun(normalizePickDigits(parts[0]))
	b := uniqueDigitRun(normalizePickDigits(parts[1]))
	if len(a) < minHead || len(b) < minTail {
		return "", false
	}
	return a + "," + b, true
}

func normalizeWuxingZuZeroPoolWire(wire string) (string, bool) {
	parts := splitCommaParts(strings.TrimSpace(wire))
	if len(parts) != 2 {
		return "", false
	}
	head := normalizePickDigits(parts[0])
	tail := normalizePickDigits(parts[1])
	if head != "0" || len(tail) != 5 {
		return "", false
	}
	return "0," + tail, true
}

func countWuxingZu60BetNums(wireContent string) int {
	wire, ok := normalizeWuxingZu60Wire(wireContent)
	if !ok {
		return 0
	}
	parts := splitCommaParts(wire)
	a := uniqueDigitRun(normalizePickDigits(parts[0]))
	b := uniqueDigitRun(normalizePickDigits(parts[1]))
	total := 0
	for i := 0; i < len(a); i++ {
		n := 0
		for j := 0; j < len(b); j++ {
			if b[j] != a[i] {
				n++
			}
		}
		total += combin(n, 3)
	}
	return total
}

// countWuxingZu30BetNums 对每个二重对 (d1,d2)，计 |单号\{d1,d2}| 并求和。
func countWuxingZu30BetNums(wireContent string) int {
	wire, ok := normalizeWuxingZu30Wire(wireContent)
	if !ok {
		return 0
	}
	parts := splitCommaParts(wire)
	a := uniqueDigitRun(normalizePickDigits(parts[0]))
	b := uniqueDigitRun(normalizePickDigits(parts[1]))
	total := 0
	for i := 0; i < len(a); i++ {
		for j := i + 1; j < len(a); j++ {
			n := 0
			for k := 0; k < len(b); k++ {
				if b[k] != a[i] && b[k] != a[j] {
					n++
				}
			}
			total += n
		}
	}
	return total
}

func countWuxingZu20BetNums(wireContent string) int {
	wire, ok := normalizeWuxingZu20Wire(wireContent)
	if !ok {
		return 0
	}
	parts := splitCommaParts(wire)
	a := uniqueDigitRun(normalizePickDigits(parts[0]))
	b := uniqueDigitRun(normalizePickDigits(parts[1]))
	total := 0
	for i := 0; i < len(a); i++ {
		n := 0
		for j := 0; j < len(b); j++ {
			if b[j] != a[i] {
				n++
			}
		}
		total += combin(n, 2)
	}
	return total
}

func countWuxingZu10BetNums(wireContent string) int {
	if n := countZu4BetNums(wireContent); n > 0 {
		return n
	}
	// 旧「0,五码池」
	if _, ok := normalizeWuxingZuZeroPoolWire(wireContent); ok {
		return 5
	}
	return 0
}

func countWuxingZu5BetNums(wireContent string) int {
	return countWuxingZu10BetNums(wireContent)
}

func countWuxingZuBetNums(mode, wireContent string) int {
	switch mode {
	case "zu60":
		return countWuxingZu60BetNums(wireContent)
	case "zu30":
		return countWuxingZu30BetNums(wireContent)
	case "zu20":
		return countWuxingZu20BetNums(wireContent)
	case "zu10":
		return countWuxingZu10BetNums(wireContent)
	case "zu5":
		return countWuxingZu5BetNums(wireContent)
	default:
		return 0
	}
}
