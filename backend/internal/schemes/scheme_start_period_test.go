package schemes



import (

	"testing"

	"time"



	"github.com/jackc/pgx/v5/pgtype"



	"caipiao/backend/internal/db/sqlcdb"

	"caipiao/backend/internal/lottery"
	"caipiao/backend/internal/timeutil"

)



func TestSchemeStartPeriodEnded_fromSnapshot(t *testing.T) {

	closeAt := time.Now().Add(20 * time.Second)

	inst := sqlcdb.SchemeInstance{

		StartSkipPeriod:  pgtype.Text{String: "P100", Valid: true},

		StartSkipCloseAt: pgtype.Timestamptz{Time: closeAt, Valid: true},

	}

	if schemeStartPeriodEnded(inst, nil, time.Now()) {

		t.Fatal("expected waiting before skip close")

	}

	if !schemeStartPeriodEnded(inst, nil, closeAt.Add(time.Second)) {

		t.Fatal("expected ended after skip close")

	}

}



func TestSchemeStartPeriodEnded_activatesWhenOpenPeriodAdvanced(t *testing.T) {

	code := "scheme_start_wait_close_test"

	old := "115010"

	newP := "115011"

	closeAt := time.Now().Add(25 * time.Second)

	lottery.UpdatePeriodsScheduleFull(code, newP, old, closeAt, time.Now().Add(40*time.Second))



	inst := sqlcdb.SchemeInstance{

		LotteryCode: code,

		StartSkipPeriod: pgtype.Text{String: old, Valid: true},

		StartSkipCloseAt: pgtype.Timestamptz{

			Time:  closeAt,

			Valid: true,

		},

	}

	if !schemeStartPeriodEnded(inst, nil, time.Now()) {

		t.Fatal("current open period after skipped period must activate")

	}

	if !schemeStartPeriodEnded(inst, nil, closeAt.Add(time.Second)) {

		t.Fatal("expected ended after skip close_at")

	}

}



func TestSchemeStartPeriodEnded_legacyUsesCacheClose(t *testing.T) {

	code := "scheme_start_legacy_test"

	period := "115003"

	closeAt := time.Now().Add(30 * time.Second)

	lottery.UpdatePeriodsScheduleFull(code, period, period, closeAt, closeAt)



	inst := sqlcdb.SchemeInstance{

		LotteryCode:      code,

		LastSettledIssue: pgtype.Text{String: period, Valid: true},

	}

	if schemeStartPeriodEnded(inst, nil, time.Now()) {

		t.Fatal("expected still waiting before close")

	}

}



func TestSchemeStartPeriodEnded_noSkipInScheduleWindow(t *testing.T) {

	now := time.Date(2026, 7, 1, 12, 0, 0, 0, timeutil.PlatformLocation())

	cfg := []byte(`{"startTime":"2026-07-01 11:00:00","endTime":"2026-07-01 20:00:00"}`)

	inst := sqlcdb.SchemeInstance{StatusReason: StatusReasonAwaitNextBet}

	if !schemeStartPeriodEnded(inst, cfg, now) {

		t.Fatal("no skip snapshot in schedule window should allow activation")

	}

	cfgBefore := []byte(`{"startTime":"2026-07-01 13:00:00","endTime":"2026-07-01 20:00:00"}`)

	if schemeStartPeriodEnded(inst, cfgBefore, now) {

		t.Fatal("before schedule start must not activate without skip ended")

	}

}



func TestAwaitNextBetCountdownSec_usesSnapshot(t *testing.T) {

	closeAt := time.Now().Add(25 * time.Second)

	inst := sqlcdb.SchemeInstance{

		StatusReason:     StatusReasonAwaitNextBet,

		StartSkipCloseAt: pgtype.Timestamptz{Time: closeAt, Valid: true},

	}

	sec, ok := awaitNextBetCountdownSec(inst, time.Now())

	if !ok || sec < 23 || sec > 25 {

		t.Fatalf("sec=%d ok=%v", sec, ok)

	}

}


func TestSchemeStartPeriodEnded_openPastSkippedWithoutCloseAt(t *testing.T) {
	code := "scheme_start_open_past_skip"
	closeAt := time.Now().Add(time.Minute)
	// 缓存已是新期，无法解析旧跳过期封盘时刻
	lottery.UpdatePeriodsScheduleFull(code, "200", "200", closeAt, closeAt)
	inst := sqlcdb.SchemeInstance{
		LotteryCode:     code,
		StartSkipPeriod: pgtype.Text{String: "100", Valid: true},
	}
	if !schemeStartPeriodEnded(inst, nil, time.Now()) {
		t.Fatal("current open after skipped period without closeAt should activate")
	}
}

func TestSchemeStartPeriodEnded_openPastSkippedWithStaleFutureSnapshot(t *testing.T) {
	code := "scheme_start_stale_snapshot_test"
	skipped := "115010"
	current := "115011"
	now := time.Now().UTC()
	lottery.UpdatePeriodsScheduleFull(code, current, skipped, now.Add(20*time.Second), now.Add(40*time.Second))

	inst := sqlcdb.SchemeInstance{
		LotteryCode:      code,
		StartSkipPeriod:  pgtype.Text{String: skipped, Valid: true},
		StartSkipCloseAt: pgtype.Timestamptz{Time: now.Add(8 * time.Hour), Valid: true},
	}
	if !schemeStartPeriodEnded(inst, nil, now) {
		t.Fatal("current open period after skipped period must override a stale future close_at snapshot")
	}
}
