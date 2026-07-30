package schemes

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// 玩法的「合法投注空间」权威定义。
//
// 立这一层的原因：号池、选项宇宙、注数这三件事此前分散在 guajibet（豹子零注、solo 上限）、
// attributeUniverse（和值宇宙）、ruleNumberPool（号池）与前端 playConfig.ts 里各推一份，
// 任何一处漏掉就是一个静默 bug。保存校验、下注前校验、对账命令、单测都应查这里。
//
// 与 attributeUniverse 的关系：后者是 worker 冷热出号**当前实际使用**的宇宙，用
// 「单号池上下界 × 位数」推导；本文件用穷举给出**理论可达**的宇宙。两者的差集即
// 「永远不可能开出、却会被当成最冷号选中」的选项，由 UnreachableHotColdOptions 暴露。

const (
	// UniversePerPosition 按位分行：每行是该位的候选号（定位胆、直选复式）
	UniversePerPosition = "perPosition"
	// UniverseTokenList 单行号池：组选复式、不定位、包胆、任选
	UniverseTokenList = "tokenList"
	// UniverseCombos 单式：逗号分隔的定长组合
	UniverseCombos = "combos"
	// UniverseAttribute 属性选项：大小单双、龙虎、和值、跨度、尾数
	UniverseAttribute = "attribute"
)

// PlayUniverse 一个玩法的合法投注空间。
type PlayUniverse struct {
	Kind      string
	Positions []string // perPosition 时的位名；其余为空
	Tokens    []string // 合法 token 全集；attribute 时为可达选项宇宙
	ComboLen  int      // combos 时每注长度
	MaxUnits  int      // 全选时的理论注数；0 表示无法推导
	Note      string   // 推导说明，便于审计输出定位
}

// 违规码。
const (
	ViolationTokenOutOfPool = "token_out_of_pool"
	ViolationZeroUnits      = "zero_units"
	ViolationUnitsOverLimit = "units_over_limit"
	ViolationEmptyContent   = "empty_content"
	ViolationTooManyLines   = "too_many_lines"
)

type Violation struct {
	Code   string `json:"code"`
	Detail string `json:"detail"`
}

// 穷举上限：超过则放弃可达性推导，退回笛卡尔边界并在 Note 说明。
// 时时彩五位 10^5、PK10 冠亚 10×9、快三 6^3 均在此之内。
const universeEnumLimit = 2_000_000

// UniverseForScheme 按方案定义推导合法投注空间。kind / config 与 FormatBetContentLines 同源。
func UniverseForScheme(kind string, config []byte) (PlayUniverse, bool) {
	if len(config) == 0 {
		return PlayUniverse{}, false
	}
	return universeForRule(parseSchemeConfig(kind, config, 0, 0).Play)
}

func universeForRule(rule playRule) (PlayUniverse, bool) {
	switch universeKindForRule(rule) {
	case "":
		return PlayUniverse{}, false

	case UniverseAttribute:
		tokens, exact := reachableAttributeUniverse(rule)
		if len(tokens) == 0 {
			return PlayUniverse{}, false
		}
		note := "穷举可达值域"
		if !exact {
			note = "组合数超限，退回上下界推导"
		}
		return PlayUniverse{
			Kind: UniverseAttribute, Tokens: tokens, MaxUnits: len(tokens), Note: note,
		}, true

	case UniverseCombos:
		pool := playNumberPool(rule)
		n := playPositionCount(rule)
		return PlayUniverse{
			Kind: UniverseCombos, Tokens: pool, ComboLen: n, Note: "单式：每注定长组合",
		}, true

	case UniverseTokenList:
		pool := playNumberPool(rule)
		// 组选/不定位的注数是组合数而非乘积，需要子玩法选码数才能算，这里不下结论。
		return PlayUniverse{
			Kind: UniverseTokenList, Tokens: pool, Note: "整体号池：注数为组合数，不在此推导",
		}, true

	default:
		pool := playNumberPool(rule)
		positions := playRuleDisplayPositions(rule)
		n := playPositionCount(rule)
		if len(positions) != n {
			// 位名推不出来不影响号池校验，用序号占位
			positions = make([]string, n)
			for i := range positions {
				positions[i] = fmt.Sprintf("第%d位", i+1)
			}
		}
		return PlayUniverse{
			Kind: UniversePerPosition, Positions: positions, Tokens: pool,
			MaxUnits: powInt(len(pool), n), Note: "按位分行：全选注数 = 号池^位数",
		}, true
	}
}

