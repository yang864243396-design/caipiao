package schemes

import (
	"context"
	"fmt"
	"math/rand"
	"sort"
	"strconv"
	"strings"

	"caipiao/backend/internal/db/sqlcdb"
)

// pickDecision 出号策略结果（v8 §4：出号体系与倍投体系独立）。
type pickDecision struct {
	Content   string // 本期下注内容（与方案内容同格式）
	Skip      bool   // 本期跳过不下注
	Direction string // 开某投某本局投向：pos / neg
}

// resolvePick 按运行类型决定本期下注内容（仅 kind=custom 分发；
// 反买/跟单与未识别类型走传统组轮换内容）。
func (w *Worker) resolvePick(
	ctx context.Context,
	cfg parsedSchemeConfig,
	inst sqlcdb.SchemeInstance,
	draw sqlcdb.LotteryDraw,
) pickDecision {
	if cfg.Contrary {
		if inv := strings.TrimSpace(cfg.ContraryPlan); inv != "" {
			return pickDecision{Content: inv}
		}
		return pickDecision{Content: cfg.GroupContent}
	}
	if cfg.Kind != "custom" || cfg.RunTypeID == "" {
		return pickDecision{Content: cfg.GroupContent}
	}
	switch cfg.RunTypeID {
	case RunTypeFixedRotate:
		return pickFixedRotate(cfg, inst)
	case RunTypeAdvFixedRotate:
		return pickJushuList(cfg, inst)
	case RunTypeAdvTriggerBet:
		return w.pickTriggerBet(ctx, cfg, inst, draw)
	case RunTypeHotColdWarm:
		return w.pickHotColdWarm(ctx, cfg, inst, draw)
	case RunTypeRandomDraw:
		return pickRandomDraw(cfg, inst)
	case RunTypeFixedNumber:
		return pickFixedNumber(cfg)
	case RunTypeBuiltinPlan:
		// 未物化的内置计画（尚未选择收藏方案）：跳过不下注
		return pickDecision{Skip: true}
	default:
		return pickDecision{Content: cfg.GroupContent}
	}
}

// AdvancePickAfterFormalSettlement 正式盘派奖后补推进出号游标。
//
// 正式盘下单时第三方尚未开奖（无中/未中结果），出号游标与投向被冻结（见 worker.go
// guajiReal 分支），因此定码轮换/高级定码轮换等运行类型会一直停在下单时的组，
// 表现为“每期都用第一组号码下注”。派奖拿到结果后在此按 advancePickState 相同语义
// 补推进，使各运行类型逐期切换下注内容。
//
// betContent 为该期实际下注内容（供冷热中奖轮换 / 随机出号保持等策略使用）。
func AdvancePickAfterFormalSettlement(
	kind string,
	definitionConfig []byte,
	inst sqlcdb.SchemeInstance,
	betContent string,
	hit bool,
) (pickIndex int32, currentPick string, lastDirection string) {
	groupIndex := 0
	if inst.RoundIndex > 0 {
		groupIndex = int(inst.RoundIndex)
	}
	cfg := parseSchemeConfig(kind, definitionConfig, int(inst.RoundIndex), groupIndex)
	cfg.Play = attachOddsBase(cfg.Play, inst.LotteryCode)
	dec := pickDecision{Content: betContent}
	if cfg.RunTypeID == RunTypeAdvTriggerBet && cfg.Trigger != nil {
		// 下单时投向未持久化（applyLastDirection=inst.LastDirection），此处按同一起点
		// 重算本期投向，再交由 advancePickState 写回状态机。
		dec.Direction = nextTriggerDirection(cfg.Trigger.Mode, inst.LastDirection)
	}
	return advancePickState(cfg, inst, dec, hit)
}

// advancePickState 结算后推进出号游标（写回 pick_index / current_pick / last_direction）。
func advancePickState(
	cfg parsedSchemeConfig,
	inst sqlcdb.SchemeInstance,
	dec pickDecision,
	hit bool,
) (pickIndex int32, currentPick string, lastDirection string) {
	pickIndex = inst.PickIndex
	currentPick = inst.CurrentPick
	lastDirection = inst.LastDirection
	if cfg.Contrary || cfg.Kind != "custom" || cfg.RunTypeID == "" {
		return pickIndex, currentPick, lastDirection
	}
	switch cfg.RunTypeID {
	case RunTypeFixedRotate:
		n := cfg.GroupCount
		if n <= 0 {
			n = 1
		}
		pickIndex = (inst.PickIndex + 1) % int32(n)
	case RunTypeAdvFixedRotate:
		// 优先按本期实际下注内容定位局号，避免游标被并发/回头复位打乱后跳错
		row := jushuRowForContent(cfg.Jushu, dec.Content, int(inst.PickIndex))
		next := row.AfterMiss
		if hit {
			next = row.AfterHit
		}
		if !jushuExists(cfg.Jushu, next) {
			next = firstJu(cfg.Jushu)
		}
		pickIndex = int32(next)
	case RunTypeAdvTriggerBet:
		if dec.Direction != "" {
			lastDirection = dec.Direction
		}
	case RunTypeHotColdWarm:
		// 换号：清空 current_pick，下期按统计期数+出号类型+容错重新取码（非池内 +1）
		strategy := hotColdStrategy(cfg.HotCold)
		switch strategy {
		case "every":
			currentPick = ""
		case "after_hit":
			if hit {
				currentPick = ""
			} else {
				currentPick = dec.Content
			}
		case "after_miss":
			if hit {
				currentPick = dec.Content
			} else {
				currentPick = ""
			}
		default: // keep 不换号：锁定本期内容
			currentPick = dec.Content
		}
	case RunTypeRandomDraw:
		strategy := "every"
		if cfg.Random != nil && cfg.Random.Strategy != "" {
			strategy = cfg.Random.Strategy
		}
		switch strategy {
		case "keep":
			currentPick = dec.Content
		case "after_hit": // 中后换：命中则下期重新随机
			if hit {
				currentPick = ""
			} else {
				currentPick = dec.Content
			}
		case "after_miss": // 挂后换：未中则下期重新随机
			if hit {
				currentPick = dec.Content
			} else {
				currentPick = ""
			}
		default: // every 每期换
			currentPick = ""
		}
	}
	return pickIndex, currentPick, lastDirection
}

