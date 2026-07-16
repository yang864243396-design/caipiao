package schemes

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"strconv"
	"strings"
)

// BetPayload is persisted on bet_orders for settlement.
type BetPayload struct {
	PlayTemplate string `json:"playTemplate,omitempty"`
	TypeID       string `json:"typeId,omitempty"`
	SubID        string `json:"subId,omitempty"`
	BetMode      string `json:"betMode,omitempty"` // 服务端从 catalog 填充，用于 SSC 规则解析
	PlayMethod   string `json:"playMethod,omitempty"`
	PlayTypeID   string `json:"playTypeId,omitempty"`
	SubPlayID    string `json:"subPlayId,omitempty"`
	GroupContent string `json:"groupContent,omitempty"`
}

// NormalizeBetPayload validates client picks; falls back only when groupContent empty.
func NormalizeBetPayload(in BetPayload) ([]byte, error) {
	playMethod := strings.TrimSpace(in.PlayMethod)
	if playMethod == "" {
		playMethod = "定位胆万位"
	}
	content := strings.TrimSpace(in.GroupContent)
	if content == "" {
		return nil, fmt.Errorf("groupContent 不能为空")
	}

	template := strings.TrimSpace(in.PlayTemplate)
	typeID := strings.TrimSpace(in.TypeID)
	subID := strings.TrimSpace(in.SubID)
	if typeID == "" {
		typeID = strings.TrimSpace(in.PlayTypeID)
	}
	if subID == "" {
		subID = strings.TrimSpace(in.SubPlayID)
	}

	rule := resolvePlayRuleFromBetPayload(BetPayload{
		PlayTemplate: template,
		TypeID:       typeID,
		SubID:        subID,
		BetMode:      in.BetMode,
		PlayMethod:   playMethod,
		PlayTypeID:   in.PlayTypeID,
		SubPlayID:    in.SubPlayID,
	})

	if err := validateGroupContent(rule, content); err != nil {
		return nil, err
	}
	out := BetPayload{
		PlayTemplate: template,
		TypeID:       typeID,
		SubID:        subID,
		BetMode:      rule.BetMode,
		PlayMethod:   playMethod,
		PlayTypeID:   rule.PlayTypeID,
		SubPlayID:    rule.SubPlayID,
		GroupContent: content,
	}
	raw, err := json.Marshal(out)
	if err != nil {
		return nil, err
	}
	return raw, nil
}

// EnsureBetPayload uses stored payload when groupContent present, else deterministic fallback.
func EnsureBetPayload(raw []byte, playMethod, orderNo string) []byte {
	var p BetPayload
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &p)
	}
	if strings.TrimSpace(p.GroupContent) != "" {
		normalized, err := NormalizeBetPayload(p)
		if err == nil {
			return normalized
		}
	}
	return BuildBetPayload(playMethod, orderNo)
}

