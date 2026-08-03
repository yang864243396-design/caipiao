package schemes

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/joho/godotenv"

	"caipiao/backend/internal/config"
	"caipiao/backend/internal/db"
	"caipiao/backend/internal/db/sqlcdb"
)

// 目标期已有记录时必须保留 from 占位，否则开放期与接单期错位会连打。
func TestMoveCloudBetRecordPeriodKeepsClaimWhenTargetExists(t *testing.T) {
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

	q := sqlcdb.New(pool)
	suffix := strconv.FormatInt(time.Now().UnixNano()%1_000_000_000, 10)
	schemeID := "tm-" + suffix
	fromPeriod := "1014163099001"
	toPeriod := "1014163099002"
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM cloud_bet_records WHERE scheme_id = $1`, schemeID)
	})

	seq := 0
	claim := func(period, content string, units int) {
		t.Helper()
		seq++
		ok, err := q.TryClaimCloudBetPeriod(ctx, sqlcdb.ReserveCloudBetPeriodParams{
			RecordNo:   "TM" + suffix + strconv.Itoa(seq),
			MemberID:   memberID,
			SimBet:     true,
			SchemeID:   schemeID,
			SchemeName: "move-test",
			PeriodNo:   period,
			PlayType:   "test",
			Multiplier: "1",
			RoundLabel: "1",
			Amount:     numericFromFloat(2),
			Pnl:        numericFromFloat(0),
			Status:     "pending",
			BetContent: content,
			BetUnits:   units,
		})
		if err != nil || !ok {
			t.Fatalf("claim %s: ok=%v err=%v", period, ok, err)
		}
	}
	claim(fromPeriod, "0,1", 1)
	claim(toPeriod, "0,1,2,3,4,5,6,7,8,9", 45)

	renamed, err := q.MoveCloudBetRecordPeriod(ctx, schemeID, fromPeriod, toPeriod)
	if err != nil {
		t.Fatalf("move: %v", err)
	}
	if renamed {
		t.Fatal("expected renamed=false when target exists")
	}
	fromTaken, err := q.CloudBetPeriodHandled(ctx, schemeID, fromPeriod)
	if err != nil || !fromTaken {
		t.Fatalf("from period must remain claimed: taken=%v err=%v", fromTaken, err)
	}
}