// ---------- 定码轮换 ----------

func pickFixedRotate(cfg parsedSchemeConfig, inst sqlcdb.SchemeInstance) pickDecision {
	if len(cfg.Groups) == 0 {
		return pickDecision{Content: cfg.GroupContent}
	}
	idx := int(inst.PickIndex) % len(cfg.Groups)
	if idx < 0 {
		idx = 0
	}
	return pickDecision{Content: cfg.Groups[idx]}
}

// ---------- 高级定码轮换（局数列表） ----------

func currentJushuRow(rows []jushuRow, cur int) jushuRow {
	if len(rows) == 0 {
		return jushuRow{Ju: 1, AfterHit: 1, AfterMiss: 1}
	}
	if cur <= 0 {
		cur = rows[0].Ju
	}
	for _, r := range rows {
		if r.Ju == cur {
			return r
		}
	}
	return rows[0]
}

// jushuRowForContent 按本期实际内容匹配局；匹配失败再回退 pick_index。
func jushuRowForContent(rows []jushuRow, content string, pickIndex int) jushuRow {
	want := normalizeSchemeGroupContent(content)
	if strings.TrimSpace(want) != "" {
		for _, r := range rows {
			if normalizeSchemeGroupContent(r.Content) == want {
				return r
			}
		}
	}
	return currentJushuRow(rows, pickIndex)
}

func jushuExists(rows []jushuRow, ju int) bool {
	for _, r := range rows {
		if r.Ju == ju {
			return true
		}
	}
	return false
}

func firstJu(rows []jushuRow) int {
	if len(rows) == 0 {
		return 1
	}
	return rows[0].Ju
}

func pickJushuList(cfg parsedSchemeConfig, inst sqlcdb.SchemeInstance) pickDecision {
	row := currentJushuRow(cfg.Jushu, int(inst.PickIndex))
	if strings.TrimSpace(row.Content) == "" {
		return pickDecision{Content: cfg.GroupContent}
	}
	return pickDecision{Content: row.Content}
}

// ---------- 固定取码 ----------

// pickFixedNumber 固定取码：每期复投 schemeGroups[0] 指定号码。
func pickFixedNumber(cfg parsedSchemeConfig) pickDecision {
	if len(cfg.Groups) > 0 && strings.TrimSpace(cfg.Groups[0]) != "" {
		return pickDecision{Content: cfg.Groups[0]}
	}
	return pickDecision{Content: cfg.GroupContent}
}

// ---------- 随机出号 ----------

func pickRandomDraw(cfg parsedSchemeConfig, inst sqlcdb.SchemeInstance) pickDecision {
	if strings.TrimSpace(inst.CurrentPick) != "" {
		return pickDecision{Content: inst.CurrentPick}
	}
	// 单式/组选单式：整注随机——随机抽 N 个完整组合（对齐竞品 GetCombinaList + 抽样模型）。
	if isWholeTicketRandom(cfg.Play) {
		n := 1
		if cfg.Random != nil && len(cfg.Random.Counts) > 0 && cfg.Random.Counts[0] > 0 {
			n = cfg.Random.Counts[0]
		}
		return pickDecision{Content: randomWholeTickets(cfg.Play, n)}
	}
	// 组合家族（组三/组六/组选N/组选复式）：号码池随机——随机选 K 个号组成号码池，
	// 由玩法评估按组选口径展开注数（与手动组选复式内容格式一致）。
	if isZuxuanPoolRandom(cfg.Play) {
		k := 0
		if cfg.Random != nil && len(cfg.Random.Counts) > 0 {
			k = cfg.Random.Counts[0]
		}
		return pickDecision{Content: randomZuxuanPool(cfg.Play, k)}
	}
	// 属性/聚合家族（大小单双/龙虎/特殊号/庄闲/和值/跨度/不定位/包胆）：
	// 从该玩法的选项宇宙随机抽 K 个（对齐竞品 GetCombinaList 宇宙 + 抽样）。
	if isAttributeRandom(cfg.Play) {
		k := 0
		if cfg.Random != nil && len(cfg.Random.Counts) > 0 {
			k = cfg.Random.Counts[0]
		}
		return pickDecision{Content: randomAttributeContent(cfg.Play, k)}
	}
	positions := playPositionCount(cfg.Play)
	lines := make([]string, 0, positions)
	for i := 0; i < positions; i++ {
		count := 1
		if cfg.Random != nil && i < len(cfg.Random.Counts) {
			count = cfg.Random.Counts[i]
		}
		lines = append(lines, randomDigits(cfg.Play, count))
	}
	return pickDecision{Content: strings.Join(lines, "\n")}
}

