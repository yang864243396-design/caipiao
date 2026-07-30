package schemes

import (
	"fmt"
	"strconv"
	"strings"
)

// MaxBetUnitsForScheme 该玩法单组注数上限；0 表示第三方未设上限（或本端尚未定义）。
//
// 覆盖直选复式 / 和值 / 跨度 / 尾数 / 直选组合等（对齐前端 betPayload.ts）。
func MaxBetUnitsForScheme(kind string, config []byte) int {
	if len(config) == 0 {
		return 0
	}
	return maxBetUnitsForPlay(parseSchemeConfig(kind, config, 0, 0).Play)
}

// ValidateSchemeBetContent 校验投注内容是否落在该玩法的合法投注空间内。
//
// maxUnits <= 0 时取该玩法自身的上限（MaxBetUnitsForScheme）；仍为 0 则不查上限。
// 判不准的情形一律不报——审计工具的误报会直接摧毁可信度，宁可漏报。
func ValidateSchemeBetContent(kind string, config []byte, content string, maxUnits int) []Violation {
	u, ok := UniverseForScheme(kind, config)
	if !ok {
		return nil
	}
	rule := parseSchemeConfig(kind, config, 0, 0).Play
	if maxUnits <= 0 {
		maxUnits = maxBetUnitsForPlay(rule)
	}
	if strings.TrimSpace(content) == "" {
		return []Violation{{Code: ViolationEmptyContent, Detail: "投注内容为空"}}
	}
	content = stripPositionLabelPrefix(rule, content)
	// 直选单式：按位号池（千\n百\n十）先展开为整注串再校验，与下注链路 normalizeZhixuanDanshiContent 一致。
	// 否则开某投某/冷热的「1,2\n3,4\n5,6」会被拆成 6 个非法「单式组合」。
	content = normalizeZhixuanDanshiContent(rule, content)

	var out []Violation
	out = append(out, validateTokens(u, rule, content)...)

	if isFushiBaoziContent(rule, content) {
		out = append(out, Violation{
			Code:   ViolationZeroUnits,
			Detail: "直选复式各位选同一个号（豹子），第三方注数为 0、无法下单",
		})
		return out
	}

	units, known := countBetUnits(u, rule, content)
	switch {
	case known && units <= 0:
		out = append(out, Violation{Code: ViolationZeroUnits, Detail: "注数为 0"})
	case known && maxUnits > 0 && units > maxUnits:
		out = append(out, Violation{
			Code:   ViolationUnitsOverLimit,
			Detail: fmt.Sprintf("注数 %d 超过上限 %d", units, maxUnits),
		})
	}
	return out
}

// CountBetUnitsForScheme 按投注内容推算注数，口径与第三方 bets_nums 一致。
//
// 第二个返回值 false 表示该玩法算不出或口径不可比：属性玩法（和值/跨度/大小单双）
// 第三方按底层号码组合计数，本端只知道选了几个选项，两者不是一个东西，不能对账。
func CountBetUnitsForScheme(kind string, config []byte, content string) (int, bool) {
	u, ok := UniverseForScheme(kind, config)
	if !ok {
		return 0, false
	}
	switch u.Kind {
	case UniversePerPosition, UniverseCombos:
	default:
		return 0, false
	}
	rule := parseSchemeConfig(kind, config, 0, 0).Play
	return countBetUnits(u, rule, stripPositionLabelPrefix(rule, content))
}

// stripPositionLabelPrefix 剥掉任选玩法的位名前缀，只留号码部分。
//
// 任选的内容形如「千,十|12,34」或「千,十\n12,34」——前半段选的是位、不是号。
// 不剥掉会把「千」「十」当成号码报越界，正常的任选方案全数误伤。
// 复用下注链路的 parseRenxuanPosPicksContent，保证与验奖同一套解析。
func stripPositionLabelPrefix(rule playRule, content string) string {
	if !isRenxuanPlayType(rule.PlayTypeID) {
		return content
	}
	n := renPickCount(rule.CatalogSubID)
	if _, picks, ok := parseRenxuanPosPicksContent(content, n); ok {
		return picks
	}
	return content
}

// UniverseKindForScheme 返回该玩法的投注内容形态，供报告分类。
func UniverseKindForScheme(kind string, config []byte) string {
	u, ok := UniverseForScheme(kind, config)
	if !ok {
		return ""
	}
	return u.Kind
}

