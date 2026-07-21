package guajibet

import (
	"sort"
	"strconv"
	"strings"
)

// ContentKind 第三方 bet_content 编码类别。
type ContentKind int

const (
	KindUnknown ContentKind = iota
	KindSSCDingweiWire
	KindSSCPositionWire
	KindSSCDanshiDigits
	KindTextTokens
	KindNumberTokens
	KindLHCNumbers
	KindLHCText
)

// SampleGroupContent 为矩阵/冒烟测试生成最小合法内部选号（groupContent）。
func SampleGroupContent(meta RuleMeta) string {
	if meta.PlayTemplate == "lhc_std" {
		return sampleLHCGroupContent(meta)
	}
	if meta.PlayTemplate == "k3_std" && k3SantongMeta(meta) {
		return "1"
	}
	if meta.PlayTemplate == "k3_std" && k3PairPickNeedsSolo(meta) {
		if strings.Contains(meta.Label, "二不同") {
			return "1,3"
		}
		return "1,2"
	}
	mode := InferBetMode(meta)
	switch mode {
	case "dingwei":
		return "7"
	case "fushi":
		if meta.PlayTemplate == "k3_std" {
			return "1,2,3"
		}
		if isRenxuanMeta(meta) {
			return sampleRenxuanContent(meta)
		}
		if isSyxwRenxuanMeta(meta) {
			return sampleSyxwRenxuanContent(meta)
		}
		if paddedFushiUsesFlatPick(meta) {
			_, segLen := segmentRange(meta)
			return samplePaddedFlatFushiContent(segLen)
		}
		_, segLen := segmentRange(meta)
		if segLen <= 1 {
			return "7"
		}
		// 各位用不同单码，避免豹子（如 1,1,1）——第三方网页对豹子计 0 注且无法下注
		lines := make([]string, segLen)
		for i := range lines {
			lines[i] = string(byte('0' + i%10))
		}
		return strings.Join(lines, "\n")
	case "zuxuan_fs":
		if isRenxuanMeta(meta) {
			return sampleRenxuanZuxuanFsContent(meta)
		}
		return sampleZuxuanFushiContent(meta)
	case "zuxuan_ds":
		if isRenxuanMeta(meta) {
			return sampleRenxuanDanshiContent(meta)
		}
		_, segLen := segmentRange(meta)
		if segLen <= 0 {
			segLen = 2
		}
		if segLen == 2 {
			return "12"
		}
		if segLen == 3 {
			return "123"
		}
		digits := make([]byte, segLen)
		for i := range digits {
			digits[i] = byte('1' + i)
		}
		return string(digits)
	case "danshi":
		if isRenxuanMeta(meta) {
			return sampleRenxuanDanshiContent(meta)
		}
		if isSyxwRenxuanMeta(meta) {
			return sampleSyxwRenxuanContent(meta)
		}
		if meta.PlayTemplate == "k3_std" {
			if k3SantongMeta(meta) {
				return "1"
			}
			if strings.Contains(meta.Label, "三连号") {
				return "123"
			}
			if strings.Contains(meta.Label, "手动输入") {
				return "112"
			}
			return "123"
		}
		if meta.PlayTemplate == "syxw_std" {
			_, segLen := segmentRange(meta)
			if segLen <= 0 {
				segLen = 1
			}
			digits := make([]byte, segLen)
			for i := range digits {
				digits[i] = byte('1' + i)
			}
			return string(digits)
		}
		_, segLen := segmentRange(meta)
		if segLen <= 0 {
			segLen = 3
		}
		if usesPaddedDigits(meta.PlayTemplate) {
			return sampleDistinctDigitString(segLen)
		}
		return sampleDistinctDigitString(segLen)
	case "hezhi", "weishu":
		if isRenxuanMeta(meta) {
			return sampleRenxuanHezhiContent(meta)
		}
		if meta.PlayTemplate == "pc28_std" {
			return "1,2"
		}
		if meta.PlayTemplate == "pk10_std" {
			if strings.Contains(meta.Label, "前三") {
				return "12"
			}
			if strings.Contains(meta.Label, "后三") {
				return "10"
			}
			return "3"
		}
		if meta.PlayTemplate == "k3_std" {
			return "6"
		}
		return "6"
	case "kuadu":
		return "0"
	case "zuhe":
		_, segLen := segmentRange(meta)
		if segLen >= 4 {
			parts := make([]string, segLen)
			for i := range parts {
				parts[i] = string(rune('1' + i))
			}
			return strings.Join(parts, ",")
		}
		return "1,2,3"
	case "zu3":
		if isRenxuanMeta(meta) {
			return sampleRenxuanZuxuanFsContent(meta)
		}
		if meta.PlayTemplate == "k3_std" && strings.Contains(meta.Label, "二同号单选") {
			return "1,2"
		}
		if meta.PlayTemplate == "k3_std" && strings.Contains(meta.Label, "二不同") {
			return "1,3"
		}
		return "1,2"
	case "zu6":
		if isRenxuanMeta(meta) {
			return sampleRenxuanZuxuanFsContent(meta)
		}
		if meta.Group == "四星" || meta.Group == "前后四" || meta.TypeID == "g013" || meta.TypeID == "g014" {
			return "1,2,3"
		}
		return "1,2,3"
	case "zu24":
		return "1,2,3,4"
	case "zu12":
		return "12,34"
	case "zu4":
		return "1,2"
	case "zu120", "zu60", "zu30", "zu20", "zu10", "zu5":
		if mode == "zu120" {
			return sampleZu120Digits()
		}
		switch mode {
		case "zu60":
			return sampleWuxingZu60Content()
		case "zu30":
			return sampleWuxingZu30Content()
		case "zu20":
			return sampleWuxingZu20Content()
		case "zu10", "zu5":
			return sampleWuxingZuZeroPoolContent()
		}
	case "baodan":
		return "3"
	case "hunhe":
		if isRenxuanMeta(meta) {
			return sampleRenxuanZuxuanFsContent(meta)
		}
		return "123"
	case "teshu":
		if meta.TypeID == "g015" || meta.Group == "五星" {
			return "6"
		}
		if meta.PlayTemplate == "pc28_std" {
			return "豹子"
		}
		return "豹子"
	case "longhu", "longhuhe":
		return "龙"
	case "zhuangxian":
		return "庄"
	case "dxds", "daxiao", "danshuang":
		return sampleDxdsContent(meta)
	case "budingwei":
		return sampleBudingweiContent(meta)
	}

	if meta.Group == "任选" {
		return sampleRenxuanContent(meta)
	}
	if isSyxwRenxuanMeta(meta) {
		return sampleSyxwRenxuanContent(meta)
	}

	kind, _, segLen := classifyRule(meta)
	switch kind {
	case KindSSCDingweiWire:
		return "7"
	case KindSSCPositionWire:
		if segLen <= 1 {
			if usesPaddedDigits(meta.PlayTemplate) {
				return "01"
			}
			return "7"
		}
		lines := make([]string, segLen)
		for i := range lines {
			if usesPaddedDigits(meta.PlayTemplate) {
				lines[i] = "01"
			} else {
				lines[i] = "1"
			}
		}
		return strings.Join(lines, "\n")
	case KindSSCDanshiDigits:
		if segLen <= 0 {
			segLen = 3
		}
		if usesPaddedDigits(meta.PlayTemplate) {
			return sampleDistinctDigitString(segLen)
		}
		return sampleDistinctDigitString(segLen)
	case KindTextTokens:
		text := meta.combinedText()
		if strings.Contains(text, "龙虎") {
			if strings.Contains(text, "和") {
				return "和"
			}
			return "龙"
		}
		if strings.Contains(text, "大小") || strings.Contains(meta.Label, "大") {
			return "大"
		}
		if strings.Contains(text, "单双") || strings.Contains(meta.Label, "单") {
			return "单"
		}
		if strings.Contains(text, "豹") {
			return "豹"
		}
		if strings.Contains(text, "家") {
			return "家"
		}
		if strings.Contains(text, "红") || strings.Contains(text, "蓝") || strings.Contains(text, "绿") {
			return "红"
		}
		if strings.Contains(text, "金") || strings.Contains(text, "木") {
			return "金"
		}
		return "龙"
	case KindNumberTokens:
		if meta.PlayTemplate == "k3_std" {
			return "3"
		}
		if usesPaddedDigits(meta.PlayTemplate) {
			return "01"
		}
		return "6"
	case KindLHCNumbers:
		return "01"
	case KindLHCText:
		if strings.Contains(meta.combinedText(), "肖") {
			return "鼠"
		}
		return "01"
	default:
		return "1"
	}
}