// isWholeTicketRandom 判定为"整注型"玩法（直选单式/组选单式）——随机产号需抽完整组合，
// 而非按位产号池。
func isWholeTicketRandom(rule playRule) bool {
	bm := strings.ToLower(strings.TrimSpace(rule.BetMode))
	sub := strings.ToLower(strings.TrimSpace(rule.SubPlayID))
	switch bm {
	case "danshi", "zhixuan_ds", "zuxuan_ds", "hunhe":
		return true
	}
	switch sub {
	case "zhixuan_ds", "zuxuan_ds":
		return true
	}
	return false
}

// isAttributeRandom 判定为"属性/聚合型"玩法（大小单双/龙虎/特殊号/庄闲/和值/跨度/不定位/包胆）——
// 随机产号为"从选项宇宙抽 K 个"，非按位号池、非整注单式。
func isAttributeRandom(rule playRule) bool {
	switch strings.ToLower(strings.TrimSpace(rule.BetMode)) {
	case "daxiao", "danshuang", "dxds", "zhuangxian",
		"longhu", "longhuhe", "longhubao", "teshu",
		"hezhi", "kuadu", "budingwei", "baodan":
		return true
	}
	return false
}

// attributeUniverse 属性/聚合玩法的合法选项宇宙（供随机抽样）。数字池型（不定位/包胆）返回 nil，另行处理。
func attributeUniverse(rule playRule) []string {
	switch strings.ToLower(strings.TrimSpace(rule.BetMode)) {
	case "daxiao":
		return []string{"大", "小"}
	case "danshuang":
		return []string{"单", "双"}
	case "dxds":
		return []string{"大", "小", "单", "双"}
	case "zhuangxian":
		return []string{"庄", "闲"}
	case "longhu":
		return []string{"龙", "虎"}
	case "longhuhe":
		return []string{"龙", "虎", "和"}
	case "longhubao":
		return []string{"龙", "虎", "豹"}
	case "teshu":
		if rule.PlayTemplate == "pc28_std" {
			return []string{"豹子", "对子", "顺子", "极大", "极小"}
		}
		return []string{"豹子", "对子", "顺子"}
	case "hezhi":
		min, max := ruleNumberPool(rule)
		segLen := rule.SegmentLen
		if segLen < 1 {
			segLen = 1
		}
		lo, hi := segLen*min, segLen*max
		// 组选和值排除仅豹子（各位同码）可组成的极值和：三星 0(000)/27(999) → 1..26，前二 0/18 → 1..17。
		if rule.HezhiZuxuan && hi-lo >= 2 {
			lo, hi = lo+1, hi-1
		}
		out := make([]string, 0, hi-lo+1)
		for v := lo; v <= hi; v++ {
			out = append(out, strconv.Itoa(v))
		}
		return out
	case "kuadu":
		min, max := ruleNumberPool(rule)
		out := make([]string, 0, max-min+1)
		for v := 0; v <= max-min; v++ {
			out = append(out, strconv.Itoa(v))
		}
		return out
	}
	return nil
}

// randomAttributeContent 从属性/聚合玩法的选项宇宙随机抽 k 个（去重、逗号分隔）。
// 不定位/包胆为数字池型：抽 k 个不重复号码（不定位下限=选码位数）。
func randomAttributeContent(rule playRule, k int) string {
	bm := strings.ToLower(strings.TrimSpace(rule.BetMode))
	if bm == "budingwei" || bm == "baodan" {
		pool := playNumberPool(rule)
		if len(pool) == 0 {
			return ""
		}
		minK := 1
		if bm == "budingwei" {
			minK = budingweiNeedCount(rule.CatalogSubID)
		}
		if k < minK {
			k = minK
		}
		if k > len(pool) {
			k = len(pool)
		}
		idx := rand.Perm(len(pool))[:k]
		sort.Ints(idx)
		out := make([]string, 0, k)
		for _, i := range idx {
			out = append(out, pool[i])
		}
		return strings.Join(out, ",")
	}
	universe := attributeUniverse(rule)
	if len(universe) == 0 {
		return ""
	}
	if k < 1 {
		k = 1
	}
	if k > len(universe) {
		k = len(universe)
	}
	idx := rand.Perm(len(universe))[:k]
	sort.Ints(idx)
	out := make([]string, 0, k)
	for _, i := range idx {
		out = append(out, universe[i])
	}
	return strings.Join(out, ",")
}

// isZuxuanPoolRandom 判定为"组选号码池型"玩法（组三/组六/组选N/组选复式）——
// 随机产号为"选 K 个号组成号码池"，非按位、也非整注单式。
func isZuxuanPoolRandom(rule playRule) bool {
	if isWholeTicketRandom(rule) {
		return false
	}
	bm := strings.ToLower(strings.TrimSpace(rule.BetMode))
	switch bm {
	case "zu3", "zu6", "zu24", "zu12", "zu60", "zu30", "zu120":
		return true
	}
	cat := strings.ToLower(rule.SubPlayID + " " + rule.CatalogSubID)
	if cat == "" {
		return false
	}
	if strings.Contains(cat, "zuxuan_fs") {
		return true
	}
	// 兼容 zu3/zu6/zuxuan/zu24… 标记出现在子玩法/目录 id 中
	for _, k := range []string{"zu3", "zu6", "zu24", "zu12", "zu60", "zu30", "zu120", "zuxuan"} {
		if strings.Contains(cat, k) {
			return true
		}
	}
	return false
}

