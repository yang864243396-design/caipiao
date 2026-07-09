package guaji

import (
	"fmt"
	"strings"
	"time"
)

// PeriodWallClockLocation 第三方 periods 墙钟时区（iyes.dev 实测）。
// hash_* / tron_ffc_15s / tron_ffc_6s 返回 UTC 墙钟；其余 tron_/eth_/bnb_/taiwan_ 为北京时间。
func PeriodWallClockLocation(lotteryCode string) *time.Location {
	code := strings.ToLower(strings.TrimSpace(lotteryCode))
	switch {
	case strings.HasPrefix(code, "hash_"),
		code == "tron_ffc_15s",
		code == "tron_ffc_6s":
		return time.UTC
	case strings.HasPrefix(code, "tron_"),
		strings.HasPrefix(code, "eth_"),
		strings.HasPrefix(code, "bnb_"),
		strings.HasPrefix(code, "taiwan_"):
		if loc, err := time.LoadLocation("Asia/Shanghai"); err == nil {
			return loc
		}
	}
	return time.UTC
}

// ParseGuajiPeriodTimeForLottery 按彩种墙钟时区解析 periods 时间，统一返回 UTC。
func ParseGuajiPeriodTimeForLottery(lotteryCode, raw string) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, fmt.Errorf("empty period time")
	}
	if t, err := time.Parse(time.RFC3339Nano, raw); err == nil {
		return t.UTC(), nil
	}
	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		return t.UTC(), nil
	}
	loc := PeriodWallClockLocation(lotteryCode)
	t, err := time.ParseInLocation(wallClockLayout, raw, loc)
	if err != nil {
		return time.Time{}, fmt.Errorf("guaji period time parse: %q: %w", raw, err)
	}
	return t.UTC(), nil
}
