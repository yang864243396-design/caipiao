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
	// OddsBase 该彩种「1 元三星直选」基准派彩（第三方 real/rate 派生）。
	// 0 表示未知，赔率按参考基准 970 计（缩放=1）。
	OddsBase float64
}

type betEvaluation struct {
	Hit      bool
	BetUnits int
	Odds     float64
	// PrizeNet 可选：以「1 元单注」为尺度的净奖金绝对值。
	// >0 时多区位汇总优先用它（嵌套一星/二星），避免其它区位亏损把派奖打成 ≤0 而与第三方不一致。
	PrizeNet float64
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
		// kind=contrary：planInverseNumbers 已是反集投注内容，按原玩法直接结算（不再二次取补）
		content := strings.TrimSpace(contraryPlan)
		if content == "" {
			content = groupContent
		}
		return evaluatePlayHit(rule, balls, content, false, "", positionIndex)
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
	return betEvaluation{Hit: hit, BetUnits: units, Odds: oddsDingweiOdds(rule.OddsBase)}
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
	return betEvaluation{Hit: hit, BetUnits: units, Odds: oddsDingweiOdds(rule.OddsBase)}
}

func splitDingweiPositionLines(content string) []string {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	return strings.Split(content, "\n")
}

func evaluateZhixuanFushi(rule playRule, balls []string, groupContent string) betEvaluation {
	lines := splitGroupLines(groupContent)
	var pools [][]string
	if len(lines) >= rule.SegmentLen && rule.SegmentLen > 0 {
		pools = make([][]string, rule.SegmentLen)
		for i := 0; i < rule.SegmentLen; i++ {
			pools[i] = parsePickTokensForRule(rule, lines[i])
		}
	} else {
		pool := parsePickTokensForRule(rule, groupContent)
		if len(pool) == 0 {
			pool = []string{"0"}
		}
		n := rule.SegmentLen
		if n <= 0 {
			n = 1
		}
		pools = make([][]string, n)
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
	// 直选复式豹子（各位同一单码）：对齐第三方网页计 0 注
	if isZhixuanFushiBaoziPools(pools) {
		units = 0
	}
	seg := drawSegment(balls, rule.SegmentStart, rule.SegmentLen)
	if len(seg) != rule.SegmentLen {
		// 无开奖号时仍返回正确注数（预览/资金校验）
		return betEvaluation{BetUnits: units, Odds: oddsZhixuan(rule.SegmentLen, rule.OddsBase)}
	}
	if units <= 0 {
		return betEvaluation{Hit: false, BetUnits: 0, Odds: oddsZhixuan(rule.SegmentLen, rule.OddsBase)}
	}
	hit := true
	for i, digit := range seg {
		if !containsDigit(pools[i], digit) {
			hit = false
			break
		}
	}
	return betEvaluation{Hit: hit, BetUnits: units, Odds: oddsZhixuan(rule.SegmentLen, rule.OddsBase)}
}

func isZhixuanFushiBaoziPools(pools [][]string) bool {
	if len(pools) < 2 {
		return false
	}
	var first string
	for i, p := range pools {
		if len(p) != 1 {
			return false
		}
		d := p[0]
		if i == 0 {
			first = d
			continue
		}
		if d != first {
			return false
		}
	}
	return first != ""
}

func evaluateZhixuanDanshi(rule playRule, balls []string, groupContent string) betEvaluation {
	// 冷热出号按位号池（每位一行）按直选复式位积评估，避免误计 0/1 注。
	if looksLikeZhixuanPositionPool(groupContent, rule.SegmentLen) {
		return evaluateZhixuanFushi(rule, balls, groupContent)
	}
	seg := drawSegment(balls, rule.SegmentStart, rule.SegmentLen)
	if len(seg) != rule.SegmentLen {
		return betEvaluation{BetUnits: 1, Odds: oddsZhixuan(rule.SegmentLen, rule.OddsBase)}
	}
	tokens := parseSegmentTokensForRule(rule, groupContent, rule.SegmentLen)
	if len(tokens) == 0 {
		tokens = parseNumberTokens(groupContent, rule.SegmentLen)
	}
	if len(tokens) == 0 && rule.SegmentLen > 0 {
		tokens = chunkDigitString(groupContent, rule.SegmentLen)
	}
	tokens = uniqueStringTokens(tokens)
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
	return betEvaluation{Hit: hit, BetUnits: units, Odds: oddsZhixuan(rule.SegmentLen, rule.OddsBase)}
}

// looksLikeZhixuanPositionPool 判断内容是否为按位号池（冷热出号 UI）：segLen 行、每行单码池。
func looksLikeZhixuanPositionPool(content string, segLen int) bool {
	if segLen <= 1 || !strings.Contains(content, "\n") {
		return false
	}
	raw := strings.ReplaceAll(content, "\r\n", "\n")
	raw = strings.ReplaceAll(raw, "\r", "\n")
	lines := strings.Split(raw, "\n")
	for len(lines) < segLen {
		lines = append(lines, "")
	}
	for i := 0; i < segLen; i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			return false
		}
		parts := strings.FieldsFunc(line, func(r rune) bool {
			return r == ',' || r == ' ' || r == '，' || r == '\t'
		})
		if len(parts) == 0 {
			return false
		}
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if len(p) != 1 || p[0] < '0' || p[0] > '9' {
				return false
			}
		}
	}
	return true
}