// randomZuxuanPool 随机选 k 个不重复号码组成组选号码池（升序，逗号分隔）。
// k 下限为段长（保证至少 1 注），上限为号池大小。
func randomZuxuanPool(rule playRule, k int) string {
	pool := playNumberPool(rule)
	if len(pool) == 0 {
		return ""
	}
	minK := playPositionCount(rule)
	if minK < 2 {
		minK = 2
	}
	if k < minK {
		k = minK
	}
	if k > len(pool) {
		k = len(pool)
	}
	idx := rand.Perm(len(pool))[:k]
	sort.Ints(idx)
	out := make([]string, 0, k)
	for _, i := range idx {
		out = append(out, pool[i])
	}
	return strings.Join(out, ",")
}

// randomWholeTickets 随机抽 n 个完整组合（每位随机取一个号拼成一注），去重。
// 组选单式（zuxuan_ds）内位号升序归一，按组合去重；直选单式保留位序。
// 内容格式与 evaluateZhixuanDanshi 兼容：逗号分隔的定长 token。
func randomWholeTickets(rule playRule, n int) string {
	positions := playPositionCount(rule)
	if positions <= 0 {
		positions = 1
	}
	pool := playNumberPool(rule)
	if len(pool) == 0 {
		return ""
	}
	if n < 1 {
		n = 1
	}
	// 上限保护：最多 200 注
	const maxN = 200
	if n > maxN {
		n = maxN
	}
	bm := strings.ToLower(strings.TrimSpace(rule.BetMode))
	sub := strings.ToLower(strings.TrimSpace(rule.SubPlayID))
	isHunhe := bm == "hunhe"
	// 组选单式 / 混合组选单式：位号升序归一（按组合去重）；混合额外排除豹子（全同号）。
	isZuxuan := isHunhe || bm == "zuxuan_ds" || sub == "zuxuan_ds"
	seen := make(map[string]struct{}, n)
	out := make([]string, 0, n)
	for attempts := 0; len(out) < n && attempts < n*100+100; attempts++ {
		digits := make([]string, positions)
		for p := 0; p < positions; p++ {
			digits[p] = pool[rand.Intn(len(pool))]
		}
		key := strings.Join(digits, "")
		if isZuxuan {
			sorted := append([]string(nil), digits...)
			sort.Strings(sorted)
			key = strings.Join(sorted, "")
			digits = sorted
		}
		if isHunhe && allSameTokens(digits) {
			// 混合组选单式排除豹子（全同号）
			continue
		}
		if _, dup := seen[key]; dup {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, strings.Join(digits, ""))
	}
	return strings.Join(out, ",")
}

// allSameTokens 判定所有 token 相同（豹子/全同号）。
func allSameTokens(tokens []string) bool {
	if len(tokens) <= 1 {
		return false
	}
	for i := 1; i < len(tokens); i++ {
		if tokens[i] != tokens[0] {
			return false
		}
	}
	return true
}

func playPositionCount(rule playRule) int {
	if len(rule.SegmentPos) > 0 {
		return len(rule.SegmentPos)
	}
	if rule.SegmentLen > 1 {
		return rule.SegmentLen
	}
	return 1
}

func playNumberPool(rule playRule) []string {
	min, max := ruleNumberPool(rule)
	if max-min > 98 {
		max = min + 98
	}
	pool := make([]string, 0, max-min+1)
	for v := min; v <= max; v++ {
		// 11 选 5（max==11）补零；六合/PK10/时时彩保持与开奖球/历史池一致的无补零或单位数形态
		if max == 11 {
			pool = append(pool, fmt.Sprintf("%02d", v))
		} else {
			pool = append(pool, strconv.Itoa(v))
		}
	}
	return pool
}

func randomDigits(rule playRule, count int) string {
	pool := playNumberPool(rule)
	if count < 1 {
		count = 1
	}
	if count > len(pool) {
		count = len(pool)
	}
	idx := rand.Perm(len(pool))[:count]
	sort.Ints(idx)
	out := make([]string, 0, count)
	for _, i := range idx {
		out = append(out, pool[i])
	}
	return strings.Join(out, ",")
}

// ---------- 冷热出号 ----------

func hotColdStrategy(hc *hotColdWarmCfg) string {
	if hc == nil {
		return "keep"
	}
	if hc.Strategy != "" {
		return hc.Strategy
	}
	if hc.WinRotate {
		return "after_hit"
	}
	return "keep"
}

func (w *Worker) pickHotColdWarm(
	ctx context.Context,
	cfg parsedSchemeConfig,
	inst sqlcdb.SchemeInstance,
	draw sqlcdb.LotteryDraw,
) pickDecision {
	if cur := strings.TrimSpace(inst.CurrentPick); cur != "" && !hotColdPickNeedsRebuild(cfg, cur) {
		return pickDecision{Content: cur}
	}
	periods := 20
	if cfg.HotCold != nil && cfg.HotCold.TotalPeriods > 0 {
		periods = cfg.HotCold.TotalPeriods
	}
	draws := w.recentDrawBalls(ctx, inst.LotteryCode, draw.IssueNo, periods)
	content := buildHotColdPickContent(cfg, draws)
	if strings.TrimSpace(content) == "" {
		if cfg.HotCold != nil && len(cfg.HotCold.Pool) > 0 {
			content = strings.Join(cfg.HotCold.Pool, "\n")
		} else {
			content = cfg.GroupContent
		}
	}
	return pickDecision{Content: normalizeZhixuanDanshiContent(cfg.Play, content)}
}

