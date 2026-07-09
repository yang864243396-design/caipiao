package timeutil

import (
	"testing"
	"time"
)

func TestNaturalDaysRangeThreeDays(t *testing.T) {
	loc := PlatformLocation()
	ref := time.Date(2026, 5, 28, 15, 30, 0, 0, loc)
	since, until := naturalDaysRangeAt(ref, 3)

	wantSince := time.Date(2026, 5, 26, 0, 0, 0, 0, loc).UTC()
	wantUntil := time.Date(2026, 5, 29, 0, 0, 0, 0, loc).UTC()
	if !since.Equal(wantSince) {
		t.Fatalf("since=%v want=%v", since, wantSince)
	}
	if !until.Equal(wantUntil) {
		t.Fatalf("until=%v want=%v", until, wantUntil)
	}

	from, to, _, _ := naturalDaysMetaAt(ref, 3)
	if from != "2026-05-26" || to != "2026-05-28" {
		t.Fatalf("meta from=%s to=%s", from, to)
	}
}

func naturalDaysRangeAt(now time.Time, days int) (time.Time, time.Time) {
	loc := PlatformLocation()
	now = now.In(loc)
	startOfToday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	sinceLocal := startOfToday.AddDate(0, 0, -(days - 1))
	untilLocal := startOfToday.AddDate(0, 0, 1)
	return sinceLocal.UTC(), untilLocal.UTC()
}

func naturalDaysMetaAt(now time.Time, days int) (string, string, time.Time, time.Time) {
	since, until := naturalDaysRangeAt(now, days)
	loc := PlatformLocation()
	dateFrom := since.In(loc).Format("2006-01-02")
	dateTo := until.Add(-time.Nanosecond).In(loc).Format("2006-01-02")
	return dateFrom, dateTo, since, until
}
