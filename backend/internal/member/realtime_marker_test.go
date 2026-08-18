package member

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/joho/godotenv"

	"caipiao/backend/internal/config"
	"caipiao/backend/internal/db"
)

type adminPauseMarkerRef struct {
	MemberID   int64
	InstanceID string
}

type adminPauseRecordingMarker struct {
	mu     sync.Mutex
	refs   []adminPauseMarkerRef
	onMark func(memberID int64, instanceID string)
}

func (m *adminPauseRecordingMarker) MarkScheme(memberID int64, instanceID string) {
	if m.onMark != nil {
		m.onMark(memberID, instanceID)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.refs = append(m.refs, adminPauseMarkerRef{MemberID: memberID, InstanceID: instanceID})
}

func adminPauseTestPool(t *testing.T) *db.Pool {
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

func TestAdminPauseMarksEveryChangedInstance(t *testing.T) {
	pool := adminPauseTestPool(t)
	ctx := context.Background()
	stamp := time.Now().UnixNano()
	account := fmt.Sprintf("rtadm%d", stamp%1e12)
	var memberID int64
	if err := pool.QueryRow(ctx, `
INSERT INTO members (account, password_hash, display_name, status)
VALUES ($1, 'test', 'admin pause marker', 'active')
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

	instanceIDs := []string{fmt.Sprintf("rtadm-a-%d", stamp), fmt.Sprintf("rtadm-b-%d", stamp)}
	statuses := []string{"running", "pending"}
	for i, instanceID := range instanceIDs {
		definitionID := fmt.Sprintf("rtadm-def-%d-%d", i, stamp)
		if _, err := pool.Exec(ctx, `
INSERT INTO scheme_definitions (id, member_id, kind, scheme_name, lottery_code, lottery_label, share_status, config)
VALUES ($1, $2, 'custom', $3, $4, 'test', 'private', '{}'::jsonb)`,
			definitionID, memberID, fmt.Sprintf("admin-pause-%d", i), lottery); err != nil {
			t.Fatalf("seed definition %d: %v", i, err)
		}
		if _, err := pool.Exec(ctx, `
INSERT INTO scheme_instances (
    id, definition_id, member_id, kind, scheme_name, lottery_code, lottery_label, status, sim_bet
) VALUES ($1, $2, $3, 'custom', $4, $5, 'test', $6, false)`,
			instanceID, definitionID, memberID, fmt.Sprintf("admin-pause-%d", i), lottery, statuses[i]); err != nil {
			t.Fatalf("seed instance %d: %v", i, err)
		}
	}

	marker := &adminPauseRecordingMarker{}
	marker.onMark = func(markMemberID int64, instanceID string) {
		var status string
		if err := pool.QueryRow(context.Background(),
			`SELECT status FROM scheme_instances WHERE member_id = $1 AND id = $2`, markMemberID, instanceID).Scan(&status); err != nil {
			t.Errorf("marker ran before admin transaction committed: %v", err)
		}
		if status != "paused" {
			t.Errorf("status at marker=%q want paused", status)
		}
	}
	svc := NewService(pool, nil)
	svc.SetRealtimeMarker(marker)
	if _, err := svc.AdminApplyOp(ctx, memberID, AdminMemberOpInput{Action: "toggle_freeze"}); err != nil {
		t.Fatalf("AdminApplyOp(freeze): %v", err)
	}
	if _, err := svc.AdminApplyOp(ctx, memberID, AdminMemberOpInput{Action: "toggle_freeze"}); err != nil {
		t.Fatalf("AdminApplyOp(unfreeze): %v", err)
	}

	marker.mu.Lock()
	got := append([]adminPauseMarkerRef(nil), marker.refs...)
	marker.mu.Unlock()
	sort.Slice(got, func(i, j int) bool { return got[i].InstanceID < got[j].InstanceID })
	want := []adminPauseMarkerRef{{MemberID: memberID, InstanceID: instanceIDs[0]}, {MemberID: memberID, InstanceID: instanceIDs[1]}}
	sort.Slice(want, func(i, j int) bool { return want[i].InstanceID < want[j].InstanceID })
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("refs=%v want=%v", got, want)
	}
}
