package schemes

import (
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"caipiao/backend/internal/db/sqlcdb"
)

func TestMaybeActivateAfterStartPeriod_noOpWhenStillWaiting(t *testing.T) {
	s := &Service{q: nil}
	row := sqlcdb.SchemeInstance{
		ID:           "inst-1",
		Status:       "running",
		StatusReason: StatusReasonAwaitNextBet,
		StartSkipCloseAt: pgtype.Timestamptz{
			Time:  time.Now().Add(20 * time.Second),
			Valid: true,
		},
	}
	out := s.maybeActivateAfterStartPeriod(nil, row, time.Now())
	if out.StatusReason != StatusReasonAwaitNextBet {
		t.Fatalf("reason=%s", out.StatusReason)
	}
}