// normalizeGroupContentEdges 清理首尾空白；含换行时保留前导/尾随空行（定位胆位次）。
func normalizeGroupContentEdges(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	if strings.Contains(s, "\n") {
		return strings.Trim(s, " \t")
	}
	return strings.TrimSpace(s)
}

// FormatBetContentForRule 将 groupContent 转为第三方 bet_content。
func FormatBetContentForRule(meta RuleMeta, groupContent string) string {
	groupContent = normalizeGroupContentEdges(groupContent)
	if strings.TrimSpace(groupContent) == "" {
		groupContent = SampleGroupContent(meta)
	}
	if meta.PlayTemplate == "k3_std" && (k3PairPickNeedsSolo(meta) || k3SantongMeta(meta)) {
		return formatCommaPickDigits(groupContent)
	}
	mode := InferBetMode(meta)
	if isRenxuanMeta(meta) {
		return formatRenxuanBetContent(meta, mode, groupContent)
	}
	if isSyxwRenxuanMeta(meta) {
		switch mode {
		case "danshi":
			k := syxwRenxuanPickN(meta)
			if k <= 0 {
				k = 1
			}
			return formatPaddedDanshiDigits(meta.PlayTemplate, k, groupContent)
		default:
			return formatPaddedPickDigits(meta.PlayTemplate, groupContent)
		}
	}
	if meta.PlayTemplate == "lhc_std" {
		switch inferLHCBetMode(meta) {
		case "fushi":
			switch strings.TrimSpace(meta.Group) {
			case "生肖连", "尾数连":
				return formatTextTokens(groupContent)
			default:
				return formatLHCPickDigits(groupContent)
			}
		case "buzhong", "xuanyi", "renzhong":
			return formatLHCPickDigits(groupContent)
		case "tema", "zhengte":
			return formatLHCTemaWire(groupContent)
		case "zongxiao":
			return formatLHCZongxiaoWire(groupContent)
		case "qima":
			return formatLHCQimaWire(groupContent)
		case "tematouwei":
			return formatLHCTematouweiWire(groupContent)
		default:
			return strings.TrimSpace(groupContent)
		}
	}
	if mode == "dxds" || mode == "daxiao" || mode == "danshuang" {
		if IsSSCPlayTemplate(meta.PlayTemplate) {
			return formatDxdsBetContent(meta, groupContent)
		}
		if meta.PlayTemplate == "pk10_std" {
			if isPK10DxdsComboMeta(meta) {
				return formatPK10DxdsComboWire(groupContent)
			}
			return formatTextTokens(groupContent)
		}
	}
	switch mode {
	case "dingwei":
		pos := dingweiPositionIndex(meta)
		return formatDingweiWire(meta.PlayTemplate, pos, groupContent)
	case "fushi":
		if meta.PlayTemplate == "k3_std" {
			return formatCommaPickDigits(groupContent)
		}
		if paddedFushiUsesFlatPick(meta) {
			return formatPaddedPickDigits(meta.PlayTemplate, groupContent)
		}
		_, segStart, segLen := classifyRule(meta)
		if usesPaddedDigits(meta.PlayTemplate) {
			return formatPositionWire(meta.PlayTemplate, segStart, segLen, groupContent)
		}
		return formatSSCFushiContent(segStart, segLen, groupContent)
	case "zuxuan_fs", "zu3", "zu6":
		if usesPaddedDigits(meta.PlayTemplate) {
			return formatPaddedPickDigits(meta.PlayTemplate, groupContent)
		}
		return formatCommaPickDigits(groupContent)
	case "zuxuan_ds":
		if usesPaddedDigits(meta.PlayTemplate) {
			_, segLen := segmentRange(meta)
			return formatPaddedDanshiDigits(meta.PlayTemplate, segLen, groupContent)
		}
		_, segLen := segmentRange(meta)
		// 组选单式：排除对子/豹子，并按组选形态去重（12 与 21 同一注）
		return formatSSCZuxuanDanshiDigits(segLen, groupContent)
	case "danshi":
		if meta.PlayTemplate == "k3_std" {
			return normalizePickDigits(groupContent)
		}
		if usesPaddedDigits(meta.PlayTemplate) {
			_, segLen := segmentRange(meta)
			return formatPaddedDanshiDigits(meta.PlayTemplate, segLen, groupContent)
		}
		_, segLen := segmentRange(meta)
		return formatSSCDanshiDigits(segLen, groupContent)
	case "hezhi", "kuadu", "weishu", "baodan", "hunhe", "budingwei":
		if mode == "hezhi" && usesPaddedDigits(meta.PlayTemplate) {
			return formatPaddedPickDigits(meta.PlayTemplate, groupContent)
		}
		if mode == "budingwei" && meta.PlayTemplate == "syxw_std" {
			return formatPaddedPickDigits(meta.PlayTemplate, groupContent)
		}
		if mode == "budingwei" {
			return formatBudingweiContent(meta, groupContent)
		}
		return formatCommaPickDigits(groupContent)
	case "zu24", "zu12", "zu4", "zu120", "zu60", "zu30", "zu20", "zu10", "zu5":
		switch mode {
		case "zu120", "zu24":
			// 单号池 C(n,4)/C(n,5)
			return formatCommaPickDigits(groupContent)
		case "zu12":
			// 双区「二重号,单号」如 12,34；扁选 1,2,3,4 → 12,34
			return formatZu12Wire(groupContent)
		case "zu4":
			// 双区「三重号,单号」如 1,2
			return formatZu4Wire(groupContent)
		default:
			return formatWuxingZuWire(mode, groupContent)
		}
	case "zuhe":
		_, segStart, segLen := classifyRule(meta)
		if usesPaddedDigits(meta.PlayTemplate) {
			return formatPositionWire(meta.PlayTemplate, segStart, segLen, groupContent)
		}
		return formatSSCZuheContent(segStart, segLen, groupContent)
	case "teshu", "longhu", "longhuhe", "zhuangxian":
		return formatTextTokens(groupContent)
	}

	kind, segStart, segLen := classifyRule(meta)
	switch kind {
	case KindSSCDingweiWire:
		pos := dingweiPositionIndex(meta)
		return formatDingweiWire(meta.PlayTemplate, pos, groupContent)
	case KindSSCPositionWire:
		if usesPaddedDigits(meta.PlayTemplate) {
			return formatPositionWire(meta.PlayTemplate, segStart, segLen, groupContent)
		}
		return formatSSCFushiContent(segStart, segLen, groupContent)
	case KindSSCDanshiDigits:
		if usesPaddedDigits(meta.PlayTemplate) {
			_, segLen := segmentRange(meta)
			return formatPaddedDanshiDigits(meta.PlayTemplate, segLen, groupContent)
		}
		return formatSSCDanshiDigits(segLen, groupContent)
	case KindTextTokens:
		return formatTextTokens(groupContent)
	case KindNumberTokens:
		if usesPaddedDigits(meta.PlayTemplate) {
			return formatPaddedPickDigits(meta.PlayTemplate, groupContent)
		}
		return normalizePickDigits(groupContent)
	case KindLHCNumbers:
		return formatLHCPickDigits(groupContent)
	case KindLHCText:
		return strings.TrimSpace(groupContent)
	default:
		return groupContent
	}
}

