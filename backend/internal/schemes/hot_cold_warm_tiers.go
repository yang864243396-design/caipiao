package schemes

import "sort"

// HotColdWarmTiersResult 冷热属性家族分档结果（v6 仅冷/热；Warm 恒为空）。
//
// 复用权威的 evaluatePlayHit 判定，避免前端重复实现各彩种（SSC/PC28/K3/PK10）
// 的大小阈值 / 和值 / 跨度 / 龙虎比较逻辑导致口径漂移。
type HotColdWarmTiersResult struct {
	Mode     string         `json:"mode"`     // attribute | unsupported
	Universe []string       `json:"universe"` // 选项宇宙（大/小、龙/虎、和值 0..N 等）
	Hot      []string       `json:"hot"`
	Warm     []string       `json:"warm"` // 兼容字段，恒为空
	Cold     []string       `json:"cold"`
	Counts   map[string]int `json:"counts"`  // 各选项最近 N 期命中次数
	Counted  int            `json:"counted"` // 有效统计期数
}

// HotColdWarmAttributeTiers 对属性/聚合家族（大小单双/龙虎/庄闲/特殊号/和值/跨度）
// 按"最近 N 期每个选项命中频次"降序二等分为热/冷（对齐 v6，无温档）。
//
// 说明：
//   - 每期对宇宙内每个选项调用 evaluatePlayHit，命中即计数——大小单双一期可同时命中
//     一个大小档与一个单双档，龙虎一期命中龙/虎/和之一，和值/跨度一期命中唯一值。
//   - 数字池型（不定位/包胆）与按位型不在此处理（前端按数字整体/按位频次分档）。
func HotColdWarmAttributeTiers(rule playRule, draws [][]string) HotColdWarmTiersResult {
	universe := attributeUniverse(rule)
	if len(universe) == 0 {
		return HotColdWarmTiersResult{Mode: "unsupported"}
	}
	origIdx := make(map[string]int, len(universe))
	for i, opt := range universe {
		origIdx[opt] = i
	}
	counts := make(map[string]int, len(universe))
	counted := 0
	for _, balls := range draws {
		if len(balls) == 0 {
			continue
		}
		any := false
		for _, opt := range universe {
			if evaluatePlayHit(rule, balls, opt, false, "", rule.PositionIdx).Hit {
				counts[opt]++
				any = true
			}
		}
		if any {
			counted++
		}
	}
	sorted := append([]string(nil), universe...)
	sort.SliceStable(sorted, func(i, j int) bool {
		if counts[sorted[i]] != counts[sorted[j]] {
			return counts[sorted[i]] > counts[sorted[j]]
		}
		return origIdx[sorted[i]] < origIdx[sorted[j]]
	})
	n := len(sorted)
	half := min((n+1)/2, n)
	return HotColdWarmTiersResult{
		Mode:     "attribute",
		Universe: universe,
		Hot:      sorted[:half],
		Warm:     []string{},
		Cold:     sorted[half:],
		Counts:   counts,
		Counted:  counted,
	}
}
