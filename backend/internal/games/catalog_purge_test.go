package games

import (
	"context"
	"testing"

	"github.com/joho/godotenv"

	"caipiao/backend/internal/config"
	"caipiao/backend/internal/db"
	"caipiao/backend/internal/db/sqlcdb"
)

func TestLegacyCatalogPurgeIdempotent(t *testing.T) {
	_ = godotenv.Load("../../.env")
	cfg := config.Load()
	if cfg.DatabaseURL == "" {
		t.Skip("DATABASE_URL not configured")
	}
	pool, err := db.Connect(context.Background(), cfg.DatabaseURL, 2, 0)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer pool.Close()
	q := sqlcdb.New(pool)
	if err := RunLegacyCatalogPurge(context.Background(), pool); err != nil {
		t.Fatalf("purge: %v", err)
	}
	catalogCount, err := q.CountLotteryCatalogWithTemplate(context.Background())
	if err != nil {
		t.Fatalf("count catalog: %v", err)
	}
	if catalogCount != expectedCatalogSeedCount {
		t.Fatalf("catalog count want %d got %d", expectedCatalogSeedCount, catalogCount)
	}
	subCount, err := q.CountSubPlays(context.Background())
	if err != nil {
		t.Fatalf("count sub_plays: %v", err)
	}
	if subCount != expectedSubPlayCount {
		t.Fatalf("sub_plays count want %d got %d", expectedSubPlayCount, subCount)
	}
	_, err = q.GetLotteryCatalogPurgeState(context.Background())
	if err != nil {
		t.Fatalf("purge marker missing: %v", err)
	}
	_, err = q.GetLotteryCatalogByCode(context.Background(), LegacyLotteryCodes[0])
	if err == nil {
		t.Fatalf("legacy lottery %s should be purged", LegacyLotteryCodes[0])
	}
	t.Logf("purge ok: catalog=%d sub_plays=%d", catalogCount, subCount)
}
