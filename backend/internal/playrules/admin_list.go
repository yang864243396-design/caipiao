package playrules

import "caipiao/backend/internal/db/sqlcdb"

// AdminRow is intentionally read-only. Rule editing and publication are not
// exposed by this rollout.
type AdminRow struct {
	ID               int64  `json:"id"`
	TemplateCode     string `json:"templateCode"`
	TypeID           string `json:"typeId"`
	SubID            string `json:"subId"`
	LotteryCode      string `json:"lotteryCode"`
	RuleVersion      int    `json:"ruleVersion"`
	EvaluatorKey     string `json:"evaluatorKey"`
	EvaluatorVersion int    `json:"evaluatorVersion"`
	StrategyEnabled  bool   `json:"strategyEnabled"`
	PublishedAt      string `json:"publishedAt"`
	UpdatedAt        string `json:"updatedAt"`
}

func MapAdminRows(rows []sqlcdb.ListPublishedPlayRuleSpecsForAdminRow) []AdminRow {
	items := make([]AdminRow, 0, len(rows))
	for _, row := range rows {
		lotteryCode := ""
		if row.LotteryCode.Valid {
			lotteryCode = row.LotteryCode.String
		}
		item := AdminRow{
			ID: row.ID, TemplateCode: row.TemplateCode, TypeID: row.TypeID, SubID: row.SubID,
			LotteryCode: lotteryCode, RuleVersion: int(row.RuleVersion), EvaluatorKey: row.EvaluatorKey,
			EvaluatorVersion: int(row.EvaluatorVersion), StrategyEnabled: row.StrategyEnabled,
		}
		if row.PublishedAt.Valid {
			item.PublishedAt = row.PublishedAt.Time.UTC().Format("2006-01-02T15:04:05Z")
		}
		if row.UpdatedAt.Valid {
			item.UpdatedAt = row.UpdatedAt.Time.UTC().Format("2006-01-02T15:04:05Z")
		}
		items = append(items, item)
	}
	return items
}