func validateGroupContent(rule playRule, content string) error {
	if strings.TrimSpace(content) == "" {
		return fmt.Errorf("groupContent 不能为空")
	}
	sub := rule.SubPlayID
	if sub == "zhixuan_ds" {
		tokens := parseNumberTokens(content, rule.SegmentLen)
		if len(tokens) == 0 {
			return fmt.Errorf("直选单式须为 %d 位数字", rule.SegmentLen)
		}
		return nil
	}
	if sub == "zhixuan_fs" && rule.SegmentLen > 1 {
		lines := splitGroupLines(content)
		if len(lines) >= rule.SegmentLen {
			for i := 0; i < rule.SegmentLen; i++ {
				if len(parseDigitTokens(lines[i])) == 0 {
					return fmt.Errorf("第 %d 位选号无效", i+1)
				}
			}
			return nil
		}
		if len(parseDigitTokens(content)) == 0 {
			return fmt.Errorf("选号池不能为空")
		}
		return nil
	}
	if rule.BetMode == "dingwei" && strings.Contains(content, "\n") {
		lines := splitDingweiPositionLines(content)
		hasAny := false
		for i := 0; i < 5; i++ {
			line := ""
			if i < len(lines) {
				line = lines[i]
			}
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			if len(parseDigitTokens(line)) == 0 {
				return fmt.Errorf("第 %d 位选号无效", i+1)
			}
			hasAny = true
		}
		if !hasAny {
			return fmt.Errorf("请至少在一位选择号码")
		}
		return nil
	}
	if rule.PlayTemplate == "lhc_std" || isLHCPlayRule(rule) {
		if validateLHCGroupContent(rule, content) {
			return nil
		}
		return fmt.Errorf("选号无效")
	}
	if isSpecialBetMode(rule.BetMode) {
		if strings.TrimSpace(content) != "" {
			return nil
		}
		return fmt.Errorf("选号无效")
	}
	if rule.PlayTypeID == "renxuan_fs" || rule.PlayTypeID == "renxuan_ds" {
		if validateSyxwRenxuanContent(rule, content) {
			return nil
		}
		return fmt.Errorf("选号无效")
	}
	_, maxPool := ruleNumberPool(rule)
	if maxPool > defaultPoolMax {
		if len(parsePickTokensForRule(rule, content)) > 0 {
			return nil
		}
		if rule.SubPlayID == "zhixuan_ds" && len(parseSegmentTokensForRule(rule, content, rule.SegmentLen)) > 0 {
			return nil
		}
		if rule.SubPlayID == "zhixuan_fs" && rule.SegmentLen > 1 {
			lines := splitGroupLines(content)
			if len(lines) >= rule.SegmentLen {
				ok := true
				for i := 0; i < rule.SegmentLen; i++ {
					if len(parsePickTokensForRule(rule, lines[i])) == 0 {
						ok = false
						break
					}
				}
				if ok {
					return nil
				}
			}
		}
	}
	if len(parseDigitTokens(content)) == 0 {
		// 和值/跨度/龙虎等特殊玩法：允许非空自由文本（P2 textarea 手输）
		if strings.TrimSpace(content) != "" {
			return nil
		}
		return fmt.Errorf("选号无效")
	}
	return nil
}

func isLHCPlayRule(rule playRule) bool {
	switch rule.BetMode {
	case "tema", "zhengte", "fushi", "tuotou", "sx_dp", "ws_dp", "sw_dp", "renyi_dp",
		"texiao", "zongxiao", "xiao", "xiao_z", "xiao_bz", "weishu", "weishu_bz", "wei_z", "wei_bz",
		"buzhong", "xuanyi", "guoguan", "tematouwei", "wuxing", "jiaye", "bose", "banbo", "banbanbo",
		"qima", "renzhong":
		return true
	default:
		return false
	}
}

func resolvePlayRuleFromBetPayload(p BetPayload) playRule {
	template := strings.TrimSpace(p.PlayTemplate)
	typeID := strings.TrimSpace(p.TypeID)
	if typeID == "" {
		typeID = strings.TrimSpace(p.PlayTypeID)
	}
	subID := strings.TrimSpace(p.SubID)
	if subID == "" {
		subID = strings.TrimSpace(p.SubPlayID)
	}
	betMode := strings.TrimSpace(p.BetMode)
	cfg := map[string]interface{}{
		"playTemplate": template,
		"typeId":       typeID,
		"subId":        subID,
		"betMode":      betMode,
		"playMethod":   strings.TrimSpace(p.PlayMethod),
	}
	if rule, ok := resolveCatalogPlayRule(cfg); ok {
		return rule
	}
	return playRule{}
}

func isSpecialBetMode(mode string) bool {
	switch strings.TrimSpace(mode) {
	case "longhu", "daxiao", "danshuang", "dxds", "teshu", "longhubao",
		"hezhi", "lianhao", "sanlian", "shoudong", "tonghao", "butong", "dantiao":
		return true
	default:
		return false
	}
}

