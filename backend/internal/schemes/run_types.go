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

// ValidateRunTypePlay 校验运行类型与玩法类型的关联矩阵。
func ValidateRunTypePlay(runTypeID, playTypeID, subPlayID, guajiGroup, subLabel string) error {
	switch runTypeID {
	case RunTypeAdvTriggerBet:
		if !SupportsAdvTriggerBet(playTypeID, subPlayID, guajiGroup, subLabel) {
			return fmt.Errorf("%w: 高级开某投某仅支持定位胆、龙虎及 PC28 和值/大小单双/龙虎豹", ErrInvalidCreateRequest)
		}
	case RunTypeHotColdWarm:
		if isLonghuPlayGroup(guajiGroup, playTypeID) {
			return fmt.Errorf("%w: 冷热温出号不支持龙虎玩法", ErrInvalidCreateRequest)
		}
	}
	return nil
}