// pickHotColdWarmFromDraws 供预览/单测：无 DB 时用传入开奖序列取码。
func pickHotColdWarmFromDraws(cfg parsedSchemeConfig, inst sqlcdb.SchemeInstance, draws [][]string) pickDecision {
	if cur := strings.TrimSpace(inst.CurrentPick); cur != "" && !hotColdPickNeedsRebuild(cfg, cur) {
		return pickDecision{Content: normalizeZhixuanDanshiContent(cfg.Play, cur)}
	}
	content := buildHotColdPickContent(cfg, draws)
	if strings.TrimSpace(content) == "" {
		if cfg.HotCold != nil && len(cfg.HotCold.Pool) > 0 {
			content = strings.Join(cfg.HotCold.Pool, "\n")
		} else {
			content = cfg.GroupContent
		}
	}
	return pickDecision{Content: normalizeZhixuanDanshiContent(cfg.Play, content)}
}

// hotColdPickNeedsRebuild 多位面板却保了单位内容（无换行）时强制重取，避免旧引擎单号锁死。
// 直选单式展开后的整注串（如 "555" / "432,435"）无换行，属合法保号，不重取。
func hotColdPickNeedsRebuild(cfg parsedSchemeConfig, currentPick string) bool {
	if playPositionCount(cfg.Play) <= 1 {
		return false
	}
	if strings.Contains(currentPick, "\n") {
		return false
	}
	seg := cfg.Play.SegmentLen
	if seg <= 0 {
		seg = playPositionCount(cfg.Play)
	}
	if seg > 1 && len(parseNumberTokens(currentPick, seg)) > 0 {
		return false
	}
	return true
}

// recentDrawBalls 取当期之前最近 N 期开奖球（不含当期）。
func (w *Worker) recentDrawBalls(ctx context.Context, lotteryCode, currentIssue string, periods int) [][]string {
	if w == nil || w.q == nil || periods <= 0 {
		return nil
	}
	if periods > 500 {
		periods = 500
	}
	rows, err := w.q.ListLotteryDraws(ctx, sqlcdb.ListLotteryDrawsParams{
		LotteryCode: lotteryCode,
		RowLimit:    int32(periods + 8),
	})
	if err != nil || len(rows) == 0 {
		return nil
	}
	out := make([][]string, 0, periods)
	for _, r := range rows {
		if currentIssue != "" && (r.IssueNo == currentIssue || r.IssueNo >= currentIssue) {
			continue
		}
		balls := sqlcdb.ParseDrawBalls(r.Balls)
		if len(balls) == 0 {
			continue
		}
		out = append(out, balls)
		if len(out) >= periods {
			break
		}
	}
	return out
}

// buildHotColdPickContent 按「名次 + 起点偏移」取码（对齐富联冷热出号逆向）：
//
//	每期用最近 N 期把号码按出现次数「最热→最冷」排序 → 出号类型定位到热端/冷端 →
//	容错=起点偏移跳过该端最极端的前 fault 名 → 连续取 pickCount 个名次的号。
//
// 混合模式：hc.Pool[i] 非空的位用手选号码覆盖，其余位按名次自动取号。
func buildHotColdPickContent(cfg parsedSchemeConfig, draws [][]string) string {
	hc := cfg.HotCold
	if hc == nil {
		return ""
	}
	fault := hc.FaultCount // 起点偏移（0=从最极端开始）
	if fault < 0 {
		fault = 0
	}
	if fault > 9 {
		fault = 9
	}
	count := hc.PickCount // 每位取几个名次
	if count < 1 {
		count = 1
	}
	types := normalizeHotColdPickTypes(hc.PickTypes)
	wantHot, wantCold := hotColdWants(types)
	pool := playNumberPool(cfg.Play)

	// 属性家族（大小单双/龙虎/和值等）
	if isHotColdAttributePlay(cfg.Play) {
		if manual := hotColdManualAt(hc.Pool, 0); manual != "" {
			return manual
		}
		if len(types) == 0 || len(draws) == 0 {
			return ""
		}
		res := HotColdWarmAttributeTiers(cfg.Play, draws)
		full := append(append([]string{}, res.Hot...), res.Cold...)
		return strings.Join(pickTokensByRank(full, wantHot, wantCold, fault, count), ",")
	}
	// 号码整体频次（组选/不定位/包胆）
	if isHotColdDigitOverall(cfg.Play) {
		if manual := hotColdManualAt(hc.Pool, 0); manual != "" {
			return manual
		}
		if len(types) == 0 || len(draws) == 0 {
			return ""
		}
		hot, cold := hotColdWarmTiersOverall(draws, cfg.Play, pool)
		full := append(append([]string{}, hot...), cold...)
		return strings.Join(pickTokensByRank(full, wantHot, wantCold, fault, count), ",")
	}
	// 按位型：逐位取号，支持手动覆盖
	n := playPositionCount(cfg.Play)
	lines := make([]string, n)
	filled := 0
	for i := 0; i < n; i++ {
		if manual := hotColdManualAt(hc.Pool, i); manual != "" {
			lines[i] = manual
			filled++
			continue
		}
		if len(types) == 0 || len(draws) == 0 {
			continue
		}
		pos := hotColdPositionIdx(cfg.Play, i)
		hot, _, cold := hotColdWarmTiers(draws, pos, pool)
		full := append(append([]string{}, hot...), cold...)
		picked := pickTokensByRank(full, wantHot, wantCold, fault, count)
		lines[i] = strings.Join(picked, ",")
		if lines[i] != "" {
			filled++
		}
	}
	if filled == 0 {
		return ""
	}
	return strings.Join(lines, "\n")
}

