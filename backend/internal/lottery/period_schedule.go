package lottery

import (
	"strings"
	"sync"
	"time"
)

const periodsScheduleMaxAge = 20 * time.Second

// PeriodsFallbackStaleAge 平台 Worker 正常 3s 刷新；超过此时间视为过期，允许按需兜底拉取。
const PeriodsFallbackStaleAge = 5 * time.Second

// PeriodsSchedule 来自第三方 /api/web_bets/lott/periods 的封盘快照。
type PeriodsSchedule struct {
	CurrentPeriod     string
	StartSkipPeriod   string    // 方案开启时无条件跳过的最近一期
	StartSkipCloseAt  time.Time // 跳过期封盘时刻
	CloseAt           time.Time // 当前可投期封盘时刻（由 CloseEndTimeRaw 按 UTC 墙钟解析）
	CloseEndTimeRaw   string    // 第三方 periods 原始 end_time 字符串
	OpenStartAt       time.Time // 当前可投期 start_time（UTC 墙钟）
	PeriodDurationSec int       // 当前可投期时长（秒），来自第三方 start/end
	UpdatedAt         time.Time
}

var periodsSchedule sync.Map // lotteryCode -> PeriodsSchedule

// periodCloseAnchor 同一期号首次写入的封盘快照；缓存被清空后仍禁止 end_time 后移。
type periodCloseAnchor struct {
	CloseAt           time.Time
	CloseEndTimeRaw   string
	OpenStartAt       time.Time
	PeriodDurationSec int
}

var periodCloseAnchors sync.Map // lotteryCode+"\x00"+period -> periodCloseAnchor

// UpdatePeriodsSchedule 写入 periods API 同步结果（current 与 skip 相同时兼容旧调用）。
func UpdatePeriodsSchedule(lotteryCode, currentPeriod string, closeAt time.Time) {
	UpdatePeriodsScheduleFull(lotteryCode, currentPeriod, currentPeriod, closeAt, closeAt)
}

// UpdatePeriodsScheduleFull 写入当前可投期与方案开启跳过期。
func UpdatePeriodsScheduleFull(lotteryCode, currentPeriod, startSkipPeriod string, startSkipCloseAt, closeAt time.Time) {
	UpdatePeriodsScheduleFullWithDuration(lotteryCode, currentPeriod, startSkipPeriod, startSkipCloseAt, closeAt, 0, "", time.Time{})
}

// UpdatePeriodsScheduleFullWithDuration 写入 periods 快照（含单期时长、原始 end_time 与开盘时刻）。
func UpdatePeriodsScheduleFullWithDuration(lotteryCode, currentPeriod, startSkipPeriod string, startSkipCloseAt, closeAt time.Time, periodDurationSec int, closeEndTimeRaw string, openStartAt time.Time) {
	_ = TryUpdatePeriodsScheduleFullWithDurationAt(
		lotteryCode, currentPeriod, startSkipPeriod, startSkipCloseAt, closeAt, periodDurationSec, closeEndTimeRaw, openStartAt, time.Time{},
	)
}

