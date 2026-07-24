package schemes

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"
)

const baseBetUnitYuan = 2.0

type schemeRound struct {
	Mult      float64 `json:"mult"`
	AfterHit  int     `json:"afterHit"`
	AfterMiss int     `json:"afterMiss"`
}

// jushuRow 高级定码轮换局数列表（v8 §3.2）。
type jushuRow struct {
	Ju        int    `json:"ju"`
	Content   string `json:"content"`
	AfterHit  int    `json:"afterHit"`
	AfterMiss int    `json:"afterMiss"`
}

// triggerRow 开某投某映射行（v8 §3.3）。
type triggerRow struct {
	Enabled bool   `json:"enabled"`
	Open    string `json:"open"`
	Pos     string `json:"pos"`
	Neg     string `json:"neg"`
}

type triggerBetCfg struct {
	Rows []triggerRow `json:"rows"`
	// always_pos / always_neg / alt_pos_first / alt_neg_first
	Mode string `json:"mode"`
	// PositionIdxs 定位胆投注位（可多选，0=万/冠军 …）；HasPosition 表示配置显式指定。
	// 兼容旧字段 positionIdx（单值）。统一「一星定位胆」默认万位。
	PositionIdxs []int `json:"positionIdxs,omitempty"`
	PositionIdx  int   `json:"positionIdx,omitempty"` // 首项/旧单值镜像
	HasPosition  bool  `json:"-"`
}

type hotColdWarmCfg struct {
	TotalPeriods int `json:"totalPeriods"`
	// Pool 手动覆盖：按位对齐（万\n千\n百…），某位非空则该位用手选号码，
	// 为空则该位按名次自动取号（混合模式）。跨位/整体型玩法取 Pool[0]。
	Pool []string `json:"pool"`
	// every 每期换 / keep 不换号 / after_hit 中后换 / after_miss 挂后换
	Strategy string `json:"strategy"`
	// WinRotate 兼容旧配置；Strategy 为空时 true→after_hit、false→keep
	WinRotate bool `json:"winRotate"`
	// PickTypes 出号类型：hot / cold（可多选；空则兼容旧 pool）
	PickTypes []string `json:"pickTypes"`
	// FaultCount 容错=起点偏移：在「最热→最冷」排序上跳过该端最极端的前 N 名（0=不跳过）。
	// 旧字段名保留兼容；语义已由「取N个」改为「起点偏移」（对齐富联，见冷热出号逆向）。
	FaultCount int `json:"faultCount"`
	// PickCount 每位取几个名次：从起点偏移处连续取 N 个号（默认 1）。
	PickCount int `json:"pickCount"`
}

type randomDrawCfg struct {
	Counts []int `json:"counts"`
	// every 每期换 / keep 不换号 / after_hit 中后换 / after_miss 挂后换
	Strategy string `json:"strategy"`
}


type parsedSchemeConfig struct {
	Kind          string
	RunTypeID     string
	PlayTypeLabel string
	Play          playRule
	BetUnitYuan   float64
	Currency      string
	GroupContent  string
	Groups        []string
	Contrary      bool
	ContraryPlan  string
	Rounds        []schemeRound
	GroupCount    int
	Jushu         []jushuRow
	Trigger *triggerBetCfg
	HotCold *hotColdWarmCfg
	Random  *randomDrawCfg
}

func parseSchemeConfig(kind string, raw []byte, roundIndex, groupIndex int) parsedSchemeConfig {
	out := parsedSchemeConfig{
		Kind:          kind,
		PlayTypeLabel: "定位胆",
		Rounds:        []schemeRound{{Mult: 1, AfterHit: 0, AfterMiss: 0}},
	}
	if len(raw) == 0 {
		out.Play = playRule{PlayTypeID: "dingwei", SegmentLen: 1}
		out.Currency = normalizeSchemeCurrency("")
		return out
	}
	var cfg map[string]interface{}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		out.Play = playRule{PlayTypeID: "dingwei", SegmentLen: 1}
		out.Currency = normalizeSchemeCurrency("")
		return out
	}

	out.PlayTypeLabel = resolvePlayTypeLabel(cfg)
	out.BetUnitYuan = schemeBetUnitFromConfig(cfg)
	out.Currency = normalizeSchemeCurrency(stringVal(cfg, "schemeCurrency"))
	if rule, ok := resolveCatalogPlayRule(cfg); ok {
		out.Play = rule
	} else {
		out.Play = resolvePlayRule(cfg, out.PlayTypeLabel)
	}
	out.Contrary = kind == "contrary" || strings.EqualFold(kind, "contrary")
	if inv, ok := cfg["planInverseNumbers"].(string); ok {
		out.ContraryPlan = inv
	}
	out.RunTypeID = resolveEffectiveRunType(kind, cfg)
	groups := extractSchemeGroups(cfg)
	out.Groups = groups
	if len(groups) > 0 {
		out.GroupCount = len(groups)
		idx := groupIndex % len(groups)
		out.GroupContent = groups[idx]
	}
	if out.GroupCount <= 0 {
		out.GroupCount = 1
	}
	out.Rounds = resolveRounds(cfg)
	if len(out.Rounds) == 0 {
		out.Rounds = []schemeRound{{Mult: 1, AfterHit: 0, AfterMiss: 0}}
	}
	out.Jushu = resolveJushuList(cfg, groups, out.Rounds)
	out.Trigger = resolveTriggerBet(cfg)
	applyTriggerBetPosition(&out)
	out.HotCold = resolveHotColdWarm(cfg)
	out.Random = resolveRandomDraw(cfg)
	_ = roundIndex
	return out
}