// CountBetNums 统计第三方 bets_nums。
func CountBetNums(meta RuleMeta, wireContent string) int {
	wireContent = strings.TrimSpace(wireContent)
	if wireContent == "" {
		return 0
	}
	mode := InferBetMode(meta)
	_, segLen := segmentRange(meta)
	if meta.PlayTemplate == "lhc_std" {
		if n := countLHCBetNums(meta, wireContent); n > 0 {
			return n
		}
	}
	if isSyxwRenxuanMeta(meta) {
		if n := countSyxwRenxuanBetNums(meta, wireContent); n > 0 {
			return n
		}
	}
	if isRenxuanMeta(meta) {
		switch mode {
		case "fushi", "danshi", "zuxuan_ds", "hezhi", "zuxuan_fs", "zu3", "zu6", "zu24", "zu12", "zu4", "hunhe":
			return countRenxuanPoolBetNums(meta, wireContent)
		}
	}
	switch mode {
	case "fushi":
		if meta.PlayTemplate == "k3_std" {
			tokens := splitPickTokens(wireContent)
			if len(tokens) == 0 {
				return 0
			}
			if strings.Contains(meta.Label, "同号") && strings.Contains(meta.Label, "复选") {
				return len(tokens)
			}
			return 1
		}
		if paddedFushiUsesFlatPick(meta) {
			tokens := splitPickTokens(wireContent)
			if len(tokens) == 0 {
				return 0
			}
			if meta.PlayTemplate == "k3_std" && strings.Contains(meta.Label, "同号") && strings.Contains(meta.Label, "复选") {
				return len(tokens)
			}
			return applySegmentMultiplier(meta, 1)
		}
		if usesPaddedDigits(meta.PlayTemplate) {
			positions := positionCountForTemplate(meta.PlayTemplate)
			if IsPositionWireContent(wireContent, positions) {
				n := countPositionProductForTemplate(meta.PlayTemplate, wireContent, positions)
				if isSSCFushiBaoziWire(wireContent) {
					return 0
				}
				return applySegmentMultiplier(meta, n)
			}
			_, segStart, segLen := classifyRule(meta)
			wire := formatPositionWire(meta.PlayTemplate, segStart, segLen, wireContent)
			n := countPositionProductForTemplate(meta.PlayTemplate, wire, positions)
			if isSSCFushiBaoziWire(wire) {
				return 0
			}
			return applySegmentMultiplier(meta, n)
		}
		n := countSSCFushiProduct(wireContent)
		if isSSCFushiBaoziWire(wireContent) {
			return 0
		}
		return applySegmentMultiplier(meta, n)
	case "zuxuan_fs":
		_, segLen := segmentRange(meta)
		if paddedFushiUsesFlatPick(meta) {
			tokens := splitPickTokens(wireContent)
			if len(tokens) == segLen {
				return applySegmentMultiplier(meta, 1)
			}
			return applySegmentMultiplier(meta, countZuxuanFushiBetNums(len(tokens), segLen))
		}
		return applySegmentMultiplier(meta, countZuxuanFushiBetNums(len(splitPickDigits(wireContent)), segLen))
	case "zuxuan_ds":
		// 与混合组选一致：排除对子/豹子，按排序形态去重（对齐第三方预览）
		n := countSSCHunheBetNums(wireContent, segLen)
		return applySegmentMultiplier(meta, n)
	case "hezhi":
		if meta.PlayTemplate == "k3_std" {
			return countK3HezhiBetNums(wireContent)
		}
		if meta.PlayTemplate == "pk10_std" {
			picks := parseIntTokenList(wireContent)
			if len(picks) == 0 {
				return 1
			}
			return len(picks)
		}
		if meta.PlayTemplate == "pc28_std" {
			picks := parseIntTokenList(wireContent)
			if len(picks) == 0 {
				return 1
			}
			return len(picks)
		}
		return countHezhiBetNums(meta, wireContent, segLen)
	case "kuadu", "weishu", "baodan":
		picks := parseIntTokenList(wireContent)
		if len(picks) == 0 {
			return 1
		}
		if mode == "kuadu" {
			total := 0
			for _, span := range picks {
				total += countOrderedSpanCombinations(span, segLen)
			}
			if total > 0 {
				return applySegmentMultiplier(meta, total)
			}
		}
		if mode == "baodan" {
			return applySegmentMultiplier(meta, countBaodanBetNums(len(picks), segLen))
		}
		return applySegmentMultiplier(meta, len(picks))
	case "budingwei":
		return countBudingweiBetNums(meta, wireContent)
	case "zu3":
		return applySegmentMultiplier(meta, zu3PoolUnits(len(splitPickDigits(wireContent))))
	case "zu6":
		if meta.Group == "四星" || meta.Group == "前后四" || meta.TypeID == "g013" || meta.TypeID == "g014" {
			return applySegmentMultiplier(meta, countSixingZu6BetNums(len(splitPickDigits(wireContent))))
		}
		return applySegmentMultiplier(meta, zu6PoolUnits(len(splitPickDigits(wireContent))))
	case "zu12":
		return applySegmentMultiplier(meta, countZu12BetNums(wireContent))
	case "zu24", "zu4", "zu120", "zu60", "zu30", "zu20", "zu10", "zu5":
		if n := countWuxingZuBetNums(mode, wireContent); n > 0 {
			return applySegmentMultiplier(meta, n)
		}
		if mode != "zu60" && mode != "zu30" && mode != "zu20" && mode != "zu10" && mode != "zu5" {
			return applySegmentMultiplier(meta, countZuGroupBetNums(mode, len(splitPickDigits(wireContent))))
		}
		return 0
	case "zuhe":
		// 直选组合：与复式同形「各位数字串」；注数 = 位积 × 段长（三星×3）
		product := countSSCFushiProduct(wireContent)
		if product <= 0 {
			return 0
		}
		if segLen <= 0 {
			_, segLen = segmentRange(meta)
		}
		if segLen <= 0 {
			segLen = 1
		}
		return applySegmentMultiplier(meta, product*segLen)
	case "hunhe":
		n := countSSCHunheBetNums(wireContent, segLen)
		return applySegmentMultiplier(meta, n)
	case "teshu", "longhu", "longhuhe", "dxds", "daxiao", "danshuang":
		if isPositionDxds(meta) {
			return 1
		}
		tokens := splitPickTokens(wireContent)
		if len(tokens) == 0 {
			return applySegmentMultiplier(meta, 1)
		}
		return applySegmentMultiplier(meta, len(tokens))
	}

	kind, _, segLen := classifyRule(meta)
	positions := positionCountForTemplate(meta.PlayTemplate)
	switch kind {
	case KindSSCDingweiWire, KindSSCPositionWire:
		if !IsPositionWireContent(wireContent, positions) {
			return 0
		}
		if kind == KindSSCDingweiWire {
			if usesPaddedDigits(meta.PlayTemplate) {
				return countPaddedDingweiPicks(wireContent, positions)
			}
			return CountPositionWireBetsNums(wireContent, positions)
		}
		return countPositionProductForTemplate(meta.PlayTemplate, wireContent, positions)
	case KindSSCDanshiDigits:
		parts := uniqueStringsPreserve(splitCommaParts(wireContent))
		n := 0
		if len(parts) == 0 {
			if len(normalizePickDigits(wireContent)) >= segLen && segLen > 0 {
				n = 1
			}
		} else {
			n = len(parts)
		}
		return applySegmentMultiplier(meta, n)
	case KindTextTokens, KindNumberTokens, KindLHCNumbers, KindLHCText:
		if n := countK3PairPickBetNums(meta, wireContent); n > 0 {
			return n
		}
		if isSyxwRenxuanMeta(meta) {
			return countSyxwRenxuanBetNums(meta, wireContent)
		}
		tokens := splitPickTokens(wireContent)
		if len(tokens) == 0 && wireContent != "" {
			return 1
		}
		return len(tokens)
	default:
		return 1
	}
}