func universeKindForRule(rule playRule) string {
	switch strings.ToLower(strings.TrimSpace(rule.BetMode)) {
	case "daxiao", "danshuang", "dxds", "zhuangxian",
		"longhu", "longhuhe", "longhubao", "hezhi", "kuadu", "weishu":
		return UniverseAttribute
	case "teshu":
		// 快三 / PC28 的特殊号是「豹子/对子/顺子」这类固定选项，形态确定。
		// 时时彩五星特殊号则两种都有——「豹子/顺子」选文字，「一帆风顺」选一个 0-9 数字，
		// 只看 betMode 分不出来，必须查子玩法表。在接上子玩法表之前一律判为未知，
		// 不做任何断言：两边都猜错过，与其误报不如挂在"未覆盖"里让人看见这个缺口。
		switch strings.TrimSpace(rule.PlayTemplate) {
		case "k3_std", "pc28_std":
			return UniverseAttribute
		}
		return ""
	case "danshi", "zhixuan_ds", "zuxuan_ds", "hunhe":
		return UniverseCombos
	case "budingwei", "baodan", "zuxuan_fs",
		"zu3", "zu6", "zu12", "zu24", "zu30", "zu60", "zu120":
		return UniverseTokenList
	}
	if isHotColdDigitOverall(rule) {
		return UniverseTokenList
	}
	return UniversePerPosition
}

// ---------- 可达值域穷举 ----------

// reachableAttributeUniverse 属性玩法的可达选项。
// 第二个返回值为 false 表示组合数超限、退回了 attributeUniverse 的上下界推导。
func reachableAttributeUniverse(rule playRule) ([]string, bool) {
	switch strings.ToLower(strings.TrimSpace(rule.BetMode)) {
	case "hezhi":
		return reachableDerivedValues(rule, sumOf)
	case "weishu":
		return reachableDerivedValues(rule, func(vals []int) int { return sumOf(vals) % 10 })
	case "kuadu":
		return reachableDerivedValues(rule, spanOf)
	}
	// 大小单双 / 龙虎 / 特殊号等固定文字选项，全部可达
	return attributeUniverse(rule), true
}

// reachableDerivedValues 穷举该位段所有合法开奖组合，收集 derive 的取值。
func reachableDerivedValues(rule playRule, derive func([]int) int) ([]string, bool) {
	min, max := ruleNumberPool(rule)
	n := rule.SegmentLen
	if n < 1 {
		n = playPositionCount(rule)
	}
	if n < 1 || max < min {
		return attributeUniverse(rule), false
	}
	distinct := templateDrawsDistinctBalls(rule.PlayTemplate)
	notAllSame := ruleExcludesAllSame(rule)

	seen := map[int]struct{}{}
	ok := enumerateSegments(min, max, n, distinct, func(vals []int) {
		if notAllSame && allSameInts(vals) {
			return
		}
		seen[derive(vals)] = struct{}{}
	})
	if !ok || len(seen) == 0 {
		return attributeUniverse(rule), false
	}
	out := make([]int, 0, len(seen))
	for v := range seen {
		out = append(out, v)
	}
	sort.Ints(out)
	tokens := make([]string, 0, len(out))
	for _, v := range out {
		tokens = append(tokens, strconv.Itoa(v))
	}
	return tokens, true
}

// enumerateSegments 穷举 n 个位置在 [min,max] 上的取值组合；distinct 时同期号码互不相同。
// 返回 false 表示组合数超过 universeEnumLimit，未完成穷举。
func enumerateSegments(min, max, n int, distinct bool, visit func([]int)) bool {
	if total := combinationCount(min, max, n, distinct); total <= 0 || total > universeEnumLimit {
		return false
	}
	vals := make([]int, n)
	used := map[int]bool{}
	var rec func(depth int)
	rec = func(depth int) {
		if depth == n {
			visit(vals)
			return
		}
		for v := min; v <= max; v++ {
			if distinct && used[v] {
				continue
			}
			vals[depth] = v
			if distinct {
				used[v] = true
			}
			rec(depth + 1)
			if distinct {
				used[v] = false
			}
		}
	}
	rec(0)
	return true
}

