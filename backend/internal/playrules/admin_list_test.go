package playrules

import (
	"testing"
	"time"

	"caipiao/backend/internal/db/sqlcdb"
	"github.com/jackc/pgx/v5/pgtype"
)

func TestMapAdminRowsKeepsDefaultAndLotteryOverrideScopes(t *testing.T) {
	rows := []sqlcdb.ListPublishedPlayRuleSpecsForAdminRow{
		{ID: 1, TemplateCode: "ssc_std", TypeID: "g001", SubID: "1", RuleVersion: 1, EvaluatorKey: "ssc.direct", EvaluatorVersion: 1, StrategyEnabled: true, PublishedAt: pgtype.Timestamptz{Time: time.Date(2026, 8, 17, 1, 2, 3, 0, time.UTC), Valid: true}},
		{ID: 2, TemplateCode: "ssc_std", TypeID: "g001", SubID: "1", LotteryCode: pgtype.Text{String: "tron_ffc_3s", Valid: true}, RuleVersion: 2, EvaluatorKey: "ssc.direct", EvaluatorVersion: 1, StrategyEnabled: true},
	}
	items := MapAdminRows(rows)
	if len(items) != 2 || items[0].LotteryCode != "" || items[1].LotteryCode != "tron_ffc_3s" {
		t.Fatalf("items = %+v, want default plus lottery override", items)
	}
	if items[0].PublishedAt != "2026-08-17T01:02:03Z" {
		t.Fatalf("publishedAt = %q", items[0].PublishedAt)
	}
}