// NeedsSoloForRule 是否须 solo=true。
func NeedsSoloForRule(meta RuleMeta, wireContent string) bool {
	if guajiGroupRequiresSoloFalse(meta) {
		return false
	}
	mode := InferBetMode(meta)
	group := strings.TrimSpace(meta.Group)
	// 前后二：实测任意玩法 solo=true →「单挑参数错误」。前后四相反，须 solo=true（见 ResolveSolo 实测）。
	if group == "前后二" || meta.TypeID == "g008" {
		return false
	}
	textQH := meta.Group + " " + meta.TypeLabel + " " + meta.Label + " " + meta.TeamLabel + " " + meta.FullName
	if strings.Contains(textQH, "前后二") &&
		!strings.Contains(textQH, "前中后三") && !strings.Contains(textQH, "前后三") &&
		!strings.Contains(textQH, "前后四") {
		return false
	}
	if mode == "weishu" || mode == "teshu" || mode == "baodan" {
		return false
	}
	// 前后三/前中后三混合组选：实测 solo=true →「单挑参数错误」
	if mode == "hunhe" {
		text := meta.Group + " " + meta.TypeLabel + " " + meta.Label + " " + meta.FullName
		if strings.Contains(text, "前后三") || strings.Contains(text, "前中后三") {
			return false
		}
	}
	if meta.PlayTemplate == "pc28_std" && mode == "hezhi" {
		return false
	}
	if mode == "zuhe" {
		// 直选组合：多区位玩法实测须 solo=false（与直选单式/组三不同）
		g := strings.TrimSpace(meta.Group)
		switch g {
		case "四星", "五星", "前后四", "前中后三", "前后三":
			return false
		}
		if strings.Contains(meta.Group+meta.TypeLabel+meta.Label, "前中后三") ||
			strings.Contains(meta.Group+meta.TypeLabel+meta.Label, "前后三") {
			return false
		}
	}
	if mode == "longhu" || mode == "longhuhe" || mode == "zhuangxian" {
		return false
	}
	if mode == "dxds" || mode == "daxiao" || mode == "danshuang" {
		return false
	}
	if mode == "zu24" || mode == "zu12" || mode == "zu4" || mode == "zu120" || mode == "zu60" || mode == "zu30" || mode == "zu20" || mode == "zu10" || mode == "zu5" {
		return false
	}
	if mode == "zu6" && (meta.Group == "四星" || meta.Group == "前后四" || meta.TypeID == "g013" || meta.TypeID == "g014") {
		return false
	}
	// SSC 前二/后二组选复式：勿走末尾默认 solo=true
	if sscErxingZuxuanFsForcesSoloFalse(meta) {
		return false
	}
	if meta.PlayTemplate == "lhc_std" {
		return false
	}
	betsNums := CountBetNums(meta, wireContent)
	if isRenxuanMeta(meta) {
		return renxuanNeedsSoloTrue(meta, mode, betsNums)
	}
	if isSyxwRenxuanMeta(meta) {
		return syxwRenxuanNeedsSoloTrue(meta, betsNums)
	}
	if mode == "hezhi" && strings.Contains(meta.Label, "组选") && group != "前后二" && group != "前后四" {
		return false
	}
	if mode == "hezhi" && !strings.Contains(meta.Label, "组选") && group != "前后二" && group != "前后四" {
		if meta.Group == "任选" || meta.TypeID == "g011" {
			return false
		}
	}
	if mode == "budingwei" {
		return budingweiNeedsSolo(meta)
	}
	if wuxingFushiNeedsSolo(meta, mode, betsNums) {
		return true
	}
	if paddedFushiNeedsSolo(meta, mode, betsNums) {
		return true
	}
	if paddedDanshiNeedsSolo(meta, mode, betsNums) {
		return true
	}
	if paddedFushiUsesFlatPick(meta) && mode == "fushi" {
		if meta.PlayTemplate == "pk10_std" {
			_, segLen := segmentRange(meta)
			if segLen <= 1 {
				return false
			}
		}
	}
	if mode == "dingwei" {
		return false
	}
	if meta.PlayTemplate == "pk10_std" && mode == "hezhi" {
		return false
	}
	if meta.PlayTemplate == "pk10_std" && (mode == "dxds" || mode == "daxiao" || mode == "danshuang") {
		return false
	}
	if meta.PlayTemplate == "k3_std" {
		if mode == "hezhi" {
			return false
		}
		if k3PairPickNeedsSolo(meta) || k3SantongMeta(meta) {
			return betsNums == 1
		}
		if strings.Contains(meta.Label, "单挑") {
			return betsNums == 1
		}
		return false
	}
	if meta.PlayTemplate == "syxw_std" {
		switch mode {
		case "budingwei":
			return false
		}
	}
	// 注意：不可用 IsSSCDingweiBetContent 反推 solo。前三复式若被误格式成
	// "013,0,0,,"（五段）会命中定位胆 wire，导致 solo=false → 单挑参数错误。
	// 定位胆已在上方 mode=="dingwei" 分支返回 false。
	return true
}

// ResolveBetsNums 计算第三方 bets_nums；meta 未填 PlayTemplate 时回退定位胆 wire 统计。
func ResolveBetsNums(meta RuleMeta, wireContent string, amount, amountUnit float64, multiplier int) int {
	if strings.TrimSpace(meta.PlayTemplate) != "" {
		if n := CountBetNums(meta, wireContent); n > 0 {
			return n
		}
		// 直选复式豹子/对子：CountBetNums 明确为 0，禁止回退成 1（对齐第三方网页无法下注）
		if IsFushiBaoziZeroBet(meta, wireContent) {
			return 0
		}
	}
	if n := CountDingweiBetsNums(wireContent); n > 0 {
		return n
	}
	unit := amountUnit
	if unit <= 0 {
		unit = 2
	}
	mult := multiplier
	if mult <= 0 {
		mult = 1
	}
	if unit > 0 && mult > 0 && amount > 0 {
		if n := int(amount / (unit * float64(mult))); n > 0 {
			return n
		}
	}
	return 1
}

// IsFushiBaoziZeroBet 直选复式各位同一单码（如 7,7,7）时第三方计 0 注且无法下注。
func IsFushiBaoziZeroBet(meta RuleMeta, wireContent string) bool {
	if InferBetMode(meta) != "fushi" {
		return false
	}
	return isSSCFushiBaoziWire(wireContent)
}

// ResolveSolo 是否须 solo=true；注数超过平台单挑上限时须 solo=false。
func ResolveSolo(meta RuleMeta, wireContent string, betsNums int) bool {
	// 前后二：实测任意注数 solo=true →「单挑参数错误」，必须 solo=false。
	// 前后四：实测须 solo=true（与前后二相反）；白名单优先于注数上限。
	if guajiGroupRequiresSoloFalse(meta) {
		return false
	}
	if guajiGroupRequiresSoloTrue(meta) {
		return true
	}
	if betsNums > guajiSoloMaxBets {
		return false
	}
	// SSC 前二/后二组选复式：实测任意注数 solo=true →「单挑参数错误」（与直选复式单注不同）。
	if sscErxingZuxuanFsForcesSoloFalse(meta) {
		return false
	}
	// 前二/后二：实测多注仍带 solo=true → guaji 40000「单挑参数错误」；仅单注可 solo。
	if erxingDuoZhuForcesSoloFalse(meta) && betsNums > 1 {
		return false
	}
	if strings.TrimSpace(meta.PlayTemplate) != "" {
		return NeedsSoloForRule(meta, wireContent)
	}
	return NeedsSoloBet(wireContent)
}