// TryUpdatePeriodsScheduleFullWithDurationAt 写入可投期快照；now 非零时拒绝已封盘时刻（锚点钳制后）。
func TryUpdatePeriodsScheduleFullWithDurationAt(
	lotteryCode, currentPeriod, startSkipPeriod string,
	startSkipCloseAt, closeAt time.Time,
	periodDurationSec int,
	closeEndTimeRaw string,
	openStartAt time.Time,
	now time.Time,
) bool {
	lotteryCode = strings.TrimSpace(lotteryCode)
	currentPeriod = strings.TrimSpace(currentPeriod)
	startSkipPeriod = strings.TrimSpace(startSkipPeriod)
	if lotteryCode == "" || closeAt.IsZero() {
		return false
	}
	if startSkipPeriod == "" {
		startSkipPeriod = currentPeriod
	}
	if startSkipCloseAt.IsZero() {
		startSkipCloseAt = closeAt
	}
	if startSkipPeriod == "" {
		return false
	}
	closeAt = closeAt.UTC()
	if v, ok := periodsSchedule.Load(lotteryCode); ok {
		if prev, ok := v.(PeriodsSchedule); ok && prev.CurrentPeriod == currentPeriod && !prev.CloseAt.IsZero() {
			oldClose := prev.CloseAt.UTC()
			// 同一期：禁止封盘时刻后移（第三方 end_time 偶发跳变会导致两次请求不一致）
			if closeAt.After(oldClose) {
				closeAt = oldClose
				if prev.CloseEndTimeRaw != "" {
					closeEndTimeRaw = prev.CloseEndTimeRaw
				}
				if !prev.OpenStartAt.IsZero() {
					openStartAt = prev.OpenStartAt
				}
				if prev.PeriodDurationSec > 0 && periodDurationSec <= 0 {
					periodDurationSec = prev.PeriodDurationSec
				}
			}
		}
	}
	closeAt, closeEndTimeRaw, openStartAt, periodDurationSec = clampCloseAtWithPeriodAnchor(
		lotteryCode,
		currentPeriod,
		closeAt,
		closeEndTimeRaw,
		openStartAt,
		periodDurationSec,
	)
	if !now.IsZero() && !now.UTC().Before(closeAt) {
		return false
	}
	periodsSchedule.Store(lotteryCode, PeriodsSchedule{
		CurrentPeriod:     currentPeriod,
		StartSkipPeriod:   startSkipPeriod,
		StartSkipCloseAt:  startSkipCloseAt.UTC(),
		CloseAt:           closeAt.UTC(),
		CloseEndTimeRaw:   strings.TrimSpace(closeEndTimeRaw),
		OpenStartAt:       openStartAt.UTC(),
		PeriodDurationSec: periodDurationSec,
		UpdatedAt:         time.Now().UTC(),
	})
	return true
}

// ClearPeriodsSchedule 清除本地 periods 封盘快照（当前期已封盘且需重新拉取时）。
func ClearPeriodsSchedule(lotteryCode string) {
	lotteryCode = strings.TrimSpace(lotteryCode)
	if lotteryCode == "" {
		return
	}
	periodsSchedule.Delete(lotteryCode)
}

// PeriodsScheduleFor 读取 periods 封盘快照。
func PeriodsScheduleFor(lotteryCode string) (PeriodsSchedule, bool) {
	lotteryCode = strings.TrimSpace(lotteryCode)
	if lotteryCode == "" {
		return PeriodsSchedule{}, false
	}
	v, ok := periodsSchedule.Load(lotteryCode)
	if !ok {
		return PeriodsSchedule{}, false
	}
	ps, ok := v.(PeriodsSchedule)
	return ps, ok
}

// PeriodsScheduleFresh 判断 periods 缓存是否在 maxAge 内更新过。
func PeriodsScheduleFresh(lotteryCode string, maxAge time.Duration, now time.Time) bool {
	ps, ok := PeriodsScheduleFor(lotteryCode)
	if !ok || ps.CurrentPeriod == "" || ps.CloseAt.IsZero() {
		return false
	}
	if maxAge <= 0 {
		return false
	}
	now = now.UTC()
	age := now.Sub(ps.UpdatedAt)
	if age < 0 {
		age = 0
	}
	return age <= maxAge
}

// PeriodsScheduleStale 缓存是否已超过兜底阈值（平台 Worker 可能故障）。
func PeriodsScheduleStale(lotteryCode string, now time.Time) bool {
	return !PeriodsScheduleFresh(lotteryCode, PeriodsFallbackStaleAge, now)
}

// StartSkipPeriodFromCache 方案开启跳过期：读平台同步缓存，禁止 WS。
func StartSkipPeriodFromCache(lotteryCode string) (string, bool) {
	ps, ok := PeriodsScheduleFor(lotteryCode)
	if !ok {
		return "", false
	}
	if time.Now().UTC().Sub(ps.UpdatedAt) > periodsScheduleMaxAge {
		return "", false
	}
	p := strings.TrimSpace(ps.StartSkipPeriod)
	if p == "" {
		p = strings.TrimSpace(ps.CurrentPeriod)
	}
	if p == "" || ps.CloseAt.IsZero() {
		return "", false
	}
	return p, true
}

