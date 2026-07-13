package schemes

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"

	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/guajibet"
)

var errGuajiRuleIDMissing = errors.New("第三方 rule_id 未配置，请执行 guaji-rules-sync")

var dingweiPositionLabels = []string{"万位", "千位", "百位", "十位", "个位"}

// lookupSubPlay 查子玩法；兼容方案存 bet_mode、legacy dingwei_* 或 rules 同步后 sub_id 变更。
func lookupSubPlay(ctx context.Context, q *sqlcdb.Queries, template, typeID, subID, betMode string, positionIdx int) (sqlcdb.GetSubPlayRow, error) {
	template = strings.TrimSpace(template)
	typeID = strings.TrimSpace(typeID)
	subID = strings.TrimSpace(subID)
	betMode = strings.TrimSpace(betMode)
	if template == "" || typeID == "" || subID == "" {
		return sqlcdb.GetSubPlayRow{}, fmt.Errorf("sub play not found: empty key")
	}
	if q == nil {
		return sqlcdb.GetSubPlayRow{}, fmt.Errorf("sub play lookup unavailable")
	}

	row, err := q.GetSubPlay(ctx, sqlcdb.GetSubPlayParams{
		TemplateCode: template,
		TypeID:       typeID,
		SubID:        subID,
	})
	if err == nil {
		return row, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return sqlcdb.GetSubPlayRow{}, err
	}

	if legacySub := legacyDingweiSubID(subID, betMode, positionIdx); legacySub != "" && legacySub != subID {
		row, err = q.GetSubPlay(ctx, sqlcdb.GetSubPlayParams{
			TemplateCode: template,
			TypeID:       "dingwei",
			SubID:        legacySub,
		})
		if err == nil {
			return row, nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return sqlcdb.GetSubPlayRow{}, err
		}
	}

	rows, err := q.ListSubPlaysByTemplate(ctx, template)
	if err != nil {
		return sqlcdb.GetSubPlayRow{}, err
	}
	converted := make([]sqlcdb.GetSubPlayRow, len(rows))
	for i, r := range rows {
		converted[i] = subPlayFromListRow(r)
	}
	return lookupSubPlayFromRows(template, converted, typeID, subID, betMode, positionIdx)
}

func subPlayFromListRow(r sqlcdb.ListSubPlaysByTemplateRow) sqlcdb.GetSubPlayRow {
	return sqlcdb.GetSubPlayRow{
		TemplateCode:     r.TemplateCode,
		TypeID:           r.TypeID,
		SubID:            r.SubID,
		Label:            r.Label,
		SortOrder:        r.SortOrder,
		BetMode:          r.BetMode,
		SegmentRule:      r.SegmentRule,
		OutboundPlayCode: r.OutboundPlayCode,
		Enabled:          r.Enabled,
	}
}

func lookupSubPlayFromRows(template string, rows []sqlcdb.GetSubPlayRow, typeID, subID, betMode string, positionIdx int) (sqlcdb.GetSubPlayRow, error) {
	mode := resolveLookupBetMode(subID, betMode)

	var candidates []sqlcdb.GetSubPlayRow
	for _, r := range rows {
		if r.TypeID != typeID || !r.Enabled {
			continue
		}
		if r.SubID == subID {
			return r, nil
		}
		bm := strings.TrimSpace(textVal(r.BetMode))
		if bm != "" && (bm == subID || (mode != "" && bm == mode)) {
			candidates = append(candidates, r)
		}
	}
	if picked, ok := pickSubPlayCandidate(candidates, mode, positionIdx); ok {
		return picked, nil
	}

	if mode != "" {
		candidates = candidates[:0]
		for _, r := range rows {
			if r.TypeID != typeID || !r.Enabled {
				continue
			}
			if subPlayLabelMatchesMode(r.Label, mode) {
				candidates = append(candidates, r)
			}
		}
		if picked, ok := pickSubPlayCandidate(candidates, mode, positionIdx); ok {
			return picked, nil
		}
	}

	if legacySub := legacyDingweiSubID(subID, betMode, positionIdx); legacySub != "" {
		for _, r := range rows {
			if !r.Enabled {
				continue
			}
			if r.TypeID == "dingwei" && r.SubID == legacySub {
				return r, nil
			}
		}
	}

	return sqlcdb.GetSubPlayRow{}, fmt.Errorf("sub play not found: %s/%s/%s", template, typeID, subID)
}