// resolveEffectiveRunType 归一化运行类型；仅 kind=custom 参与分发，
// 内置计画取物化时记录的实际类型；**未物化的内置计画保留 builtin_plan，
// 引擎按期跳过不下注**（v8 §0/§3.6）。
func resolveEffectiveRunType(kind string, cfg map[string]interface{}) string {
	if kind != "custom" {
		return ""
	}
	raw, _ := cfg["runTypeId"].(string)
	rt := NormalizeRunTypeID(raw)
	if rt != RunTypeBuiltinPlan {
		return rt
	}
	if bp, ok := cfg["builtinPlan"].(map[string]interface{}); ok {
		if actual, ok := bp["runTypeId"].(string); ok && strings.TrimSpace(actual) != "" {
			inner := NormalizeRunTypeID(actual)
			if inner != RunTypeBuiltinPlan {
				return inner
			}
		}
	}
	return RunTypeBuiltinPlan
}

// resolveJushuList 局数列表；无配置时由存量 schemeGroups（+rounds 跳转）运行时换形（v8 §8）。
func resolveJushuList(cfg map[string]interface{}, groups []string, rounds []schemeRound) []jushuRow {
	if raw, ok := cfg["jushuList"].([]interface{}); ok && len(raw) > 0 {
		out := make([]jushuRow, 0, len(raw))
		for _, item := range raw {
			m, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			row := jushuRow{
				Ju:        toInt(m["ju"], 0),
				AfterHit:  toInt(m["afterHit"], 1),
				AfterMiss: toInt(m["afterMiss"], 1),
			}
			if c, ok := m["content"].(string); ok {
				row.Content = normalizeSchemeGroupContent(c)
			}
			if row.Ju > 0 && strings.TrimSpace(row.Content) != "" {
				out = append(out, row)
			}
		}
		if len(out) > 0 {
			return out
		}
	}
	// 存量换形：局 i+1 = 第 i 组号码；有 rounds 时沿用其跳转（0-based → 局号），否则回第 1 局
	out := make([]jushuRow, 0, len(groups))
	for i, g := range groups {
		row := jushuRow{Ju: i + 1, Content: g, AfterHit: 1, AfterMiss: 1}
		if i < len(rounds) {
			row.AfterHit = rounds[i].AfterHit + 1
			row.AfterMiss = rounds[i].AfterMiss + 1
		}
		out = append(out, row)
	}
	return out
}

func resolveTriggerBet(cfg map[string]interface{}) *triggerBetCfg {
	raw, ok := cfg["triggerBet"].(map[string]interface{})
	if !ok {
		return nil
	}
	out := &triggerBetCfg{Mode: "always_pos"}
	if m, ok := raw["mode"].(string); ok && strings.TrimSpace(m) != "" {
		out.Mode = strings.TrimSpace(m)
	}
	seenPos := map[int]bool{}
	appendPos := func(idx int) {
		if idx < 0 || seenPos[idx] {
			return
		}
		seenPos[idx] = true
		out.PositionIdxs = append(out.PositionIdxs, idx)
		out.HasPosition = true
	}
	if arr, ok := raw["positionIdxs"].([]interface{}); ok {
		for _, item := range arr {
			appendPos(toInt(item, -1))
		}
	}
	if _, ok := raw["positionIdx"]; ok {
		appendPos(toInt(raw["positionIdx"], -1))
	}
	rows, _ := raw["rows"].([]interface{})
	for _, item := range rows {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		row := triggerRow{Enabled: true}
		if v, ok := m["enabled"].(bool); ok {
			row.Enabled = v
		}
		if v, ok := m["open"].(string); ok {
			row.Open = strings.TrimSpace(v)
		}
		if v, ok := m["pos"].(string); ok {
			row.Pos = strings.TrimSpace(v)
		}
		if v, ok := m["neg"].(string); ok {
			row.Neg = strings.TrimSpace(v)
		}
		if row.Open != "" {
			out.Rows = append(out.Rows, row)
		}
	}
	if len(out.Rows) == 0 {
		return nil
	}
	return out
}

