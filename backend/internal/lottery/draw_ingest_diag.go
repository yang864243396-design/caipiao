package lottery

import (
	"log/slog"
	"strings"
	"time"
)

// PeriodCloseAtFromAnchor 读取某期 periods 锚点封盘时刻（clampCloseAtWithPeriodAnchor 写入）。
func PeriodCloseAtFromAnchor(lotteryCode, period string) (time.Time, bool) {
	lotteryCode = strings.TrimSpace(lotteryCode)
	period = strings.TrimSpace(period)
	if lotteryCode == "" || period == "" {
		return time.Time{}, false
	}
	v, ok := periodCloseAnchors.Load(periodAnchorKey(lotteryCode, period))
	if !ok {
		return time.Time{}, false
	}
	anchor, ok := v.(periodCloseAnchor)
	if !ok || anchor.CloseAt.IsZero() {
		return time.Time{}, false
	}
	return anchor.CloseAt.UTC(), true
}

// resolvePeriodCloseAt 封盘时刻：优先锚点，其次当前 periods 缓存中同号 closeAt。
func resolvePeriodCloseAt(lotteryCode, period string) (time.Time, bool) {
	if ca, ok := PeriodCloseAtFromAnchor(lotteryCode, period); ok {
		return ca, true
	}
	period = strings.TrimSpace(period)
	if ps, ok := PeriodsScheduleFor(lotteryCode); ok {
		if strings.TrimSpace(ps.CurrentPeriod) == period && !ps.CloseAt.IsZero() {
			return ps.CloseAt.UTC(), true
		}
	}
	return time.Time{}, false
}

// LogDrawCloseToIngestLatency 诊断：从 periods 封盘到开奖入库的耗时（区分第三方推送 vs 我们链路）。
// source: draw_ws | history_rest
func LogDrawCloseToIngestLatency(lotteryCode, issue, source string, ingestedAt time.Time) {
	lotteryCode = strings.TrimSpace(lotteryCode)
	issue = strings.TrimSpace(issue)
	source = strings.TrimSpace(source)
	if lotteryCode == "" || issue == "" {
		return
	}
	if ingestedAt.IsZero() {
		ingestedAt = time.Now().UTC()
	} else {
		ingestedAt = ingestedAt.UTC()
	}
	closeAt, hasClose := resolvePeriodCloseAt(lotteryCode, issue)
	if !hasClose {
		slog.Info("draw ingest (no close anchor)",
			"lotteryCode", lotteryCode,
			"issue", issue,
			"source", source,
			"ingestedAt", ingestedAt.Format(time.RFC3339),
		)
		return
	}
	latencySec := int(ingestedAt.Sub(closeAt).Round(time.Second).Seconds())
	if latencySec < 0 {
		latencySec = 0
	}
	slog.Info("draw close-to-ingest latency",
		"lotteryCode", lotteryCode,
		"issue", issue,
		"source", source,
		"closeAt", closeAt.Format(time.RFC3339),
		"ingestedAt", ingestedAt.Format(time.RFC3339),
		"latencySec", latencySec,
	)
}