func validateSyxwRenxuanContent(rule playRule, content string) bool {
	if rule.BetMode == "danshi" || strings.HasSuffix(rule.CatalogSubID, "_ds") {
		for _, line := range splitGroupLines(content) {
			if len(parsePickTokensForRule(rule, line)) > 0 {
				return true
			}
		}
		return strings.TrimSpace(content) != ""
	}
	return len(parsePickTokensForRule(rule, content)) > 0
}

func validateLHCGroupContent(rule playRule, content string) bool {
	content = strings.TrimSpace(content)
	if content == "" {
		return false
	}
	if rule.BetMode == "tuotou" || strings.Contains(rule.BetMode, "_dp") {
		return strings.Contains(content, "|") || strings.Contains(content, "#") || len(parseLHCNumbers(content)) > 0
	}
	if rule.BetMode == "qima" {
		picks := parseLHCQimaPicks(content)
		return len(picks) > 0
	}
	if rule.BetMode == "zongxiao" {
		return len(parseLHCZongxiaoPicks(content)) > 0
	}
	if len(parseLHCNumbers(content)) > 0 {
		return true
	}
	if len(parseLHCZodiacs(content)) > 0 {
		return true
	}
	return len(parseTextTokens(content)) > 0
}

// BuildBetPayload builds deterministic settlement payload for a manual bet order.
func BuildBetPayload(playMethod, orderNo string) []byte {
	playMethod = strings.TrimSpace(playMethod)
	if playMethod == "" {
		playMethod = "定位胆万位"
	}
	cfg := configFromPlayMethod(playMethod)
	rule := resolvePlayRule(cfg, playMethod)
	payload := BetPayload{
		PlayMethod:   playMethod,
		PlayTypeID:   rule.PlayTypeID,
		SubPlayID:    rule.SubPlayID,
		GroupContent: demoGroupContent(orderNo, rule),
	}
	raw, _ := json.Marshal(payload)
	return raw
}

// EvaluateBetPayload judges hit/odds using the same engine as the scheme worker.
func EvaluateBetPayload(payload []byte, balls []string) (hit bool, odds float64) {
	var p BetPayload
	if len(payload) > 0 {
		_ = json.Unmarshal(payload, &p)
	}
	playMethod := strings.TrimSpace(p.PlayMethod)
	if playMethod == "" {
		playMethod = "定位胆万位"
	}

	typeID := strings.TrimSpace(p.TypeID)
	if typeID == "" {
		typeID = strings.TrimSpace(p.PlayTypeID)
	}
	subID := strings.TrimSpace(p.SubID)
	if subID == "" {
		subID = strings.TrimSpace(p.SubPlayID)
	}

	rule := resolvePlayRuleFromBetPayload(p)
	content := strings.TrimSpace(p.GroupContent)
	if content == "" {
		content = demoGroupContent("fallback", rule)
	}
	ev := evaluatePlayHit(rule, balls, content, false, "", rule.PositionIdx)
	return ev.Hit, ev.Odds
}

func CalcOrderPnL(amount float64, hit bool, odds float64) float64 {
	return calcPnLWithOdds(amount, hit, odds)
}

