package sqlcdb

import (
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
)

func TestPlayRuleQueryContractsExposeLocatorAndStrategyClaim(t *testing.T) {
	locator := ResolvePublishedPlayRuleSpecParams{
		TemplateCode: "ssc_std",
		TypeID:       "g001",
		SubID:        "1",
		LotteryCode:  pgtype.Text{String: "tron_ffc_3s", Valid: true},
	}
	if locator.TemplateCode == "" || !locator.LotteryCode.Valid {
		t.Fatal("published rule locator fields must be generated")
	}

	claim := TryClaimSchemeStrategyEvaluationParams{
		InstanceID:  "inst-test",
		LotteryCode: "tron_ffc_3s",
		PeriodNo:    "85421401",
	}
	if claim.PeriodNo == "" {
		t.Fatal("strategy claim period field must be generated")
	}
}
