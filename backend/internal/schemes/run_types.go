package schemes

import (
	"strings"
)

// 运行类型定版（docs/run-types-implementation-plan.md v8 §2）。
const (
	RunTypeFixedRotate    = "fixed_rotate"     // 定码轮换
	RunTypeAdvFixedRotate = "adv_fixed_rotate" // 高级定码轮换（局数列表）
	RunTypeAdvTriggerBet  = "adv_trigger_bet"  // 高级开某投某
	RunTypeHotColdWarm    = "hot_cold_warm"    // 冷热出号（v6 仅冷/热两档；id 兼容保留）
	RunTypeRandomDraw     = "random_draw"      // 随机出号
	RunTypeBuiltinPlan    = "builtin_plan"     // 内置计划
	RunTypeFixedNumber    = "fixed_number"     // 固定号码（每期复投指定号码）
)

// RunTypeLabels 运行类型展示名（与 lottery_scheme_option_sets 种子同源）。
var RunTypeLabels = map[string]string{
	RunTypeFixedRotate:    "定码轮换",
	RunTypeAdvFixedRotate: "高级定码轮换",
	RunTypeAdvTriggerBet:  "高级开某投某",
	RunTypeHotColdWarm:    "冷热出号",
	RunTypeRandomDraw:     "随机出号",
	RunTypeBuiltinPlan:    "内置计划",
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

// SupportsPositionSourceSubPlay 冷热出号/随机出号仅支持按位产号玩法。
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

// ValidateRunTypePlay 校验运行类型与玩法类型的关联。
//
// 对齐第三方（V8）：运行类型与玩法**正交、无门禁**——任意运行类型可配任意玩法。
// 出号引擎按玩法自适应产号（随机=选项宇宙抽样、冷热=频次分档、开某投某=按位触发+玩法内容、
// 固定取码=条件规则、定码/局数=用户填号）。因此此处不再限制玩法。
// 保留 SupportsHotColdWarmSubPlay / SupportsRandomDrawSubPlay 等辅助函数供前端 UI 模式判断复用。
func ValidateRunTypePlay(runTypeID, playTypeID, subPlayID, guajiGroup, subLabel string) error {
	_ = runTypeID
	_ = playTypeID
	_ = subPlayID
	_ = guajiGroup
	_ = subLabel
	return nil
}

// SupportsHotColdWarmSubPlay 冷热出号支持的子玩法：
//   - 按位型：直选复式/直选组合/定位胆/任选直选复式（按位频次分档）
//   - 号码池型：组三/组六/组选N/组选复式/不定位/包胆（号码整体频次分档）
//   - 属性/聚合型：大小单双/龙虎/特殊号/庄闲/和值/跨度（选项命中频次分档）
//
// 不含单式（组合频次≈0，不适合分档）。
func SupportsHotColdWarmSubPlay(guajiGroup, subLabel string) bool {
	if SupportsPositionSourceSubPlay(guajiGroup, subLabel) {
		return true
	}
	sub := strings.TrimSpace(subLabel)
	if strings.Contains(sub, "单式") {
		return false
	}
	// 属性/聚合家族：选项命中频次分档（HotColdWarmAttributeTiers）
	for _, kw := range []string{
		"大小单双", "特殊号", "庄闲", "龙虎豹", "直选和值", "组选和值", "和值尾数", "跨度", "龙虎", "和值",
	} {
		if strings.Contains(sub, kw) || strings.Contains(guajiGroup, kw) {
			return true
		}
	}
	// 号码池型：组选家族 + 不定位 + 包胆
	if strings.Contains(sub, "组三") || strings.Contains(sub, "组六") || strings.Contains(sub, "组选") ||
		strings.Contains(sub, "不定位") || strings.Contains(sub, "包胆") {
		return true
	}
	return false
}

// SupportsRandomDrawSubPlay 随机出号支持的子玩法：
//   - 按位型（同冷热出号）：直选复式/直选组合/定位胆/任选直选复式
//   - 单式（直选/组选单式）：整注随机
//   - 组合家族（组三/组六/组选N/组选复式）：号码池随机（选 K 个号）
//
// 仍排除和值/跨度/包胆/不定位/龙虎/大小单双/特殊号等无法随机产号或补集无意义的玩法。
func SupportsRandomDrawSubPlay(guajiGroup, subLabel string) bool {
	if SupportsPositionSourceSubPlay(guajiGroup, subLabel) {
		return true
	}
	sub := strings.TrimSpace(subLabel)
	// 直选单式 / 组选单式 / 混合组选单式
	if strings.Contains(sub, "单式") || strings.Contains(sub, "混合") {
		return true
	}
	// 组合家族：组三 / 组六 / 组选N / 组选复式（号码池随机）
	if strings.Contains(sub, "组三") || strings.Contains(sub, "组六") || strings.Contains(sub, "组选") {
		return true
	}
	// 属性/聚合家族：大小单双 / 龙虎 / 特殊号 / 庄闲 / 和值 / 跨度 / 不定位 / 包胆
	for _, kw := range []string{
		"大小单双", "大小", "单双", "龙虎", "庄闲", "特殊号", "豹子", "对子", "顺子",
		"和值", "跨度", "不定位", "包胆",
	} {
		if strings.Contains(sub, kw) {
			return true
		}
	}
	return false
}
