package schemes

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/joho/godotenv"

	"caipiao/backend/internal/config"
	"caipiao/backend/internal/db"
	"caipiao/backend/internal/db/sqlcdb"
)

func TestFinalizeClaimedCloudBetRecordGuajiMetaRequiresAndUpdatesClaim(t *testing.T) {
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
	schemeID := "tm-meta-" + suffix
	period := "test-period-" + suffix
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM cloud_bet_records WHERE scheme_id = $1`, schemeID)
	})

	found, err := q.FinalizeClaimedCloudBetRecordGuajiMeta(ctx, schemeID, period,
		pgtype.Text{String: "tp-missing", Valid: true}, pgtype.Text{}, pgtype.Text{},
		numericFromFloat(0), "pending", numericFromFloat(0.252), 252, "五星 · 五星组选120", "0,1,2,3,4,5,6,7,8,9")
	if err != nil || found {
		t.Fatalf("missing claim update: found=%v err=%v", found, err)
	}

	claimed, err := q.TryClaimCloudBetPeriod(ctx, sqlcdb.ReserveCloudBetPeriodParams{
		RecordNo:   "TM" + suffix,
		MemberID:   memberID,
		SimBet:     false,
		SchemeID:   schemeID,
		SchemeName: "meta-test",
		PeriodNo:   period,
		PlayType:   "五星 · 五星组选120",
		Multiplier: "1",
		RoundLabel: "1",
		Amount:     numericFromFloat(0.01),
		Pnl:        numericFromFloat(0),
		Status:     "pending",
		BetContent: "0,1,2,3,4,5,6,7,8,9",
		BetUnits:   10,
	})
	if err != nil || !claimed {
		t.Fatalf("claim: claimed=%v err=%v", claimed, err)
	}

	found, err = q.FinalizeClaimedCloudBetRecordGuajiMeta(ctx, schemeID, period,
		pgtype.Text{String: "126312220", Valid: true}, pgtype.Text{String: "BO-test", Valid: true}, pgtype.Text{String: "1014217800349", Valid: true},
		numericFromFloat(0), "pending", numericFromFloat(0.252), 252, "五星 · 五星组选120", "0,1,2,3,4,5,6,7,8,9")
	if err != nil || !found {
		t.Fatalf("claimed update: found=%v err=%v", found, err)
	}

	var thirdPartyID, orderNo string
	var amount float64
	var units int
	if err := pool.QueryRow(ctx, `
SELECT COALESCE(third_party_bet_id, ''), COALESCE(bet_order_no, ''), amount::float8, COALESCE(bet_units, 0)
FROM cloud_bet_records WHERE scheme_id = $1 AND period_no = $2`, schemeID, period).Scan(&thirdPartyID, &orderNo, &amount, &units); err != nil {
		t.Fatal(err)
	}
	if thirdPartyID != "126312220" || orderNo != "BO-test" || amount != 0.25 || units != 252 {
		t.Fatalf("unexpected finalized cloud record: id=%q order=%q amount=%v units=%d", thirdPartyID, orderNo, amount, units)
	}
}
