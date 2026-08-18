package schemes

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"caipiao/backend/internal/db/sqlcdb"
)

type realtimeTestQueryCall struct {
	sql  string
	args []interface{}
}

type realtimeTestDB struct {
	queries   []pgx.Rows
	queryRows []pgx.Row
	calls     []realtimeTestQueryCall
}

func (db *realtimeTestDB) Exec(context.Context, string, ...interface{}) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, errors.New("unexpected database write")
}

func (db *realtimeTestDB) Query(_ context.Context, query string, args ...interface{}) (pgx.Rows, error) {
	db.calls = append(db.calls, realtimeTestQueryCall{sql: query, args: args})
	if len(db.queries) == 0 {
		return nil, errors.New("unexpected database query")
	}
	rows := db.queries[0]
	db.queries = db.queries[1:]
	return rows, nil
}

func (db *realtimeTestDB) QueryRow(context.Context, string, ...interface{}) pgx.Row {
	if len(db.queryRows) == 0 {
		return realtimeTestRow(func(...interface{}) error { return errors.New("unexpected database query row") })
	}
	row := db.queryRows[0]
	db.queryRows = db.queryRows[1:]
	return row
}

type realtimeTestRow func(...interface{}) error

func (row realtimeTestRow) Scan(dest ...interface{}) error { return row(dest...) }

type realtimeTestRows struct {
	scans  []realtimeTestRow
	index  int
	closed bool
}

func (rows *realtimeTestRows) Close()                                       { rows.closed = true }
func (rows *realtimeTestRows) Err() error                                   { return nil }
func (rows *realtimeTestRows) CommandTag() pgconn.CommandTag                { return pgconn.CommandTag{} }
func (rows *realtimeTestRows) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (rows *realtimeTestRows) Values() ([]interface{}, error)               { return nil, nil }
func (rows *realtimeTestRows) RawValues() [][]byte                          { return nil }
func (rows *realtimeTestRows) Conn() *pgx.Conn                              { return nil }

func (rows *realtimeTestRows) Next() bool {
	if rows.index >= len(rows.scans) {
		rows.closed = true
		return false
	}
	rows.index++
	return true
}

func (rows *realtimeTestRows) Scan(dest ...interface{}) error {
	if rows.index == 0 || rows.index > len(rows.scans) {
		return errors.New("scan called without current row")
	}
	return rows.scans[rows.index-1](dest...)
}

func realtimeInstanceScan(id, definitionID string, memberID int64, updatedAt time.Time) realtimeTestRow {
	return func(dest ...interface{}) error {
		if len(dest) != 23 {
			return fmt.Errorf("instance scan destinations=%d want 23", len(dest))
		}
		*(dest[0].(*string)) = id
		*(dest[1].(*string)) = definitionID
		*(dest[2].(*int64)) = memberID
		*(dest[3].(*string)) = "custom"
		*(dest[22].(*pgtype.Timestamptz)) = pgtype.Timestamptz{Time: updatedAt, Valid: true}
		return nil
	}
}

func realtimeMetaScan(id string, memberID int64, runType, currency string) realtimeTestRow {
	return func(dest ...interface{}) error {
		if len(dest) != 4 {
			return fmt.Errorf("meta scan destinations=%d want 4", len(dest))
		}
		*(dest[0].(*string)) = id
		*(dest[1].(*int64)) = memberID
		*(dest[2].(*string)) = runType
		*(dest[3].(*string)) = currency
		return nil
	}
}

func realtimeChangeScan(memberID int64, instanceID string, updatedAt time.Time) realtimeTestRow {
	return func(dest ...interface{}) error {
		if len(dest) != 3 {
			return fmt.Errorf("change scan destinations=%d want 3", len(dest))
		}
		*(dest[0].(*int64)) = memberID
		*(dest[1].(*string)) = instanceID
		*(dest[2].(*time.Time)) = updatedAt
		return nil
	}
}

func realtimeStatsScan(memberID int64, formalTurnover, formalTotal, formalRunning, simTurnover, simTotal, simRunning float64, running, starts int) realtimeTestRow {
	return func(dest ...interface{}) error {
		if len(dest) != 9 {
			return fmt.Errorf("stats scan destinations=%d want 9", len(dest))
		}
		*(dest[0].(*int64)) = memberID
		for index, value := range []float64{formalTurnover, formalTotal, formalRunning, simTurnover, simTotal, simRunning} {
			if err := dest[index+1].(*pgtype.Numeric).Scan(strconv.FormatFloat(value, 'f', -1, 64)); err != nil {
				return err
			}
		}
		*(dest[7].(*int32)) = int32(running)
		*(dest[8].(*int32)) = int32(starts)
		return nil
	}
}