func resolveLookupBetMode(subID, betMode string) string {
	subID = strings.TrimSpace(subID)
	betMode = strings.TrimSpace(betMode)
	if betMode != "" {
		return legacySubMode(subID, betMode)
	}
	if m := legacySubMode(subID, subID); m != "" {
		return m
	}
	return ""
}

func subPlayLabelMatchesMode(label, mode string) bool {
	label = strings.TrimSpace(label)
	if label == "" || mode == "" {
		return false
	}
	for _, kw := range betModeLabelKeywords(mode) {
		if kw != "" && strings.Contains(label, kw) {
			return true
		}
	}
	return false
}

func betModeLabelKeywords(mode string) []string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "dingwei":
		return []string{"定位胆"}
	case "fushi", "zhixuan_fs":
		return []string{"直选复式", "复式"}
	case "danshi", "zhixuan_ds":
		return []string{"直选单式", "单式"}
	case "zuxuan_fs", "zu3", "zu6":
		return []string{"组选复式", "组三", "组六", "组选"}
	case "hezhi":
		return []string{"和值"}
	case "longhu", "longhuhe":
		return []string{"龙虎"}
	case "dxds", "daxiao", "danshuang":
		return []string{"大小", "单双"}
	case "budingwei":
		return []string{"不定位"}
	default:
		return nil
	}
}

func isSemanticPlayToken(s string) bool {
	s = strings.ToLower(strings.TrimSpace(s))
	switch s {
	case "zhixuan_fs", "zhixuan_ds", "zuxuan_fs", "zuxuan_ds", "dingwei",
		"fushi", "danshi", "hezhi", "kuadu", "baodan", "hunhe", "zuhe",
		"zu3", "zu6", "budingwei", "dxds", "longhu", "longhuhe", "teshu", "weishu":
		return true
	}
	return strings.Contains(s, "zhixuan") || strings.Contains(s, "zuxuan") ||
		strings.HasPrefix(s, "dingwei_") || strings.HasPrefix(s, "sub_")
}

func pickSubPlayCandidate(candidates []sqlcdb.GetSubPlayRow, mode string, positionIdx int) (sqlcdb.GetSubPlayRow, bool) {
	if len(candidates) == 0 {
		return sqlcdb.GetSubPlayRow{}, false
	}
	if len(candidates) == 1 {
		return candidates[0], true
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].SortOrder == candidates[j].SortOrder {
			return candidates[i].SubID < candidates[j].SubID
		}
		return candidates[i].SortOrder < candidates[j].SortOrder
	})
	mode = strings.ToLower(strings.TrimSpace(mode))
	if mode == "dingwei" || strings.HasPrefix(mode, "dingwei") {
		idx := positionIdx
		if idx < 0 {
			idx = 0
		}
		for _, r := range candidates {
			if dingweiLabelMatchesPosition(r.Label, idx) {
				return r, true
			}
		}
		if idx >= len(candidates) {
			idx = len(candidates) - 1
		}
		return candidates[idx], true
	}
	if narrowed := narrowSubPlayByModeLabel(candidates, mode); len(narrowed) == 1 {
		return narrowed[0], true
	} else if len(narrowed) > 1 {
		candidates = narrowed
	}
	if best, ok := bestSubPlayByModeScore(candidates, mode); ok {
		return best, true
	}
	return sqlcdb.GetSubPlayRow{}, false
}

func narrowSubPlayByModeLabel(candidates []sqlcdb.GetSubPlayRow, mode string) []sqlcdb.GetSubPlayRow {
	want, avoid := modeLabelPreferAvoid(mode)
	if len(want) == 0 && len(avoid) == 0 {
		return nil
	}
	var preferred []sqlcdb.GetSubPlayRow
	for _, r := range candidates {
		label := strings.TrimSpace(r.Label)
		if labelHasAny(label, avoid) && !labelHasAny(label, want) {
			continue
		}
		if labelHasAny(label, want) {
			preferred = append(preferred, r)
		}
	}
	if len(preferred) > 0 {
		return preferred
	}
	var filtered []sqlcdb.GetSubPlayRow
	for _, r := range candidates {
		if labelHasAny(r.Label, avoid) {
			continue
		}
		filtered = append(filtered, r)
	}
	return filtered
}

