package schemes_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"
)

func TestStopAndStartInstanceIntegration(t *testing.T) {
	if os.Getenv("RUN_SCHEME_LIFECYCLE_INTEGRATION") != "1" {
		t.Skip("set RUN_SCHEME_LIFECYCLE_INTEGRATION=1 to run the isolated simulated-scheme lifecycle check")
	}
	ctx := context.Background()
	env := newE2EEnv(t)
	targetID := env.createRunningInstance(t, fmt.Sprintf("E2E-resume-%d", time.Now().UnixNano()), map[string]interface{}{
		"runTypeId":    "fixed_rotate",
		"betMode":      "dingwei",
		"schemeGroups": []string{"1,2", "3,4"},
	})
	t.Logf("created isolated simulated instance %s", targetID)
	if _, err := env.svc.StopInstance(ctx, env.account, targetID); err != nil {
		t.Fatalf("StopInstance: %v", err)
	}
	inst, err := env.svc.StartInstance(ctx, env.account, targetID)
	if err != nil {
		t.Fatalf("StartInstance(%s): %v", targetID, err)
	}
	if inst.Turnover != 0 || inst.PnL != 0 || inst.LookbackPnL != 0 {
		t.Fatalf("expected reset metrics, got turnover=%v pnl=%v lookback=%v", inst.Turnover, inst.PnL, inst.LookbackPnL)
	}
}
