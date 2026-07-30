package schemes

import "strings"

// 投注详情「我的投注」按位名分行。位名表与 client/src/utils/playInputProfile.ts 对齐，
// 位窗口沿用结算侧的 playRule，避免展示与实际投注位错位。

var (
	sscDisplayPositions  = []string{"万位", "千位", "百位", "十位", "个位"}
	pk10DisplayPositions = []string{"冠军", "亚军", "季军", "第四", "第五", "第六", "第七", "第八", "第九", "第十"}
	syxwDisplayPositions = []string{"一位", "二位", "三位", "四位", "五位"}
)

// FormatBetContentLines 把 cloud_bet_records.bet_content 拆成详情页展示行。
// kind / config 取自该注单的方案定义；玩法无「位」概念或解析不出位窗口时，
// 按原样分行且不加位名——宁可不标，也不标错。
func FormatBetContentLines(kind string, config []byte, betContent string) []string {
	raw := strings.ReplaceAll(betContent, "\r\n", "\n")
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	lines := strings.Split(raw, "\n")
	if len(lines) == 1 {
		picks := normalizeBetContentPicks(lines[0])
		if picks == "" {
			return nil
		}
		return []string{picks}
	}

	labels := betContentPositionLabels(kind, config, len(lines))
	out := make([]string, 0, len(lines))
	for i, line := range lines {
		picks := normalizeBetContentPicks(line)
		if picks == "" {
			// 定位胆等用空行编码未选的位，跳过而不是占一行
			continue
		}
		if labels != nil {
			out = append(out, labels[i]+" "+picks)
			continue
		}
		out = append(out, picks)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// betContentPositionLabels 返回与行数严格等长的位名；对不上就整体放弃标注。
func betContentPositionLabels(kind string, config []byte, lineCount int) []string {
	if lineCount <= 1 || len(config) == 0 {
		return nil
	}
	labels := playRuleDisplayPositions(parseSchemeConfig(kind, config, 0, 0).Play)
	if len(labels) != lineCount {
		return nil
	}
	return labels
}

func playRuleDisplayPositions(rule playRule) []string {
	base := displayPositionBase(rule.PlayTemplate)
	if len(base) == 0 || !isPositionalDisplayPlay(rule) {
		return nil
	}
	if idx := displayPositionIndexes(rule); len(idx) > 0 {
		out := make([]string, 0, len(idx))
		for _, i := range idx {
			if i < 0 || i >= len(base) {
				return nil
			}
			out = append(out, base[i])
		}
		return out
	}
	start, n := rule.SegmentStart, rule.SegmentLen
	if n <= 1 || start < 0 || start+n > len(base) {
		return nil
	}
	return append([]string(nil), base[start:start+n]...)
}

func displayPositionBase(playTemplate string) []string {
	switch strings.TrimSpace(playTemplate) {
	case "pk10_std":
		return pk10DisplayPositions
	case "syxw_std":
		return syxwDisplayPositions
	case "ssc_std", "fast_ssc_std", "":
		return sscDisplayPositions
	default:
		// k3 / lhc / pc28 等无「位」概念
		return nil
	}
}

// displayPositionIndexes 覆盖非连续位段，以及展示位与结算段起点不一致的玩法。
// 取值与 client sscSegmentLabelsForMeta 保持一致：会员在方案编辑页正是按这些列名选的号。
func displayPositionIndexes(rule playRule) []int {
	if len(rule.SegmentPos) > 0 {
		return rule.SegmentPos
	}
	switch strings.TrimSpace(rule.PlayTemplate) {
	case "ssc_std", "fast_ssc_std", "":
	default:
		return nil
	}
	switch strings.TrimSpace(rule.PlayTypeID) {
	case "qianzhonghou3", "g007": // 前中后三
		return []int{0, 1, 2}
	case "qianhou3", "g012": // 前后三
		return []int{0, 2, 4}
	case "g008": // 前后二
		return []int{0, 4}
	case "g014": // 前后四
		return []int{0, 1, 3, 4}
	case "g013", "sixing", "hou4": // 四星
		return []int{1, 2, 3, 4}
	default:
		return nil
	}
}

// isPositionalDisplayPlay 排除行与「位」不对应的玩法：任选按选码数分行、
// 和值/跨度/龙虎/不定位/组选等本就无按位语义。
func isPositionalDisplayPlay(rule playRule) bool {
	switch strings.TrimSpace(rule.PlayTypeID) {
	case "renxuan", "g011", "renxuan_fs", "renxuan_ds",
		"longhu", "g010", "budingwei", "g009", "pc28_20", "pc28_28":
		return false
	}
	switch strings.TrimSpace(rule.BetMode) {
	case "hezhi", "kuadu", "longhu", "longhuhe", "budingwei", "zuhe", "hunhe",
		"weishu", "teshu", "baodan", "danshi",
		"zuxuan_fs", "zuxuan_ds", "zu3", "zu6", "zu12", "zu24", "zu30", "zu60", "zu120":
		return false
	}
	switch strings.TrimSpace(rule.SubPlayID) {
	case "zhixuan_ds", "zuxuan_fs", "zuxuan_ds", "hezhi", "kuadu", "longhu", "longhuhe",
		"budingwei", "zuhe", "hunhe", "weishu", "teshu", "baodan":
		return false
	}
	return true
}

func normalizeBetContentPicks(line string) string {
	fields := strings.FieldsFunc(line, func(r rune) bool {
		return r == ',' || r == '，' || r == ' ' || r == '\t' || r == '|'
	})
	return strings.Join(fields, " ")
}