// expandZhixuanPositionPoolToDanshi 按位号池 → 直选单式票（笛卡尔积）。
// 例：segLen=3, "5\n5\n5" → "555"；"4,5\n3,5\n2,5" → "432,435,..."。
func expandZhixuanPositionPoolToDanshi(content string, segLen int) (string, bool) {
	if !looksLikeZhixuanPositionPool(content, segLen) {
		return "", false
	}
	raw := strings.ReplaceAll(content, "\r\n", "\n")
	raw = strings.ReplaceAll(raw, "\r", "\n")
	lines := strings.Split(raw, "\n")
	for len(lines) < segLen {
		lines = append(lines, "")
	}
	pools := make([][]string, segLen)
	for i := 0; i < segLen; i++ {
		parts := strings.FieldsFunc(strings.TrimSpace(lines[i]), func(r rune) bool {
			return r == ',' || r == ' ' || r == '，' || r == '\t'
		})
		seen := make(map[string]struct{}, len(parts))
		out := make([]string, 0, len(parts))
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if len(p) != 1 {
				return "", false
			}
			if _, ok := seen[p]; ok {
				continue
			}
			seen[p] = struct{}{}
			out = append(out, p)
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
	return strings.Join(uniqueStringTokens(cur), ","), true
}

// joinPositionPoolGroupsIfNeeded 将「每位一组」的 schemeGroups 拼成按位号池。
func joinPositionPoolGroupsIfNeeded(cfg parsedSchemeConfig, content string) string {
	seg := cfg.Play.SegmentLen
	if seg <= 1 {
		seg = playPositionCount(cfg.Play)
	}
	if seg <= 1 || len(cfg.Groups) < seg {
		return content
	}
	bm := strings.ToLower(strings.TrimSpace(cfg.Play.BetMode))
	sub := strings.ToLower(strings.TrimSpace(cfg.Play.SubPlayID))
	isDanshi := bm == "danshi" || bm == "zhixuan_ds" || sub == "zhixuan_ds" || strings.HasSuffix(sub, "_ds")
	isFushi := bm == "fushi" || sub == "zhixuan_fs"
	if !isDanshi && !isFushi {
		return content
	}
	if strings.Contains(content, "\n") && looksLikeZhixuanPositionPool(content, seg) {
		return content
	}
	joined := strings.Join(cfg.Groups[:seg], "\n")
	if looksLikeZhixuanPositionPool(joined, seg) {
		return joined
	}
	return content
}

// normalizeZhixuanDanshiContent 冷热按位号池在进入单式校验/落库前展开为整注串。
func normalizeZhixuanDanshiContent(rule playRule, content string) string {
	bm := strings.ToLower(strings.TrimSpace(rule.BetMode))
	sub := strings.ToLower(strings.TrimSpace(rule.SubPlayID))
	if bm == "zuxuan_ds" || sub == "zuxuan_ds" {
		return content
	}
	// 复式/组选复式按位内容勿展成单式票
	if bm == "fushi" || bm == "zhixuan_fs" || sub == "zhixuan_fs" ||
		bm == "zuxuan_fs" || sub == "zuxuan_fs" || bm == "zuhe" {
		return content
	}
	isDanshi := bm == "danshi" || bm == "zhixuan_ds" || sub == "zhixuan_ds" || strings.HasSuffix(sub, "_ds")
	seg := rule.SegmentLen
	if seg <= 0 {
		seg = playPositionCount(rule)
	}
	// betMode 偶发丢失时：内容已是按位号池且段长≥2，仍按单式展开（对齐 guajibet Format）
	if !isDanshi {
		if seg <= 1 || !looksLikeZhixuanPositionPool(content, seg) {
			return content
		}
	}
	if expanded, ok := expandZhixuanPositionPoolToDanshi(content, seg); ok {
		return expanded
	}
	return content
}

func uniqueStringTokens(items []string) []string {
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

func chunkDigitString(raw string, segLen int) []string {
	if segLen <= 0 {
		return nil
	}
	var b strings.Builder
	for _, r := range raw {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	digits := b.String()
	if len(digits) < segLen || len(digits)%segLen != 0 {
		return nil
	}
	out := make([]string, 0, len(digits)/segLen)
	for i := 0; i+segLen <= len(digits); i += segLen {
		out = append(out, digits[i:i+segLen])
	}
	return out
}

func evaluateZuxuanFushi(rule playRule, balls []string, groupContent string) betEvaluation {
	seg := drawSegment(balls, rule.SegmentStart, rule.SegmentLen)
	if len(seg) != rule.SegmentLen {
		return betEvaluation{BetUnits: 1, Odds: oddsZuxuan(rule.SegmentLen, rule.OddsBase)}
	}
	tokens := parseNumberTokens(groupContent, rule.SegmentLen)
	if len(tokens) == 0 {
		pool := parsePickTokensForRule(rule, groupContent)
		if len(pool) == 0 {
			pool = parseDigitTokens(groupContent)
		}
		hit := zuxuanPoolHit(seg, pool)
		units := zuxuanPoolUnitsForRule(rule, pool)
		if units <= 0 {
			units = 1
		}
		return betEvaluation{Hit: hit, BetUnits: units, Odds: oddsZuxuan(rule.SegmentLen, rule.OddsBase)}
	}
	tokens = uniqueStringTokens(tokens)
	drawnSorted := sortDigits(seg)
	hit := false
	for _, t := range tokens {
		if sortStringDigits(t) == drawnSorted {
			hit = true
			break
		}
	}
	return betEvaluation{Hit: hit, BetUnits: len(tokens), Odds: oddsZuxuan(rule.SegmentLen, rule.OddsBase)}
}

// evaluateContraryHit 由正集计划内容取补后，按同玩法结算（用于详情「计划反集」注数/奖金预估）。
func evaluateContraryHit(rule playRule, balls []string, planContent string, positionIndex int) betEvaluation {
	_ = positionIndex
	inv := complementPlanContent(rule, planContent)
	if inv == "" {
		return betEvaluation{BetUnits: 0, Odds: oddsDingweiOdds(rule.OddsBase)}
	}
	return evaluatePlayHit(rule, balls, inv, false, "", rule.PositionIdx)
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

/** 保留空行并补齐到 n（直选复式按位校验用，禁止把缺位压成单行号池） */
func splitGroupLinesPad(content string, n int) []string {
	if n <= 0 {
		n = 1
	}
	raw := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
	out := make([]string, n)
	for i := 0; i < n; i++ {
		if i < len(raw) {
			out[i] = strings.TrimSpace(raw[i])
		}
	}
	return out
}

/** 是否含至少 1 个 0-9 单 digit（与 parseDigitTokens 不同：空串不算） */
func hasDigitPickToken(raw string) bool {
	raw = strings.NewReplacer("\n", ",", "，", ",", " ", ",").Replace(raw)
	for _, p := range strings.Split(raw, ",") {
		p = strings.TrimSpace(p)
		if len(p) == 1 && p[0] >= '0' && p[0] <= '9' {
			return true
		}
	}
	return false
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
		// 通用组选复式：组三 n*(n-1) + 组六 C(n,3)
		if n < 3 {
			if n < 2 {
				return n
			}
			return n * (n - 1)
		}
		return n*(n-1) + n*(n-1)*(n-2)/6
	}
	if segLen == 4 && n >= 4 {
		return n * (n - 1) / 2
	}
	return n
}

// zuxuanPoolUnitsForRule 按 betMode/catalog 区分组三、组六与通用组选复式。
func zuxuanPoolUnitsForRule(rule playRule, pool []string) int {
	mode := strings.ToLower(strings.TrimSpace(rule.BetMode))
	cat := strings.ToLower(rule.CatalogSubID + " " + rule.SubPlayID)
	if mode == "zu6" || (strings.Contains(cat, "zu6") && !strings.Contains(cat, "zu60") && !strings.Contains(cat, "zu120")) {
		return zu6PoolUnits(pool)
	}
	if mode == "zu3" || strings.Contains(cat, "zu3") {
		return zu3PoolUnits(pool)
	}
	return zuxuanPoolUnits(pool, rule.SegmentLen)
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

// oddsDingwei 定位胆参考赔率（base=970 时的值）。实际按 oddsDingweiOdds(base) 缩放。
const oddsDingwei = 9.0

// 组选包胆单注派奖（1 元模式近似第三方，base=970 参考值）
const (
	oddsBaodanZu6 = 161.666
	oddsBaodanZu3 = 323.333
)

// oddsDingweiOdds 定位胆赔率，随第三方赔率线缩放。
func oddsDingweiOdds(base float64) float64 { return oddsDingwei * oddsScale(base) }

// oddsZhixuan 直选单注赔率（1 元模式「可中」尺度，对齐 V6 展示/派彩）。
// base 为该彩种第三方基准（1 元三星直选）；未知时按参考基准 970。
// 例：前三直选复式 base=970 → 970；base=980 → 980（随赔率线走）。
func oddsZhixuan(segLen int, base float64) float64 {
	var ref float64
	switch segLen {
	case 5:
		ref = 97000.0
	case 4:
		ref = 9700.0
	case 3:
		ref = 970.0
	case 2:
		// 二星直选 / 组合嵌套后二：对齐 V6 实测净额 ≈19.4
		ref = 19.4
	default:
		ref = 9.0
	}
	return ref * oddsScale(base)
}

func oddsZuxuan(segLen int, base float64) float64 {
	var ref float64
	switch segLen {
	case 4:
		ref = 24.0
	case 3:
		ref = 16.0
	default:
		ref = 9.0
	}
	return ref * oddsScale(base)
}

func calcPnLWithOdds(amount float64, hit bool, odds float64) float64 {
	if hit {
		return round2(amount * odds)
	}
	return round2(-amount)
}
