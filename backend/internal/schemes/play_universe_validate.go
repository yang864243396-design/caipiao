package schemes

import (
	"fmt"
	"strconv"
	"strings"

	"caipiao/backend/internal/guajibet"
)

// MaxBetUnitsForScheme 该玩法单组注数上限；0 表示第三方未设上限（或本端尚未定义）。
//
// 覆盖直选复式/单式 / 和值 / 跨度 / 尾数 / 直选组合等（对齐前端 betPayload.ts）。
func MaxBetUnitsForScheme(kind string, config []byte) int {
	if len(config) == 0 {
		return 0
	}
	return maxBetUnitsForPlay(parseSchemeConfig(kind, config, 0, 0).Play)
}

// isBaodanPlayRule 组选包胆（含中三组选包胆）：每组仅允许 1 个胆码。
func isBaodanPlayRule(rule playRule) bool {
	bm := strings.ToLower(strings.TrimSpace(rule.BetMode))
	if bm == "baodan" {
		return true
	}
	sub := strings.ToLower(rule.SubPlayID + " " + rule.CatalogSubID)
	return strings.Contains(sub, "baodan") || strings.Contains(sub, "_bd")
}

// isYimaBudingweiPlayRule 一码不定位：第三方最多 2 个号（超过 →「投注数字不可超过两位数」）。
func isYimaBudingweiPlayRule(rule playRule) bool {
	bm := strings.ToLower(strings.TrimSpace(rule.BetMode))
	sub := strings.ToLower(rule.SubPlayID + " " + rule.CatalogSubID)
	isBdw := bm == "budingwei" || strings.Contains(sub, "budingwei")
	if !isBdw {
		return false
	}
	// 二码/三码：need≥2；一码（含目录 id 如 113）need=1
	return budingweiNeedCount(rule.CatalogSubID) <= 1 && budingweiNeedCount(rule.SubPlayID) <= 1
}