func TestGroupRealtimeSchemeSnapshotsIsolatesMembersAndMarksMissing(t *testing.T) {
	refs := []RealtimeInstanceRef{
		{MemberID: 7, InstanceID: "inst-a"},
		{MemberID: 7, InstanceID: "inst-gone"},
		{MemberID: 8, InstanceID: "inst-b"},
	}
	rows := []sqlcdb.SchemeInstance{
		{ID: "inst-a", MemberID: 7, UpdatedAt: pgtype.Timestamptz{Time: time.Unix(20, 0), Valid: true}},
		{ID: "inst-b", MemberID: 999, UpdatedAt: pgtype.Timestamptz{Time: time.Unix(30, 0), Valid: true}},
	}

	got := groupRealtimeSchemeSnapshots(refs, rows, nil, time.Unix(40, 0))

	if len(got.ItemsByMember[7]) != 1 || got.ItemsByMember[7][0].ID != "inst-a" {
		t.Fatalf("items=%v", got.ItemsByMember)
	}
	if !reflect.DeepEqual(got.RemovedByMember[7], []string{"inst-gone"}) {
		t.Fatalf("removed=%v", got.RemovedByMember)
	}
	if !reflect.DeepEqual(got.RemovedByMember[8], []string{"inst-b"}) {
		t.Fatalf("removed=%v", got.RemovedByMember)
	}
}

func TestGroupRealtimeSchemeSnapshotsSortsItemsAndTombstonesByID(t *testing.T) {
	refs := []RealtimeInstanceRef{
		{MemberID: 7, InstanceID: "inst-z"},
		{MemberID: 7, InstanceID: "inst-gone-z"},
		{MemberID: 7, InstanceID: "inst-a"},
		{MemberID: 7, InstanceID: "inst-gone-a"},
	}
	rows := []sqlcdb.SchemeInstance{
		{ID: "inst-z", MemberID: 7},
		{ID: "inst-a", MemberID: 7},
	}

	got := groupRealtimeSchemeSnapshots(refs, rows, nil, time.Unix(40, 0))

	if items := got.ItemsByMember[7]; len(items) != 2 || items[0].ID != "inst-a" || items[1].ID != "inst-z" {
		t.Fatalf("items=%v", items)
	}
	if want := []string{"inst-gone-a", "inst-gone-z"}; !reflect.DeepEqual(got.RemovedByMember[7], want) {
		t.Fatalf("removed=%v want=%v", got.RemovedByMember[7], want)
	}
}

func TestGroupRealtimeSchemeSnapshotsRetainsUpdatedAtAndIsolatesDefinitionMeta(t *testing.T) {
	updatedAt := time.Unix(20, 0)
	refs := []RealtimeInstanceRef{
		{MemberID: 7, InstanceID: "inst-a"},
		{MemberID: 7, InstanceID: "inst-z"},
	}
	rows := []sqlcdb.SchemeInstance{
		{ID: "inst-a", DefinitionID: "def-a", MemberID: 7, Kind: "custom", UpdatedAt: pgtype.Timestamptz{Time: updatedAt, Valid: true}},
		{ID: "inst-z", DefinitionID: "def-z", MemberID: 7, Kind: "custom", UpdatedAt: pgtype.Timestamptz{Time: updatedAt, Valid: true}},
	}
	meta := []sqlcdb.CloudRealtimeDefinitionMeta{
		{ID: "def-a", MemberID: 7, RunType: "fixed_rotate", SchemeCurrency: "USDT"},
		{ID: "def-z", MemberID: 999, RunType: "random_draw", SchemeCurrency: "TRX"},
	}

	got := groupRealtimeSchemeSnapshots(refs, rows, meta, time.Unix(40, 0))

	a := got.ItemsByMember[7][0]
	if a.MemberID != 7 || a.UpdatedAt != "1970-01-01T00:00:20Z" {
		t.Fatalf("memberId=%d updatedAt=%q", a.MemberID, a.UpdatedAt)
	}
	if a.RunTypeID != "fixed_rotate" || a.SchemeCurrency != "USDT" {
		t.Fatalf("matching meta mapped as runType=%q currency=%q", a.RunTypeID, a.SchemeCurrency)
	}
	z := got.ItemsByMember[7][1]
	if z.RunTypeID != "" || z.SchemeCurrency != "USDT" {
		t.Fatalf("cross-member meta leaked as runType=%q currency=%q", z.RunTypeID, z.SchemeCurrency)
	}
}