// StartSkipCloseAtFromCache 方案开启跳过期封盘时刻（与 StartSkipPeriod 同源快照）。
func StartSkipCloseAtFromCache(lotteryCode string) (time.Time, bool) {
	ps, ok := PeriodsScheduleFor(lotteryCode)
	if !ok {
		return time.Time{}, false
	}
	if time.Now().UTC().Sub(ps.UpdatedAt) > periodsScheduleMaxAge {
		return time.Time{}, false
	}
	if ps.StartSkipCloseAt.IsZero() {
		if ps.CloseAt.IsZero() {
			return time.Time{}, false
		}
		return ps.CloseAt, true
	}
	return ps.StartSkipCloseAt, true
}

// StartSkipSnapshotFromCache 开启跳过期号 + 封盘时刻（开启瞬间写入 DB 用）。
func StartSkipSnapshotFromCache(lotteryCode string) (string, time.Time, bool) {
	p, ok := StartSkipPeriodFromCache(lotteryCode)
	if !ok {
		return "", time.Time{}, false
	}
	ca, ok := StartSkipCloseAtFromCache(lotteryCode)
	if !ok {
		return "", time.Time{}, false
	}
	return p, ca, true
}

// PeriodsSnapshotFresh periods 展示/下注用缓存是否在有效期内（20s）。
func PeriodsSnapshotFresh(lotteryCode string, now time.Time) bool {
	return PeriodsScheduleFresh(lotteryCode, periodsScheduleMaxAge, now.UTC())
}
func PeriodsBetCloseAt(lotteryCode string, now time.Time) (time.Time, bool) {
	ps, ok := PeriodsScheduleFor(lotteryCode)
	if !ok || ps.CloseAt.IsZero() || !PeriodsScheduleFresh(lotteryCode, periodsScheduleMaxAge, now.UTC()) {
		return time.Time{}, false
	}
	return periodsCloseAtIfOpen(ps, now)
}

// PeriodsDisplayCloseAt 展示用封盘时刻：缓存中 closeAt 尚未到达即有效，不依赖 UpdatedAt 新鲜度。
// 下注/防重仍用 PeriodsBetCloseAt（须缓存新鲜）。
func PeriodsDisplayCloseAt(lotteryCode string, now time.Time) (time.Time, bool) {
	ps, ok := PeriodsScheduleFor(lotteryCode)
	if !ok || ps.CloseAt.IsZero() {
		return time.Time{}, false
	}
	return periodsCloseAtIfOpen(ps, now)
}

func periodsCloseAtIfOpen(ps PeriodsSchedule, now time.Time) (time.Time, bool) {
	closeAt := ps.CloseAt.UTC()
	if !now.UTC().Before(closeAt) {
		return time.Time{}, false
	}
	return closeAt, true
}

// PeriodsDisplayCountdownSec 展示用距封盘秒数（封顶为单期投注窗口，与第三方页面一致不超过 60s）。
func PeriodsDisplayCountdownSec(lotteryCode string, now time.Time) (int, bool) {
	ps, ok := PeriodsScheduleFor(lotteryCode)
	if !ok || ps.CloseAt.IsZero() {
		return 0, false
	}
	if _, ok := periodsCloseAtIfOpen(ps, now); !ok {
		return 0, false
	}
	return BetCountdownSecFromSchedule(ps, now)
}

// PeriodsCloseEndTimeRaw 当前可投期第三方原始 end_time（UTC 墙钟字符串）。
func PeriodsCloseEndTimeRaw(lotteryCode string, now time.Time) (string, bool) {
	ps, ok := PeriodsScheduleFor(lotteryCode)
	if !ok || ps.CloseEndTimeRaw == "" || !PeriodsScheduleFresh(lotteryCode, periodsScheduleMaxAge, now.UTC()) {
		return "", false
	}
	if _, ok := periodsCloseAtIfOpen(ps, now); !ok {
		return "", false
	}
	return ps.CloseEndTimeRaw, true
}