// sscErxingZuxuanFsForcesSoloFalse 时时彩前二/后二组选复式须 solo=false。
func sscErxingZuxuanFsForcesSoloFalse(meta RuleMeta) bool {
	tpl := strings.TrimSpace(meta.PlayTemplate)
	if tpl != "" && tpl != "ssc_std" && tpl != "fast_ssc_std" {
		return false
	}
	mode := InferBetMode(meta)
	if mode != "zuxuan_fs" {
		return false
	}
	return erxingDuoZhuForcesSoloFalse(meta)
}

// erxingDuoZhuForcesSoloFalse 二星（前二/后二）多注不可 solo。
func erxingDuoZhuForcesSoloFalse(meta RuleMeta) bool {
	g := strings.TrimSpace(meta.Group)
	switch g {
	case "前二码", "后二码", "前二", "后二":
		return true
	case "前后二", "前后四":
		return false
	}
	text := meta.Group + " " + meta.TypeLabel + " " + meta.Label + " " + meta.TeamLabel + " " + meta.FullName
	if strings.Contains(text, "前后二") || strings.Contains(text, "前后四") {
		return false
	}
	return strings.Contains(text, "前二") || strings.Contains(text, "后二")
}

func classifyRule(meta RuleMeta) (ContentKind, int, int) {
	k := classifyRuleKind(meta)
	start, length := segmentRange(meta)
	return k, start, length
}

// segmentBetMultiplier 同一组选号覆盖的段数（前中后三=3，前后二/三/四=2，其余=1）。
func segmentBetMultiplier(meta RuleMeta) int {
	switch strings.TrimSpace(meta.Group) {
	case "前中后三":
		return 3
	case "前后三", "前后二", "前后四":
		return 2
	default:
		return 1
	}
}

func applySegmentMultiplier(meta RuleMeta, n int) int {
	if n <= 0 {
		return n
	}
	if m := segmentBetMultiplier(meta); m > 1 {
		return n * m
	}
	return n
}

// guajiGroupRequiresSoloFalse 第三方要求 solo=false 的区位组合玩法。
// 实测：前后二任意注数 solo=true →「单挑参数错误」。
// 注意：前后四 / 前中后三 / 前后三须 solo=true（勿列入此处）。
func guajiGroupRequiresSoloFalse(meta RuleMeta) bool {
	// rules/v2：g008=前后二（segment 偶发缺 guajiGroup 文案）
	if strings.TrimSpace(meta.TypeID) == "g008" {
		return true
	}
	g := strings.TrimSpace(meta.Group)
	if g == "前后二" {
		return true
	}
	text := meta.Group + " " + meta.TypeLabel + " " + meta.Label + " " + meta.TeamLabel + " " + meta.FullName
	if strings.Contains(text, "前中后三") || strings.Contains(text, "前后三") || strings.Contains(text, "前后四") {
		return false
	}
	return strings.Contains(text, "前后二")
}

// guajiGroupRequiresSoloTrue 第三方要求 solo=true 的区位组合玩法。
// 实测（2026-07）：前后四直选复式/单式 bets=段积×2 时 solo=false →「单挑参数错误」，solo=true 才过。
// 直选组合 / 组选24 等仍走 solo=false（勿一律 true）。
func guajiGroupRequiresSoloTrue(meta RuleMeta) bool {
	if !isQianhou4Meta(meta) {
		return false
	}
	mode := InferBetMode(meta)
	switch mode {
	case "fushi", "danshi", "zuxuan_ds":
		return true
	default:
		return false
	}
}

func isQianhou4Meta(meta RuleMeta) bool {
	if strings.TrimSpace(meta.TypeID) == "g014" {
		return true
	}
	if strings.TrimSpace(meta.Group) == "前后四" {
		return true
	}
	text := meta.Group + " " + meta.TypeLabel + " " + meta.Label + " " + meta.TeamLabel + " " + meta.FullName
	return strings.Contains(text, "前后四")
}

func classifyRuleKind(meta RuleMeta) ContentKind {
	text := meta.combinedText()
	label := meta.Label

	if meta.PlayTemplate == "lhc_std" {
		if strings.Contains(text, "肖") && !strings.Contains(text, "特码") {
			return KindLHCText
		}
		if strings.Contains(label, "波") || strings.Contains(label, "家") {
			return KindLHCText
		}
		return KindLHCNumbers
	}
	if meta.PlayTemplate == "pc28_std" {
		if strings.Contains(label, "大小") || strings.Contains(label, "单双") || strings.Contains(label, "龙虎") {
			return KindTextTokens
		}
		return KindNumberTokens
	}
	if meta.PlayTemplate == "k3_std" {
		if strings.Contains(label, "大小") || strings.Contains(label, "单双") {
			return KindTextTokens
		}
		return KindNumberTokens
	}
	if meta.PlayTemplate == "syxw_std" {
		mode := InferBetMode(meta)
		if mode == "dingwei" || meta.TypeID == "dingwei" {
			return KindSSCDingweiWire
		}
		if strings.Contains(label, "单式") {
			return KindSSCDanshiDigits
		}
		if strings.Contains(label, "复式") {
			_, length := segmentRange(meta)
			if length > 1 {
				return KindSSCPositionWire
			}
		}
		return KindNumberTokens
	}
	if meta.PlayTemplate == "pk10_std" {
		mode := InferBetMode(meta)
		if mode == "dingwei" || meta.TypeID == "dingwei" {
			return KindSSCDingweiWire
		}
		if strings.Contains(text, "龙虎") || strings.Contains(label, "龙虎") {
			return KindTextTokens
		}
		if strings.Contains(label, "单式") {
			return KindSSCDanshiDigits
		}
		if strings.Contains(label, "复式") {
			_, length := segmentRange(meta)
			if length > 1 {
				return KindSSCPositionWire
			}
		}
		return KindNumberTokens
	}

	// ssc_std / default
	mode := InferBetMode(meta)
	switch mode {
	case "dingwei":
		return KindSSCDingweiWire
	case "fushi":
		_, length := segmentRange(meta)
		if length > 1 {
			return KindSSCPositionWire
		}
	case "zuxuan_fs":
		return KindNumberTokens
	case "zuxuan_ds":
		return KindSSCDanshiDigits
	case "danshi":
		return KindSSCDanshiDigits
	case "hezhi", "kuadu", "zu3", "zu6", "zuhe", "baodan", "hunhe", "weishu", "budingwei":
		return KindNumberTokens
	case "teshu", "longhu", "longhuhe", "dxds", "daxiao", "danshuang", "zhuangxian":
		return KindTextTokens
	}

	if strings.Contains(label, "定位胆") || meta.Group == "一星" || strings.Contains(text, "定位胆") {
		return KindSSCDingweiWire
	}
	if strings.Contains(label, "直选单式") || strings.Contains(label, "单式") {
		return KindSSCDanshiDigits
	}
	if strings.Contains(label, "直选复式") || (strings.Contains(label, "复式") && strings.Contains(label, "直选")) {
		_, length := segmentRange(meta)
		if length > 1 {
			return KindSSCPositionWire
		}
	}
	_, length := segmentRange(meta)
	if length > 1 && (strings.Contains(meta.TeamLabel, "直选") || strings.Contains(meta.TeamLabel, "复式")) {
		return KindSSCPositionWire
	}
	return KindNumberTokens
}

func isFushiKind(kind ContentKind) bool {
	return kind == KindSSCPositionWire
}

func countHezhiBetNums(meta RuleMeta, wireContent string, segLen int) int {
	if segLen <= 0 {
		segLen = 3
	}
	picks := parseIntTokenList(wireContent)
	if len(picks) == 0 {
		return 1
	}
	zuxuan := strings.Contains(meta.Label, "组选")
	total := 0
	for _, sum := range picks {
		if zuxuan {
			total += countZuxuanSumCombinations(sum, segLen)
		} else {
			total += countOrderedSumCombinations(sum, segLen)
		}
	}
	if total <= 0 {
		return applySegmentMultiplier(meta, 1)
	}
	return applySegmentMultiplier(meta, total)
}

