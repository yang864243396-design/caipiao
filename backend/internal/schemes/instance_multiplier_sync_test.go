package schemes_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/schemes"
)

// TestDefinitionMultiplierAndCloudInstanceStayInSync guards the regression
// where config.multCoeff and scheme_instances.multiplier drifted apart, while
// the worker only used the latter for real bet calculation.
func TestDefinitionMultiplierAndCloudInstanceStayInSync(t *testing.T) {
	env := newE2EEnv(t)
	ctx := context.Background()
	name := fmt.Sprintf("multiplier-sync-%d", time.Now().UnixNano())

	def, err := env.svc.CreateDefinition(ctx, env.account, schemes.CreateDefinitionInput{
		Kind:        "custom",
		SchemeName:  name,
		LotteryCode: env.lottery,
		RunTypeID:   "fixed_rotate",
		PlayTypeID:  env.playTypeID,
		SubPlayID:   env.subPlayID,
	})
	if err != nil {
		t.Fatalf("CreateDefinition: %v", err)
	}
	t.Cleanup(func() {
		_, _ = env.pool.Exec(context.Background(), `DELETE FROM scheme_instances WHERE definition_id = $1`, def.ID)
		_, _ = env.pool.Exec(context.Background(), `DELETE FROM scheme_definitions WHERE id = $1`, def.ID)
	})

	if _, err := env.svc.UpdateDefinition(ctx, env.account, def.ID, schemes.UpdateDefinitionPatch{
		HasMultCoeff: true,
		MultCoeff:    "3",
	}); err != nil {
		t.Fatalf("UpdateDefinition(multCoeff=3): %v", err)
	}
	added, err := env.svc.AddDefinitionToCloud(ctx, env.account, def.ID, "private", schemes.AddToCloudConfigPatch{
		SchemeFunds: "1000",
		BetUnit:     "1",
	})
	if err != nil {
		t.Fatalf("AddDefinitionToCloud: %v", err)
	}
	if got := added.Instance.Multiplier; got != 3 {
		t.Fatalf("new cloud instance multiplier=%v, want 3 from config.multCoeff", got)
	}

	if _, err := env.svc.UpdateDefinition(ctx, env.account, def.ID, schemes.UpdateDefinitionPatch{
		HasMultCoeff: true,
		MultCoeff:    "4",
	}); err != nil {
		t.Fatalf("UpdateDefinition(multCoeff=4): %v", err)
	}
	inst, err := env.q.GetSchemeInstanceByDefinitionID(ctx, def.ID)
	if err != nil {
		t.Fatalf("GetSchemeInstanceByDefinitionID: %v", err)
	}
	value, valueErr := inst.Multiplier.Float64Value()
	if valueErr != nil || !value.Valid {
		t.Fatalf("read saved instance multiplier: %v", valueErr)
	}
	if got := value.Float64; got != 4 {
		t.Fatalf("saved instance multiplier=%v, want 4 after config save", got)
	}

	if _, err := env.svc.UpdateInstanceMultiplier(ctx, env.account, added.Instance.ID, 5); err != nil {
		t.Fatalf("UpdateInstanceMultiplier(5): %v", err)
	}
	updatedDef, err := env.q.GetSchemeDefinitionByIDAndMember(ctx, sqlcdb.GetSchemeDefinitionByIDAndMemberParams{
		ID:       def.ID,
		MemberID: env.memberID,
	})
	if err != nil {
		t.Fatalf("GetSchemeDefinitionByIDAndMember: %v", err)
	}
	var cfg map[string]interface{}
	if err := json.Unmarshal(updatedDef.Config, &cfg); err != nil {
		t.Fatalf("unmarshal updated config: %v", err)
	}
	if got := fmt.Sprint(cfg["multCoeff"]); got != "5" {
		t.Fatalf("config.multCoeff=%q, want 5 after cloud-card save", got)
	}
}
