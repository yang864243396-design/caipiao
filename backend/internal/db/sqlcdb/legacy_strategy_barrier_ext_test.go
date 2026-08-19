package sqlcdb

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type scriptedBarrierRow struct {
	values []any
	err    error
}

func (r scriptedBarrierRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	if len(dest) != len(r.values) {
		return fmt.Errorf("scan destinations = %d, want %d", len(dest), len(r.values))
	}
	for i, value := range r.values {
		switch target := dest[i].(type) {
		case *int64:
			*target = value.(int64)
		case *string:
			*target = value.(string)
		case *bool:
			*target = value.(bool)
		case *time.Time:
			*target = value.(time.Time)
		default:
			return fmt.Errorf("unsupported scan target %T", dest[i])
		}
	}
	return nil
}

type scriptedBarrierDB struct {
	rows    []pgx.Row
	queries []string
	args    [][]any
}

func (db *scriptedBarrierDB) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}

func (db *scriptedBarrierDB) Query(context.Context, string, ...any) (pgx.Rows, error) {
	return nil, fmt.Errorf("unexpected Query call")
}

func (db *scriptedBarrierDB) QueryRow(_ context.Context, query string, args ...any) pgx.Row {
	db.queries = append(db.queries, query)
	db.args = append(db.args, args)
	if len(db.rows) == 0 {
		return scriptedBarrierRow{err: fmt.Errorf("unexpected QueryRow call")}
	}
	row := db.rows[0]
	db.rows = db.rows[1:]
	return row
}

func TestGetLegacyStrategyBarrierLocatesPartitionBeforeReadingRecord(t *testing.T) {
	placedAt := time.Date(2026, 8, 19, 12, 0, 0, 0, time.UTC)
	db := &scriptedBarrierDB{rows: []pgx.Row{
		scriptedBarrierRow{values: []any{int64(44), placedAt}},
		scriptedBarrierRow{values: []any{int64(44), "tron_ffc_6s", "1001", false, true}},
	}}

	barrier, found, err := New(db).GetLegacyStrategyBarrier(context.Background(), " scheme-1 ")
	if err != nil {
		t.Fatal(err)
	}
	if !found || barrier.RecordID != 44 {
		t.Fatalf("barrier = %+v, found = %v", barrier, found)
	}
	if len(db.queries) != 2 {
		t.Fatalf("queries = %d, want identity lookup plus exact partition read", len(db.queries))
	}
	if !strings.Contains(db.queries[0], "FROM cloud_bet_record_identity") {
		t.Fatalf("first query must use identity locator: %s", db.queries[0])
	}
	if !strings.Contains(db.queries[1], "c.placed_at = $2") ||
		!strings.Contains(db.queries[1], "c.id = $3") {
		t.Fatalf("record query must contain exact partition key and id: %s", db.queries[1])
	}
	if got := db.args[1][0]; got != pgx.QueryExecModeExec {
		t.Fatalf("partition read mode = %v, want per-call custom plan", got)
	}
	if got := db.args[0][0]; got != "scheme-1" {
		t.Fatalf("identity scheme arg = %v", got)
	}
	if got := db.args[1][2]; got != placedAt {
		t.Fatalf("partition key arg = %v", got)
	}
	if got := db.args[1][3]; got != int64(44) {
		t.Fatalf("record id arg = %v", got)
	}
}