func countOrderedSpanCombinations(span, positions int) int {
	if positions <= 0 || span < 0 {
		return 0
	}
	count := 0
	digits := make([]int, positions)
	var dfs func(idx, min, max int)
	dfs = func(idx, min, max int) {
		if idx == positions {
			if max-min == span {
				count++
			}
			return
		}
		for d := 0; d <= 9; d++ {
			nmin, nmax := min, max
			if idx == 0 {
				nmin, nmax = d, d
			} else {
				if d < nmin {
					nmin = d
				}
				if d > nmax {
					nmax = d
				}
			}
			digits[idx] = d
			dfs(idx+1, nmin, nmax)
		}
	}
	dfs(0, 0, 0)
	return count
}

func countOrderedSumCombinations(targetSum, positions int) int {
	if positions <= 0 || targetSum < 0 {
		return 0
	}
	ways := make([][]int, positions+1)
	for i := range ways {
		ways[i] = make([]int, targetSum+1)
	}
	ways[0][0] = 1
	for pos := 0; pos < positions; pos++ {
		for sum := 0; sum <= targetSum; sum++ {
			n := ways[pos][sum]
			if n == 0 {
				continue
			}
			for d := 0; d <= 9 && sum+d <= targetSum; d++ {
				ways[pos+1][sum+d] += n
			}
		}
	}
	return ways[positions][targetSum]
}

// countBaodanBetNums 组选包胆：每胆覆盖组六 C(9,n-1) + 组三 9*(n-1) 注（n=segLen；三码实测 54）。
func countBaodanBetNums(pickCount, segLen int) int {
	if pickCount <= 0 {
		return 1
	}
	if segLen <= 0 {
		segLen = 3
	}
	return pickCount * baodanUnitsPerDan(segLen)
}

func baodanUnitsPerDan(segLen int) int {
	if segLen < 2 {
		return 1
	}
	if segLen == 2 {
		return 9
	}
	if segLen == 4 {
		return combin(9, 3)
	}
	zu6 := combin(9, segLen-1)
	zu3 := 9 * (segLen - 1)
	return zu6 + zu3
}

func combin(n, k int) int {
	if k < 0 || k > n {
		return 0
	}
	if k > n-k {
		k = n - k
	}
	r := 1
	for i := 0; i < k; i++ {
		r = r * (n - i) / (i + 1)
	}
	return r
}

func countZuxuanSumCombinations(targetSum, segLen int) int {
	if segLen == 4 {
		// 四星组选和值：3 个不同数字之和 × 4 种排列位置。
		return countZuxuanSumMultiset(targetSum, 3) * 4
	}
	return countZuxuanSumMultiset(targetSum, segLen)
}

func countZuxuanSumMultiset(targetSum, segLen int) int {
	if segLen <= 0 || targetSum < 0 {
		return 0
	}
	count := 0
	digits := make([]int, segLen)
	var dfs func(pos, minVal, sum int)
	dfs = func(pos, minVal, sum int) {
		if pos == segLen {
			if sum != targetSum {
				return
			}
			for i := 1; i < segLen; i++ {
				if digits[i] != digits[0] {
					count++
					return
				}
			}
			return
		}
		for d := minVal; d <= 9; d++ {
			if sum+d > targetSum {
				break
			}
			digits[pos] = d
			dfs(pos+1, d, sum+d)
		}
	}
	dfs(0, 0, 0)
	return count
}

func zu3PoolUnits(n int) int {
	if n < 2 {
		if n <= 0 {
			return 1
		}
		return n
	}
	return n * (n - 1)
}

func zu6PoolUnits(n int) int {
	if n < 3 {
		if n <= 0 {
			return 1
		}
		return n
	}
	return n * (n - 1) * (n - 2) / 6
}

// countSixingZu6BetNums 四星/前后四组选6：C(n,2)（实测 n=3→3, n=4→6, n=5→10）。
func countSixingZu6BetNums(poolSize int) int {
	if poolSize < 3 {
		if poolSize == 2 {
			return 1
		}
		return 0
	}
	return combin(poolSize, 2)
}

func countZuGroupBetNums(mode string, poolSize int) int {
	switch mode {
	case "zu24":
		if poolSize < 4 {
			return 0
		}
		return combin(poolSize, 4)
	case "zu12":
		if poolSize < 4 {
			return 0
		}
		return combin(poolSize, 2) * 2
	case "zu4":
		if poolSize < 2 {
			return 0
		}
		if poolSize == 2 {
			return 1
		}
		return 0
	case "zu120":
		if poolSize < 5 {
			return 0
		}
		return combin(poolSize, 5)
	case "zu60":
		if poolSize < 5 {
			return 0
		}
		return combin(poolSize, 2) * combin(poolSize-2, 3)
	case "zu30":
		if poolSize < 5 {
			return 0
		}
		return combin(poolSize, 2) * (poolSize - 2)
	case "zu20":
		if poolSize < 5 {
			return 0
		}
		return combin(poolSize, 5)
	case "zu10":
		if poolSize < 5 {
			return 0
		}
		return combin(poolSize, 5)
	case "zu5":
		if poolSize < 5 {
			return 0
		}
		return combin(poolSize, 5)
	default:
		return 0
	}
}

func countRenxuanPoolBetNums(meta RuleMeta, wireContent string) int {
	k := renxuanSegmentLen(meta)
	if k <= 0 {
		k = 2
	}
	mode := InferBetMode(meta)
	switch mode {
	case "danshi", "zuxuan_ds", "hunhe":
		return countRenxuanDanshiWire(wireContent, k)
	case "hezhi", "weishu":
		if n := countRenxuanHezhiWire(meta, wireContent, k); n > 0 {
			return n
		}
		return countHezhiBetNums(meta, wireContent, k)
	case "zuxuan_fs", "zu3", "zu6", "zu24", "zu12", "zu4":
		if n := countRenxuanZuxuanPickWire(meta, wireContent, k); n > 0 {
			return n
		}
	}
	if IsPositionWireContent(wireContent, sscPositionCount) {
		return countRenxuanFromSegments(strings.Split(wireContent, ","), k)
	}
	return countRenxuanPoolBetNumsFlat(meta, wireContent, k)
}

func countRenxuanFromSegments(segments []string, k int) int {
	pools := make([]int, 0, len(segments))
	for _, seg := range segments {
		seg = strings.TrimSpace(seg)
		if seg == "" {
			continue
		}
		pools = append(pools, len([]rune(seg)))
	}
	if len(pools) < k {
		return 0
	}
	total := 0
	var walk func(start, depth, prod int)
	walk = func(start, depth, prod int) {
		if depth == k {
			total += prod
			return
		}
		for i := start; i <= len(pools)-k+depth; i++ {
			walk(i+1, depth+1, prod*pools[i])
		}
	}
	walk(0, 0, 1)
	return total
}

func isRenxuanMeta(meta RuleMeta) bool {
	// LHC 生肖连复用 type_id=g011，与时时彩「任选」不同玩法域。
	if meta.PlayTemplate == "lhc_std" {
		return false
	}
	if meta.Group == "任选" {
		return true
	}
	switch strings.TrimSpace(meta.TypeID) {
	case "g011", "renxuan":
		return true
	default:
		return false
	}
}

func formatRenxuanWire(groupContent string) string {
	lines := splitPositionLines(groupContent)
	if len(lines) >= sscPositionCount {
		segments := make([]string, sscPositionCount)
		for i := 0; i < sscPositionCount; i++ {
			segments[i] = normalizePickDigits(lines[i])
		}
		return strings.Join(segments, ",")
	}
	if parts := splitCommaParts(groupContent); len(parts) == sscPositionCount {
		return strings.Join(parts, ",")
	}
	segments := make([]string, sscPositionCount)
	for i, t := range splitPickTokens(groupContent) {
		if i >= sscPositionCount {
			break
		}
		segments[i] = t
	}
	return strings.Join(segments, ",")
}

