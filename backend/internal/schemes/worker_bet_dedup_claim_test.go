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
	"caipiao/backend/internal/lottery"
)

func TestHasUnsettledGuajiBetIgnoresHistoricalUnconfirmedClaim(t *testing.T) {
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
	q := sqlcdb.New(tx)
	suffix := strconv.FormatInt(time.Now().UnixNano(), 10)
	schemeID := "test-stale-claim-" + suffix
	lotteryCode := "test-stale-claim-lottery-" + suffix
	oldPeriod := "100"
	currentPeriod := "101"

	claimed, err := q.TryClaimCloudBetPeriod(ctx, sqlcdb.ReserveCloudBetPeriodParams{
		RecordNo:   "TH" + suffix,
		MemberID:   memberID,
		SimBet:     false,
		SchemeID:   schemeID,
		SchemeName: "stale-claim-test",
		PeriodNo:   oldPeriod,
		PlayType:   "test",
		Multiplier: "1",
		RoundLabel: "1",
		Amount:     numericFromFloat(0.02),
		Pnl:        numericFromFloat(0),
		Status:     "pending",
		BetContent: "1",
		BetUnits:   1,
	})
	if err != nil || !claimed {
		t.Fatalf("claim: claimed=%v err=%v", claimed, err)
	}

	lottery.UpdatePeriodsSchedule(lotteryCode, currentPeriod, time.Now().UTC().Add(time.Minute))
	t.Cleanup(func() { lottery.ClearPeriodsSchedule(lotteryCode) })

	w := &Worker{q: q}
	inst := sqlcdb.SchemeInstance{ID: schemeID, LotteryCode: lotteryCode, SimBet: false}
	if w.hasUnsettledGuajiBet(ctx, inst) {
		t.Fatalf("historical unconfirmed claim %s must not block current open period %s", oldPeriod, currentPeriod)
	}
	dedup, err := w.evaluateGuajiBetDedup(ctx, q, inst)
	if err != nil {
		t.Fatal(err)
	}
	if dedup.Skip {
		t.Fatalf("historical unconfirmed claim %s must not dedup current open period %s: %+v", oldPeriod, currentPeriod, dedup)
	}

	if _, err := tx.Exec(ctx, `
		UPDATE cloud_bet_records
		SET third_party_bet_id = 'accepted-test-bet', third_party_period = $3
		WHERE scheme_id = $1 AND period_no = $2`, schemeID, oldPeriod, oldPeriod); err != nil {
		t.Fatal(err)
	}
	if w.hasUnsettledGuajiBet(ctx, inst) {
		t.Fatalf("accepted historical bet from third-party period %s must not block later open period %s", oldPeriod, currentPeriod)
	}
	dedup, err = w.evaluateGuajiBetDedup(ctx, q, inst)
	if err != nil {
		t.Fatal(err)
	}
	if dedup.Skip {
		t.Fatalf("accepted historical bet from period %s must not dedup current open period %s: %+v", oldPeriod, currentPeriod, dedup)
	}

	if _, err := tx.Exec(ctx, `
		UPDATE cloud_bet_records
		SET third_party_period = NULL
		WHERE scheme_id = $1 AND period_no = $2`, schemeID, oldPeriod); err != nil {
		t.Fatal(err)
	}
	if !w.hasUnsettledGuajiBet(ctx, inst) {
		t.Fatal("accepted pending bet without authoritative third-party period must remain blocking")
	}
}

// 回归：第三方请求刚发出、尚未回写 third_party_bet_id 时，下一 tick 也必须被占位拦住。
// 否则本地期号跳动时，两次请求可能被第三方接到同一期。
func TestSchemeUnsettledGuajiPeriodIncludesUnconfirmedClaim(t *testing.T) {
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
	q := sqlcdb.New(tx)
	suffix := strconv.FormatInt(time.Now().UnixNano(), 10)
	schemeID := "test-pending-claim-" + suffix
	period := "test-period-" + suffix

	claimed, err := q.TryClaimCloudBetPeriod(ctx, sqlcdb.ReserveCloudBetPeriodParams{
		RecordNo:   "TC" + suffix,
		MemberID:   memberID,
		SimBet:     false,
		SchemeID:   schemeID,
		SchemeName: "pending-claim-test",
		PeriodNo:   period,
		PlayType:   "test",
		Multiplier: "1",
		RoundLabel: "1",
		Amount:     numericFromFloat(0.02),
		Pnl:        numericFromFloat(0),
		Status:     "pending",
		BetContent: "1",
		BetUnits:   1,
	})
	if err != nil || !claimed {
		t.Fatalf("claim: claimed=%v err=%v", claimed, err)
	}

	gotPeriod, pending, err := q.SchemeUnsettledGuajiPeriod(ctx, schemeID)
	if err != nil {
		t.Fatal(err)
	}
	if !pending || gotPeriod != period {
		t.Fatalf("unconfirmed claim must block next real bet: pending=%v period=%q want=%q", pending, gotPeriod, period)
	}
}

func TestSchemeUnsettledGuajiPeriodIgnoresSimClaim(t *testing.T) {
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
	q := sqlcdb.New(tx)
	suffix := strconv.FormatInt(time.Now().UnixNano(), 10)
	schemeID := "test-sim-claim-" + suffix

	claimed, err := q.TryClaimCloudBetPeriod(ctx, sqlcdb.ReserveCloudBetPeriodParams{
		RecordNo:   "TS" + suffix,
		MemberID:   memberID,
		SimBet:     true,
		SchemeID:   schemeID,
		SchemeName: "sim-claim-test",
		PeriodNo:   "test-period-" + suffix,
		PlayType:   "test",
		Multiplier: "1",
		RoundLabel: "1",
		Amount:     numericFromFloat(0.02),
		Pnl:        numericFromFloat(0),
		Status:     "pending",
		BetContent: "1",
		BetUnits:   1,
	})
	if err != nil || !claimed {
		t.Fatalf("claim: claimed=%v err=%v", claimed, err)
	}

	if gotPeriod, pending, err := q.SchemeUnsettledGuajiPeriod(ctx, schemeID); err != nil || pending || gotPeriod != "" {
		t.Fatalf("sim claim must not block real Guaji bet: pending=%v period=%q err=%v", pending, gotPeriod, err)
	}
}
