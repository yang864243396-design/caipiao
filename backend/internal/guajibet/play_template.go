package guajibet

import "strings"

// IsSSCPlayTemplate 时时彩/快速彩共用 SSC 编码与注数逻辑。
func IsSSCPlayTemplate(template string) bool {
	switch strings.TrimSpace(template) {
	case "", "ssc_std", "fast_ssc_std":
		return true
	default:
		return false
	}
}