func parseIntTokenList(raw string) []int {
	raw = strings.NewReplacer("\n", ",", "，", ",", " ", ",").Replace(raw)
	parts := strings.Split(raw, ",")
	out := make([]int, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		n := 0
		for _, r := range p {
			if r < '0' || r > '9' {
				n = -1
				break
			}
			n = n*10 + int(r-'0')
		}
		if n >= 0 {
			out = append(out, n)
		}
	}
	return out
}

func splitPickDigits(content string) []string {
	return splitPickTokens(content)
}

func parseZuhePairs(content string) [][2]string {
	raw := strings.NewReplacer("\n", ",", "，", ",", " ", ",").Replace(content)
	parts := strings.Split(raw, ",")
	out := make([][2]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if len(p) == 2 && isAllDigits(p) {
			out = append(out, [2]string{string(p[0]), string(p[1])})
		}
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

func segmentRange(meta RuleMeta) (start, length int) {
	group := meta.Group
	typeLabel := meta.TypeLabel
	text := group + " " + typeLabel + " " + meta.FullName

	switch meta.PlayTemplate {
	case "syxw_std":
		return syxwSegmentRange(typeLabel, meta.TypeID)
	case "pk10_std":
		return pk10SegmentRange(meta.TypeLabel, meta.TypeID, meta.Label)
	}

	// guaji Group 精确匹配优先（避免「前后三」误匹配「前三」、「前中后三」误匹配「后三」）。
	switch group {
	case "前中后三", "前后三":
		return 0, 3
	case "前后二":
		return 0, 2
	case "前后四":
		return 0, 4
	case "四星":
		return 0, 4
	case "五星":
		return 0, 5
	case "任选":
		return 0, renxuanSegmentLen(meta)
	}

	switch {
	case strings.Contains(group, "前中后三") || strings.Contains(text, "前中后三"):
		return 0, 3
	case strings.Contains(group, "前后四") || strings.Contains(text, "前后四"):
		return 0, 4
	case strings.Contains(group, "前后三") || strings.Contains(text, "前后三"):
		return 0, 3
	case strings.Contains(group, "前后二") || strings.Contains(text, "前后二"):
		return 0, 2
	case strings.Contains(group, "五星") || strings.Contains(text, "五星"):
		return 0, 5
	case strings.Contains(group, "四星") || strings.Contains(text, "四星"):
		return 0, 4
	case strings.Contains(group, "前三") || strings.Contains(text, "前三"):
		return 0, 3
	case strings.Contains(group, "中三") || strings.Contains(text, "中三"):
		return 1, 3
	case strings.Contains(group, "后三") || strings.Contains(text, "后三"):
		return 2, 3
	case strings.Contains(group, "前二") || strings.Contains(text, "前二"):
		return 0, 2
	case strings.Contains(group, "后二") || strings.Contains(text, "后二"):
		return 3, 2
	case strings.Contains(text, "后四"):
		return 1, 4
	case strings.Contains(text, "前四"):
		return 0, 4
	case strings.Contains(group, "前一") || strings.Contains(text, "冠军") || strings.Contains(text, "前一"):
		return 0, 1
	case strings.Contains(group, "前二名") || strings.Contains(text, "前二"):
		return 0, 2
	case meta.Group == "一星" || strings.Contains(text, "定位胆"):
		return dingweiPositionIndex(meta), 1
	default:
		return legacyTypeSegmentRange(meta.TypeID)
	}
}

func legacyTypeSegmentRange(typeID string) (int, int) {
	switch strings.TrimSpace(typeID) {
	case "g001":
		return 0, 3
	case "g002":
		return 1, 3
	case "g003":
		return 2, 3
	case "g004":
		return 0, 2
	case "g005":
		return 3, 2
	case "g007", "g012":
		return 0, 3
	case "g008":
		return 0, 2
	case "g013", "g014":
		return 0, 4
	case "g015":
		return 0, 5
	default:
		return 0, 1
	}
}

func renxuanSegmentLen(meta RuleMeta) int {
	text := meta.TeamLabel + meta.FullName + meta.Label + meta.SubID
	switch {
	case strings.Contains(text, "任选四"), strings.Contains(text, "任四"), strings.Contains(text, "ren4"):
		return 4
	case strings.Contains(text, "任选三"), strings.Contains(text, "任三"), strings.Contains(text, "ren3"):
		return 3
	}
	if id, err := strconv.Atoi(strings.TrimSpace(meta.RuleID)); err == nil && meta.TypeID == "g011" {
		switch {
		case id >= 141 && id <= 145:
			return 4
		case id >= 80 && id <= 88:
			return 3
		case id >= 74 && id <= 79:
			return 2
		}
	}
	return 2
}

func syxwSegmentRange(typeLabel, typeID string) (int, int) {
	switch strings.TrimSpace(typeID) {
	case "g001":
		return 0, 3
	case "g002":
		return 0, 2
	case "g003":
		return 0, 1
	}
	text := typeLabel + typeID
	if strings.Contains(text, "前三") {
		return 0, 3
	}
	if strings.Contains(text, "前二") {
		return 0, 2
	}
	return 0, 1
}

func pk10SegmentRange(typeLabel, typeID, label string) (int, int) {
	text := typeLabel + typeID + label
	if strings.Contains(text, "冠亚") {
		return 0, 2
	}
	if strings.Contains(text, "前五") {
		return 0, 5
	}
	if strings.Contains(text, "前四") {
		return 0, 4
	}
	if strings.Contains(text, "前三") {
		return 0, 3
	}
	if strings.Contains(text, "前二") {
		return 0, 2
	}
	if strings.Contains(text, "前一") || strings.Contains(text, "冠军") {
		return 0, 1
	}
	switch strings.TrimSpace(typeID) {
	case "g001":
		return 0, 2
	case "g003":
		return 0, 1
	case "g004":
		return 0, 2
	case "g005":
		return 0, 3
	case "g006":
		return 0, 4
	case "g007":
		return 0, 5
	}
	return 0, 1
}

func dingweiPositionIndex(meta RuleMeta) int {
	text := meta.FullName + meta.Label + meta.SubID
	for idx, r := range []rune{'万', '千', '百', '十', '个'} {
		if strings.ContainsRune(text, r) {
			return idx
		}
	}
	return 0
}

func formatSSCFushiContent(start, length int, groupContent string) string {
	if length <= 0 {
		length = 1
	}
	lines := splitPositionLines(groupContent)
	parts := make([]string, length)
	for i := 0; i < length; i++ {
		line := ""
		if i < len(lines) {
			line = lines[i]
		} else if len(lines) == 1 {
			line = lines[0]
		}
		parts[i] = normalizePickDigits(line)
	}
	return strings.Join(parts, ",")
}

// formatSSCZuheContent 三/四/五星直选组合：wire 与直选复式同形。
// 多行按位；无换行时「1,2,3」表示三位各 1 码（勿压成 123,123,123）。
func formatSSCZuheContent(start, length int, groupContent string) string {
	if length <= 0 {
		length = 1
	}
	if strings.ContainsAny(groupContent, "\n\r") {
		return formatSSCFushiContent(start, length, groupContent)
	}
	tokens := splitPickTokens(groupContent)
	if len(tokens) == length {
		parts := make([]string, length)
		for i := 0; i < length; i++ {
			parts[i] = normalizePickDigits(tokens[i])
		}
		return strings.Join(parts, ",")
	}
	return formatSSCFushiContent(start, length, groupContent)
}

func countSSCFushiProduct(wireContent string) int {
	parts := splitCommaParts(wireContent)
	if len(parts) == 0 {
		return 0
	}
	product := 1
	for _, p := range parts {
		if p == "" {
			continue
		}
		product *= len([]rune(p))
	}
	if product <= 0 {
		return 0
	}
	return product
}

// isSSCFushiBaoziWire 直选复式各位均为同一单码（如 7,7,7 / 1,1）。
// 第三方网页预览计 0 注且无法下注，本平台对齐。
func isSSCFushiBaoziWire(wireContent string) bool {
	parts := splitCommaParts(wireContent)
	nonEmpty := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			nonEmpty = append(nonEmpty, p)
		}
	}
	if len(nonEmpty) < 2 {
		return false
	}
	first := nonEmpty[0]
	if len([]rune(first)) != 1 {
		return false
	}
	for _, p := range nonEmpty[1:] {
		if p != first {
			return false
		}
	}
	return true
}

