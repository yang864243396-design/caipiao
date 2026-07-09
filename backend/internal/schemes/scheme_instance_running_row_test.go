package schemes

import (
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"caipiao/backend/internal/db/sqlcdb"
)

func TestSchemeInstanceFromRunningRow_carriesStartSkipSnapshot(t *testing.T) {
	closeAt := time.Now().Add(-time.Minute)
	row := sqlcdb.ListRunningSchemeInstancesRow{
		ID:           "inst-test",
		Status:       "running",
		StatusReason: StatusReasonAwaitNextBet,
		StartSkipPeriod: pgtype.Text{
			String: "111202606240645",
			Valid:  true,
		},
		StartSkipCloseAt: pgtype.Timestamptz{
			Time:  closeAt,
			Valid: true,
		},
		LastSettledIssue: pgtype.Text{
			String: "111202606240645",
			Valid:  true,
		},
	}
	inst := sqlcdb.SchemeInstanceFromRunningRow(row)
	if !inst.StartSkipCloseAt.Valid {
		t.Fatal("expected start_skip_close_at on worker instance row")
	}
	if !schemeStartPeriodEnded(inst, nil, time.Now()) {
		t.Fatal("expected activation after skip close when snapshot loaded")
	}
}