// hotColdWants 出号类型 → 是否取热端/冷端。
func hotColdWants(types []string) (wantHot, wantCold bool) {
	for _, t := range types {
		if t == "hot" {
			wantHot = true
		}
		if t == "cold" {
			wantCold = true
		}
	}
	return wantHot, wantCold
}

// hotColdManualAt 取某位的手动覆盖号码（空=该位自动取号）。
func hotColdManualAt(pool []string, i int) string {
	if i < 0 || i >= len(pool) {
		return ""
	}
	return strings.TrimSpace(pool[i])
}

// pickTokensByRank 在「最热→最冷」全序 full 上，按出号类型 + 起点偏移 + 名次个数取号。
//   - 热端：从 full[offset] 起连续取 count 个（offset=1,count=2 → 第2、第3热）。
//   - 冷端：从最冷 full[n-1] 起、跳过 offset 名后连续取 count 个（冷号在前）。
//   - 同时选热+冷：热端取号在前、冷端在后，去重保序。
func pickTokensByRank(full []string, wantHot, wantCold bool, offset, count int) []string {
	n := len(full)
	if n == 0 {
		return nil
	}
	if count < 1 {
		count = 1
	}
	if offset < 0 {
		offset = 0
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, count*2)
	add := func(idx int) {
		if idx < 0 || idx >= n {
			return
		}
		t := strings.TrimSpace(full[idx])
		if t == "" {
			return
		}
		if _, dup := seen[t]; dup {
			return
		}
		seen[t] = struct{}{}
		out = append(out, t)
	}
	if wantHot {
		for k := 0; k < count; k++ {
			add(offset + k)
		}
	}
	if wantCold {
		for k := 0; k < count; k++ {
			add(n - 1 - offset - k)
		}
	}
	return out
}

func normalizeHotColdPickTypes(raw []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, 2)
	for _, t := range raw {
		t = strings.ToLower(strings.TrimSpace(t))
		if (t == "hot" || t == "cold") && !seen[t] {
			seen[t] = true
			out = append(out, t)
		}
	}
	return out
}

func isHotColdDigitOverall(rule playRule) bool {
	bm := strings.ToLower(strings.TrimSpace(rule.BetMode))
	switch bm {
	case "zu3", "zu6", "zu24", "zu12", "zu60", "zu30", "zu120", "budingwei", "baodan":
		return true
	}
	sub := strings.ToLower(strings.TrimSpace(rule.SubPlayID) + " " + strings.TrimSpace(rule.CatalogSubID))
	return strings.Contains(sub, "zuxuan") || strings.Contains(sub, "zu3") || strings.Contains(sub, "zu6") ||
		strings.Contains(sub, "budingwei") || strings.Contains(sub, "baodan")
}

func isHotColdAttributePlay(rule playRule) bool {
	switch strings.ToLower(strings.TrimSpace(rule.BetMode)) {
	case "daxiao", "danshuang", "dxds", "zhuangxian",
		"longhu", "longhuhe", "longhubao", "teshu",
		"hezhi", "kuadu":
		return true
	}
	return false
}

func hotColdPositionIdx(rule playRule, lineIdx int) int {
	if len(rule.SegmentPos) > 0 {
		if lineIdx >= 0 && lineIdx < len(rule.SegmentPos) {
			return rule.SegmentPos[lineIdx]
		}
		return rule.SegmentPos[0]
	}
	if rule.SegmentLen <= 1 {
		return rule.PositionIdx
	}
	return rule.SegmentStart + lineIdx
}

// hotColdWarmTiersOverall 跨位合并频次后二等分热/冷（组选/不定位/包胆）。
func hotColdWarmTiersOverall(draws [][]string, rule playRule, pool []string) (hot, cold []string) {
	counts := make(map[string]int, len(pool))
	positions := playPositionCount(rule)
	for _, balls := range draws {
		for i := 0; i < positions; i++ {
			pos := hotColdPositionIdx(rule, i)
			if pos >= 0 && pos < len(balls) {
				counts[strings.TrimSpace(balls[pos])]++
			}
		}
	}
	sorted := append([]string(nil), pool...)
	sort.SliceStable(sorted, func(i, j int) bool {
		if counts[sorted[i]] != counts[sorted[j]] {
			return counts[sorted[i]] > counts[sorted[j]]
		}
		return sorted[i] < sorted[j]
	})
	n := len(sorted)
	half := (n + 1) / 2
	if half > n {
		half = n
	}
	return sorted[:half], sorted[half:]
}

// hotColdWarmTiers 按最近 N 期频次排序二等分（热/冷；对齐 v6 第三方，无温档）。
// warm 恒为空切片，保留返回值以兼容旧调用方。
func hotColdWarmTiers(draws [][]string, positionIdx int, pool []string) (hot, warm, cold []string) {
	counts := make(map[string]int, len(pool))
	for _, balls := range draws {
		if positionIdx >= 0 && positionIdx < len(balls) {
			counts[strings.TrimSpace(balls[positionIdx])]++
		}
	}
	sorted := append([]string(nil), pool...)
	sort.SliceStable(sorted, func(i, j int) bool {
		if counts[sorted[i]] != counts[sorted[j]] {
			return counts[sorted[i]] > counts[sorted[j]]
		}
		return sorted[i] < sorted[j]
	})
	n := len(sorted)
	half := (n + 1) / 2
	if half > n {
		half = n
	}
	hot = sorted[:half]
	warm = []string{}
	cold = sorted[half:]
	return hot, warm, cold
}

