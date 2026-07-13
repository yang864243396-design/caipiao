package rulessync

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"caipiao/backend/internal/db"
)

// TemplateBinding 本地 play_template ↔ 第三方 rules/v2 顶层键。
type TemplateBinding struct {
	TemplateCode     string
	GuajiRulesTypeID string
}

// DefaultBindings 本地 play_template ↔ 第三方 rules/v2 顶层键。
//
// type 6（3D）、type 11（加拿大28）暂不接入。
// 波场 3/6/15 秒彩等极速盘使用 type 7（快速彩），见 fast_ssc_std。
var DefaultBindings = []TemplateBinding{
	{TemplateCode: "ssc_std", GuajiRulesTypeID: "1"},
	{TemplateCode: "fast_ssc_std", GuajiRulesTypeID: "7"},
	{TemplateCode: "syxw_std", GuajiRulesTypeID: "2"},
	{TemplateCode: "pk10_std", GuajiRulesTypeID: "3"},
	{TemplateCode: "k3_std", GuajiRulesTypeID: "4"},
	{TemplateCode: "pc28_std", GuajiRulesTypeID: "5"},
	{TemplateCode: "lhc_std", GuajiRulesTypeID: "8"},
}

type PlayTypeRow struct {
	TypeID    string
	Label     string
	SortOrder int
}

type SubPlayRow struct {
	TypeID           string
	SubID            string
	Label            string
	SortOrder        int
	OutboundPlayCode string
	SegmentRule      json.RawMessage
}

type SyncPlan struct {
	TemplateCode     string
	GuajiRulesTypeID string
	RulesTypeName    string
	PlayTypes        []PlayTypeRow
	SubPlays         []SubPlayRow
}

// BuildPlan 将 rules/v2 单模板转为本地 play_types + sub_plays（1:1）。
//
// 对应关系（以台湾5分彩 / ssc_std / type_id=1 为例）：
//   - 彩种类型名 = data["1"].name（时时彩）→ play_templates.label
//   - 玩法类型   = groups[i].name（前三码）→ play_types.label
//   - 子玩法     = rule.full_name（前三直选复式）→ sub_plays.label
//     无 full_name 时回退 rule.name；segment_rule 仍保留 guajiFullName。
//   - 对外 rule_id = rule.id → sub_plays.outbound_play_code
func BuildPlan(templateCode, rulesTypeID string, tpl RulesTemplate) (SyncPlan, error) {
	templateCode = strings.TrimSpace(templateCode)
	rulesTypeID = strings.TrimSpace(rulesTypeID)
	if templateCode == "" || rulesTypeID == "" {
		return SyncPlan{}, fmt.Errorf("templateCode/rulesTypeID 不能为空")
	}
	if strings.TrimSpace(tpl.Name) == "" {
		return SyncPlan{}, fmt.Errorf("rules/v2 type %s 无 name", rulesTypeID)
	}

	plan := SyncPlan{
		TemplateCode:     templateCode,
		GuajiRulesTypeID: rulesTypeID,
		RulesTypeName:    strings.TrimSpace(tpl.Name),
	}

	for gi, group := range tpl.Groups {
		groupName := strings.TrimSpace(group.Name)
		if groupName == "" {
			continue
		}
		typeID := fmt.Sprintf("g%03d", gi+1)
		plan.PlayTypes = append(plan.PlayTypes, PlayTypeRow{
			TypeID:    typeID,
			Label:     groupName,
			SortOrder: gi + 1,
		})

		subOrder := 0
		for _, team := range group.Team {
			teamName := strings.TrimSpace(team.Name)
			for _, rule := range team.Rule {
				if !rule.Active {
					continue
				}
				ruleID := strings.TrimSpace(rule.ID)
				ruleName := strings.TrimSpace(rule.Name)
				if ruleID == "" || ruleName == "" {
					continue
				}
				subOrder++
				fullName := strings.TrimSpace(rule.FullName)
				label := fullName
				if label == "" {
					label = ruleName
				}
				seg, _ := json.Marshal(map[string]string{
					"guajiGroup":    groupName,
					"guajiTeam":     teamName,
					"guajiFullName": fullName,
					"guajiRuleId":   ruleID,
				})
				plan.SubPlays = append(plan.SubPlays, SubPlayRow{
					TypeID:           typeID,
					SubID:            ruleID,
					Label:            label,
					SortOrder:        subOrder,
					OutboundPlayCode: ruleID,
					SegmentRule:      seg,
				})
			}
		}
	}
	if len(plan.PlayTypes) == 0 || len(plan.SubPlays) == 0 {
		return SyncPlan{}, fmt.Errorf("rules/v2 type %s 无有效玩法", rulesTypeID)
	}
	return plan, nil
}

func ApplyPlan(ctx context.Context, pool *db.Pool, plan SyncPlan) error {
	if pool == nil {
		return fmt.Errorf("db pool is nil")
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `
		INSERT INTO play_templates (code, label, version, guaji_rules_type_id)
		VALUES ($1, $2, 1, $3)
		ON CONFLICT (code) DO UPDATE SET
			label = EXCLUDED.label,
			guaji_rules_type_id = EXCLUDED.guaji_rules_type_id`,
		plan.TemplateCode, plan.RulesTypeName, plan.GuajiRulesTypeID); err != nil {
		return fmt.Errorf("upsert play_templates: %w", err)
	}

	if _, err := tx.Exec(ctx, `DELETE FROM sub_plays WHERE template_code = $1`, plan.TemplateCode); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM play_types WHERE template_code = $1`, plan.TemplateCode); err != nil {
		return err
	}

	for _, pt := range plan.PlayTypes {
		if _, err := tx.Exec(ctx, `
			INSERT INTO play_types (template_code, type_id, label, sort_order, enabled)
			VALUES ($1, $2, $3, $4, true)`,
			plan.TemplateCode, pt.TypeID, pt.Label, pt.SortOrder); err != nil {
			return fmt.Errorf("insert play_type %s: %w", pt.TypeID, err)
		}
	}

	for _, sp := range plan.SubPlays {
		if _, err := tx.Exec(ctx, `
			INSERT INTO sub_plays (template_code, type_id, sub_id, label, sort_order, segment_rule, outbound_play_code, enabled)
			VALUES ($1, $2, $3, $4, $5, $6, $7, true)`,
			plan.TemplateCode, sp.TypeID, sp.SubID, sp.Label, sp.SortOrder, sp.SegmentRule, sp.OutboundPlayCode); err != nil {
			return fmt.Errorf("insert sub_play %s/%s: %w", sp.TypeID, sp.SubID, err)
		}
	}

	return tx.Commit(ctx)
}