// applyTriggerBetPosition 规范化开某投某投注位。
// 定位胆：写回 Play.PositionIdx/SegmentStart（单位投注）。
// 前三直选复式等：只保留 Trigger.PositionIdxs 供按位匹配/出号，不改写玩法段。
func applyTriggerBetPosition(out *parsedSchemeConfig) {
	if out == nil || out.Trigger == nil || !out.Trigger.HasPosition {
		return
	}
	if !triggerBetUsesPosition(out.Play) {
		return
	}
	max := 4
	if out.Play.PlayTemplate == "pk10_std" {
		max = 9
	}
	norm := make([]int, 0, len(out.Trigger.PositionIdxs))
	seen := map[int]bool{}
	for _, idx := range out.Trigger.PositionIdxs {
		if idx < 0 {
			idx = 0
		}
		if idx > max {
			idx = max
		}
		if seen[idx] {
			continue
		}
		seen[idx] = true
		norm = append(norm, idx)
	}
	if len(norm) == 0 {
		norm = []int{0}
	}
	sort.Ints(norm)
	out.Trigger.PositionIdxs = norm
	out.Trigger.PositionIdx = norm[0]
	if !isDingweiTriggerPlay(out.Play) {
		return
	}
	out.Play.PositionIdx = norm[0]
	out.Play.SegmentStart = norm[0]
	out.Play.SegmentLen = 1
}

func isDingweiTriggerPlay(rule playRule) bool {
	bm := strings.TrimSpace(rule.BetMode)
	tid := strings.TrimSpace(rule.PlayTypeID)
	return bm == "dingwei" || tid == "dingwei" || tid == "g006"
}

func triggerBetUsesPosition(rule playRule) bool {
	if isLonghuPlay(rule) {
		return false
	}
	if rule.PlayTemplate == "pc28_std" {
		return false
	}
	bm := strings.TrimSpace(rule.BetMode)
	tid := strings.TrimSpace(rule.PlayTypeID)
	sub := strings.TrimSpace(rule.SubPlayID)
	if isDingweiTriggerPlay(rule) {
		return true
	}
	// 前三/中三/后三直选复式等：SegmentLen>=2 的按位数字玩法
	if rule.SegmentLen >= 2 {
		if bm == "fushi" || bm == "zhixuan_fs" || bm == "zuhe" {
			return true
		}
		if sub == "zhixuan_fs" || strings.Contains(sub, "zhixuan_fs") {
			return true
		}
	}
	_ = tid
	return false
}

// layoutTriggerBetDingweiContent 将单位号码编排到选定投注位（多行），
// 避免统一「一星定位胆」子玩法在 wire 时始终压到万位。仅定位胆使用。
func layoutTriggerBetDingweiContent(cfg parsedSchemeConfig, content string) string {
	content = strings.TrimSpace(content)
	if content == "" || cfg.Trigger == nil || !cfg.Trigger.HasPosition {
		return content
	}
	if !isDingweiTriggerPlay(cfg.Play) || !triggerBetUsesPosition(cfg.Play) {
		return content
	}
	if strings.Contains(content, "\n") {
		return content
	}
	positions := 5
	if cfg.Play.PlayTemplate == "pk10_std" {
		positions = 10
	}
	// 已是「号,,,,」稀疏五段 wire（含空位）：原样保留。
	// 注意：正投多号「1,2,3」或满号「1,2,3,4,5」逗号数也可能是 positions-1，不能仅按逗号数判断。
	if parts := strings.Split(content, ","); len(parts) == positions {
		hasEmpty := false
		for _, p := range parts {
			if strings.TrimSpace(p) == "" {
				hasEmpty = true
				break
			}
		}
		if hasEmpty {
			return content
		}
	}
	idxs := cfg.Trigger.PositionIdxs
	if len(idxs) == 0 {
		idxs = []int{cfg.Play.PositionIdx}
	}
	lines := make([]string, positions)
	for _, idx := range idxs {
		if idx < 0 {
			idx = 0
		}
		if idx >= positions {
			idx = positions - 1
		}
		lines[idx] = content
	}
	return strings.Join(lines, "\n")
}

