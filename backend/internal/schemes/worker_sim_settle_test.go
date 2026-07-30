package schemes

import (
	"context"
	"encoding/json"
	"math"
	"strconv"
	"testing"
	"time"

	"github.com/joho/godotenv"

	"caipiao/backend/internal/config"
	"caipiao/backend/internal/db"
	"caipiao/backend/internal/db/sqlcdb"
)

func simCfg(t *testing.T, m map[string]interface{}) []byte {
	t.Helper()
	raw, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}
	return raw
}

// decideSimSettlement 是全系统唯一写下 status = hit|miss 的地方。
func TestDecideSimSettlement(t *testing.T) {
	dingwei := map[string]interface{}{
		"playTemplate": "ssc_std", "playTypeId": "dingwei", "subPlayId": "sub_ge",
		"playMethod": "定位胆", "betMode": "dingwei", "schemeGroups": []string{"7"},
	}
	dxds := map[string]interface{}{
		"playTemplate": "ssc_std", "playTypeId": "g016", "subPlayId": "266",
		"playMethod": "后二大小单双", "betMode": "dxds", "schemeGroups": []string{"双"},
	}

	cases := []struct {
		name       string
		cfg        map[string]interface{}
		betContent string
		balls      []string
		amount     float64
		wantOK     bool
		wantStatus string
		wantHit    bool
		// wantPnl < 0 表示亏满本金；wantPayout 只在中奖时校验
		wantPnlIsLoss bool
	}{
		{
			name: "中奖：定位胆个位押中",
			cfg:  dingwei, betContent: "7", balls: []string{"1", "2", "3", "4", "7"},
			amount: 10, wantOK: true, wantStatus: "hit", wantHit: true,
		},
		{
			name: "未中奖：定位胆个位落空",
			cfg:  dingwei, betContent: "7", balls: []string{"1", "2", "3", "4", "5"},
			amount: 10, wantOK: true, wantStatus: "miss", wantPnlIsLoss: true,
		},
		{
			name: "开奖号缺失：不结算",
			cfg:  dingwei, betContent: "7", balls: nil,
			amount: 10, wantOK: false,
		},
		{
			name: "投注内容为空时回退方案配置内容",
			cfg:  dingwei, betContent: "", balls: []string{"1", "2", "3", "4", "7"},
			amount: 10, wantOK: true, wantStatus: "hit", wantHit: true,
		},
		// 后二大小单双是按位判定（十位、个位各一档），不是和值。开奖 [5 2 7 3 1] 的
		// 十位=3、个位=1 都是单。
		{
			name: "按位大小单双：两位都押中",
			cfg:  dxds, betContent: "单\n单", balls: []string{"5", "2", "7", "3", "1"},
			amount: 8, wantOK: true, wantStatus: "hit", wantHit: true,
		},
		{
			name: "按位大小单双：一位押错即未中",
			cfg:  dxds, betContent: "双\n单", balls: []string{"5", "2", "7", "3", "1"},
			amount: 8, wantOK: true, wantStatus: "miss", wantPnlIsLoss: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := decideSimSettlement(
				"custom", simCfg(t, tc.cfg), "tron_ffc_1m", tc.betContent, 0, tc.balls, tc.amount,
			)
			if ok != tc.wantOK {
				t.Fatalf("ok=%v want %v", ok, tc.wantOK)
			}
			if !ok {
				if got != (simSettlement{}) {
					t.Fatalf("不结算时应返回零值，got %+v", got)
				}
				return
			}
			if got.Status != tc.wantStatus || got.Hit != tc.wantHit {
				t.Fatalf("status=%s hit=%v，want %s/%v", got.Status, got.Hit, tc.wantStatus, tc.wantHit)
			}

			// 资金守恒：未中奖亏满本金且不返奖；中奖返奖 = 本金 + 盈亏
			if tc.wantPnlIsLoss {
				if math.Abs(got.Pnl+tc.amount) > 0.005 {
					t.Fatalf("未中奖应亏满本金：pnl=%.2f amount=%.2f", got.Pnl, tc.amount)
				}
				if got.Payout != 0 {
					t.Fatalf("未中奖不应有返奖：%.2f", got.Payout)
				}
				return
			}
			if got.Pnl <= 0 {
				t.Fatalf("中奖盈亏应为正：%.2f", got.Pnl)
			}
			if math.Abs(got.Payout-(tc.amount+got.Pnl)) > 0.005 {
				t.Fatalf("返奖 %.2f ≠ 本金 %.2f + 盈亏 %.2f", got.Payout, tc.amount, got.Pnl)
			}
		})
	}
}

// 重复结算靠 SQL 的 status='pending' 前置条件挡住，第二次必须影响 0 行。
// 这一条只能真连库验——它保证的是并发下两个 tick 不会把盈亏记两遍。
func TestUpdateCloudBetRecordFromSettlementByID_idempotent(t *testing.T) {
	_ = godotenv.Load("../../.env")
	cfg := config.Load()
	if cfg.DatabaseURL == "" {
		t.Skip("DATABASE_URL not set")
	}
	ctx := context.Background()
	pool, err := db.Connect(ctx, cfg.DatabaseURL, cfg.DBMaxConns, cfg.DBMinConns)
	if err != nil {
		t.Skip(err)
	}
	t.Cleanup(func() { pool.Close() })

	var memberID int64
	if err := pool.QueryRow(ctx, `SELECT id FROM members ORDER BY id LIMIT 1`).Scan(&memberID); err != nil {
		t.Skipf("no member: %v", err)
	}

	recordNo := "AUDITTEST" + strconv.FormatInt(time.Now().UnixNano(), 10)
	var id int64
	err = pool.QueryRow(ctx, `
INSERT INTO cloud_bet_records
  (record_no, member_id, sim_bet, scheme_id, scheme_name, period_no, play_type,
   multiplier, round_label, amount, pnl, status, bet_content)
VALUES ($1, $2, true, 'audit-test', 'audit-test', '0', '定位胆', '1', '1/1', 10, 0, 'pending', '7')
RETURNING id`, recordNo, memberID).Scan(&id)
	if err != nil {
		t.Fatalf("insert: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM cloud_bet_records WHERE id = $1`, id)
	})

	q := sqlcdb.New(pool)
	pnl, payout := numericFromFloat(80), numericFromFloat(90)

	n, err := q.UpdateCloudBetRecordFromSettlementByID(ctx, id, "hit", pnl, payout)
	if err != nil || n != 1 {
		t.Fatalf("首次结算应影响 1 行：n=%d err=%v", n, err)
	}
	n, err = q.UpdateCloudBetRecordFromSettlementByID(ctx, id, "miss", numericFromFloat(-10), numericFromFloat(0))
	if err != nil {
		t.Fatalf("重复结算不应报错：%v", err)
	}
	if n != 0 {
		t.Fatalf("重复结算应影响 0 行，实际 %d —— 盈亏会被记两遍", n)
	}

	var status string
	var gotPnl float64
	if err := pool.QueryRow(ctx,
		`SELECT status, pnl::float8 FROM cloud_bet_records WHERE id = $1`, id).Scan(&status, &gotPnl); err != nil {
		t.Fatalf("readback: %v", err)
	}
	if status != "hit" || math.Abs(gotPnl-80) > 0.005 {
		t.Fatalf("重复结算不应覆盖首次结果：status=%s pnl=%.2f", status, gotPnl)
	}
}