// ---------- 高级开某投某 ----------

func (w *Worker) pickTriggerBet(
	ctx context.Context,
	cfg parsedSchemeConfig,
	inst sqlcdb.SchemeInstance,
	draw sqlcdb.LotteryDraw,
) pickDecision {
	prevBalls := w.previousDrawBalls(ctx, inst.LotteryCode, draw)
	return resolveTriggerBetDecision(cfg, prevBalls, inst.LastDirection)
}

// resolveTriggerBetDecision 高级开某投某出号。
// 定位胆多选位：每位按该位上期开奖各自查映射下注（例：上期 17232、选万/百/个 → 1,,2,,2），
// 不可把某一命中行的号码复制到所有位。
func resolveTriggerBetDecision(cfg parsedSchemeConfig, prevBalls []string, lastDirection string) pickDecision {
	if cfg.Trigger == nil || len(cfg.Trigger.Rows) == 0 {
		return pickDecision{Skip: true}
	}
	enabled := make([]triggerRow, 0, len(cfg.Trigger.Rows))
	for _, r := range cfg.Trigger.Rows {
		if r.Enabled {
			enabled = append(enabled, r)
		}
	}
	if len(enabled) == 0 {
		return pickDecision{Skip: true}
	}

	direction := nextTriggerDirection(cfg.Trigger.Mode, lastDirection)

	// 定位胆多选位 / 前三直选复式等按位玩法：按位独立映射
	if triggerBetUsesPosition(cfg.Play) {
		if isDingweiTriggerPlay(cfg.Play) && !cfg.Trigger.HasPosition {
			// 旧定位胆配置未写 positionIdxs：仍走单行映射 + 默认位编排
		} else {
			return pickTriggerBetPerPosition(cfg, enabled, prevBalls, direction)
		}
	}

	// 龙虎 / PC28 等：整期一个开出条件 → 一行映射
	row := enabled[0] // Q4c：无匹配走启用第 1 行
	if len(prevBalls) > 0 {
		for _, r := range enabled {
			if triggerOpenMatches(cfg.Play, prevBalls, r.Open) {
				row = r
				break
			}
		}
	}
	content, dir := triggerRowPickContent(row, direction)
	if content == "" {
		return pickDecision{Skip: true}
	}
	return pickDecision{Content: content, Direction: dir}
}

func pickTriggerBetPerPosition(
	cfg parsedSchemeConfig,
	enabled []triggerRow,
	prevBalls []string,
	direction string,
) pickDecision {
	// 前三直选复式等：按玩法段出多行（万\n千\n百），而非五星定位胆稀疏行
	if !isDingweiTriggerPlay(cfg.Play) && cfg.Play.SegmentLen >= 2 {
		return pickTriggerBetPerSegment(cfg, enabled, prevBalls, direction)
	}
	positions := 5
	if cfg.Play.PlayTemplate == "pk10_std" {
		positions = 10
	}
	idxs := cfg.Trigger.PositionIdxs
	if len(idxs) == 0 {
		idxs = []int{cfg.Play.PositionIdx}
	}
	lines := make([]string, positions)
	filled := 0
	outDir := direction
	for _, idx := range idxs {
		if idx < 0 {
			idx = 0
		}
		if idx >= positions {
			idx = positions - 1
		}
		row := enabled[0]
		if idx < len(prevBalls) {
			open := normalizeTriggerToken(strings.TrimSpace(prevBalls[idx]))
			if r, ok := findEnabledTriggerRowByOpen(enabled, open); ok {
				row = r
			}
		}
		content, dir := triggerRowPickContent(row, direction)
		if content == "" {
			continue
		}
		lines[idx] = content
		filled++
		outDir = dir
	}
	if filled == 0 {
		return pickDecision{Skip: true}
	}
	// 单位：仍压到选定投注位（兼容仅选一位）
	if filled == 1 && len(idxs) == 1 {
		return pickDecision{
			Content:   layoutTriggerBetDingweiContent(cfg, lines[idxs[0]]),
			Direction: outDir,
		}
	}
	return pickDecision{Content: strings.Join(lines, "\n"), Direction: outDir}
}

// pickTriggerBetPerSegment 直选复式等：段内每位按绝对球位开奖查映射，输出 segmentLen 行。
// 未勾选的段内位用启用第 1 行正/反投补齐，避免复式缺位。
func pickTriggerBetPerSegment(
	cfg parsedSchemeConfig,
	enabled []triggerRow,
	prevBalls []string,
	direction string,
) pickDecision {
	segStart := cfg.Play.SegmentStart
	segLen := cfg.Play.SegmentLen
	if segLen <= 0 {
		segLen = 1
	}
	selected := map[int]bool{}
	for _, abs := range cfg.Trigger.PositionIdxs {
		selected[abs] = true
	}
	if len(selected) == 0 {
		for i := 0; i < segLen; i++ {
			selected[segStart+i] = true
		}
	}
	lines := make([]string, segLen)
	filled := 0
	outDir := direction
	for rel := 0; rel < segLen; rel++ {
		abs := segStart + rel
		row := enabled[0]
		if selected[abs] && abs < len(prevBalls) {
			open := normalizeTriggerToken(strings.TrimSpace(prevBalls[abs]))
			if r, ok := findEnabledTriggerRowByOpen(enabled, open); ok {
				row = r
			}
		}
		content, dir := triggerRowPickContentAt(row, direction, rel, segLen)
		if content == "" {
			continue
		}
		lines[rel] = content
		filled++
		outDir = dir
	}
	if filled == 0 {
		return pickDecision{Skip: true}
	}
	return pickDecision{Content: strings.Join(lines, "\n"), Direction: outDir}
}