// PeriodsDisplayCloseEndTimeRaw 展示用原始 end_time，允许缓存略旧。
func PeriodsDisplayCloseEndTimeRaw(lotteryCode string, now time.Time) (string, bool) {
	ps, ok := PeriodsScheduleFor(lotteryCode)
	if !ok || ps.CloseEndTimeRaw == "" {
		return "", false
	}
	if _, ok := periodsCloseAtIfOpen(ps, now); !ok {
		return "", false
	}
	return ps.CloseEndTimeRaw, true
}

// FormatUTCWallClock 将时刻格式化为 UTC 墙钟字符串（与第三方 periods 字段一致）。
func FormatUTCWallClock(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format("2006-01-02 15:04:05")
}

// BetCountdownSecFromSchedule 展示用距封盘秒数：不超过单期投注窗口（start→end）。
func BetCountdownSecFromSchedule(ps PeriodsSchedule, now time.Time) (int, bool) {
	if ps.CloseAt.IsZero() {
		return 0, false
	}
	now = now.UTC()
	if !now.Before(ps.CloseAt.UTC()) {
		return 0, false
	}
	rem := int(ps.CloseAt.Sub(now).Round(time.Second).Seconds())
	window := ps.PeriodDurationSec
	if !ps.OpenStartAt.IsZero() {
		if w := int(ps.CloseAt.Sub(ps.OpenStartAt.UTC()).Round(time.Second).Seconds()); w > 0 {
			window = w
		}
	}
	if window > 0 && rem > window {
		rem = window
	}
	return rem, true
}

// PeriodsScheduleNeedsRefresh 缓存过期或当前期 end_time 已过，需要重新拉取 periods。
func PeriodsScheduleNeedsRefresh(lotteryCode string, now time.Time) bool {
	if !PeriodsScheduleFresh(lotteryCode, PeriodsFallbackStaleAge, now.UTC()) {
		return true
	}
	ps, ok := PeriodsScheduleFor(lotteryCode)
	if !ok || ps.CloseAt.IsZero() {
		return true
	}
	return !now.UTC().Before(ps.CloseAt.UTC())
}

// PeriodsCountdownSec 距 periods API 封盘时刻剩余秒数；过期或未同步返回 false。
func PeriodsCountdownSec(lotteryCode string, now time.Time) (int, bool) {
	ps, ok := PeriodsScheduleFor(lotteryCode)
	if !ok || ps.CloseAt.IsZero() {
		return 0, false
	}
	now = now.UTC()
	if now.Sub(ps.UpdatedAt) > periodsScheduleMaxAge {
		return 0, false
	}
	rem := int(ps.CloseAt.Sub(now).Round(time.Second).Seconds())
	if rem < 0 {
		return 0, false
	}
	return rem, true
}

// CurrentOpenIssue 返回当前可投注期号（periods 快照优先，其次开奖 WS 当期）。
func CurrentOpenIssue(lotteryCode string) (string, bool) {
	lotteryCode = strings.TrimSpace(lotteryCode)
	if lotteryCode == "" {
		return "", false
	}
	if ps, ok := PeriodsScheduleFor(lotteryCode); ok && ps.CurrentPeriod != "" {
		if PeriodsScheduleFresh(lotteryCode, periodsScheduleMaxAge, time.Now()) {
			return ps.CurrentPeriod, true
		}
	}
	if st, ok := PeriodStateFor(lotteryCode); ok {
		if issue := strings.TrimSpace(st.CurrentIssue); issue != "" {
			return issue, true
		}
	}
	return "", false
}

// StrictOpenIssueForGuajiBet 仅认 periods API 未封盘期号；禁止 WS 回退（防重与下单必须一致）。
func StrictOpenIssueForGuajiBet(lotteryCode string) (string, bool) {
	return openIssueFromPeriodsSchedule(lotteryCode, time.Now().UTC())
}

