package schemes

import (
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"caipiao/backend/internal/db/sqlcdb"
)

func TestDisplayRunTimeSec_running(t *testing.T) {
	since := time.Now().Add(-125 * time.Second)
	row := sqlcdb.SchemeInstance{
		Status:       "running",
		RunTimeSec:   300,
		RunningSince: pgtype.Timestamptz{Time: since, Valid: true},
	}
	got := displayRunTimeSec(row, time.Now())
	if got < 424 || got > 426 {
		t.Fatalf("got %d want ~425", got)
	}
}

func TestDisplayRunTimeSec_paused(t *testing.T) {
	row := sqlcdb.SchemeInstance{Status: "paused", RunTimeSec: 120}
	if got := displayRunTimeSec(row, time.Now()); got != 120 {
		t.Fatalf("got %d want 120", got)
	}
}