func modeLabelPreferAvoid(mode string) (want, avoid []string) {
	switch mode {
	case "zhixuan_fs", "fushi":
		return []string{"直选复式"}, []string{"组选"}
	case "zhixuan_ds", "danshi":
		return []string{"直选单式"}, []string{"组选"}
	case "zuxuan_fs":
		return []string{"组选复式", "组选"}, nil
	case "zu3":
		return []string{"组三"}, []string{"组六"}
	case "zu6":
		return []string{"组六"}, []string{"组三"}
	default:
		return nil, nil
	}
}

func labelHasAny(label string, keys []string) bool {
	for _, k := range keys {
		if k != "" && strings.Contains(label, k) {
			return true
		}
	}
	return false
}

func bestSubPlayByModeScore(candidates []sqlcdb.GetSubPlayRow, mode string) (sqlcdb.GetSubPlayRow, bool) {
	if len(candidates) == 0 {
		return sqlcdb.GetSubPlayRow{}, false
	}
	bestIdx := -1
	bestScore := -1
	for i, r := range candidates {
		score := subPlayModeScore(r.Label, mode)
		if score > bestScore {
			bestScore = score
			bestIdx = i
		}
	}
	if bestIdx < 0 || bestScore <= 0 {
		return sqlcdb.GetSubPlayRow{}, false
	}
	return candidates[bestIdx], true
}

func subPlayModeScore(label, mode string) int {
	label = strings.TrimSpace(label)
	if label == "" || mode == "" {
		return 0
	}
	score := 0
	want, avoid := modeLabelPreferAvoid(mode)
	for _, w := range want {
		if strings.Contains(label, w) {
			score += 10
		}
	}
	for _, a := range avoid {
		if strings.Contains(label, a) {
			score -= 5
		}
	}
	for _, kw := range betModeLabelKeywords(mode) {
		if kw != "" && strings.Contains(label, kw) {
			score += 1
		}
	}
	return score
}

func dingweiLabelMatchesPosition(label string, positionIdx int) bool {
	if positionIdx < 0 || positionIdx >= len(dingweiPositionLabels) {
		return false
	}
	return strings.Contains(label, dingweiPositionLabels[positionIdx])
}

func legacyDingweiSubID(subID, betMode string, positionIdx int) string {
	subID = strings.TrimSpace(subID)
	betMode = strings.TrimSpace(betMode)
	mode := legacySubMode(subID, betMode)
	if mode == "" {
		mode = legacySubMode(subID, subID)
	}
	if mode != "dingwei" && !strings.HasPrefix(strings.ToLower(subID), "dingwei") {
		return ""
	}
	if strings.HasPrefix(strings.ToLower(subID), "dingwei_") {
		return subID
	}
	switch {
	case strings.HasPrefix(subID, "sub_"):
		return dingweiSubFromSchemeSub(subID)
	default:
		return dingweiSubFromPositionIndex(positionIdx)
	}
}

func dingweiSubFromSchemeSub(subID string) string {
	switch subID {
	case "sub_wan":
		return "dingwei_wan"
	case "sub_qian":
		return "dingwei_qian"
	case "sub_bai":
		return "dingwei_bai"
	case "sub_shi":
		return "dingwei_shi"
	case "sub_ge":
		return "dingwei_ge"
	default:
		return ""
	}
}

func dingweiSubFromPositionIndex(positionIdx int) string {
	switch positionIdx {
	case 0:
		return "dingwei_wan"
	case 1:
		return "dingwei_qian"
	case 2:
		return "dingwei_bai"
	case 3:
		return "dingwei_shi"
	case 4:
		return "dingwei_ge"
	default:
		return "dingwei_wan"
	}
}

func resolveGuajiRuleIDFromSubPlay(row sqlcdb.GetSubPlayRow) string {
	return guajibet.ExtractGuajiRuleID(textVal(row.OutboundPlayCode), row.SegmentRule, row.SubID)
}