func resolveHotColdWarm(cfg map[string]interface{}) *hotColdWarmCfg {
	raw, ok := cfg["hotColdWarm"].(map[string]interface{})
	if !ok {
		return nil
	}
	out := &hotColdWarmCfg{TotalPeriods: 20, Strategy: "keep", FaultCount: 0, PickCount: 1}
	if v := toInt(raw["totalPeriods"], 0); v > 0 {
		out.TotalPeriods = v
	}
	if v, ok := raw["winRotate"].(bool); ok {
		out.WinRotate = v
	}
	if s, ok := raw["strategy"].(string); ok {
		switch strings.TrimSpace(s) {
		case "every", "keep", "after_hit", "after_miss":
			out.Strategy = strings.TrimSpace(s)
		}
	} else if out.WinRotate {
		out.Strategy = "after_hit"
	}
	// 容错=起点偏移（0..9），允许显式 0；旧配置的正整数按新语义直接当偏移。
	if _, has := raw["faultCount"]; has {
		fc := toInt(raw["faultCount"], 0)
		if fc < 0 {
			fc = 0
		}
		if fc > 9 {
			fc = 9
		}
		out.FaultCount = fc
	}
	if pc := toInt(raw["pickCount"], 0); pc > 0 {
		if pc > 10 {
			pc = 10
		}
		out.PickCount = pc
	}
	if arr, ok := raw["pickTypes"].([]interface{}); ok {
		for _, item := range arr {
			s := strings.ToLower(strings.TrimSpace(fmt.Sprint(item)))
			if s == "hot" || s == "cold" {
				out.PickTypes = append(out.PickTypes, s)
			}
		}
	}
	if pool, ok := raw["pool"].([]interface{}); ok {
		for _, item := range pool {
			if s, ok := item.(string); ok {
				s = strings.TrimSpace(s)
				if s != "" {
					out.Pool = append(out.Pool, s)
				}
			}
		}
	}
	// 新口径：出号类型+容错即可运行；旧口径：仅有 pool 也可
	if len(out.PickTypes) == 0 && len(out.Pool) == 0 {
		return nil
	}
	return out
}

func resolveRandomDraw(cfg map[string]interface{}) *randomDrawCfg {
	raw, ok := cfg["randomDraw"].(map[string]interface{})
	if !ok {
		return nil
	}
	out := &randomDrawCfg{Strategy: "every"}
	if s, ok := raw["strategy"].(string); ok && strings.TrimSpace(s) != "" {
		out.Strategy = strings.TrimSpace(s)
	}
	if counts, ok := raw["counts"].([]interface{}); ok {
		for _, item := range counts {
			n := toInt(item, 1)
			if n < 1 {
				n = 1
			}
			if n > 10 {
				n = 10
			}
			out.Counts = append(out.Counts, n)
		}
	}
	return out
}

func resolvePlayTypeLabel(cfg map[string]interface{}) string {
	if v, ok := cfg["playMethod"].(string); ok {
		if pm := strings.TrimSpace(v); pm != "" && !isBarePlayToken(pm) {
			return pm
		}
	}
	playTypeID, _ := cfg["playTypeId"].(string)
	if playTypeID == "" {
		playTypeID, _ = cfg["typeId"].(string)
	}
	subPlayID, _ := cfg["subPlayId"].(string)
	if subPlayID == "" {
		subPlayID, _ = cfg["subId"].(string)
	}
	betMode, _ := cfg["betMode"].(string)
	template, _ := cfg["playTemplate"].(string)
	if template == "lhc_std" || isLHCTypeID(playTypeID) {
		if label := formatLHCPlayLabel(playTypeID, subPlayID); label != "" {
			return label
		}
	}
	playLabels := map[string]string{
		"dingwei": "定位胆", "g006": "定位胆",
		"g001": "前三码", "g002": "中三码", "g003": "后三码",
		"g004": "前二码", "g005": "后二码",
		"g007": "前中后三", "g008": "前后二", "g009": "不定位",
		"g010": "龙虎", "g011": "任选", "g012": "前后三",
		"g013": "四星", "g014": "前后四", "g015": "五星", "g016": "大小单双",
		"dxds": "大小单双", "hou4": "后四", "qian3": "前三", "zhong3": "中三",
	}
	subLabels := map[string]string{
		"zhixuan_fs": "直选复式", "zhixuan_ds": "直选单式", "zuxuan_fs": "组选复式",
	}
	label := playLabels[playTypeID]
	if label == "" {
		label = playTypeID
	}
	if sub := subLabels[subPlayID]; sub != "" {
		if label != "" && !isBarePlayToken(label) {
			return label + sub
		}
		return sub
	}
	if label != "" && !isBarePlayToken(label) {
		return label
	}
	if betMode == "dingwei" || playTypeID == "g006" {
		return "定位胆"
	}
	// 未知 typeId 时不要默认「定位胆」，避免污染 guajibet.InferBetMode
	if label != "" {
		return label
	}
	return ""
}

