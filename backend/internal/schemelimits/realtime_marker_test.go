package schemelimits

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/joho/godotenv"

	"caipiao/backend/internal/config"
	"caipiao/backend/internal/db"
	"caipiao/backend/internal/db/sqlcdb"
)

type schemeLimitMarkerRef struct {
	MemberID   int64
	InstanceID string
}

type schemeLimitRecordingMarker struct {
	mu     sync.Mutex
	refs   []schemeLimitMarkerRef
	onMark func(memberID int64, instanceID string)
}

func (m *schemeLimitRecordingMarker) MarkScheme(memberID int64, instanceID string) {
	if m.onMark != nil {
		m.onMark(memberID, instanceID)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.refs = append(m.refs, schemeLimitMarkerRef{MemberID: memberID, InstanceID: instanceID})
}

func schemeLimitTestPool(t *testing.T) *db.Pool {
	t.Helper()
	_ = godotenv.Load("../../.env")
	cfg := config.Load()
	if cfg.DatabaseURL == "" {
		t.Skip("DATABASE_URL not set")
	}
	pool, err := db.Connect(context.Background(), cfg.DatabaseURL, cfg.DBMaxConns, cfg.DBMinConns)
	if err != nil {
		t.Skip(err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func TestSchemeLimitPauseMarksChangedInstance(t *testing.T) {
	pool := schemeLimitTestPool(t)
	ctx := context.Background()
	stamp := time.Now().UnixNano()
	account := fmt.Sprintf("rtsl%d", stamp%1e12)
	var memberID int64
	if err := pool.QueryRow(ctx, `
INSERT INTO members (account, password_hash, display_name, status)
VALUES ($1, 'test', 'scheme limit marker', 'active')
RETURNING id`, account).Scan(&memberID); err != nil {
		t.Fatalf("seed member: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM scheme_definitions WHERE member_id = $1`, memberID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM members WHERE id = $1`, memberID)
	})
	var lottery string
	if err := pool.QueryRow(ctx, `SELECT code FROM lottery_catalog ORDER BY code LIMIT 1`).Scan(&lottery); err != nil {
		t.Skipf("lottery fixture unavailable: %v", err)
	}
	definitionID := fmt.Sprintf("rtsl-def-%d", stamp)
	instanceID := fmt.Sprintf("rtsl-inst-%d", stamp)
	configJSON := []byte(`{"stopLoss":"10"}`)
	if _, err := pool.Exec(ctx, `
INSERT INTO scheme_definitions (id, member_id, kind, scheme_name, lottery_code, lottery_label, share_status, config)
VALUES ($1, $2, 'custom', 'scheme-limit', $3, 'test', 'private', $4::jsonb)`,
		definitionID, memberID, lottery, configJSON); err != nil {
		t.Fatalf("seed definition: %v", err)
	}
	if _, err := pool.Exec(ctx, `
INSERT INTO scheme_instances (
    id, definition_id, member_id, kind, scheme_name, lottery_code, lottery_label,
    status, session_pnl, sim_bet, running_since
) VALUES ($1, $2, $3, 'custom', 'scheme-limit', $4, 'test', 'running', -11, false, now())`,
		instanceID, definitionID, memberID, lottery); err != nil {
		t.Fatalf("seed instance: %v", err)
	}
	q := sqlcdb.New(pool)
	inst, err := q.GetSchemeInstanceFull(ctx, instanceID)
	if err != nil {
		t.Fatalf("load instance: %v", err)
	}

	marker := &schemeLimitRecordingMarker{}
	marker.onMark = func(markMemberID int64, markInstanceID string) {
		var status string
		if err := pool.QueryRow(context.Background(),
			`SELECT status FROM scheme_instances WHERE member_id = $1 AND id = $2`, markMemberID, markInstanceID).Scan(&status); err != nil {
			t.Errorf("marker ran before pause was visible: %v", err)
		}
		if status != "pending" {
			t.Errorf("status at marker=%q want pending", status)
		}
	}
	if !PauseRunningInstanceIfHit(ctx, q, nil, inst, configJSON, marker) {
		t.Fatal("PauseRunningInstanceIfHit returned false")
	}
	if PauseRunningInstanceIfHit(ctx, q, nil, inst, configJSON, marker) {
		t.Fatal("second PauseRunningInstanceIfHit unexpectedly changed rows")
	}

	marker.mu.Lock()
	defer marker.mu.Unlock()
	want := []schemeLimitMarkerRef{{MemberID: memberID, InstanceID: instanceID}}
	if !reflect.DeepEqual(marker.refs, want) {
		t.Fatalf("refs=%v want=%v", marker.refs, want)
	}
}