func combinationCount(min, max, n int, distinct bool) int {
	size := max - min + 1
	if size <= 0 || n <= 0 {
		return 0
	}
	total := 1
	for i := 0; i < n; i++ {
		f := size
		if distinct {
			f = size - i
			if f <= 0 {
				return 0
			}
		}
		if total > universeEnumLimit/f {
			return universeEnumLimit + 1
		}
		total *= f
	}
	return total
}

// templateDrawsDistinctBalls 该彩种同一期各位号码是否互不相同。
// PK10 是名次排列、11 选 5 与六合彩是不重复抽号；时时彩/快三/PC28 各位独立可重复。
func templateDrawsDistinctBalls(template string) bool {
	switch strings.TrimSpace(template) {
	case "pk10_std", "syxw_std", "lhc_std":
		return true
	}
	return false
}

// ruleExcludesAllSame 组选不计顺序，全同号（豹子）不构成组选注，其和值/跨度不可达。
func ruleExcludesAllSame(rule playRule) bool {
	if rule.HezhiZuxuan {
		return true
	}
	text := strings.ToLower(rule.BetMode + " " + rule.SubPlayID + " " + rule.CatalogSubID)
	return strings.Contains(text, "zuxuan") || strings.Contains(text, "zu3") || strings.Contains(text, "zu6")
}

func sumOf(vals []int) int {
	s := 0
	for _, v := range vals {
		s += v
	}
	return s
}

func spanOf(vals []int) int {
	if len(vals) == 0 {
		return 0
	}
	lo, hi := vals[0], vals[0]
	for _, v := range vals[1:] {
		if v < lo {
			lo = v
		}
		if v > hi {
			hi = v
		}
	}
	return hi - lo
}

func allSameInts(vals []int) bool {
	if len(vals) <= 1 {
		return false
	}
	for _, v := range vals[1:] {
		if v != vals[0] {
			return false
		}
	}
	return true
}

func powInt(base, exp int) int {
	if base <= 0 || exp <= 0 {
		return 0
	}
	out := 1
	for i := 0; i < exp; i++ {
		if out > universeEnumLimit/base {
			return 0 // 溢出保护：注数已远超任何合理上限，交由调用方按 0=未知处理
		}
		out *= base
	}
	return out
}

// ---------- 冷热出号一致性 ----------

// UnreachableHotColdOptions 返回冷热候选宇宙里理论上永远开不出的选项。
//
// 这类选项频次恒为 0，在「最热→最冷」排序上永远垫底，只要方案取冷号就会稳定押中它们。
// 典型：PK10 冠亚和的 2 / 20（两个名次不会相同，实际只有 3..19）、
// 组选和值的 0 / 27（仅豹子可组成，组选不可下）。
func UnreachableHotColdOptions(kind string, config []byte) []string {
	if len(config) == 0 {
		return nil
	}
	return unreachableOptionsForRule(parseSchemeConfig(kind, config, 0, 0).Play)
}

func unreachableOptionsForRule(rule playRule) []string {
	if !isHotColdAttributePlay(rule) {
		return nil
	}
	reachable, exact := reachableAttributeUniverse(rule)
	if !exact {
		return nil
	}
	ok := make(map[string]struct{}, len(reachable))
	for _, t := range reachable {
		ok[t] = struct{}{}
	}
	var bad []string
	for _, t := range attributeUniverse(rule) {
		if _, hit := ok[t]; !hit {
			bad = append(bad, t)
		}
	}
	return bad
}

// HotColdRouting 返回冷热出号实际走的分支与按玩法语义应走的分支。
//
// 只判「属性玩法」这条界线，两侧任一越界都会让计频统计错对象：
//   - 该走属性却走了按位：统计的是原始球号频次，而不是和值/尾数的频次（和值尾数即此类）
//   - 该走号池却走了属性：候选宇宙是一组文字选项，根本不是该玩法能下的内容
//
// 按位 / 号池 / 单式三者之间不做区分——冷热出号本就会把按位号池展开成整注单式，
// 那不是缺陷。actual 为空表示无从判断。
func HotColdRouting(kind string, config []byte) (actual, expected string) {
	if len(config) == 0 {
		return "", ""
	}
	rule := parseSchemeConfig(kind, config, 0, 0).Play
	shape := universeKindForRule(rule)
	if shape == "" {
		return "", "" // 形态未知，不下结论
	}
	actual = UniversePerPosition
	if isHotColdAttributePlay(rule) {
		actual = UniverseAttribute
	}
	expected = UniversePerPosition
	if shape == UniverseAttribute {
		expected = UniverseAttribute
	}
	return actual, expected
}
