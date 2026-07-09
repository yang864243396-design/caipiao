package schemes_test

import (
	"context"
	"testing"

	"github.com/joho/godotenv"

	"caipiao/backend/internal/config"
	"caipiao/backend/internal/db"
	"caipiao/backend/internal/schemes"
)

func TestUpdateInstanceMultiplierIntegration(t *testing.T) {
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
	if len(rows.Items) == 0 {
		t.Skip("no instances for demo account")
	}
	id := rows.Items[0].ID
	_, err = svc.UpdateInstanceMultiplier(context.Background(), cfg.ClientDemoAccount, id, 2)
	if err != nil {
		t.Fatalf("UpdateInstanceMultiplier(%s): %v", id, err)
	}
}
