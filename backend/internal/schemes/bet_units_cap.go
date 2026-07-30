package schemes

import (
	"strings"
)

// 第三方单组注数上限（对齐 client/src/utils/betPayload.ts）。
const (
	hezhiKuaduMaxBetUnits = 900
	weishuMaxBetUnitsCap  = 9
	zuheMaxBetUnitsBase   = 2700
)

// maxBetUnitsForPlay 该玩法单组最大注数；0 表示本端尚未定义上限（不拦截）。
// 按 BetMode / 区位独立取值，供随机出号重抽、真下单与审计共用。
func maxBetUnitsForPlay(rule playRule) int {
	bm := strings.ToLower(strings.TrimSpace(rule.BetMode))
	z := playZoneMultiplier(rule)
	if z < 1 {
		z = 1
	}
	switch bm {
	case "fushi", "zhixuan_fs":
		base := zhixuanFushiMaxBetUnits(rule)
		if base <= 0 {
			return 0
		}
		return base * z
	case "hezhi", "kuadu":
		// 与前端 HEZHI_MAX_BET_UNITS 一致：对比区位倍乘后的组合注数
		return hezhiKuaduMaxBetUnits
	case "weishu":
		return weishuMaxBetUnitsCap
	case "zuhe":
		return zuheMaxBetUnitsBase * z
	default:
		return 0
	}
}

// playZoneMultiplier 多区位玩法注数倍乘（前中后三×3、前后三/二/四×2）。
func playZoneMultiplier(rule playRule) int {
	if n := len(rule.SegmentPos); n > 1 {
		return n
	}
	switch strings.ToLower(strings.TrimSpace(rule.PlayTypeID)) {
	case "qianzhonghou3", "g007":
		return 3
	case "qianhou3", "g012", "qianhou2", "g008", "qianhou4", "g014":
		return 2
	}
	return 1
}

// contentExceedsBetUnitsMax 内容组合注数是否超过该玩法上限。
// 上限为 0（未定义）时视为不超限。
func contentExceedsBetUnitsMax(rule playRule, content string) bool {
	max := maxBetUnitsForPlay(rule)
	if max <= 0 {
		return false
	}
	n := countPlayWireBetUnits(rule, content)
	return n > max
}

// countPlayWireBetUnits 按第三方 bets_nums 口径统计注数（用已解析的 SegmentLen，
// 不依赖 guajibet 对 typeId 文案的区位推断）。
func countPlayWireBetUnits(rule playRule, content string) int {
	content = strings.TrimSpace(normalizeZhixuanDanshiContent(rule, content))
	if content == "" {
		return 0
	}
	bm := strings.ToLower(strings.TrimSpace(rule.BetMode))
	seg := rule.SegmentLen
	if seg <= 0 {
		seg = 1
	}
	z := playZoneMultiplier(rule)
	if z < 1 {
		z = 1
	}
	var base int
	switch bm {
	case "fushi", "zhixuan_fs":
		base = countZhixuanFushiBetUnits(content, seg)
	case "kuadu":
		base = countKuaduCombinatorialUnits(content, seg)
	case "hezhi":
		base = countHezhiCombinatorialUnits(content, seg, rule.HezhiZuxuan)
	case "weishu":
		base = len(parseIntTokens(content))
	case "zuhe":
		// 直选组合：每位一注形态的展开注数与复式位积同口径（再×区位）
		base = countZhixuanFushiBetUnits(content, seg)
	case "danshi", "zhixuan_ds", "zuxuan_ds", "hunhe":
		base = len(parseSegmentTokensForRule(rule, content, seg))
		if base <= 0 {
			// 整注逗号串
			base = len(splitContentTokens(content))
		}
	default:
		return 0
	}
	if base <= 0 {
		return 0
	}
	return base * z
}

func countKuaduCombinatorialUnits(content string, segLen int) int {
	if segLen <= 0 {
		segLen = 3
	}
	picks := parseIntTokens(content)
	if len(picks) == 0 {
		return 0
	}
	total := 0
	for _, span := range picks {
		total += countOrderedSpanCombinationsLocal(span, segLen)
	}
	return total
}

func countHezhiCombinatorialUnits(content string, segLen int, zuxuan bool) int {
	if segLen <= 0 {
		segLen = 3
	}
	picks := parseIntTokens(content)
	if len(picks) == 0 {
		return 0
	}
	total := 0
	for _, sum := range picks {
		if zuxuan {
			total += countZuxuanSumCombinationsLocal(sum, segLen)
		} else {
			total += countOrderedSumCombinationsLocal(sum, segLen)
		}
	}
	return total
}

func countOrderedSpanCombinationsLocal(span, positions int) int {
	if positions <= 0 || span < 0 {
		return 0
	}
	count := 0
	var dfs func(idx, min, max int)
	dfs = func(idx, min, max int) {
		if idx == positions {
			if max-min == span {
				count++
			}
			return
		}
		for d := 0; d <= 9; d++ {
			nmin, nmax := min, max
			if idx == 0 {
				nmin, nmax = d, d
			} else {
				if d < nmin {
					nmin = d
				}
				if d > nmax {
					nmax = d
				}
			}
			dfs(idx+1, nmin, nmax)
		}
	}
	dfs(0, 0, 0)
	return count
}

func countOrderedSumCombinationsLocal(targetSum, positions int) int {
	if positions <= 0 || targetSum < 0 {
		return 0
	}
	count := 0
	var dfs func(idx, sum int)
	dfs = func(idx, sum int) {
		if idx == positions {
			if sum == targetSum {
				count++
			}
			return
		}
		for d := 0; d <= 9; d++ {
			dfs(idx+1, sum+d)
		}
	}
	dfs(0, 0)
	return count
}

// countZuxuanSumCombinationsLocal 组选和值组合数（三星：无序、排除豹子；与 guajibet 常用口径对齐的简化版）。
func countZuxuanSumCombinationsLocal(targetSum, segLen int) int {
	if segLen != 3 {
		return countOrderedSumCombinationsLocal(targetSum, segLen)
	}
	count := 0
	for a := 0; a <= 9; a++ {
		for b := a; b <= 9; b++ {
			for c := b; c <= 9; c++ {
				if a+b+c != targetSum {
					continue
				}
				if a == b && b == c {
					continue // 豹子
				}
				// 有序排列数
				if a == b || b == c || a == c {
					count += 3 // 组三
				} else {
					count += 6 // 组六
				}
			}
		}
	}
	return count
}
