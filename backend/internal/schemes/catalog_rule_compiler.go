package schemes

import (
	"encoding/json"
	"fmt"
	"strings"

	"caipiao/backend/internal/playrules"
)

// CatalogRuleCompileInput contains the stable catalogue identity plus the
// stored bet mode required to reproduce the existing settlement resolver.
type CatalogRuleCompileInput struct {
	Locator    playrules.Locator
	BetMode    string
	PlayMethod string
}

// EvaluateCatalogRule is the read-only replay entry point. It compiles the
// same declarative specification used for publication, then uses the frozen
// evaluator path without persisting a snapshot or changing a bet record.
func EvaluateCatalogRule(input CatalogRuleCompileInput, balls []string, content string) (betEvaluation, error) {
	compiled, err := CompileCatalogRule(input)
	if err != nil {
		return betEvaluation{}, err
	}
	spec, err := json.Marshal(compiled.Spec)
	if err != nil {
		return betEvaluation{}, fmt.Errorf("marshal compiled rule: %w", err)
	}
	return evaluateFrozenRule(playrules.Snapshot{
		Locator:          input.Locator,
		RuleVersion:      1,
		EvaluatorVersion: 1,
		EvaluatorKey:     compiled.EvaluatorKey,
		EvaluationSpec:   spec,
		StrategyEnabled:  true,
	}, playRule{}, balls, content, false, "")
}

// CompiledCatalogRule is a declarative, executable rule specification. The
// evaluator key selects existing Go code; it never stores executable code.
type CompiledCatalogRule struct {
	EvaluatorKey string
	Spec         playrules.EvaluationSpec
}

// CompileCatalogRule converts an enabled catalogue play into the same rule
// shape consumed by frozen strategy settlement.
func CompileCatalogRule(input CatalogRuleCompileInput) (CompiledCatalogRule, error) {
	locator := input.Locator
	locator.TemplateCode = strings.TrimSpace(locator.TemplateCode)
	locator.TypeID = strings.TrimSpace(locator.TypeID)
	locator.SubID = strings.TrimSpace(locator.SubID)
	if locator.TemplateCode == "" || locator.TypeID == "" || locator.SubID == "" {
		return CompiledCatalogRule{}, fmt.Errorf("catalog rule locator is incomplete")
	}
	betMode := strings.TrimSpace(input.BetMode)
	if betMode == "" {
		betMode = inferCatalogBetMode(input.PlayMethod)
	}
	base, ok := resolveCatalogPlayRule(map[string]interface{}{
		"playTemplate": locator.TemplateCode,
		"typeId":       locator.TypeID,
		"subId":        locator.SubID,
		"betMode":      betMode,
		"playMethod":   strings.TrimSpace(input.PlayMethod),
	})
	if !ok || strings.TrimSpace(base.BetMode) == "" {
		return CompiledCatalogRule{}, fmt.Errorf("catalog rule %s/%s/%s has no executable bet mode", locator.TemplateCode, locator.TypeID, locator.SubID)
	}
	key, err := evaluatorKeyForCatalogRule(base)
	if err != nil {
		return CompiledCatalogRule{}, err
	}
	min, max := base.NumberPoolMin, base.NumberPoolMax
	if min == 0 && max == 0 {
		min, max = catalogNumberRange(locator.TemplateCode)
	}
	return CompiledCatalogRule{
		EvaluatorKey: key,
		Spec: playrules.EvaluationSpec{
			Mode:         base.BetMode,
			NumberMin:    min,
			NumberMax:    max,
			SegmentStart: base.SegmentStart,
			SegmentLen:   base.SegmentLen,
			PositionIdx:  base.PositionIdx,
			BetMode:      base.BetMode,
			CatalogSubID: base.CatalogSubID,
			HezhiZuxuan:  base.HezhiZuxuan,
		},
	}, nil
}

func inferCatalogBetMode(playMethod string) string {
	name := strings.TrimSpace(playMethod)
	switch {
	case strings.Contains(name, "过关"):
		return "guoguan"
	case strings.Contains(name, "任意对碰"):
		return "renyi_dp"
	case strings.Contains(name, "生肖对碰"):
		return "sx_dp"
	case strings.Contains(name, "尾数对碰"):
		return "ws_dp"
	case strings.Contains(name, "生尾对碰"):
		return "sw_dp"
	case strings.Contains(name, "组选和值"), strings.Contains(name, "直选和值"), strings.Contains(name, "和值"):
		return "hezhi"
	case strings.Contains(name, "跨度"):
		return "kuadu"
	case strings.Contains(name, "不定位"):
		return "budingwei"
	case strings.Contains(name, "大小单双"):
		return "dxds"
	case strings.Contains(name, "龙虎"):
		return "longhu"
	case strings.Contains(name, "庄闲"):
		return "zhuangxian"
	case strings.Contains(name, "混合"):
		return "hunhe"
	case strings.Contains(name, "包胆"):
		return "baodan"
	case strings.Contains(name, "组选单式"):
		return "zuxuan_ds"
	case strings.Contains(name, "组选复式"):
		return "zuxuan_fs"
	case strings.Contains(name, "组三"):
		return "zu3"
	case strings.Contains(name, "组六"):
		return "zu6"
	case strings.Contains(name, "直选单式"), strings.HasSuffix(name, "单式"):
		return "danshi"
	case strings.Contains(name, "复式"):
		return "fushi"
	default:
		return ""
	}
}

func evaluatorKeyForCatalogRule(rule playRule) (string, error) {
	mode := strings.ToLower(strings.TrimSpace(rule.BetMode))
	switch rule.PlayTemplate {
	case "ssc_std", "fast_ssc_std":
		switch mode {
		case "hezhi", "kuadu":
			return "ssc.sum", nil
		case "dxds", "daxiao", "danshuang", "longhu", "longhuhe", "zhuangxian", "weishu", "teshu":
			return "ssc.attribute", nil
		case "zuxuan_fs", "zuxuan_ds", "zu3", "zu6", "zuhe", "baodan", "hunhe", "zu24", "zu12", "zu4", "zu60", "zu30", "zu20", "zu10", "zu5", "zu120":
			return "ssc.group", nil
		default:
			return "ssc.direct", nil
		}
	case "lhc_std":
		if mode == "guoguan" {
			return "lhc.guoguan", nil
		}
		if strings.HasSuffix(mode, "_dp") || mode == "renyi_dp" {
			return "lhc.duipeng", nil
		}
		return "lhc.attribute", nil
	case "pk10_std":
		return "pk10.direct", nil
	case "syxw_std":
		return "syxw.renxuan", nil
	case "k3_std":
		return "k3.standard", nil
	case "pc28_std":
		return "pc28.standard", nil
	default:
		return "", fmt.Errorf("catalog rule %s/%s/%s has unsupported template", rule.PlayTemplate, rule.PlayTypeID, rule.CatalogSubID)
	}
}

func catalogNumberRange(template string) (int, int) {
	switch strings.TrimSpace(template) {
	case "lhc_std":
		return 1, 49
	case "pk10_std":
		return 1, 10
	case "syxw_std":
		return 1, 11
	case "k3_std":
		return 1, 6
	case "pc28_std":
		return 0, 27
	default:
		return 0, 9
	}
}