func validateTokens(u PlayUniverse, rule playRule, content string) []Violation {
	switch u.Kind {
	case UniverseAttribute:
		legal := make(map[string]struct{}, len(u.Tokens))
		for _, t := range u.Tokens {
			legal[canonAttrToken(t)] = struct{}{}
		}
		var bad []string
		for _, tok := range splitContentTokens(content) {
			if _, ok := legal[canonAttrToken(tok)]; !ok {
				bad = append(bad, tok)
			}
		}
		return outOfPoolViolation(bad, u.Tokens)

	case UniverseCombos:
		// 复用下注链路的单式解析：无法解析的组合即为非法
		parts := splitContentTokens(content)
		valid := parseSegmentTokensForRule(rule, content, u.ComboLen)
		if n := len(parts) - len(valid); n > 0 {
			return []Violation{{
				Code:   ViolationTokenOutOfPool,
				Detail: fmt.Sprintf("%d 个单式组合不合法（应为 %d 位、且每位落在号池内）", n, u.ComboLen),
			}}
		}
		return nil

	default: // perPosition / tokenList 都是号池 token
		min, max := ruleNumberPool(rule)
		var bad []string
		for _, tok := range splitContentTokens(content) {
			if _, ok := normalizePoolToken(tok, min, max); ok {
				continue
			}
			// 同一玩法的内容既可能是按位号池，也可能已展开成整注单式
			// （冷热出号在下注前会走 normalizeZhixuanDanshiContent）。
			// 逐位可解的定长组合按合法处理，避免把 "104,904" 报成越界。
			if isPoolCombo(tok, rule.SegmentLen, min, max) {
				continue
			}
			bad = append(bad, tok)
		}
		return outOfPoolViolation(bad, u.Tokens)
	}
}

// isPoolCombo token 是否为一注定长组合，且每位都落在号池内。
// 仅在单字符号池（0-9 等）下成立；多字符号池无法无歧义切分，一律不认。
func isPoolCombo(tok string, segLen, min, max int) bool {
	if segLen < 2 || min < 0 || max > 9 {
		return false
	}
	r := []rune(tok)
	if len(r) != segLen {
		return false
	}
	for _, ch := range r {
		if _, ok := normalizePoolToken(string(ch), min, max); !ok {
			return false
		}
	}
	return true
}

// canonAttrToken 属性选项的比较键：纯数字按数值归一，其余原样。
// 和值/跨度这类数值属性的号池是 0..27，而内容常写成补零的 04、07——
// 数值相同却字符串不等，直接比会把正常方案全报成越界。
func canonAttrToken(tok string) string {
	tok = strings.TrimSpace(tok)
	if n, err := strconv.Atoi(tok); err == nil {
		return strconv.Itoa(n)
	}
	return tok
}

func outOfPoolViolation(bad, pool []string) []Violation {
	if len(bad) == 0 {
		return nil
	}
	return []Violation{{
		Code:   ViolationTokenOutOfPool,
		Detail: fmt.Sprintf("号码 %s 不在合法号池 [%s] 内", strings.Join(dedupStrings(bad), ","), joinPoolPreview(pool)),
	}}
}

func countBetUnits(u PlayUniverse, rule playRule, content string) (int, bool) {
	switch u.Kind {
	case UniverseAttribute:
		return len(splitContentTokens(content)), true

	case UniverseCombos:
		return len(parseSegmentTokensForRule(rule, content, u.ComboLen)), true

	case UniversePerPosition:
		units := 1
		filled := 0
		for _, line := range strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n") {
			n := len(splitContentTokens(line))
			if n == 0 {
				continue // 定位胆用空行表示该位未选
			}
			filled++
			units *= n
		}
		if filled == 0 {
			return 0, true
		}
		// 定位胆各位独立成注，不是笛卡尔乘积
		if strings.EqualFold(strings.TrimSpace(rule.BetMode), "dingwei") {
			units = 0
			for _, line := range strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n") {
				units += len(splitContentTokens(line))
			}
		}
		return units, true

	default:
		// 组选/不定位/包胆的注数是组合数，需子玩法选码数，这里不猜
		return 0, false
	}
}

// isFushiBaoziContent 直选复式各位只选同一个号 → 第三方注数为 0。
// 必须与 guajibet.IsFushiBaoziZeroBet 保持一致，见 play_universe_test.go 的交叉校验。
func isFushiBaoziContent(rule playRule, content string) bool {
	switch strings.ToLower(strings.TrimSpace(rule.BetMode)) {
	case "fushi", "zhixuan_fs":
	default:
		return false
	}
	var picks []string
	for _, line := range strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n") {
		toks := splitContentTokens(line)
		if len(toks) == 0 {
			continue
		}
		if len(toks) > 1 {
			return false
		}
		picks = append(picks, toks[0])
	}
	if len(picks) < 2 || len([]rune(picks[0])) != 1 {
		return false
	}
	for _, p := range picks[1:] {
		if p != picks[0] {
			return false
		}
	}
	return true
}

func splitContentTokens(raw string) []string {
	return strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == '，' || r == ' ' || r == '\t' || r == '\n' || r == '\r' || r == '|'
	})
}

func dedupStrings(in []string) []string {
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, s := range in {
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}

func joinPoolPreview(pool []string) string {
	if len(pool) <= 12 {
		return strings.Join(pool, ",")
	}
	return strings.Join(pool[:6], ",") + ",…," + strings.Join(pool[len(pool)-3:], ",")
}
