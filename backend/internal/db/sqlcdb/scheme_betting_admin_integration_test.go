package sqlcdb_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/joho/godotenv"

	"caipiao/backend/internal/config"
	"caipiao/backend/internal/db"
	"caipiao/backend/internal/db/sqlcdb"
)

func TestSchemeBettingAdminQueriesAgainstMigratedSchema(t *testing.T) {
	_ = godotenv.Load("../../../.env")
	cfg := config.Load()
	if cfg.DatabaseURL == "" {
		t.Skip("DATABASE_URL not set")
	}
	pool, err := db.Connect(context.Background(), cfg.DatabaseURL, 2, 0)
	if err != nil {
		t.Skipf("database unavailable: %v", err)
	}
	defer pool.Close()

	ctx := context.Background()
	q := sqlcdb.New(pool)
	if err := q.ActivateSchemeBettingChain(ctx, "__missing_scheme__", "probe-chain", true); err == nil {
		t.Fatal("missing scheme owner transition unexpectedly succeeded")
	}
	if _, err := q.CancelSchemeBetOutbox(ctx, -1, time.Now().UTC()); err == nil {
		t.Fatal("missing outbox cancellation unexpectedly succeeded")
	}
	if _, err := q.ListStrategyReadyCandidates(ctx, "__schema_probe__", "period", 0, 10); err != nil {
		t.Fatalf("strategy ready expansion query: %v", err)
	}
	if _, found, err := q.PendingFormalStrategyRowForSchemeDraw(ctx, -1, "__missing_scheme__", "__schema_probe__", "period", 0); err != nil || found {
		t.Fatalf("strategy ready exact query found=%v err=%v", found, err)
	}

	var triggerExists bool
	if err := pool.QueryRow(ctx, `
SELECT EXISTS (
    SELECT 1 FROM pg_trigger
    WHERE tgname = 'trg_scheme_betting_admin_actions_append_only' AND NOT tgisinternal
)`).Scan(&triggerExists); err != nil || !triggerExists {
		t.Fatalf("append-only trigger exists=%v err=%v", triggerExists, err)
	}
	var constraintDefinition string
	if err := pool.QueryRow(ctx, `
SELECT pg_get_constraintdef(oid)
FROM pg_constraint
WHERE conname = 'scheme_betting_admin_actions_action_check'`).Scan(&constraintDefinition); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(constraintDefinition, "enable_event") {
		t.Fatalf("action constraint = %s", constraintDefinition)
	}
}
