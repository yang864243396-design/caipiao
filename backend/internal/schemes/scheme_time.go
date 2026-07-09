package schemes

import (
	"encoding/json"
	"strings"
	"time"
)

func isUnsetSchemeConfigDateTime(raw string) bool {
	raw = strings.TrimSpace(strings.ReplaceAll(raw, "：", ":"))
	if raw == "" {
		return true
	}
	if strings.HasPrefix(raw, "0000-00-00") || strings.Contains(raw, "0000-00-00") {
		return true
	}
	return false
}

func isLegacyDefaultSchemeTimePair(startRaw, endRaw string) bool {
	return strings.TrimSpace(startRaw) == "00:00" && strings.TrimSpace(endRaw) == "23:59"
}

func parseSchemeConfigDateTime(raw string) (time.Time, bool) {
	raw = strings.TrimSpace(strings.ReplaceAll(raw, "：", ":"))
	if isUnsetSchemeConfigDateTime(raw) {
		return time.Time{}, false
	}
	if full := strings.Split(raw, " "); len(full) >= 2 {
		raw = full[0] + " " + full[1]
	}
	layouts := []string{
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
		"2006-01-02T15:04:05",
		"15:04:05",
		"15:04",
	}
	for _, layout := range layouts {
		var t time.Time
		var err error
		if strings.Contains(layout, "2006") {
			t, err = time.ParseInLocation(layout, raw, time.Local)
		} else {
			now := time.Now()
			t, err = time.ParseInLocation("2006-01-02 "+layout, now.Format("2006-01-02")+" "+raw, time.Local)
		}
		if err == nil && !t.IsZero() {
			return t, true
		}
	}
	return time.Time{}, false
}

func schemeConfigStartTime(cfgBytes []byte) (time.Time, bool) {
	if len(cfgBytes) == 0 {
		return time.Time{}, false
	}
	var cfg map[string]interface{}
	if err := json.Unmarshal(cfgBytes, &cfg); err != nil {
		return time.Time{}, false
	}
	startRaw, _ := cfg["startTime"].(string)
	endRaw, _ := cfg["endTime"].(string)
	if isLegacyDefaultSchemeTimePair(startRaw, endRaw) {
		return time.Time{}, false
	}
	return parseSchemeConfigDateTime(startRaw)
}

// schemeConfigStartTimeNotReached 当前未到方案配置的开始时间。
func schemeConfigStartTimeNotReached(cfgBytes []byte, now time.Time) bool {
	startAt, ok := schemeConfigStartTime(cfgBytes)
	if !ok {
		return false
	}
	return now.Before(startAt)
}

func schemeConfigEndTimeReached(cfgBytes []byte, now time.Time) bool {
	if len(cfgBytes) == 0 {
		return false
	}
	var cfg map[string]interface{}
	if err := json.Unmarshal(cfgBytes, &cfg); err != nil {
		return false
	}
	startRaw, _ := cfg["startTime"].(string)
	endRaw, _ := cfg["endTime"].(string)
	if isLegacyDefaultSchemeTimePair(startRaw, endRaw) {
		return false
	}
	endAt, ok := parseSchemeConfigDateTime(endRaw)
	if !ok {
		return false
	}
	return !now.Before(endAt)
}