// PlayMethodDisplay 将库内 play_method / playTypeId / subPlayId 解析为中文玩法展示名。
func PlayMethodDisplay(playMethod, playTypeID, subPlayID string) string {
	pm := strings.TrimSpace(playMethod)
	if pm != "" && !isBarePlayToken(pm) {
		return pm
	}
	cfg := map[string]interface{}{
		"playMethod": pm,
		"playTypeId": strings.TrimSpace(playTypeID),
		"subPlayId":  strings.TrimSpace(subPlayID),
	}
	label := strings.TrimSpace(resolvePlayTypeLabel(cfg))
	if label != "" && !isBarePlayToken(label) {
		return label
	}
	if pm != "" {
		return pm
	}
	return label
}

func isBarePlayToken(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return true
	}
	if _, err := strconv.Atoi(s); err == nil {
		return true
	}
	if len(s) >= 2 && strings.HasPrefix(strings.ToLower(s), "g") {
		allDigit := true
		for _, r := range s[1:] {
			if r < '0' || r > '9' {
				allDigit = false
				break
			}
		}
		if allDigit {
			return true
		}
	}
	return false
}

func resolvePositionIndex(cfg map[string]interface{}, playLabel string) int {
	if sub, ok := cfg["subPlayId"].(string); ok {
		switch sub {
		case "sub_wan":
			return 0
		case "sub_qian":
			return 1
		case "sub_bai":
			return 2
		case "sub_shi":
			return 3
		case "sub_ge":
			return 4
		}
	}
	for idx, r := range []rune{'万', '千', '百', '十', '个'} {
		if strings.ContainsRune(playLabel, r) {
			return idx
		}
	}
	return 0
}

// normalizeSchemeGroupContent 清理分组内容边缘空白，但保留换行空位（定位胆万千百十个）。
// 禁止 TrimSpace：否则 "\n\n1,2\n\n"（百位）会被压成 "1,2" → 第三方变成万位 "12,,,,"。
func normalizeSchemeGroupContent(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	if strings.Contains(s, "\n") {
		return strings.Trim(s, " \t")
	}
	return strings.TrimSpace(s)
}

func extractSchemeGroups(cfg map[string]interface{}) []string {
	raw, ok := cfg["schemeGroups"].([]interface{})
	if !ok || len(raw) == 0 {
		return nil
	}
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		s, ok := item.(string)
		if !ok {
			continue
		}
		s = normalizeSchemeGroupContent(s)
		if strings.TrimSpace(s) != "" {
			out = append(out, s)
		}
	}
	return out
}

func parseDigitTokens(raw string) []string {
	raw = strings.NewReplacer("\n", ",", "，", ",", " ", ",").Replace(raw)
	parts := strings.Split(raw, ",")
	seen := map[string]struct{}{}
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if len(p) != 1 || p[0] < '0' || p[0] > '9' {
			continue
		}
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		out = append(out, p)
	}
	if len(out) == 0 {
		return []string{"0"}
	}
	return out
}


func resolveRounds(cfg map[string]interface{}) []schemeRound {
	return normalizeSchemeRounds(parseSchemeRoundsFromRaw(cfg["rounds"]))
}

// roundsUseOneBasedTargets 高级倍投轮次页以「第 N 局」存储跳转目标（≥1）；
// 简单倍投编译结果使用 0-based 索引（含 afterHit=0 或末轮 afterMiss=0）。
func roundsUseOneBasedTargets(rounds []schemeRound) bool {
	if len(rounds) == 0 {
		return false
	}
	for _, r := range rounds {
		if r.AfterHit == 0 || r.AfterMiss == 0 {
			return false
		}
	}
	return true
}

