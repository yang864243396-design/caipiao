package schemes_test

import (
	"context"
	"testing"

	"github.com/joho/godotenv"

	"caipiao/backend/internal/config"
	"caipiao/backend/internal/db"
	"caipiao/backend/internal/schemes"
)

func TestStopAndStartInstanceIntegration(t *testing.T) {
	_ = godotenv.Load("../../.env")
	cfg := config.Load()
	if cfg.DatabaseURL == "" {
		t.Skip("DATABASE_URL not set")
	}
	pool, err := db.Connect(context.Background(), cfg.DatabaseURL, cfg.DBMaxConns, cfg.DBMinConns)
	if err != nil {
		t.Skip(err)
	}
	defer pool.Close()

	svc := schemes.NewService(pool, nil)
	rows, err := svc.ListInstances(context.Background(), cfg.ClientDemoAccount, "")
	if err != nil {
		t.Fatalf("ListInstances: %v", err)
	}
	var targetID string
	var targetStatus string
	for _, item := range rows.Items {
		if item.ID == "inst-1-1781164314120" {
			targetID = item.ID
			targetStatus = item.Status
			break
		}
	}
	if targetID == "" && len(rows.Items) > 0 {
		targetID = rows.Items[0].ID
		targetStatus = rows.Items[0].Status
	}
	if targetID == "" {
		t.Skip("no instances for demo account")
	}
	t.Logf("instance %s status=%s", targetID, targetStatus)
	if targetStatus == "running" {
		if _, err = svc.StopInstance(context.Background(), cfg.ClientDemoAccount, targetID); err != nil {
			t.Fatalf("StopInstance: %v", err)
		}
	} else if targetStatus != "pending" && targetStatus != "paused" {
		t.Skipf("instance %s status=%s not startable", targetID, targetStatus)
	}
	inst, err := svc.StartInstance(context.Background(), cfg.ClientDemoAccount, targetID)
	if err != nil {
		t.Fatalf("StartInstance(%s): %v", targetID, err)
	}
	if inst.Turnover != 0 || inst.PnL != 0 || inst.LookbackPnL != 0 {
		t.Fatalf("expected reset metrics, got turnover=%v pnl=%v lookback=%v", inst.Turnover, inst.PnL, inst.LookbackPnL)
	}
}
