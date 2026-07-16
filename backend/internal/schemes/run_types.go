package schemes

import (
	"fmt"
	"strings"
)

// 运行类型定版（docs/run-types-implementation-plan.md v8 §2）。
const (
	RunTypeFixedRotate    = "fixed_rotate"     // 定码轮换
	RunTypeAdvFixedRotate = "adv_fixed_rotate" // 高级定码轮换（局数列表）
	RunTypeAdvTriggerBet  = "adv_trigger_bet"  // 高级开某投某
	RunTypeHotColdWarm    = "hot_cold_warm"    // 冷热温出号
	RunTypeRandomDraw     = "random_draw"      // 随机出号
	RunTypeBuiltinPlan    = "builtin_plan"     // 内置计画
	RunTypeFixedNumber    = "fixed_number"     // 固定号码
)

// RunTypeLabels 运行类型展示名（与 lottery_scheme_option_sets 种子同源）。
var RunTypeLabels = map[string]string{
	RunTypeFixedRotate:    "定码轮换",
	RunTypeAdvFixedRotate: "高级定码轮换",
	RunTypeAdvTriggerBet:  "高级开某投某",
	RunTypeHotColdWarm:    "冷热温出号",
	RunTypeRandomDraw:     "随机出号",
	RunTypeBuiltinPlan:    "内置计画",
	RunTypeFixedNumber:    "固定号码",
}

// legacyRunTypeMap 废弃枚举映射（Q9=B：统一映射高级定码轮换）。
var legacyRunTypeMap = map[string]string{
	"batch_fixed":   RunTypeAdvFixedRotate,
	"dynamic_chase": RunTypeAdvFixedRotate,
	"plan_follow":   RunTypeAdvFixedRotate,
}

// NormalizeRunTypeID 归一化运行类型：废弃值映射、未识别/缺省兜底高级定码轮换。
func NormalizeRunTypeID(raw string) string {
	v := strings.TrimSpace(raw)
	if mapped, ok := legacyRunTypeMap[v]; ok {
		return mapped
	}
	if _, ok := RunTypeLabels[v]; ok {
		return v
	}
	return RunTypeAdvFixedRotate
}

// IsKnownRunTypeID 是否为定版 7 种之一（不含废弃值）。
func IsKnownRunTypeID(raw string) bool {
	_, ok := RunTypeLabels[strings.TrimSpace(raw)]
	return ok
}

// 玩法关联矩阵（Q1=代码常量；docx 定版口径）：
//   - adv_trigger_bet 支持定位胆、龙虎，及 PC28 和值/大小单双/龙虎豹
//   - hot_cold_warm 不支持龙虎
//   - builtin_plan 不选玩法（创建校验放宽）
//   - 其余支持所有玩法
const (
	playTypeDingwei = "dingwei"
	playTypeLonghu  = "longhu"
)

// SupportsPositionSourceSubPlay 冷热温/随机出号仅支持按位产号玩法。
func SupportsPositionSourceSubPlay(guajiGroup, subLabel string) bool {
	sub := strings.TrimSpace(subLabel)
	group := strings.TrimSpace(guajiGroup)
	if group == "" && sub == "" {
		return false
	}
	text := group + " " + sub
	if strings.Contains(text, "龙虎") {
		return false
	}
	if strings.Contains(text, "大小单双") || strings.Contains(sub, "和值单双") || strings.Contains(sub, "和值大小") {
		return false
	}
	if strings.Contains(text, "不定位") {
		return false
	}
	if strings.Contains(sub, "单式") || strings.Contains(sub, "混合组选") {
		return false
	}
	for _, bad := range []string{"和值", "跨度", "包胆", "组三", "组六", "特殊号", "趣味", "一帆风顺", "好事成双", "三星报喜", "四季发财"} {
		if strings.Contains(sub, bad) {
			return false
		}
	}
	if strings.Contains(sub, "组选") && !strings.Contains(sub, "组合") {
		return false
	}
	if group == "任选" || strings.Contains(group, "任选") {
		return strings.Contains(sub, "直选复式") || (strings.Contains(sub, "直选") && strings.Contains(sub, "复式"))
	}
	if strings.Contains(sub, "组合") && !strings.Contains(sub, "组选") {
		return true
	}
	if strings.Contains(sub, "直选复式") || (strings.Contains(sub, "复式") && strings.Contains(sub, "直选")) {
		return true
	}
	if strings.Contains(sub, "定位") || group == "一星" {
		return true
	}
	return false
}

// ValidateRunTypePlay 校验运行类型与玩法类型的关联矩阵。
func ValidateRunTypePlay(runTypeID, playTypeID, subPlayID, guajiGroup, subLabel string) error {
	switch runTypeID {
	case RunTypeAdvTriggerBet:
		if !SupportsAdvTriggerBet(playTypeID, subPlayID, guajiGroup, subLabel) {
			return fmt.Errorf("%w: 高级开某投某仅支持定位胆、龙虎及 PC28 和值/大小单双/龙虎豹", ErrInvalidCreateRequest)
		}
	case RunTypeHotColdWarm, RunTypeRandomDraw:
		if isLonghuPlayGroup(guajiGroup, playTypeID) {
			return fmt.Errorf("%w: 冷热温/随机出号不支持龙虎玩法", ErrInvalidCreateRequest)
		}
		// 有子玩法标签时校验按位兼容；空标签留给旧调用兼容
		if strings.TrimSpace(subLabel) != "" && !SupportsPositionSourceSubPlay(guajiGroup, subLabel) {
			return fmt.Errorf("%w: 冷热温/随机出号仅支持直选复式、组合、定位胆及任选直选复式", ErrInvalidCreateRequest)
		}
	}
	return nil
}
