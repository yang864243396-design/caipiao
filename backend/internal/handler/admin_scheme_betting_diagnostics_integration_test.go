package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/joho/godotenv"

	"caipiao/backend/internal/config"
	"caipiao/backend/internal/db"
	"caipiao/backend/internal/db/sqlcdb"
)

func TestSchemeBettingDiagnosticsAgainstMigratedSchema(t *testing.T) {
	_ = godotenv.Load("../../.env")
	cfg := config.Load()
	if cfg.DatabaseURL == "" {
		t.Skip("DATABASE_URL not set")
	}
	pool, err := db.Connect(context.Background(), cfg.DatabaseURL, 2, 0)
	if err != nil {
		t.Skipf("database unavailable: %v", err)
	}
	defer pool.Close()

	h := &Handler{db: pool}
	for name, endpoint := range map[string]func(http.ResponseWriter, *http.Request){
		"summary":        h.AdminSchemeBettingSummary,
		"events":         h.AdminSchemeBettingEvents,
		"core-partition": h.AdminCorePartitionStatus,
	} {
		t.Run(name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, "/?limit=5", nil)
			endpoint(recorder, request)
			if recorder.Code != http.StatusOK {
				t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
			}
			if name == "summary" {
				var envelope struct {
					Data map[string]any `json:"data"`
				}
				if err := json.Unmarshal(recorder.Body.Bytes(), &envelope); err != nil {
					t.Fatal(err)
				}
				for _, key := range []string{
					"drawToStrategyP99Ms", "strategyToAcceptedP99Ms",
					"safeDeadlineCompletionRate", "providerPeriodConsistencyRate",
				} {
					if _, ok := envelope.Data[key]; !ok {
						t.Fatalf("summary missing %s: %s", key, recorder.Body.String())
					}
				}
			}
			if name == "core-partition" {
				var envelope struct {
					Data map[string]any `json:"data"`
				}
				if err := json.Unmarshal(recorder.Body.Bytes(), &envelope); err != nil {
					t.Fatal(err)
				}
				for _, key := range []string{
					"phase", "forwardSync", "reverseSync", "restartRequired",
					"lastValidation", "activeTablesPartitioned",
				} {
					if _, ok := envelope.Data[key]; !ok {
						t.Fatalf("partition status missing %s: %s", key, recorder.Body.String())
					}
				}
			}
		})
	}

	q := sqlcdb.New(pool)
	if _, err := q.ListUnpublishedBetReady(context.Background(), 1); err != nil {
		t.Fatalf("ready event recovery query: %v", err)
	}
	if _, err := q.ListUnpublishedBetReconcile(context.Background(), 1); err != nil {
		t.Fatalf("reconcile event recovery query: %v", err)
	}
	if err := q.MarkBetReadyPublishFailed(context.Background(), -1); err != nil {
		t.Fatalf("ready event retry backoff query: %v", err)
	}
	if err := q.MarkBetReconcilePublishFailed(context.Background(), -1, "rejected"); err != nil {
		t.Fatalf("reconcile event retry backoff query: %v", err)
	}
	var archived int
	if err := pool.QueryRow(
		context.Background(),
		`SELECT archive_terminal_scheme_bets(now() - interval '100 years', 1)`,
	).Scan(&archived); err != nil {
		t.Fatalf("partition archive function: %v", err)
	}
	if err := pool.QueryRow(
		context.Background(),
		`SELECT archive_scheme_betting_business_history(now() - interval '100 years', 1)`,
	).Scan(&archived); err != nil {
		t.Fatalf("business history archive function: %v", err)
	}
	var ledgerExists bool
	if err := pool.QueryRow(
		context.Background(),
		`SELECT EXISTS(SELECT 1 FROM wallet_ledger)`,
	).Scan(&ledgerExists); err != nil {
		t.Fatal(err)
	}
	if ledgerExists {
		tx, err := pool.Begin(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		_, mutationErr := tx.Exec(
			context.Background(),
			`UPDATE wallet_ledger SET remark = remark WHERE id = (SELECT min(id) FROM wallet_ledger)`,
		)
		_ = tx.Rollback(context.Background())
		if mutationErr == nil {
			t.Fatal("wallet ledger update unexpectedly bypassed append-only trigger")
		}
	}
}
