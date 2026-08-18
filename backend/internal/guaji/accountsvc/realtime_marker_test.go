package accountsvc

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

type payoutMarkerRef struct {
	MemberID   int64
	InstanceID string
}

type payoutRecordingMarker struct {
	mu     sync.Mutex
	refs   []payoutMarkerRef
	onMark func(memberID int64, instanceID string)
}

func (m *payoutRecordingMarker) MarkScheme(memberID int64, instanceID string) {
	if m.onMark != nil {
		m.onMark(memberID, instanceID)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.refs = append(m.refs, payoutMarkerRef{MemberID: memberID, InstanceID: instanceID})
}

func payoutMarkerTestPool(t *testing.T) *db.Pool {
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

func TestPayoutSettlementMarkerDeduplicatesComposedMutations(t *testing.T) {
	marker := &payoutRecordingMarker{}
	settlementMarker := newSettlementMarker(marker)

	settlementMarker.MarkScheme(7, "inst-a")
	settlementMarker.MarkScheme(7, "inst-a")
	settlementMarker.MarkScheme(7, "inst-b")

	marker.mu.Lock()
	defer marker.mu.Unlock()
	want := []payoutMarkerRef{{MemberID: 7, InstanceID: "inst-a"}, {MemberID: 7, InstanceID: "inst-b"}}
	if !reflect.DeepEqual(marker.refs, want) {
		t.Fatalf("refs=%v want=%v", marker.refs, want)
	}
}

func TestPayoutSettlementMarksRealtime(t *testing.T) {
	pool := payoutMarkerTestPool(t)
	ctx := context.Background()
	stamp := time.Now().UnixNano()
	account := fmt.Sprintf("rtpay%d", stamp%1e12)
	var memberID int64
	if err := pool.QueryRow(ctx, `
INSERT INTO members (account, password_hash, display_name, status)
VALUES ($1, 'test', 'payout marker', 'active')
RETURNING id`, account).Scan(&memberID); err != nil {
		t.Fatalf("seed member: %v", err)
	}
	definitionID := fmt.Sprintf("rtpay-def-%d", stamp)
	instanceID := fmt.Sprintf("rtpay-inst-%d", stamp)
	orderNo := fmt.Sprintf("RTPAY-%d", stamp)
	recordNo := fmt.Sprintf("RP%d", stamp%1e18)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM cloud_bet_records WHERE bet_order_no = $1`, orderNo)
		_, _ = pool.Exec(context.Background(), `DELETE FROM bet_orders WHERE order_no = $1`, orderNo)
		_, _ = pool.Exec(context.Background(), `DELETE FROM scheme_definitions WHERE id = $1`, definitionID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM members WHERE id = $1`, memberID)
	})
	var lottery string
	if err := pool.QueryRow(ctx, `SELECT code FROM lottery_catalog ORDER BY code LIMIT 1`).Scan(&lottery); err != nil {
		t.Skipf("lottery fixture unavailable: %v", err)
	}
	if _, err := pool.Exec(ctx, `
INSERT INTO scheme_definitions (id, member_id, kind, scheme_name, lottery_code, lottery_label, share_status, config)
VALUES ($1, $2, 'custom', 'payout-marker', $3, 'test', 'private', '{}'::jsonb)`,
		definitionID, memberID, lottery); err != nil {
		t.Fatalf("seed definition: %v", err)
	}
	if _, err := pool.Exec(ctx, `
INSERT INTO scheme_instances (
    id, definition_id, member_id, kind, scheme_name, lottery_code, lottery_label, status, sim_bet
) VALUES ($1, $2, $3, 'custom', 'payout-marker', $4, 'test', 'paused', false)`,
		instanceID, definitionID, memberID, lottery); err != nil {
		t.Fatalf("seed instance: %v", err)
	}
	var orderID int64
	if err := pool.QueryRow(ctx, `
INSERT INTO bet_orders (
    order_no, member_id, lottery_code, lottery_name, lottery_category,
    issue_no, amount, status, play_method, bet_payload, currency
) VALUES ($1, $2, $3, 'test', 'other', '20260818001', 10, 'pending', 'test', '{}'::jsonb, 'CNY')
RETURNING id`, orderNo, memberID, lottery).Scan(&orderID); err != nil {
		t.Fatalf("seed bet order: %v", err)
	}
	if _, err := pool.Exec(ctx, `
INSERT INTO cloud_bet_records (
    record_no, member_id, scheme_id, scheme_name, period_no, play_type,
    multiplier, round_label, amount, pnl, status, bet_content, sim_bet,
    currency, lottery_code, lottery_label, definition_id, bet_units, bet_order_no
) VALUES ($1, $2, $3, 'payout-marker', '20260818001', 'test',
          '1', '1/1', 10, 0, 'pending', '1', false,
          'CNY', $4, 'test', $5, 1, $6)`,
		recordNo, memberID, instanceID, lottery, definitionID, orderNo); err != nil {
		t.Fatalf("seed cloud bet record: %v", err)
	}

	marker := &payoutRecordingMarker{}
	marker.onMark = func(markMemberID int64, markInstanceID string) {
		var betStatus, cloudStatus string
		if err := pool.QueryRow(context.Background(), `
SELECT b.status, c.status
FROM bet_orders b
JOIN cloud_bet_records c ON c.bet_order_no = b.order_no
WHERE b.id = $1 AND b.member_id = $2 AND c.scheme_id = $3`,
			orderID, markMemberID, markInstanceID).Scan(&betStatus, &cloudStatus); err != nil {
			t.Errorf("marker ran before settlement commit was visible: %v", err)
		}
		if betStatus != "lose" || cloudStatus != "miss" {
			t.Errorf("status at marker bet=%q cloud=%q want lose/miss", betStatus, cloudStatus)
		}
	}
	worker := &PayoutSyncWorker{svc: &Service{pool: pool}, q: sqlcdb.New(pool)}
	worker.SetRealtimeMarker(marker)
	row := sqlcdb.ListPendingGuajiBetOrdersRow{
		ID:       orderID,
		OrderNo:  orderNo,
		MemberID: memberID,
		Amount:   10,
		Currency: "CNY",
	}
	if err := worker.commitSettlement(ctx, row, "lose", -10, 0, "CNY", 0, false); err != nil {
		t.Fatalf("commitSettlement: %v", err)
	}
	if err := worker.commitSettlement(ctx, row, "lose", -10, 0, "CNY", 0, false); err != nil {
		t.Fatalf("idempotent commitSettlement: %v", err)
	}

	marker.mu.Lock()
	defer marker.mu.Unlock()
	want := []payoutMarkerRef{{MemberID: memberID, InstanceID: instanceID}}
	if !reflect.DeepEqual(marker.refs, want) {
		t.Fatalf("refs=%v want=%v", marker.refs, want)
	}
}