func findEnabledTriggerRowByOpen(enabled []triggerRow, open string) (triggerRow, bool) {
	open = normalizeTriggerToken(open)
	if open == "" {
		return triggerRow{}, false
	}
	for _, r := range enabled {
		if normalizeTriggerToken(r.Open) == open {
			return r, true
		}
	}
	return triggerRow{}, false
}

// triggerRowPickContent 按投向取正/反投；反投为空时退回正投。
func triggerRowPickContent(row triggerRow, direction string) (content, dir string) {
	return triggerRowPickContentAt(row, direction, 0, 1)
}

// triggerRowPickContentAt 取某一段位的正/反投。
// pos/neg 以换行分位（万\n千\n百）；单行旧值则各位共用。
func triggerRowPickContentAt(row triggerRow, direction string, rel, segLen int) (content, dir string) {
	dir = direction
	pick := func(raw string) string {
		raw = strings.ReplaceAll(raw, "\r\n", "\n")
		raw = strings.ReplaceAll(raw, "\r", "\n")
		parts := strings.Split(raw, "\n")
		if segLen <= 1 || len(parts) <= 1 {
			return strings.TrimSpace(raw)
		}
		if rel >= 0 && rel < len(parts) {
			return strings.TrimSpace(parts[rel])
		}
		return ""
	}
	content = pick(row.Pos)
	if direction == "neg" {
		content = pick(row.Neg)
	}
	if content == "" {
		if alt := pick(row.Pos); alt != "" {
			return alt, "pos"
		}
		return "", direction
	}
	return content, dir
}

// nextTriggerDirection 投向状态机（Q4b：按上一局投向交替）。
func nextTriggerDirection(mode, last string) string {
	switch mode {
	case "always_neg":
		return "neg"
	case "alt_pos_first": // 前正后反：自正投起始，逐局交替
		if last == "pos" {
			return "neg"
		}
		if last == "neg" {
			return "pos"
		}
		return "pos"
	case "alt_neg_first": // 前反后正
		if last == "neg" {
			return "pos"
		}
		if last == "pos" {
			return "neg"
		}
		return "neg"
	default: // always_pos
		return "pos"
	}
}

// previousDrawBalls 取上一期开奖球（不含当期）。
// 必须按期号精确取「issue_no < 当期」的最近一期；不可用 drawn_at 最近 N 条扫描，
// 否则上期尚未进库/不在窗口内时会误用更早开奖（开某投某会跟错位）。
func (w *Worker) previousDrawBalls(ctx context.Context, lotteryCode string, draw sqlcdb.LotteryDraw) []string {
	if w.q != nil && strings.TrimSpace(draw.IssueNo) != "" {
		prev, err := w.q.GetPreviousLotteryDrawByIssue(ctx, sqlcdb.GetPreviousLotteryDrawByIssueParams{
			LotteryCode: lotteryCode,
			IssueNo:     draw.IssueNo,
		})
		if err == nil {
			return sqlcdb.ParseDrawBalls(prev.Balls)
		}
	}
	rows, err := w.q.ListLotteryDraws(ctx, sqlcdb.ListLotteryDrawsParams{
		LotteryCode: lotteryCode,
		RowLimit:    64,
	})
	if err != nil || len(rows) == 0 {
		return nil
	}
	var bestIssue string
	var bestBalls []byte
	for _, r := range rows {
		if r.IssueNo == draw.IssueNo || r.IssueNo >= draw.IssueNo {
			continue
		}
		if bestIssue == "" || r.IssueNo > bestIssue {
			bestIssue = r.IssueNo
			bestBalls = r.Balls
		}
	}
	if bestIssue == "" {
		return nil
	}
	return sqlcdb.ParseDrawBalls(bestBalls)
}

func isLonghuPlay(rule playRule) bool {
	bm := strings.ToLower(strings.TrimSpace(rule.BetMode))
	if bm == "longhubao" {
		return false
	}
	if bm == "longhu" || bm == "longhuhe" {
		return true
	}
	return strings.TrimSpace(rule.PlayTypeID) == playTypeLonghu
}

// longhuResult 计算上期龙虎结果（复用 longhuPositions 的位映射；无子玩法时取万 vs 个）。
func longhuResult(rule playRule, balls []string) string {
	p1, p2, _ := longhuPositions(rule.CatalogSubID)
	if p1 < 0 || p2 < 0 {
		p1, p2 = 0, len(balls)-1
	}
	if p1 >= len(balls) || p2 >= len(balls) || p1 < 0 || p2 < 0 {
		return ""
	}
	a, b := atoiBall(balls[p1]), atoiBall(balls[p2])
	switch {
	case a > b:
		return "龙"
	case a < b:
		return "虎"
	default:
		return "和"
	}
}

func normalizeTriggerToken(s string) string {
	s = strings.TrimSpace(s)
	switch strings.ToLower(s) {
	case "long", "dragon":
		return "龙"
	case "hu", "tiger":
		return "虎"
	case "he", "tie", "draw":
		return "和"
	}
	// 数字 token 按数值归一（兼容 "07" 与 "7"）
	if n, err := strconv.Atoi(s); err == nil {
		return strconv.Itoa(n)
	}
	return s
}
