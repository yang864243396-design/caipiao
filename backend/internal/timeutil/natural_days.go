package timeutil

import "time"

// PlatformLocation is UTC+8 (Asia/Shanghai) for natural-day statistics.
func PlatformLocation() *time.Location {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		return time.FixedZone("CST", 8*3600)
	}
	return loc
}

// NaturalDaysRange returns [since, until) in UTC for the last `days` natural days
// ending today, using platform UTC+8 midnight boundaries.
// Example: days=3 on 2026-05-28 CST → since=2026-05-26 00:00 CST, until=2026-05-29 00:00 CST.
func NaturalDaysRange(days int) (since, until time.Time) {
	if days <= 0 {
		days = 3
	}
	loc := PlatformLocation()
	now := time.Now().In(loc)
	startOfToday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	sinceLocal := startOfToday.AddDate(0, 0, -(days - 1))
	untilLocal := startOfToday.AddDate(0, 0, 1)
	return sinceLocal.UTC(), untilLocal.UTC()
}

// NaturalDaysMeta returns API-friendly date strings (YYYY-MM-DD, CST) for the range.
func NaturalDaysMeta(days int) (dateFrom, dateTo string, since, until time.Time) {
	since, until = NaturalDaysRange(days)
	loc := PlatformLocation()
	dateFrom = since.In(loc).Format("2006-01-02")
	// until is exclusive; last included day is until - 1ns
	dateTo = until.Add(-time.Nanosecond).In(loc).Format("2006-01-02")
	return dateFrom, dateTo, since, until
}