func configFromPlayMethod(playMethod string) map[string]interface{} {
	cfg := map[string]interface{}{"playMethod": playMethod}
	switch {
	case strings.Contains(playMethod, "五星"):
		cfg["playTypeId"] = "wuxing"
	case strings.Contains(playMethod, "四星"), strings.Contains(playMethod, "后四"):
		cfg["playTypeId"] = "sixing"
	case strings.Contains(playMethod, "前中后三"):
		cfg["playTypeId"] = "qianzhonghou3"
	case strings.Contains(playMethod, "前后三"):
		cfg["playTypeId"] = "qianhou3"
	case strings.Contains(playMethod, "前三"):
		cfg["playTypeId"] = "qian3"
	case strings.Contains(playMethod, "中三"):
		cfg["playTypeId"] = "zhong3"
	case strings.Contains(playMethod, "后三"):
		cfg["playTypeId"] = "hou3"
	case strings.Contains(playMethod, "前二"):
		cfg["playTypeId"] = "qian2"
	case strings.Contains(playMethod, "后二"):
		cfg["playTypeId"] = "hou2"
	default:
		cfg["playTypeId"] = "dingwei"
	}
	if posSub := dingweiSubFromMethod(playMethod); posSub != "" {
		cfg["subPlayId"] = posSub
	}
	switch {
	case strings.Contains(playMethod, "直选复式"):
		cfg["subPlayId"] = "zhixuan_fs"
	case strings.Contains(playMethod, "直选单式"):
		cfg["subPlayId"] = "zhixuan_ds"
	case strings.Contains(playMethod, "组三"):
		cfg["subPlayId"] = "zuxuan_fs"
	case strings.Contains(playMethod, "组六"):
		cfg["subPlayId"] = "zuxuan_fs"
	case strings.Contains(playMethod, "组选"):
		cfg["subPlayId"] = "zuxuan_fs"
	}
	return cfg
}

func dingweiSubFromMethod(playMethod string) string {
	switch {
	case strings.Contains(playMethod, "万位"):
		return "dingwei_wan"
	case strings.Contains(playMethod, "千位"):
		return "dingwei_qian"
	case strings.Contains(playMethod, "百位"):
		return "dingwei_bai"
	case strings.Contains(playMethod, "十位"):
		return "dingwei_shi"
	case strings.Contains(playMethod, "个位"):
		return "dingwei_ge"
	default:
		return ""
	}
}

// PlayIDsFromMethod resolves playTypeId/subPlayId from a Chinese playMethod label.
func PlayIDsFromMethod(playMethod string) (playTypeID, subPlayID string) {
	cfg := configFromPlayMethod(playMethod)
	if v, ok := cfg["playTypeId"].(string); ok && v != "" {
		playTypeID = v
	} else {
		playTypeID = "dingwei"
	}
	if v, ok := cfg["subPlayId"].(string); ok {
		subPlayID = v
	}
	return playTypeID, subPlayID
}

// DemoGroupContentForSubPlay 为矩阵/冒烟测试生成最小合法 groupContent。
func DemoGroupContentForSubPlay(template, typeID, subID, betMode, label, seed string) string {
	rule := resolvePlayRuleFromBetPayload(BetPayload{
		PlayTemplate: template,
		TypeID:       typeID,
		SubID:        subID,
		BetMode:      betMode,
		PlayMethod:   label,
	})
	return demoGroupContent(seed, rule)
}

func demoGroupContent(orderNo string, rule playRule) string {
	h := fnv.New32a()
	_, _ = h.Write([]byte(orderNo))
	seed := h.Sum32()
	nextDigit := func() string {
		seed = seed*1664525 + 1013904223
		return strconv.Itoa(int(seed % 10))
	}
	if rule.SegmentLen == 1 {
		picks := make([]string, 3)
		for i := range picks {
			picks[i] = nextDigit()
		}
		return strings.Join(picks, ",")
	}
	sub := rule.SubPlayID
	if sub == "zhixuan_ds" {
		digits := make([]string, rule.SegmentLen)
		for i := range digits {
			digits[i] = nextDigit()
		}
		return strings.Join(digits, "")
	}
	if sub == "zuxuan_fs" {
		pool := make([]string, 5)
		for i := range pool {
			pool[i] = nextDigit()
		}
		return strings.Join(pool, ",")
	}
	if rule.SegmentLen >= 4 {
		lines := make([]string, rule.SegmentLen)
		for i := range lines {
			lines[i] = nextDigit()
		}
		return strings.Join(lines, "\n")
	}
	pool := make([]string, 5)
	for i := range pool {
		pool[i] = nextDigit()
	}
	return strings.Join(pool, ",")
}
