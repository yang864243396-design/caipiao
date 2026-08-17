package schemes

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"caipiao/backend/internal/playrules"
)

// ErrStrategyRuleUnavailable is intentionally terminal for local strategy
// evaluation. Callers must not silently fall back to display-name heuristics.
var ErrStrategyRuleUnavailable = errors.New("strategy rule unavailable")

func evaluateFrozenRule(
	snapshot playrules.Snapshot,
	base playRule,
	balls []string,
	content string,
	contrary bool,
	contraryContent string,
) (betEvaluation, error) {
	rule, err := compileFrozenRule(snapshot, base)
	if err != nil {
		return betEvaluation{}, err
	}
	if contrary {
		content = strings.TrimSpace(contraryContent)
	}
	if strings.TrimSpace(content) == "" {
		return betEvaluation{}, fmt.Errorf("%w: empty frozen bet content", ErrStrategyRuleUnavailable)
	}

	switch snapshot.EvaluatorKey {
	case "ssc.direct", "ssc.group", "ssc.sum", "ssc.attribute":
		if evaluation, ok := evaluateSSCByBetMode(rule, balls, content); ok {
			return evaluation, nil
		}
	case "lhc.guoguan", "lhc.duipeng", "lhc.attribute":
		if evaluation, ok := evaluateLHCByBetMode(rule, balls, content); ok {
			return evaluation, nil
		}
	case "pk10.direct":
		if evaluation, ok := evaluatePK10ByBetMode(rule, balls, content); ok {
			return evaluation, nil
		}
	case "syxw.renxuan":
		if evaluation, ok := evaluateSYXWByBetMode(rule, balls, content); ok {
			return evaluation, nil
		}
	case "k3.standard":
		if evaluation, ok := evaluateK3ByBetMode(rule, balls, content); ok {
			return evaluation, nil
		}
	case "pc28.standard":
		if evaluation, ok := evaluatePC28ByBetMode(rule, balls, content); ok {
			return evaluation, nil
		}
	default:
		return betEvaluation{}, fmt.Errorf("%w: unsupported evaluator %q", ErrStrategyRuleUnavailable, snapshot.EvaluatorKey)
	}
	return betEvaluation{}, fmt.Errorf("%w: evaluator %q cannot evaluate %s/%s/%s", ErrStrategyRuleUnavailable, snapshot.EvaluatorKey, snapshot.Locator.TemplateCode, snapshot.Locator.TypeID, snapshot.Locator.SubID)
}

func compileFrozenRule(snapshot playrules.Snapshot, base playRule) (playRule, error) {
	if strings.TrimSpace(snapshot.Locator.TemplateCode) == "" || strings.TrimSpace(snapshot.EvaluatorKey) == "" {
		return playRule{}, fmt.Errorf("%w: missing frozen rule identity", ErrStrategyRuleUnavailable)
	}
	var spec playrules.EvaluationSpec
	decoder := json.NewDecoder(bytes.NewReader(snapshot.EvaluationSpec))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&spec); err != nil {
		return playRule{}, fmt.Errorf("%w: invalid frozen rule spec: %v", ErrStrategyRuleUnavailable, err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return playRule{}, fmt.Errorf("%w: invalid frozen rule trailing data", ErrStrategyRuleUnavailable)
	}
	if strings.TrimSpace(spec.Mode) == "" || spec.NumberMin > spec.NumberMax {
		return playRule{}, fmt.Errorf("%w: incomplete frozen rule spec", ErrStrategyRuleUnavailable)
	}
	mode := strings.TrimSpace(spec.BetMode)
	if mode == "" {
		mode = strings.TrimSpace(spec.Mode)
	}
	if mode == "" {
		return playRule{}, fmt.Errorf("%w: frozen rule has no bet mode", ErrStrategyRuleUnavailable)
	}

	rule := base
	rule.PlayTemplate = strings.TrimSpace(snapshot.Locator.TemplateCode)
	rule.PlayTypeID = strings.TrimSpace(snapshot.Locator.TypeID)
	rule.SubPlayID = strings.TrimSpace(snapshot.Locator.SubID)
	rule.BetMode = mode
	rule.NumberPoolMin = spec.NumberMin
	rule.NumberPoolMax = spec.NumberMax
	rule.HezhiZuxuan = spec.HezhiZuxuan
	if strings.TrimSpace(spec.CatalogSubID) != "" {
		rule.CatalogSubID = strings.TrimSpace(spec.CatalogSubID)
	}
	if spec.SegmentLen > 0 {
		rule.SegmentStart = spec.SegmentStart
		rule.SegmentLen = spec.SegmentLen
		rule.PositionIdx = spec.PositionIdx
	}
	return rule, nil
}