// budingweiMinPoolForRule 不定位最少选号数；非不定位或一码返回 0。
// 二码=2；三码=3；五星二/三码=4（对齐 guajibet.budingweiMinPoolSize）。
func budingweiMinPoolForRule(rule playRule) int {
	bm := strings.ToLower(strings.TrimSpace(rule.BetMode))
	sub := strings.ToLower(rule.SubPlayID + " " + rule.CatalogSubID)
	if bm != "budingwei" && !strings.Contains(sub, "budingwei") {
		return 0
	}
	need := budingweiNeedCount(rule.CatalogSubID)
	if n := budingweiNeedCount(rule.SubPlayID); n > need {
		need = n
	}
	if need <= 1 {
		return 0
	}
	sid := strings.TrimSpace(rule.CatalogSubID)
	if sid == "" {
		sid = strings.TrimSpace(rule.SubPlayID)
	}
	if sid == "151" || sid == "152" || strings.Contains(sub, "wuxing") || strings.Contains(sub, "五星") {
		return 4
	}
	return need
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
	// 任选选位注数依赖位名前缀（C(选位数,k)×内层）；剥位前先算好
	renDanshiUnits, renDanshiKnown := 0, false
	if isRenxuanNeedsPositionRule(rule) {
		renDanshiUnits = countRenxuanNeedsPositionBetUnits(rule, content)
		renDanshiKnown = true
	}
	// 先 normalize 再剥位：任选组选单式 Format 会保留/补位名前缀（如「万,个\n12」→「万个|12」）。
	// 若先剥位再 Format，会丢选位并误补默认「千个|」，把合法「12,34」校验成非法组合。
	// 直选单式：按位号池（千\n百\n十）先展开为整注串再校验，与下注链路一致。
	content = normalizeZhixuanDanshiContent(rule, content)
	content = stripPositionLabelPrefix(rule, content)

	var out []Violation
	// 五星趣味：0–9 数字池（勿当豹子/对子/顺子文字特殊号）；一帆风顺最多 2 码
	if isWuxingQuweiDigitPlay(rule) {
		out = append(out, validateWuxingQuweiDigitContent(rule, content)...)
		units := len(parseQuweiDigitTokens(content))
		if renDanshiKnown {
			units = renDanshiUnits
		}
		if len(out) == 0 && units <= 0 {
			detail := wuxingQuweiFormatDetail
			if isWuxingYifanPlay(rule) {
				detail = wuxingYifanFormatDetail
			}
			out = append(out, Violation{Code: ViolationZeroUnits, Detail: detail})
		}
		if maxUnits > 0 && units > maxUnits {
			out = append(out, Violation{
				Code:   ViolationUnitsOverLimit,
				Detail: errMaxBetUnitsExceeded(maxUnits).Error(),
			})
		}
		return out
	}
	// 组选12：双区「二重,单号」，勿走扁选 TokenList（「12」「34」会被当成越界）
	if isZu12PlayRule(rule) {
		out = append(out, validateZu12DualZone(content)...)
		units := countZu12DualZoneBetUnits(content)
		if renDanshiKnown {
			units = renDanshiUnits
		}
		if len(out) == 0 && units <= 0 {
			out = append(out, Violation{Code: ViolationZeroUnits, Detail: zu12FormatDetail})
		}
		if maxUnits > 0 && units > maxUnits {
			out = append(out, Violation{
				Code:   ViolationUnitsOverLimit,
				Detail: errMaxBetUnitsExceeded(maxUnits).Error(),
			})
		}
		return out
	}
	// 组选4：双区「三重,单号」，勿走扁选 TokenList
	if isZu4PlayRule(rule) {
		out = append(out, validateZu4DualZone(content)...)
		units := countZu4DualZoneBetUnits(content)
		if renDanshiKnown {
			units = renDanshiUnits
		}
		if len(out) == 0 && units <= 0 {
			out = append(out, Violation{Code: ViolationZeroUnits, Detail: zu4FormatDetail})
		}
		if maxUnits > 0 && units > maxUnits {
			out = append(out, Violation{
				Code:   ViolationUnitsOverLimit,
				Detail: errMaxBetUnitsExceeded(maxUnits).Error(),
			})
		}
		return out
	}
	// 五星组选60/30/20/10/5：双区内容，勿走扁选 TokenList（zu60「1,234」展开成 4 码会误报「号码池至少选择 5」）
	if isZu60PlayRule(rule) {
		out = append(out, validateZu60DualZone(content)...)
		units := countZu60DualZoneBetUnits(content)
		if renDanshiKnown {
			units = renDanshiUnits
		}
		if len(out) == 0 && units <= 0 {
			out = append(out, Violation{Code: ViolationZeroUnits, Detail: zu60FormatDetail})
		}
		if maxUnits > 0 && units > maxUnits {
			out = append(out, Violation{
				Code:   ViolationUnitsOverLimit,
				Detail: errMaxBetUnitsExceeded(maxUnits).Error(),
			})
		}
		return out
	}
	if isZu30PlayRule(rule) {
		out = append(out, validateZu30DualZone(content)...)
		units := countZu30DualZoneBetUnits(content)
		if renDanshiKnown {
			units = renDanshiUnits
		}
		if len(out) == 0 && units <= 0 {
			out = append(out, Violation{Code: ViolationZeroUnits, Detail: zu30FormatDetail})
		}
		if maxUnits > 0 && units > maxUnits {
			out = append(out, Violation{
				Code:   ViolationUnitsOverLimit,
				Detail: errMaxBetUnitsExceeded(maxUnits).Error(),
			})
		}
		return out
	}
	if isZu20PlayRule(rule) {
		out = append(out, validateZu20DualZone(content)...)
		units := countZu20DualZoneBetUnits(content)
		if renDanshiKnown {
			units = renDanshiUnits
		}
		if len(out) == 0 && units <= 0 {
			out = append(out, Violation{Code: ViolationZeroUnits, Detail: zu20FormatDetail})
		}
		if maxUnits > 0 && units > maxUnits {
			out = append(out, Violation{
				Code:   ViolationUnitsOverLimit,
				Detail: errMaxBetUnitsExceeded(maxUnits).Error(),
			})
		}
		return out
	}
	if isZu10PlayRule(rule) || isZu5PlayRule(rule) {
		formatDetail := zu10FormatDetail
		overlapDetail := zu10OverlapDetail
		if isZu5PlayRule(rule) {
			formatDetail = zu5FormatDetail
			overlapDetail = zu5OverlapDetail
		}
		out = append(out, validateZuPairDualZone(content, formatDetail, overlapDetail)...)
		units := countZuPairDualZoneBetUnits(content)
		if renDanshiKnown {
			units = renDanshiUnits
		}
		if len(out) == 0 && units <= 0 {
			out = append(out, Violation{Code: ViolationZeroUnits, Detail: formatDetail})
		}
		if maxUnits > 0 && units > maxUnits {
			out = append(out, Violation{
				Code:   ViolationUnitsOverLimit,
				Detail: errMaxBetUnitsExceeded(maxUnits).Error(),
			})
		}
		return out
	}
	// 特码/正特：号码|属性|波色（可只选大/小/单等属性，勿按 1–49 号池拒掉）
	if isLHCTemaAttrPlayRule(rule) {
		out = append(out, validateLHCTemaBetContent(content)...)
		nums, attrs, waves := guajibet.ParseLHCTemaParts(content)
		units := len(nums) + len(attrs) + len(waves)
		if renDanshiKnown {
			units = renDanshiUnits
		}
		if len(out) == 0 && units <= 0 {
			out = append(out, Violation{Code: ViolationZeroUnits, Detail: "请选择属性或输入 1–49 号码"})
		}
		if maxUnits > 0 && units > maxUnits {
			out = append(out, Violation{
				Code:   ViolationUnitsOverLimit,
				Detail: errMaxBetUnitsExceeded(maxUnits).Error(),
			})
		}
		return out
	}
	// 任意对碰：A区|B区，区内逗号分隔的 1–49 号码，两区不可重复且合计最多 10 个。
	if isLHCRenyiDuipengPlayRule(rule) {
		out = append(out, validateLHCRenyiDuipengBetContent(content)...)
		units := countLHCRenyiDuipengBetUnits(content)
		if len(out) == 0 && units <= 0 {
			out = append(out, Violation{Code: ViolationZeroUnits, Detail: "任意对碰：A区、B区均须至少填写 1 个 01–49 号码"})
		}
		if maxUnits > 0 && units > maxUnits {
			out = append(out, Violation{
				Code:   ViolationUnitsOverLimit,
				Detail: errMaxBetUnitsExceeded(maxUnits).Error(),
			})
		}
		return out
	}
	// 生肖对碰：肖A|肖B（勿按 1–49 号池把「马」「蛇」拒掉）
	if isLHCSxDuipengPlayRule(rule) {
		out = append(out, validateLHCSxDuipengBetContent(content)...)
		units := countLHCSxDuipengBetUnits(content)
		if renDanshiKnown {
			units = renDanshiUnits
		}
		if len(out) == 0 && units <= 0 {
			out = append(out, Violation{Code: ViolationZeroUnits, Detail: "生肖对碰：请选择 2 个生肖"})
		}
		if maxUnits > 0 && units > maxUnits {
			out = append(out, Violation{
				Code:   ViolationUnitsOverLimit,
				Detail: errMaxBetUnitsExceeded(maxUnits).Error(),
			})
		}
		return out
	}
	// 尾数对碰：尾A|尾B（勿按 1–49 号池把「0」「1」拒掉）
	if isLHCWsDuipengPlayRule(rule) {
		out = append(out, validateLHCWsDuipengBetContent(content)...)
		units := countLHCWsDuipengBetUnits(content)
		if renDanshiKnown {
			units = renDanshiUnits
		}
		if len(out) == 0 && units <= 0 {
			out = append(out, Violation{Code: ViolationZeroUnits, Detail: "尾数对碰：请选择 2 个尾数"})
		}
		if maxUnits > 0 && units > maxUnits {
			out = append(out, Violation{
				Code:   ViolationUnitsOverLimit,
				Detail: errMaxBetUnitsExceeded(maxUnits).Error(),
			})
		}
		return out
	}
	// 生尾对碰：恰好 1 生肖 + 1 尾（肖|尾）
	if isLHCSwDuipengPlayRule(rule) {
		out = append(out, validateLHCSwDuipengBetContent(content)...)
		units := countLHCSwDuipengBetUnits(content)
		if renDanshiKnown {
			units = renDanshiUnits
		}
		if len(out) == 0 && units <= 0 {
			out = append(out, Violation{Code: ViolationZeroUnits, Detail: "生尾对碰：请各选择 1 个生肖和 1 个尾数"})
		}
		if maxUnits > 0 && units > maxUnits {
			out = append(out, Violation{
				Code:   ViolationUnitsOverLimit,
				Detail: errMaxBetUnitsExceeded(maxUnits).Error(),
			})
		}
		return out
	}
	// 生尾对碰：肖|尾（各恰好 1 个）
	if isLHCSwDuipengPlayRule(rule) {
		out = append(out, validateLHCSwDuipengBetContent(content)...)
		units := countLHCSwDuipengBetUnits(content)
		if renDanshiKnown {
			units = renDanshiUnits
		}
		if len(out) == 0 && units <= 0 {
			out = append(out, Violation{Code: ViolationZeroUnits, Detail: "生尾对碰：请各选 1 个生肖与 1 个尾数"})
		}
		if maxUnits > 0 && units > maxUnits {
			out = append(out, Violation{
				Code:   ViolationUnitsOverLimit,
				Detail: errMaxBetUnitsExceeded(maxUnits).Error(),
			})
		}
		return out
	}
	out = append(out, validateTokens(u, rule, content)...)
	// 组选包胆：第三方仅允许单胆（中三组选包胆等）
	if isBaodanPlayRule(rule) && u.Kind == UniverseTokenList {
		n := len(digitPoolTokensForValidate(rule, content))
		if n > 1 {
			out = append(out, Violation{
				Code:   ViolationZeroUnits,
				Detail: "包胆：只能选择一个 0–9 的号码",
			})
		}
	}
	// 组三/组六/组选6 号池：保存与审计强制最低选号（组三≥2、三星组六≥3、四星/任四组选6≥2）
	if minPick := zuxuanPoolMinPick(rule); minPick >= 2 && u.Kind == UniverseTokenList {
		n := len(digitPoolTokensForValidate(rule, content))
		if n > 0 && n < minPick {
			label := "号码池"
			bm := strings.ToLower(strings.TrimSpace(rule.BetMode))
			sub := strings.ToLower(rule.SubPlayID + " " + rule.CatalogSubID)
			switch {
			case bm == "zu24" || strings.Contains(sub, "zu24") || strings.Contains(sub, "组选24"):
				label = "组选24"
			case isSixingZu6PlayRule(rule):
				label = "组选6"
			case bm == "zu6" || (strings.Contains(sub, "zu6") && !strings.Contains(sub, "zu60") && !strings.Contains(sub, "zu120")):
				label = "组六"
			case bm == "zu3" || (strings.Contains(sub, "zu3") && !strings.Contains(sub, "zu30")):
				label = "组三"
			}
			out = append(out, Violation{
				Code:   ViolationZeroUnits,
				Detail: fmt.Sprintf("%s至少选择 %d 个号码", label, minPick),
			})
		}
	}
	// 二码/三码不定位：最少选号（二码第三方「投注数字不能低于两个」）
	if minPick := budingweiMinPoolForRule(rule); minPick >= 2 && u.Kind == UniverseTokenList {
		n := len(digitPoolTokensForValidate(rule, content))
		if n > 0 && n < minPick {
			detail := fmt.Sprintf("不定位至少选择 %d 个号码", minPick)
			if minPick == 2 {
				detail = "投注数字不能低于两个"
			} else if minPick == 4 {
				detail = "五星二码不定位：至少选择 4 个号码"
				if budingweiNeedCount(rule.CatalogSubID) >= 3 || budingweiNeedCount(rule.SubPlayID) >= 3 {
					detail = "五星三码不定位：至少选择 4 个号码"
				}
			}
			out = append(out, Violation{
				Code:   ViolationZeroUnits,
				Detail: detail,
			})
		}
	}

	if isFushiBaoziContent(rule, content) {
		out = append(out, Violation{
			Code:   ViolationZeroUnits,
			Detail: "直选复式各位选同一个号（豹子），第三方注数为 0、无法下单",
		})
		return out
	}

	units, known := countBetUnits(u, rule, content)
	if renDanshiKnown {
		units, known = renDanshiUnits, true
	}
	switch {
	case known && units <= 0:
		detail := "注数为 0"
		if isZu3DanshiPlayRule(rule) {
			detail = "组三单式：每注须为两个相同号码和一个不同号码（如 112），不含豹子与组六"
		}
		if isZu6DanshiPlayRule(rule) {
			detail = "组六单式：每注须为三个各不相同的号码（如 012），不含豹子与组三"
		}
		if isHunhePlayRule(rule) {
			detail = "混合组选：每注须为三个号码且不含豹子；组选形态相同只计 1 注，顺序不限"
		}
		out = append(out, Violation{Code: ViolationZeroUnits, Detail: detail})
	case known && maxUnits > 0 && units > maxUnits:
		out = append(out, Violation{
			Code: ViolationUnitsOverLimit,
			// 与前端 / errMaxBetUnitsExceeded 文案一致，便于弹窗原样展示
			Detail: errMaxBetUnitsExceeded(maxUnits).Error(),
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
	if isRenxuanNeedsPositionRule(rule) {
		return countRenxuanNeedsPositionBetUnits(rule, content), true
	}
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
		// 0–9 号池：粘连「12」「1234567890」按位拆开再验（对齐下注 formatSSCZuxuanPoolDigits）
		for _, tok := range expandDigitPoolTokenParts(content, min, max) {
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

// expandDigitPoolTokenParts 拆号池 token：0–9 池下把纯数字粘连串按位展开；否则保持原 token。
// 例：「1234567890,1234567890」→ 十个单码；「1,2,a」→「1」「2」「a」（非法留给后续校验）。
func expandDigitPoolTokenParts(content string, min, max int) []string {
	parts := splitContentTokens(content)
	if max > defaultPoolMax || min < 0 {
		return parts
	}
	out := make([]string, 0, len(parts))
	for _, tok := range parts {
		tok = strings.TrimSpace(tok)
		if tok == "" {
			continue
		}
		if len(tok) > 1 && isAllDigitASCII(tok) {
			for i := 0; i < len(tok); i++ {
				out = append(out, tok[i:i+1])
			}
			continue
		}
		out = append(out, tok)
	}
	return out
}

// digitPoolTokensForValidate 号池计数用：展开粘连后去重保序（与下注拆码口径一致）。
func digitPoolTokensForValidate(rule playRule, content string) []string {
	min, max := ruleNumberPool(rule)
	return dedupStrings(expandDigitPoolTokenParts(content, min, max))
}

func isAllDigitASCII(s string) bool {
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
		// 任选直选复式：C(5,n) 计注（勿五位乘积误成 72900）
		if isRenxuanPlayType(rule.PlayTypeID) && isRenxuanZhixuanFushiRule(rule) {
			return countRenxuanZhixuanFushiBetUnits(rule, content), true
		}
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

func isLHCTemaAttrPlayRule(rule playRule) bool {
	if strings.TrimSpace(rule.PlayTemplate) != "" && strings.TrimSpace(rule.PlayTemplate) != "lhc_std" {
		return false
	}
	bm := strings.ToLower(strings.TrimSpace(rule.BetMode))
	if bm == "tematouwei" {
		return false
	}
	if bm == "tema" || bm == "zhengte" {
		return true
	}
	tid := strings.TrimSpace(rule.PlayTypeID)
	if tid == "g001" || tid == "g002" || tid == "tema" || tid == "zhengte" {
		return true
	}
	sub := strings.ToLower(strings.TrimSpace(rule.SubPlayID))
	return sub == "tema" || sub == "zhengte"
}

// isLHCSxDuipengPlayRule 二全中生肖对碰（目录 281 / betMode=sx_dp）。
func isLHCSxDuipengPlayRule(rule playRule) bool {
	if strings.TrimSpace(rule.PlayTemplate) != "" && strings.TrimSpace(rule.PlayTemplate) != "lhc_std" {
		return false
	}
	if strings.EqualFold(strings.TrimSpace(rule.BetMode), "sx_dp") {
		return true
	}
	sub := strings.TrimSpace(rule.CatalogSubID)
	if sub == "" {
		sub = strings.TrimSpace(rule.SubPlayID)
	}
	return sub == "281"
}

// isLHCWsDuipengPlayRule 二全中尾数对碰（目录 282 / betMode=ws_dp；兼容 288/294）。
func isLHCWsDuipengPlayRule(rule playRule) bool {
	if strings.TrimSpace(rule.PlayTemplate) != "" && strings.TrimSpace(rule.PlayTemplate) != "lhc_std" {
		return false
	}
	if strings.EqualFold(strings.TrimSpace(rule.BetMode), "ws_dp") {
		return true
	}
	sub := strings.TrimSpace(rule.CatalogSubID)
	if sub == "" {
		sub = strings.TrimSpace(rule.SubPlayID)
	}
	return sub == "282" || sub == "288" || sub == "294"
}

// isLHCSwDuipengPlayRule 二全中生尾对碰（目录 283 / betMode=sw_dp；兼容 289/295）。
func isLHCSwDuipengPlayRule(rule playRule) bool {
	if strings.TrimSpace(rule.PlayTemplate) != "" && strings.TrimSpace(rule.PlayTemplate) != "lhc_std" {
		return false
	}
	if strings.EqualFold(strings.TrimSpace(rule.BetMode), "sw_dp") {
		return true
	}
	sub := strings.TrimSpace(rule.CatalogSubID)
	if sub == "" {
		sub = strings.TrimSpace(rule.SubPlayID)
	}
	return sub == "283" || sub == "289" || sub == "295"
}

// isLHCRenyiDuipengPlayRule 二全中/二中特/特串任意对碰（目录 284/290/296）。
func isLHCRenyiDuipengPlayRule(rule playRule) bool {
	if strings.TrimSpace(rule.PlayTemplate) != "" && strings.TrimSpace(rule.PlayTemplate) != "lhc_std" {
		return false
	}
	if strings.EqualFold(strings.TrimSpace(rule.BetMode), "renyi_dp") {
		return true
	}
	sub := strings.TrimSpace(rule.CatalogSubID)
	if sub == "" {
		sub = strings.TrimSpace(rule.SubPlayID)
	}
	return sub == "284" || sub == "290" || sub == "296"
}

const lhcRenyiDuipengMaxPicks = 10

// validateLHCRenyiDuipengBetContent 校验任意对碰的 A区|B区号码格式与总数上限。
func validateLHCRenyiDuipengBetContent(content string) []Violation {
	content = strings.TrimSpace(content)
	sep := ""
	switch {
	case strings.Count(content, "|") == 1:
		sep = "|"
	case strings.Count(content, "#") == 1:
		sep = "#"
	default:
		return []Violation{{Code: ViolationZeroUnits, Detail: "任意对碰：请用 | 分隔 A区 与 B区（如 01,13|02,14）"}}
	}

	parts := strings.Split(content, sep)
	left, leftInvalid := parseLHCRenyiDuipengNumbers(parts[0])
	right, rightInvalid := parseLHCRenyiDuipengNumbers(parts[1])
	invalid := append(leftInvalid, rightInvalid...)
	if len(invalid) > 0 {
		return []Violation{{
			Code:   ViolationTokenOutOfPool,
			Detail: fmt.Sprintf("任意对碰：无效号码 %s（请选择 01–49）", strings.Join(invalid, ",")),
		}}
	}
	if len(left) == 0 || len(right) == 0 {
		return []Violation{{Code: ViolationZeroUnits, Detail: "任意对碰：A区、B区均须至少填写 1 个 01–49 号码"}}
	}

	rightSet := make(map[int]struct{}, len(right))
	for _, n := range right {
		rightSet[n] = struct{}{}
	}
	overlap := make([]string, 0)
	for _, n := range left {
		if _, ok := rightSet[n]; ok {
			overlap = append(overlap, fmt.Sprintf("%02d", n))
		}
	}
	if len(overlap) > 0 {
		return []Violation{{
			Code:   ViolationZeroUnits,
			Detail: fmt.Sprintf("任意对碰：A区与B区号码不可重复（重复：%s）", strings.Join(overlap, ",")),
		}}
	}
	if len(left)+len(right) > lhcRenyiDuipengMaxPicks {
		return []Violation{{Code: ViolationZeroUnits, Detail: "任意对碰：A区和B区合计最多选择 10 个号码"}}
	}
	return nil
}

func countLHCRenyiDuipengBetUnits(content string) int {
	content = strings.TrimSpace(content)
	sep := "|"
	if !strings.Contains(content, sep) {
		sep = "#"
	}
	if strings.Count(content, sep) != 1 {
		return 0
	}
	parts := strings.Split(content, sep)
	left, leftInvalid := parseLHCRenyiDuipengNumbers(parts[0])
	right, rightInvalid := parseLHCRenyiDuipengNumbers(parts[1])
	if len(leftInvalid) > 0 || len(rightInvalid) > 0 || len(left) == 0 || len(right) == 0 {
		return 0
	}
	return len(left) * len(right)
}

func parseLHCRenyiDuipengNumbers(raw string) (numbers []int, invalid []string) {
	seen := make(map[int]struct{})
	for _, token := range parseTextTokens(raw) {
		n, err := strconv.Atoi(token)
		if err != nil || n < 1 || n > 49 {
			invalid = append(invalid, token)
			continue
		}
		if _, ok := seen[n]; ok {
			continue
		}
		seen[n] = struct{}{}
		numbers = append(numbers, n)
	}
	return numbers, invalid
}

// validateLHCSwDuipengBetContent 生尾对碰：恰好 1 生肖 + 1 尾（马|0 / 0|马 / 马,0）。
func validateLHCSwDuipengBetContent(content string) []Violation {
	content = strings.TrimSpace(content)
	if content == "" {
		return []Violation{{Code: ViolationEmptyContent, Detail: "投注内容为空"}}
	}
	if _, _, ok := guajibet.ParseLHCSwDuipengSides(content); !ok {
		return []Violation{{Code: ViolationZeroUnits, Detail: "生尾对碰：请各选择 1 个生肖和 1 个尾数（如 马|0）"}}
	}
	return nil
}

// countLHCSwDuipengBetUnits 注数=生肖展开数 × 尾数展开数 − 两侧共有号码数
//（与 guajibet.CountBetNums / 第三方一致：狗|5 → 19）。
func countLHCSwDuipengBetUnits(content string) int {
	z, t, ok := guajibet.ParseLHCSwDuipengSides(content)
	if !ok {
		return 0
	}
	left := lhcZodiacNumbers[z]
	if len(left) == 0 {
		return 0
	}
	rightSet := make(map[int]struct{}, 8)
	for n := 1; n <= 49; n++ {
		if n%10 == t {
			rightSet[n] = struct{}{}
		}
	}
	if len(rightSet) == 0 {
		return 0
	}
	overlap := 0
	for _, n := range left {
		if _, ok := rightSet[n]; ok {
			overlap++
		}
	}
	return len(left)*len(rightSet) - overlap
}

// validateLHCSxDuipengBetContent 生肖对碰：恰好 2 个合法生肖（肖A|肖B 或扁选 马,蛇）。
func validateLHCSxDuipengBetContent(content string) []Violation {
	content = strings.TrimSpace(content)
	if content == "" {
		return []Violation{{Code: ViolationEmptyContent, Detail: "投注内容为空"}}
	}
	flat := strings.NewReplacer("|", ",", "#", ",").Replace(content)
	parts := parseTextTokens(flat)
	var zs []string
	var bad []string
	seen := make(map[string]bool, len(parts))
	for _, p := range parts {
		if _, ok := lhcZodiacNumbers[p]; !ok {
			bad = append(bad, p)
			continue
		}
		if seen[p] {
			continue
		}
		seen[p] = true
		zs = append(zs, p)
	}
	if len(bad) > 0 {
		return []Violation{{
			Code:   ViolationTokenOutOfPool,
			Detail: fmt.Sprintf("生肖对碰：无效选项 %s（请选择十二生肖）", strings.Join(dedupStrings(bad), ",")),
		}}
	}
	if len(zs) < 2 {
		return []Violation{{Code: ViolationZeroUnits, Detail: "生肖对碰：请选择 2 个生肖"}}
	}
	if len(zs) > 2 {
		return []Violation{{Code: ViolationZeroUnits, Detail: "生肖对碰：最多选择 2 个生肖"}}
	}
	return nil
}

// countLHCSxDuipengBetUnits 注数=两侧展开号码数之积（马5×其它4=20；两肖均非马=16）。
func countLHCSxDuipengBetUnits(content string) int {
	flat := strings.NewReplacer("|", ",", "#", ",").Replace(content)
	zs := parseLHCZodiacs(flat)
	if len(zs) < 2 {
		return 0
	}
	a := len(lhcZodiacNumbers[zs[0]])
	b := len(lhcZodiacNumbers[zs[1]])
	if a <= 0 || b <= 0 {
		return 0
	}
	return a * b
}

// parseLHCTailTokens 解析尾数对碰内容为 0–9（兼容「0尾」「0|1」「0,1」）。
func parseLHCTailTokens(raw string) []string {
	flat := strings.NewReplacer("|", ",", "#", ",", "尾", "").Replace(raw)
	parts := parseTextTokens(flat)
	seen := make(map[string]bool, len(parts))
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 || n > 9 {
			continue
		}
		tok := strconv.Itoa(n)
		if seen[tok] {
			continue
		}
		seen[tok] = true
		out = append(out, tok)
	}
	return out
}

// validateLHCWsDuipengBetContent 尾数对碰：恰好 2 个合法尾数（尾A|尾B 或扁选 0,1）。
func validateLHCWsDuipengBetContent(content string) []Violation {
	content = strings.TrimSpace(content)
	if content == "" {
		return []Violation{{Code: ViolationEmptyContent, Detail: "投注内容为空"}}
	}
	flat := strings.NewReplacer("|", ",", "#", ",").Replace(content)
	parts := parseTextTokens(flat)
	var tails []string
	var bad []string
	seen := make(map[string]bool, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(p), "尾"))
		if p == "" {
			continue
		}
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 || n > 9 {
			bad = append(bad, p)
			continue
		}
		tok := strconv.Itoa(n)
		if seen[tok] {
			continue
		}
		seen[tok] = true
		tails = append(tails, tok)
	}
	if len(bad) > 0 {
		return []Violation{{
			Code:   ViolationTokenOutOfPool,
			Detail: fmt.Sprintf("尾数对碰：无效选项 %s（请选择 0–9 尾）", strings.Join(dedupStrings(bad), ",")),
		}}
	}
	if len(tails) < 2 {
		return []Violation{{Code: ViolationZeroUnits, Detail: "尾数对碰：请选择 2 个尾数"}}
	}
	if len(tails) > 2 {
		return []Violation{{Code: ViolationZeroUnits, Detail: "尾数对碰：最多选择 2 个尾数"}}
	}
	return nil
}

// countLHCWsDuipengBetUnits 注数=两侧展开号码数之积（0尾4×1尾5=20；1尾×2尾=25）。
func countLHCWsDuipengBetUnits(content string) int {
	tails := parseLHCTailTokens(content)
	if len(tails) < 2 {
		return 0
	}
	a := lhcTailNumberCount(tails[0])
	b := lhcTailNumberCount(tails[1])
	if a <= 0 || b <= 0 {
		return 0
	}
	return a * b
}

func lhcTailNumberCount(tail string) int {
	n, err := strconv.Atoi(strings.TrimSpace(tail))
	if err != nil || n < 0 || n > 9 {
		return 0
	}
	c := 0
	for i := 1; i <= 49; i++ {
		if i%10 == n {
			c++
		}
	}
	return c
}

// validateLHCTemaBetContent 特码三段内容：只允许 1–49 号码与已知属性/波色。
func validateLHCTemaBetContent(content string) []Violation {
	content = strings.TrimSpace(content)
	if content == "" {
		return []Violation{{Code: ViolationEmptyContent, Detail: "投注内容为空"}}
	}
	nums, attrs, waves := guajibet.ParseLHCTemaParts(content)
	legal := make(map[string]struct{}, len(nums)+len(attrs)+len(waves))
	for _, t := range nums {
		legal[t] = struct{}{}
		if n, err := strconv.Atoi(t); err == nil {
			legal[strconv.Itoa(n)] = struct{}{}
		}
	}
	for _, t := range attrs {
		legal[t] = struct{}{}
	}
	for _, t := range waves {
		legal[t] = struct{}{}
	}
	var bad []string
	for _, tok := range splitContentTokens(content) {
		tok = strings.TrimSpace(strings.TrimSuffix(tok, "||"))
		if tok == "" {
			continue
		}
		switch tok {
		case "洪波":
			tok = "红波"
		case "绿播":
			tok = "绿波"
		}
		if _, ok := legal[tok]; ok {
			continue
		}
		// 未规范化号码（如 07）已在 legal；00 / 越界单独提示
		if n, err := strconv.Atoi(tok); err == nil {
			if n < 1 || n > 49 {
				bad = append(bad, tok)
			}
			continue
		}
		bad = append(bad, tok)
	}
	if len(bad) > 0 {
		return []Violation{{
			Code:   ViolationTokenOutOfPool,
			Detail: fmt.Sprintf("特码选项无效：%s（仅支持 1–49 号码与属性/波色）", strings.Join(dedupStrings(bad), ",")),
		}}
	}
	if len(nums)+len(attrs)+len(waves) == 0 {
		return []Violation{{Code: ViolationZeroUnits, Detail: "请选择属性或输入 1–49 号码"}}
	}
	return nil
}

const zu12FormatDetail = "组选12：从0-9中输入1个及以上二重号码、2个及以上单号，两区用逗号分隔，如：12,3234"
const zu12OverlapDetail = "组选12：每个二重号须能与单号区凑成至少 1 注（选该二重时单号区去掉该码后仍≥2；如 23,123 计 2 注，1,12 为 0 注）"

func isZu12PlayRule(rule playRule) bool {
	bm := strings.ToLower(strings.TrimSpace(rule.BetMode))
	if bm == "zu12" {
		return true
	}
	sub := strings.ToLower(rule.SubPlayID + " " + rule.CatalogSubID)
	if strings.Contains(sub, "zu120") || strings.Contains(sub, "组选120") {
		return false
	}
	return strings.Contains(sub, "zu12") || strings.Contains(sub, "组选12")
}

// parseZu12DualZones 二重≥1、单号区≥2（区内去重；跨区重叠保留）。
func parseZu12DualZones(content string) (doubles, singles string, ok bool) {
	content = strings.TrimSpace(strings.ReplaceAll(content, "，", ","))
	if content == "" {
		return "", "", false
	}
	parts := strings.Split(content, ",")
	if len(parts) != 2 {
		return "", "", false
	}
	doubles = uniqueDigitRunSchemes(parts[0])
	singles = uniqueDigitRunSchemes(parts[1])
	if len(doubles) < 1 || len(singles) < 2 {
		return "", "", false
	}
	return doubles, singles, true
}

func uniqueDigitRunSchemes(s string) string {
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

func validateZu12DualZone(content string) []Violation {
	doubles, singles, ok := parseZu12DualZones(content)
	if !ok {
		return []Violation{{Code: ViolationZeroUnits, Detail: zu12FormatDetail}}
	}
	if countZu12DualZoneBetUnits(content) > 0 {
		return nil
	}
	if hasZu12Overlap(doubles, singles) {
		return []Violation{{Code: ViolationZeroUnits, Detail: zu12OverlapDetail}}
	}
	return []Violation{{Code: ViolationZeroUnits, Detail: zu12FormatDetail}}
}

func hasZu12Overlap(doubles, singles string) bool {
	set := make(map[byte]bool, len(doubles))
	for i := 0; i < len(doubles); i++ {
		set[doubles[i]] = true
	}
	for i := 0; i < len(singles); i++ {
		if set[singles[i]] {
			return true
		}
	}
	return false
}

// countZu12DualZoneBetUnits 对每个二重 d，C(|单号\{d}|, 2) 求和。
func countZu12DualZoneBetUnits(content string) int {
	a, b, ok := parseZu12DualZones(content)
	if !ok {
		return 0
	}
	total := 0
	for i := 0; i < len(a); i++ {
		n := 0
		for j := 0; j < len(b); j++ {
			if b[j] != a[i] {
				n++
			}
		}
		total += combinInt(n, 2)
	}
	return total
}

const zu4FormatDetail = "组选4：从0-9中输入1个及以上三重号码、1个及以上单号，两区用逗号分隔，如：1,2"
const zu4OverlapDetail = "组选4：每个三重号须能与单号区凑成至少 1 注（选该三重时单号区去掉该码后仍≥1；如 12,34 计 4 注，1,1 为 0 注）"

func isZu4PlayRule(rule playRule) bool {
	bm := strings.ToLower(strings.TrimSpace(rule.BetMode))
	if bm == "zu4" {
		return true
	}
	sub := strings.ToLower(rule.SubPlayID + " " + rule.CatalogSubID)
	if strings.Contains(sub, "zu24") || strings.Contains(sub, "组选24") {
		return false
	}
	if strings.Contains(sub, "zu12") || strings.Contains(sub, "组选12") {
		return false
	}
	return strings.Contains(sub, "zu4") || strings.Contains(sub, "组选4")
}

// parseZu4DualZones 三重≥1、单号区≥1（区内去重；跨区重叠保留）。
func parseZu4DualZones(content string) (triples, singles string, ok bool) {
	content = strings.TrimSpace(strings.ReplaceAll(content, "，", ","))
	if content == "" {
		return "", "", false
	}
	parts := strings.Split(content, ",")
	if len(parts) != 2 {
		return "", "", false
	}
	triples = uniqueDigitRunSchemes(parts[0])
	singles = uniqueDigitRunSchemes(parts[1])
	if len(triples) < 1 || len(singles) < 1 {
		return "", "", false
	}
	return triples, singles, true
}

func validateZu4DualZone(content string) []Violation {
	triples, singles, ok := parseZu4DualZones(content)
	if !ok {
		return []Violation{{Code: ViolationZeroUnits, Detail: zu4FormatDetail}}
	}
	if countZu4DualZoneBetUnits(content) > 0 {
		return nil
	}
	if hasZu4Overlap(triples, singles) {
		return []Violation{{Code: ViolationZeroUnits, Detail: zu4OverlapDetail}}
	}
	return []Violation{{Code: ViolationZeroUnits, Detail: zu4FormatDetail}}
}

func hasZu4Overlap(triples, singles string) bool {
	set := make(map[byte]bool, len(triples))
	for i := 0; i < len(triples); i++ {
		set[triples[i]] = true
	}
	for i := 0; i < len(singles); i++ {
		if set[singles[i]] {
			return true
		}
	}
	return false
}

// countZu4DualZoneBetUnits 对每个三重 t，统计单号 s 中 s≠t 的个数并求和。
func countZu4DualZoneBetUnits(content string) int {
	a, b, ok := parseZu4DualZones(content)
	if !ok {
		return 0
	}
	total := 0
	for i := 0; i < len(a); i++ {
		n := 0
		for j := 0; j < len(b); j++ {
			if b[j] != a[i] {
				n++
			}
		}
		total += n
	}
	return total
}

const zu60FormatDetail = "组选60：从0-9中输入1个及以上二重号码、3个及以上单号，两区用逗号分隔，如：1,234"
const zu60OverlapDetail = "组选60：每个二重号须能与单号区凑成至少 1 注（选该二重时单号区去掉该码后仍≥3）"

func isZu60PlayRule(rule playRule) bool {
	bm := strings.ToLower(strings.TrimSpace(rule.BetMode))
	if bm == "zu60" {
		return true
	}
	sub := strings.ToLower(rule.SubPlayID + " " + rule.CatalogSubID)
	if strings.Contains(sub, "zu120") || strings.Contains(sub, "组选120") {
		return false
	}
	if strings.TrimSpace(rule.SubPlayID) == "157" || strings.TrimSpace(rule.CatalogSubID) == "157" {
		return true
	}
	return strings.Contains(sub, "zu60") || strings.Contains(sub, "组选60")
}

func isZu20PlayRule(rule playRule) bool {
	bm := strings.ToLower(strings.TrimSpace(rule.BetMode))
	if bm == "zu20" {
		return true
	}
	sub := strings.ToLower(rule.SubPlayID + " " + rule.CatalogSubID)
	if strings.TrimSpace(rule.SubPlayID) == "159" || strings.TrimSpace(rule.CatalogSubID) == "159" {
		return true
	}
	return strings.Contains(sub, "zu20") || strings.Contains(sub, "组选20")
}

func isZu10PlayRule(rule playRule) bool {
	bm := strings.ToLower(strings.TrimSpace(rule.BetMode))
	if bm == "zu10" {
		return true
	}
	sub := strings.ToLower(rule.SubPlayID + " " + rule.CatalogSubID)
	if strings.TrimSpace(rule.SubPlayID) == "160" || strings.TrimSpace(rule.CatalogSubID) == "160" {
		return true
	}
	return strings.Contains(sub, "zu10") || strings.Contains(sub, "组选10")
}

func isZu5PlayRule(rule playRule) bool {
	bm := strings.ToLower(strings.TrimSpace(rule.BetMode))
	if bm == "zu5" {
		return true
	}
	sub := strings.ToLower(rule.SubPlayID + " " + rule.CatalogSubID)
	if strings.TrimSpace(rule.SubPlayID) == "161" || strings.TrimSpace(rule.CatalogSubID) == "161" {
		return true
	}
	if strings.Contains(sub, "zu50") || strings.Contains(sub, "组选50") {
		return false
	}
	return strings.Contains(sub, "zu5") || strings.Contains(sub, "组选5")
}

// parseZu60DualZones 二重≥1、单号区≥3（区内去重；跨区重叠保留）。
func parseZu60DualZones(content string) (doubles, singles string, ok bool) {
	content = strings.TrimSpace(strings.ReplaceAll(content, "，", ","))
	if content == "" {
		return "", "", false
	}
	parts := strings.Split(content, ",")
	if len(parts) != 2 {
		return "", "", false
	}
	doubles = uniqueDigitRunSchemes(parts[0])
	singles = uniqueDigitRunSchemes(parts[1])
	if len(doubles) < 1 || len(singles) < 3 {
		return "", "", false
	}
	return doubles, singles, true
}

func validateZu60DualZone(content string) []Violation {
	doubles, singles, ok := parseZu60DualZones(content)
	if !ok {
		return []Violation{{Code: ViolationZeroUnits, Detail: zu60FormatDetail}}
	}
	if countZu60DualZoneBetUnits(content) > 0 {
		return nil
	}
	if hasZu12Overlap(doubles, singles) {
		return []Violation{{Code: ViolationZeroUnits, Detail: zu60OverlapDetail}}
	}
	return []Violation{{Code: ViolationZeroUnits, Detail: zu60FormatDetail}}
}

// countZu60DualZoneBetUnits 对每个二重 d，C(|单号\{d}|, 3) 求和。
func countZu60DualZoneBetUnits(content string) int {
	a, b, ok := parseZu60DualZones(content)
	if !ok {
		return 0
	}
	total := 0
	for i := 0; i < len(a); i++ {
		n := 0
		for j := 0; j < len(b); j++ {
			if b[j] != a[i] {
				n++
			}
		}
		total += combinInt(n, 3)
	}
	return total
}

const zu30FormatDetail = "组选30：从0-9中输入3个及以上二重号码、1个及以上单号，两区用逗号分隔，如：123,1"
const zu30OverlapDetail = "组选30：每组二重号须能与单号区凑成至少 1 注（选该对二重时单号区去掉这两码后仍≥1）"

func isZu30PlayRule(rule playRule) bool {
	bm := strings.ToLower(strings.TrimSpace(rule.BetMode))
	if bm == "zu30" {
		return true
	}
	sub := strings.ToLower(rule.SubPlayID + " " + rule.CatalogSubID)
	return strings.Contains(sub, "zu30") || strings.Contains(sub, "组选30")
}

// parseZu30DualZones 二重≥3、单号区≥1（区内去重；跨区重叠保留）。
func parseZu30DualZones(content string) (doubles, singles string, ok bool) {
	content = strings.TrimSpace(strings.ReplaceAll(content, "，", ","))
	if content == "" {
		return "", "", false
	}
	parts := strings.Split(content, ",")
	if len(parts) != 2 {
		return "", "", false
	}
	doubles = uniqueDigitRunSchemes(parts[0])
	singles = uniqueDigitRunSchemes(parts[1])
	if len(doubles) < 3 || len(singles) < 1 {
		return "", "", false
	}
	return doubles, singles, true
}

func validateZu30DualZone(content string) []Violation {
	doubles, singles, ok := parseZu30DualZones(content)
	if !ok {
		return []Violation{{Code: ViolationZeroUnits, Detail: zu30FormatDetail}}
	}
	if countZu30DualZoneBetUnits(content) > 0 {
		return nil
	}
	if hasZu12Overlap(doubles, singles) {
		return []Violation{{Code: ViolationZeroUnits, Detail: zu30OverlapDetail}}
	}
	return []Violation{{Code: ViolationZeroUnits, Detail: zu30FormatDetail}}
}

// countZu30DualZoneBetUnits 对每个二重对 (d1,d2)，计 |单号\{d1,d2}| 并求和。
func countZu30DualZoneBetUnits(content string) int {
	a, b, ok := parseZu30DualZones(content)
	if !ok {
		return 0
	}
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

const zu20FormatDetail = "组选20：三重号与单号个数须相同，至少各 2 个，两区用逗号分隔，如：12,34"
const zu20OverlapDetail = "组选20：每个三重号须能与单号区凑成至少 1 注（选该三重时单号区去掉该码后仍≥2）"
const zu10FormatDetail = "组选10：从0-9中输入1个及以上三重号码、1个及以上二重号码，两区用逗号分隔，如：1,2"
const zu10OverlapDetail = "组选10：每个三重号须能与二重号区凑成至少 1 注（选该三重时二重区去掉该码后仍≥1）"
const zu5FormatDetail = "组选5：从0-9中输入1个及以上四重号码、1个及以上单号，两区用逗号分隔，如：1,2"
const zu5OverlapDetail = "组选5：每个四重号须能与单号区凑成至少 1 注（选该四重时单号区去掉该码后仍≥1）"

// parseZu20DualZones 三重与单号个数相同且各≥2。
func parseZu20DualZones(content string) (head, tail string, ok bool) {
	content = strings.TrimSpace(strings.ReplaceAll(content, "，", ","))
	if content == "" {
		return "", "", false
	}
	parts := strings.Split(content, ",")
	if len(parts) != 2 {
		return "", "", false
	}
	head = uniqueDigitRunSchemes(parts[0])
	tail = uniqueDigitRunSchemes(parts[1])
	if len(head) < 2 || len(tail) < 2 || len(head) != len(tail) {
		return "", "", false
	}
	return head, tail, true
}

func validateZu20DualZone(content string) []Violation {
	head, tail, ok := parseZu20DualZones(content)
	if !ok {
		return []Violation{{Code: ViolationZeroUnits, Detail: zu20FormatDetail}}
	}
	if countZu20DualZoneBetUnits(content) > 0 {
		return nil
	}
	if hasZu12Overlap(head, tail) {
		return []Violation{{Code: ViolationZeroUnits, Detail: zu20OverlapDetail}}
	}
	return []Violation{{Code: ViolationZeroUnits, Detail: zu20FormatDetail}}
}

func countZu20DualZoneBetUnits(content string) int {
	a, b, ok := parseZu20DualZones(content)
	if !ok {
		return 0
	}
	total := 0
	for i := 0; i < len(a); i++ {
		n := 0
		for j := 0; j < len(b); j++ {
			if b[j] != a[i] {
				n++
			}
		}
		total += combinInt(n, 2)
	}
	return total
}

// parseZuPairDualZones 头/尾各≥1（组选10/5）。
func parseZuPairDualZones(content string) (head, tail string, ok bool) {
	content = strings.TrimSpace(strings.ReplaceAll(content, "，", ","))
	if content == "" {
		return "", "", false
	}
	parts := strings.Split(content, ",")
	if len(parts) != 2 {
		return "", "", false
	}
	head = uniqueDigitRunSchemes(parts[0])
	tail = uniqueDigitRunSchemes(parts[1])
	if len(head) < 1 || len(tail) < 1 {
		return "", "", false
	}
	return head, tail, true
}

func validateZuPairDualZone(content, formatDetail, overlapDetail string) []Violation {
	head, tail, ok := parseZuPairDualZones(content)
	if !ok {
		return []Violation{{Code: ViolationZeroUnits, Detail: formatDetail}}
	}
	if countZuPairDualZoneBetUnits(content) > 0 {
		return nil
	}
	if hasZu12Overlap(head, tail) {
		return []Violation{{Code: ViolationZeroUnits, Detail: overlapDetail}}
	}
	return []Violation{{Code: ViolationZeroUnits, Detail: formatDetail}}
}

func countZuPairDualZoneBetUnits(content string) int {
	a, b, ok := parseZuPairDualZones(content)
	if !ok {
		return 0
	}
	total := 0
	for i := 0; i < len(a); i++ {
		for j := 0; j < len(b); j++ {
			if b[j] != a[i] {
				total++
			}
		}
	}
	return total
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
