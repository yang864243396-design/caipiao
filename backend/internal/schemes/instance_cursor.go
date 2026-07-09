package schemes

import (
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"caipiao/backend/internal/timeutil"
)

func encodeInstanceCursor(updatedAt time.Time, id string) string {
	return timeutil.FormatISO(updatedAt.UTC()) + "|" + strings.TrimSpace(id)
}

func decodeInstanceCursor(cursor string) (pgtype.Timestamptz, string, error) {
	cursor = strings.TrimSpace(cursor)
	if cursor == "" {
		return pgtype.Timestamptz{}, "", nil
	}
	parts := strings.SplitN(cursor, "|", 2)
	if len(parts) != 2 || strings.TrimSpace(parts[1]) == "" {
		return pgtype.Timestamptz{}, "", errors.New("invalid instance cursor")
	}
	rawTime := strings.TrimSpace(parts[0])
	t, err := time.Parse(time.RFC3339, rawTime)
	if err != nil {
		t, err = time.Parse("2006-01-02T15:04:05Z07:00", rawTime)
	}
	if err != nil {
		return pgtype.Timestamptz{}, "", err
	}
	return pgtype.Timestamptz{Time: t.UTC(), Valid: true}, strings.TrimSpace(parts[1]), nil
}
