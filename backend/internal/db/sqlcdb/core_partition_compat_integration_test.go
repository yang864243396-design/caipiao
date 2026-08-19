package sqlcdb

import (
	"context"
	"math/big"
	"strconv"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/joho/godotenv"

	"caipiao/backend/internal/config"
	"caipiao/backend/internal/db"
)

func TestPartitionedCloudBetWritesPreserveBusinessKeySemantics(t *testing.T) {
	_ = godotenv.Load("../../../.env")
	cfg := config.Load()
	if cfg.DatabaseURL == "" {
		t.Skip("DATABASE_URL not set")
	}

	ctx := context.Background()
	pool, err := db.Connect(ctx, cfg.DatabaseURL, cfg.DBMaxConns, cfg.DBMinConns)
	if err != nil {
		t.Skip(err)
	}
	t.Cleanup(pool.Close)

	var memberID int64
	if err := pool.QueryRow(ctx, `SELECT id FROM members ORDER BY id LIMIT 1`).Scan(&memberID); err != nil {
		t.Skipf("no member: %v", err)
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = tx.Rollback(context.Background()) })
	q := New(tx)
	suffix := strconv.FormatInt(time.Now().UnixNano(), 10)
	schemeID := "partition-compat-" + suffix
	periodNo := "period-" + suffix

	claim := ReserveCloudBetPeriodParams{
		RecordNo: "PC" + suffix, MemberID: memberID, SchemeID: schemeID,
		SchemeName: "partition-compat", PeriodNo: periodNo, PlayType: "test",
		Multiplier: "1", RoundLabel: "1", Amount: testPartitionNumeric(1, -2),
		Pnl: testPartitionNumeric(0, 0), Status: "pending", BetContent: "1",
		Currency: "USDT", LotteryCode: "test", LotteryLabel: "test",
		DefinitionID: "test", BetUnits: 1,
	}
	claimed, err := q.TryClaimCloudBetPeriod(ctx, claim)
	if err != nil || !claimed {
		t.Fatalf("first claim: claimed=%v err=%v", claimed, err)
	}
	if err := q.FreezeCloudBetRecordRuleSnapshot(
		ctx,
		schemeID,
		periodNo,
		[]byte(`{"version":1}`),
		1,
		"partition-compat",
	); err != nil {
		t.Fatalf("freeze rule snapshot: %v", err)
	}
	claimed, err = q.TryClaimCloudBetPeriod(ctx, claim)
	if err != nil || claimed {
		t.Fatalf("duplicate claim: claimed=%v err=%v", claimed, err)
	}

	err = q.InsertCloudBetRecordEx(ctx, InsertCloudBetRecordExParams{
		RecordNo: "ignored-" + suffix, MemberID: memberID, SchemeID: schemeID,
		SchemeName: "partition-compat", PeriodNo: periodNo, PlayType: "updated",
		Multiplier: "2", RoundLabel: "2", Amount: testPartitionNumeric(25, -2),
		Pnl: testPartitionNumeric(0, 0), Status: "pending", BetContent: "2",
		ThirdPartyBetID: pgtype.Text{String: "tp-" + suffix, Valid: true},
		BetOrderNo:      pgtype.Text{String: "bo-" + suffix, Valid: true},
		Currency:        "USDT", LotteryCode: "test", LotteryLabel: "test",
		DefinitionID: "test", BetUnits: 2,
	})
	if err != nil {
		t.Fatal(err)
	}

	var count int
	var playType, thirdPartyBetID string
	var amount float64
	if err := tx.QueryRow(ctx, `
SELECT count(*), max(play_type), max(COALESCE(third_party_bet_id, '')), max(amount)::float8
FROM cloud_bet_records
WHERE scheme_id = $1 AND period_no = $2`, schemeID, periodNo).
		Scan(&count, &playType, &thirdPartyBetID, &amount); err != nil {
		t.Fatal(err)
	}
	if count != 1 || playType != "updated" || thirdPartyBetID != "tp-"+suffix || amount != 0.25 {
		t.Fatalf("unexpected upsert result: count=%d play=%q third_party=%q amount=%v",
			count, playType, thirdPartyBetID, amount)
	}
}

func testPartitionNumeric(coefficient int64, exponent int32) pgtype.Numeric {
	return pgtype.Numeric{Int: big.NewInt(coefficient), Exp: exponent, Valid: true}
}