// normalizeSchemeRounds 将 1-based 跳转目标转为引擎使用的 0-based 轮次索引。
func normalizeSchemeRounds(rounds []schemeRound) []schemeRound {
	if len(rounds) == 0 || !roundsUseOneBasedTargets(rounds) {
		return rounds
	}
	out := make([]schemeRound, len(rounds))
	for i, r := range rounds {
		out[i] = schemeRound{
			Mult:      r.Mult,
			AfterHit:  r.AfterHit - 1,
			AfterMiss: r.AfterMiss - 1,
		}
	}
	return out
}

func parseSchemeRoundsFromRaw(raw interface{}) []schemeRound {
	items, ok := raw.([]interface{})
	if !ok {
		return nil
	}
	out := make([]schemeRound, 0, len(items))
	for _, item := range items {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		r := schemeRound{
			Mult:      toFloat(m["mult"], 1),
			AfterHit:  toInt(m["afterHit"], 0),
			AfterMiss: toInt(m["afterMiss"], 0),
		}
		out = append(out, r)
	}
	return out
}

func roundLabel(roundIndex, total int) string {
	if total <= 0 {
		total = 1
	}
	idx := roundIndex
	if idx < 0 {
		idx = 0
	}
	if idx >= total {
		idx = idx % total
	}
	return fmt.Sprintf("%d/%d", idx+1, total)
}

func nextRoundIndex(rounds []schemeRound, cur int, hit bool) int {
	if len(rounds) == 0 {
		return 0
	}
	if cur < 0 || cur >= len(rounds) {
		cur = 0
	}
	r := rounds[cur]
	if hit {
		return clampRoundIndex(r.AfterHit, len(rounds))
	}
	return clampRoundIndex(r.AfterMiss, len(rounds))
}

func clampRoundIndex(v, n int) int {
	if n <= 0 {
		return 0
	}
	if v < 0 {
		return 0
	}
	if v >= n {
		return v % n
	}
	return v
}

func toFloat(v interface{}, fallback float64) float64 {
	switch n := v.(type) {
	case float64:
		if n > 0 {
			return n
		}
	case int:
		if n > 0 {
			return float64(n)
		}
	case json.Number:
		f, err := n.Float64()
		if err == nil && f > 0 {
			return f
		}
	}
	return fallback
}

func toInt(v interface{}, fallback int) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	case json.Number:
		i, err := n.Int64()
		if err == nil {
			return int(i)
		}
	}
	return fallback
}

func evaluatePositionHit(balls []string, positionIndex int, picks []string) bool {
	if len(balls) == 0 {
		return false
	}
	if positionIndex < 0 || positionIndex >= len(balls) {
		return false
	}
	drawn := strings.TrimSpace(balls[positionIndex])
	if drawn == "" {
		return false
	}
	return containsDigit(picks, drawn)
}

func calcBetAmount(betUnits int, mult float64, unitYuan float64) float64 {
	if betUnits <= 0 {
		betUnits = 1
	}
	if mult <= 0 {
		mult = 1
	}
	if unitYuan <= 0 {
		unitYuan = baseBetUnitYuan
	}
	return round2(unitYuan * float64(betUnits) * mult)
}

// instanceBaseCoef 单方案卡片上的倍数系数。
func instanceBaseCoef(mult pgtype.Numeric) float64 {
	m := numericToFloat(mult)
	if m <= 0 {
		return 1
	}
	return m
}

// planBaseCoef 云端中心全局「方案倍数系数」。
func planBaseCoef(planMult float64) float64 {
	if planMult <= 0 {
		return 1
	}
	return planMult
}

// combinedBaseCoef = 全局方案倍数系数 × 单方案卡片倍数系数。
func combinedBaseCoef(instMult pgtype.Numeric, planMult float64) float64 {
	return round2(planBaseCoef(planMult) * instanceBaseCoef(instMult))
}

// effectiveBetMultiple = 云端倍数系数（全局×卡片）× 当前轮次方案倍投倍数 → 投注 multiple 参数。
func effectiveBetMultiple(baseCoef float64, round schemeRound) float64 {
	rm := round.Mult
	if rm <= 0 {
		rm = 1
	}
	if baseCoef <= 0 {
		baseCoef = 1
	}
	return round2(baseCoef * rm)
}

func betMultipleAsInt(mult float64) int {
	m := int(math.Round(mult))
	if m <= 0 {
		return 1
	}
	return m
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}
