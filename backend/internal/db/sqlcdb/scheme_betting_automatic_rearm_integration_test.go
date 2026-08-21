package sqlcdb_test

import (
	"context"
	"testing"

	"github.com/joho/godotenv"

	"caipiao/backend/internal/config"
	"caipiao/backend/internal/db"
	"caipiao/backend/internal/db/sqlcdb"
)

func TestAutomaticRearmQueriesAgainstCurrentSchema(t *testing.T) {
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

	q := sqlcdb.New(pool)
	if _, _, err := q.GetAutomaticRearmCandidate(context.Background(), -1); err != nil {
		t.Fatalf("exact automatic rearm query: %v", err)
	}
	if _, err := q.ListAutomaticRearmCandidates(
		context.Background(), []string{"__automatic_rearm_schema_probe__"}, []int32{0}, 1,
	); err != nil {
		t.Fatalf("bounded automatic rearm query: %v", err)
	}
}