func TestLoadRealtimeSchemeSnapshotsUsesTwoBulkQueriesAndDeduplicatesRefs(t *testing.T) {
	db := &realtimeTestDB{queries: []pgx.Rows{
		&realtimeTestRows{scans: []realtimeTestRow{realtimeInstanceScan("inst-a", "def-a", 7, time.Unix(20, 0))}},
		&realtimeTestRows{scans: []realtimeTestRow{realtimeMetaScan("def-a", 7, "fixed_rotate", "USDT")}},
	}}
	svc := &Service{q: sqlcdb.New(db)}

	got, err := svc.LoadRealtimeSchemeSnapshots(context.Background(), []RealtimeInstanceRef{
		{MemberID: 7, InstanceID: "inst-a"},
		{MemberID: 7, InstanceID: "inst-a"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(db.calls) != 2 {
		t.Fatalf("query calls=%d want 2", len(db.calls))
	}
	if ids := db.calls[0].args[0]; !reflect.DeepEqual(ids, []string{"inst-a"}) {
		t.Fatalf("instance query ids=%v", ids)
	}
	if items := got.ItemsByMember[7]; len(items) != 1 || items[0].ID != "inst-a" {
		t.Fatalf("items=%v", items)
	}
}

func TestLoadRealtimeStatsMapsAllMembersInOneBulkQuery(t *testing.T) {
	db := &realtimeTestDB{queries: []pgx.Rows{
		&realtimeTestRows{scans: []realtimeTestRow{
			realtimeStatsScan(7, 0.176, 0.176, 0.176, 1.239, 1.25, 0, 2, 3),
			realtimeStatsScan(8, 0, 0, 0, 0, 0, 0, 0, 0),
		}},
	}}
	svc := &Service{q: sqlcdb.New(db)}

	got, err := svc.LoadRealtimeStats(context.Background(), []int64{8, 7, 7})
	if err != nil {
		t.Fatal(err)
	}
	if len(db.calls) != 1 {
		t.Fatalf("query calls=%d want 1", len(db.calls))
	}
	if !strings.Contains(db.calls[0].sql, "FROM unnest($1::bigint[])") || !strings.Contains(db.calls[0].sql, "FILTER") {
		t.Fatalf("stats query is not a bulk filtered aggregate: %s", db.calls[0].sql)
	}
	if ids := db.calls[0].args[0]; !reflect.DeepEqual(ids, []int64{7, 8}) {
		t.Fatalf("stats query member ids=%v", ids)
	}
	if got[7].Formal != (CloudCenterChannelStats{TotalTurnover: 0.17, TotalSessionPnl: 0.2, RunningSessionPnl: 0.2}) {
		t.Fatalf("formal=%+v", got[7].Formal)
	}
	if got[7].Sim != (CloudCenterChannelStats{TotalTurnover: 1.23, TotalSessionPnl: 1.3}) {
		t.Fatalf("sim=%+v", got[7].Sim)
	}
	if got[7].SimQuota != (SimSchemeQuota{TodayStarts: 3, TodayStartsLimit: 5, Running: 2, RunningLimit: 5}) {
		t.Fatalf("quota=%+v", got[7].SimQuota)
	}
	if _, ok := got[8]; !ok {
		t.Fatal("member 8 missing from result")
	}
}

func TestRealtimeAndRESTStatsUseSameProjection(t *testing.T) {
	newStatsRows := func() pgx.Rows {
		return &realtimeTestRows{scans: []realtimeTestRow{
			realtimeStatsScan(7, 0.176, 0.176, 0.176, 1.239, 1.25, 0, 2, 3),
		}}
	}
	realtimeDB := &realtimeTestDB{queries: []pgx.Rows{newStatsRows()}}
	realtimeSvc := &Service{q: sqlcdb.New(realtimeDB)}
	realtimeStats, err := realtimeSvc.LoadRealtimeStats(context.Background(), []int64{7})
	if err != nil {
		t.Fatal(err)
	}

	restDB := &realtimeTestDB{
		queries: []pgx.Rows{newStatsRows()},
		queryRows: []pgx.Row{realtimeTestRow(func(dest ...interface{}) error {
			*(dest[0].(*int64)) = 7
			return nil
		})},
	}
	restSvc := &Service{q: sqlcdb.New(restDB)}
	restStats, err := restSvc.GetCloudCenterStats(context.Background(), "member-7")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(restStats, realtimeStats[7]) {
		t.Fatalf("REST=%+v realtime=%+v", restStats, realtimeStats[7])
	}
}

func TestListSchemeRealtimeChangesUsesCompositeCursorAndLimit(t *testing.T) {
	after := time.Unix(20, 0).UTC()
	db := &realtimeTestDB{queries: []pgx.Rows{
		&realtimeTestRows{scans: []realtimeTestRow{
			realtimeChangeScan(7, "inst-b", after),
		}},
	}}

	got, err := sqlcdb.New(db).ListSchemeRealtimeChanges(context.Background(), after, "inst-a", 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(db.calls) != 1 || !reflect.DeepEqual(db.calls[0].args, []interface{}{after, "inst-a", 2}) {
		t.Fatalf("query calls=%v", db.calls)
	}
	if !strings.Contains(db.calls[0].sql, "WHERE (updated_at, id) > ($1, $2)") ||
		!strings.Contains(db.calls[0].sql, "ORDER BY updated_at ASC, id ASC") {
		t.Fatalf("query does not use the composite cursor: %s", db.calls[0].sql)
	}
	if len(got) != 1 || got[0].MemberID != 7 || got[0].InstanceID != "inst-b" || !got[0].UpdatedAt.Equal(after) {
		t.Fatalf("changes=%v", got)
	}
}
