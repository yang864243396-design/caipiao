package catalogsync

import "strings"

// NormalizeLotteryName 将第三方与平台彩种名归一化后比对（如 一分彩 ↔ 1分彩）。
func NormalizeLotteryName(name string) string {
	name = strings.TrimSpace(name)
	repl := strings.NewReplacer(
		"一分", "1分",
		"三分", "3分",
		"五分", "5分",
		"六分", "6分",
		"秒彩", "秒彩",
	)
	return repl.Replace(name)
}