func formatSSCPositionWire(start, length int, groupContent string) string {
	return formatPositionWire("ssc_std", start, length, groupContent)
}

func formatTextTokens(groupContent string) string {
	return strings.ReplaceAll(strings.TrimSpace(groupContent), "，", ",")
}

func formatCommaPickDigits(groupContent string) string {
	tokens := splitPickTokens(groupContent)
	if len(tokens) == 0 {
		return strings.TrimSpace(groupContent)
	}
	return strings.Join(tokens, ",")
}

// formatBudingweiContent 不定位选号：逗号分隔；一码最多 2 个号（第三方限制）。
func formatBudingweiContent(meta RuleMeta, groupContent string) string {
	tokens := uniqueStringsPreserve(splitPickTokens(groupContent))
	if len(tokens) == 0 {
		return strings.TrimSpace(groupContent)
	}
	if budingweiPickCount(meta) == 1 && len(tokens) > 2 {
		tokens = tokens[:2]
	}
	return strings.Join(tokens, ",")
}

func formatZuheContent(groupContent string) string {
	groupContent = strings.TrimSpace(groupContent)
	if groupContent == "" {
		return groupContent
	}
	pairs := parseZuhePairs(groupContent)
	if len(pairs) == 1 {
		p := pairs[0]
		return p[0] + p[1]
	}
	return formatCommaPickDigits(groupContent)
}

func formatLHCPickDigits(groupContent string) string {
	tokens := splitPickTokens(groupContent)
	if len(tokens) == 0 {
		return strings.TrimSpace(groupContent)
	}
	out := make([]string, 0, len(tokens))
	for _, t := range tokens {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		if len(t) == 1 {
			t = "0" + t
		}
		out = append(out, t)
	}
	return strings.Join(out, ",")
}

func splitCommaParts(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// uniqueStringsPreserve 保序去重（单式注数对齐第三方）。
func uniqueStringsPreserve(items []string) []string {
	if len(items) <= 1 {
		return items
	}
	seen := make(map[string]struct{}, len(items))
	out := make([]string, 0, len(items))
	for _, raw := range items {
		t := strings.TrimSpace(raw)
		if t == "" {
			continue
		}
		if _, ok := seen[t]; ok {
			continue
		}
		seen[t] = struct{}{}
		out = append(out, t)
	}
	return out
}

// countSSCHunheBetNums 混合组选 / 组选单式注数（对齐第三方）：
// 排除对子/豹子（各位相同），按组选形态去重（123 与 321、12 与 21 计 1 注）。
// 例：123,321,232,222,333,444,542 → 3；11,12,22,13 → 2
func countSSCHunheBetNums(wireContent string, segLen int) int {
	if segLen <= 0 {
		segLen = 3
	}
	parts := splitCommaParts(wireContent)
	if len(parts) == 0 {
		digits := normalizePickDigits(wireContent)
		if len(digits) == segLen && !isBaoziDigits(digits) {
			return 1
		}
		return 0
	}
	seen := make(map[string]struct{}, len(parts))
	n := 0
	for _, p := range parts {
		digits := normalizePickDigits(p)
		if len(digits) != segLen {
			continue
		}
		if isBaoziDigits(digits) {
			continue
		}
		key := sortDigitRunes(digits)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		n++
	}
	return n
}

func isBaoziDigits(s string) bool {
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

func sortDigitRunes(s string) string {
	runes := []rune(s)
	sort.Slice(runes, func(i, j int) bool { return runes[i] < runes[j] })
	return string(runes)
}

const guajiSoloMaxBets = 28

func sampleZuxuanFushiContent(meta RuleMeta) string {
	_, segLen := segmentRange(meta)
	if segLen <= 0 {
		segLen = 2
	}
	if usesPaddedDigits(meta.PlayTemplate) {
		parts := make([]string, segLen)
		for i := range parts {
			parts[i] = string(rune('1' + i))
		}
		return strings.Join(parts, ",")
	}
	if segLen <= 2 {
		return "1,2"
	}
	return "1,2"
}

func k3SantongMeta(meta RuleMeta) bool {
	return meta.PlayTemplate == "k3_std" && strings.TrimSpace(meta.Label) == "三同号"
}

func k3PairPickNeedsSolo(meta RuleMeta) bool {
	label := strings.TrimSpace(meta.Label)
	return strings.Contains(label, "二同号单选") || strings.Contains(label, "二不同")
}

func countK3PairPickBetNums(meta RuleMeta, wireContent string) int {
	if !k3PairPickNeedsSolo(meta) {
		return 0
	}
	if strings.TrimSpace(wireContent) == "" {
		return 0
	}
	return 1
}

func sampleBudingweiContent(meta RuleMeta) string {
	need := budingweiPickCount(meta)
	if need <= 1 {
		if meta.PlayTemplate == "syxw_std" {
			return "01"
		}
		return "3"
	}
	poolSize := budingweiMinPoolSize(meta)
	parts := make([]string, poolSize)
	for i := range parts {
		parts[i] = string(rune('1' + i))
	}
	return strings.Join(parts, ",")
}

func budingweiPickCount(meta RuleMeta) int {
	text := meta.Label + meta.FullName + meta.Group
	switch {
	case strings.Contains(text, "三码"):
		return 3
	case strings.Contains(text, "二码"):
		return 2
	default:
		return 1
	}
}

func budingweiMinPoolSize(meta RuleMeta) int {
	need := budingweiPickCount(meta)
	if strings.Contains(meta.Label, "五星") && need >= 2 {
		return 4
	}
	if need >= 3 {
		return need + 1
	}
	return need
}

func budingweiNeedsSolo(meta RuleMeta) bool {
	// 实测一码/二码不定位：solo=true →「单挑参数错误」，须 solo=false。
	_ = meta
	return false
}

func countBudingweiBetNums(meta RuleMeta, wireContent string) int {
	need := budingweiPickCount(meta)
	parts := splitCommaParts(wireContent)
	if strings.Contains(meta.Label, "五星") && need >= 2 {
		if len(parts) < 4 {
			return 0
		}
		return combin(len(parts), need)
	}
	if need == 1 {
		// 一码：选几个号计几注；第三方最多允许 2 个号（超过报「不可超过两位」）
		n := len(parts)
		if n == 0 {
			if len(normalizePickDigits(wireContent)) > 0 {
				return 1
			}
			return 0
		}
		if n > 2 {
			n = 2
		}
		return n
	}
	if need == 2 && len(parts) == 2 && !strings.Contains(meta.Label, "五星") {
		return 1
	}
	picks := splitPickDigits(wireContent)
	if len(picks) < need {
		return 0
	}
	return combin(len(picks), need)
}

func countZu12BetNums(wireContent string) int {
	parts := splitCommaParts(wireContent)
	n := 0
	for _, p := range parts {
		if len(normalizePickDigits(p)) >= 2 {
			n++
		}
	}
	return n
}

func countZuxuanFushiBetNums(poolSize, segLen int) int {
	if poolSize <= 0 {
		return 1
	}
	if segLen == 2 {
		return poolSize * (poolSize - 1) / 2
	}
	if segLen == 3 {
		if poolSize < 2 {
			return poolSize
		}
		return zu3PoolUnits(poolSize) + zu6PoolUnits(poolSize)
	}
	if poolSize < 2 {
		return poolSize
	}
	return zu3PoolUnits(poolSize)
}
