package timeutil

import (
	"fmt"
	"time"
)

// ParseDateRange parses YYYY-MM-DD bounds in Asia/Shanghai; empty both => today.
func ParseDateRange(from, to string) (time.Time, time.Time, error) {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		loc = time.FixedZone("CST", 8*3600)
	}
	now := time.Now().In(loc)
	if from == "" && to == "" {
		start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
		return start.UTC(), start.Add(24 * time.Hour).UTC(), nil
	}
	if from == "" || to == "" {
		return time.Time{}, time.Time{}, fmt.Errorf("dateFrom 与 dateTo 须同时提供")
	}
	start, err := time.ParseInLocation("2006-01-02", from, loc)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("dateFrom 格式须为 YYYY-MM-DD")
	}
	endDay, err := time.ParseInLocation("2006-01-02", to, loc)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("dateTo 格式须为 YYYY-MM-DD")
	}
	if endDay.Before(start) {
		return time.Time{}, time.Time{}, fmt.Errorf("dateTo 不能早于 dateFrom")
	}
	return start.UTC(), endDay.Add(24 * time.Hour).UTC(), nil
}

func FormatISO(ts time.Time) string {
	return ts.UTC().Format("2006-01-02T15:04:05Z07:00")
}

func FormatDisplayCST(ts time.Time) string {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		loc = time.FixedZone("CST", 8*3600)
	}
	return ts.In(loc).Format("2006-01-02 15:04:05")
}