// openIssueFromPeriodsSchedule 返回 periods 缓存中仍在开盘窗口内的期号。
func openIssueFromPeriodsSchedule(lotteryCode string, now time.Time) (string, bool) {
	lotteryCode = strings.TrimSpace(lotteryCode)
	if lotteryCode == "" {
		return "", false
	}
	ps, ok := PeriodsScheduleFor(lotteryCode)
	if !ok || ps.CurrentPeriod == "" || ps.CloseAt.IsZero() {
		return "", false
	}
	now = now.UTC()
	if !now.Before(ps.CloseAt) {
		return "", false
	}
	return strings.TrimSpace(ps.CurrentPeriod), true
}

// GuajiBetWindowOpen 是否处于可尝试下注时间窗（仅 periods 缓存，禁止 WS 回退）。
// 缓存封盘时刻已过时仍返回 true，以便 Worker 触发兜底刷新与防重判定。
func GuajiBetWindowOpen(lotteryCode string, now time.Time) bool {
	if GuajiPeriodsNotProvided(lotteryCode) {
		return true
	}
	if _, ok := openIssueFromPeriodsSchedule(lotteryCode, now); ok {
		return true
	}
	rem, ok := PeriodsCountdownSec(lotteryCode, now)
	if ok && rem > 0 {
		return true
	}
	return PeriodsScheduleNeedsRefresh(lotteryCode, now)
}

func periodAnchorKey(lotteryCode, period string) string {
	return strings.TrimSpace(lotteryCode) + "\x00" + strings.TrimSpace(period)
}

// clampCloseAtWithPeriodAnchor 同一期号禁止封盘时刻后移；缓存清空后仍生效。
func clampCloseAtWithPeriodAnchor(
	lotteryCode, period string,
	closeAt time.Time,
	closeEndTimeRaw string,
	openStartAt time.Time,
	periodDurationSec int,
) (time.Time, string, time.Time, int) {
	period = strings.TrimSpace(period)
	if period == "" || closeAt.IsZero() {
		return closeAt, closeEndTimeRaw, openStartAt, periodDurationSec
	}
	closeAt = closeAt.UTC()
	key := periodAnchorKey(lotteryCode, period)

	if v, ok := periodCloseAnchors.Load(key); ok {
		anchor := v.(periodCloseAnchor)
		if !anchor.CloseAt.IsZero() {
			oldClose := anchor.CloseAt.UTC()
			if closeAt.After(oldClose) {
				closeAt = oldClose
				if anchor.CloseEndTimeRaw != "" {
					closeEndTimeRaw = anchor.CloseEndTimeRaw
				}
				if !anchor.OpenStartAt.IsZero() {
					openStartAt = anchor.OpenStartAt.UTC()
				}
				if anchor.PeriodDurationSec > 0 && periodDurationSec <= 0 {
					periodDurationSec = anchor.PeriodDurationSec
				}
				return closeAt, closeEndTimeRaw, openStartAt, periodDurationSec
			}
			if closeAt.Before(oldClose) {
				anchor = periodCloseAnchor{
					CloseAt:           closeAt,
					CloseEndTimeRaw:   strings.TrimSpace(closeEndTimeRaw),
					OpenStartAt:       openStartAt.UTC(),
					PeriodDurationSec: periodDurationSec,
				}
				periodCloseAnchors.Store(key, anchor)
				return closeAt, closeEndTimeRaw, openStartAt, periodDurationSec
			}
			return closeAt, closeEndTimeRaw, openStartAt, periodDurationSec
		}
	}

	periodCloseAnchors.Store(key, periodCloseAnchor{
		CloseAt:           closeAt,
		CloseEndTimeRaw:   strings.TrimSpace(closeEndTimeRaw),
		OpenStartAt:       openStartAt.UTC(),
		PeriodDurationSec: periodDurationSec,
	})
	return closeAt, closeEndTimeRaw, openStartAt, periodDurationSec
}

func periodIssueForSmooth(lotteryCode string) string {
	if ps, ok := PeriodsScheduleFor(lotteryCode); ok && ps.CurrentPeriod != "" {
		return ps.CurrentPeriod
	}
	if st, ok := PeriodStateFor(lotteryCode); ok {
		if st.NextIssue != "" {
			return st.NextIssue
		}
		return st.CurrentIssue
	}
	return ""
}
