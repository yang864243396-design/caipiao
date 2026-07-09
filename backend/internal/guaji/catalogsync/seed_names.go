package catalogsync

import "strings"

// seedDisplayNames 来自 docs/seeds/lottery_catalog.csv，用于 display_name 损坏时的回退匹配。
var seedDisplayNames = map[string]string{
	"tron_ffc_1m":     "波场1分彩",
	"tron_ffc_3m":     "波场3分彩",
	"tron_ffc_5m":     "波场5分彩",
	"hash_ffc_1m":     "哈希1分彩",
	"hash_ffc_3m":     "哈希3分彩",
	"hash_ffc_5m":     "哈希5分彩",
	"eth_ffc_1m":      "以太坊1分彩",
	"eth_ffc_3m":      "以太坊3分彩",
	"eth_ffc_5m":      "以太坊5分彩",
	"bnb_ffc_1m":      "币安1分彩",
	"bnb_ffc_3m":      "币安3分彩",
	"bnb_ffc_5m":      "币安5分彩",
	"eth_ffc_new":     "新以太坊分分彩",
	"tron_jisu":       "波场极速彩",
	"hash_jisu":       "哈希极速彩",
	"eth_jisu":        "以太坊极速彩",
	"tron_lhc_1m":     "波场1分六合彩",
	"tron_lhc_3m":     "波场3分六合彩",
	"tron_lhc_5m":     "波场5分六合彩",
	"tron_syxw":       "波场11选5",
	"tron_syxw_3m":    "波场3分11选5",
	"tron_syxw_5m":    "波场5分11选5",
	"eth_syxw":        "以太坊11选5",
	"eth_syxw_3m":     "以太坊3分11选5",
	"eth_syxw_5m":     "以太坊5分11选5",
	"bnb_syxw":        "币安11选5",
	"bnb_syxw_3m":     "币安3分11选5",
	"bnb_syxw_5m":     "币安5分11选5",
	"eth_pk10_jisu":   "以太极速赛车",
	"eth_pk10_5m":     "以太5分赛车",
	"bnb_pk10_jisu":   "币安极速飞艇",
	"bnb_pk10_5m":     "币安5分飞艇",
	"tron_pk10_jisu":  "波场极速赛车",
	"eth_k3":          "以太坊快三",
	"eth_k3_3m":       "以太坊3分快三",
	"eth_k3_5m":       "以太坊5分快三",
	"tron_k3_jisu":    "波场极速快三",
	"tron_k3_1m":      "波场1分快三",
	"tron_k3_3m":      "波场3分快三",
	"tron_k3_5m":      "波场5分快三",
	"bnb_k3_1m":       "币安1分快三",
	"bnb_k3_3m":       "币安3分快三",
	"bnb_k3_5m":       "币安5分快三",
	"tron_ffc_3s":     "波场3秒彩",
	"tron_ffc_6s":     "波场6秒彩",
	"tron_ffc_15s":    "波场15秒彩",
}

func localMatchKey(code, displayName string) string {
	key := NormalizeLotteryName(displayName)
	if key != "" && !looksCorrupted(displayName) {
		return key
	}
	if seed, ok := seedDisplayNames[code]; ok {
		return NormalizeLotteryName(seed)
	}
	return key
}

// chainHintFromCode 从平台 code 前缀推断链/品牌关键词，防止波场/哈希等跨链误匹配。
func chainHintFromCode(code string) string {
	code = strings.ToLower(strings.TrimSpace(code))
	switch {
	case strings.HasPrefix(code, "tron_"):
		return "波场"
	case strings.HasPrefix(code, "hash_"):
		return "哈希"
	case strings.HasPrefix(code, "eth_"):
		return "以太坊"
	case strings.HasPrefix(code, "bnb_"):
		return "币安"
	case strings.HasPrefix(code, "taiwan_"):
		return "台湾"
	default:
		return ""
	}
}

// intervalHintFromCode 从 code 后缀推断开奖间隔关键词（1分/3分/极速等）。
func intervalHintFromCode(code string) string {
	code = strings.ToLower(strings.TrimSpace(code))
	if i := strings.LastIndex(code, "_"); i >= 0 && i < len(code)-1 {
		switch code[i+1:] {
		case "1m":
			return "1分"
		case "3m":
			return "3分"
		case "5m":
			return "5分"
		case "3s":
			return "3秒"
		case "6s":
			return "6秒"
		case "15s":
			return "15秒"
		case "jisu":
			return "极速"
		}
	}
	if strings.HasSuffix(code, "_new") {
		return "分分"
	}
	return ""
}

// productHintFromCode 彩种形态关键词，防止同地区/同链误匹配。
func productHintFromCode(code string) string {
	code = strings.ToLower(strings.TrimSpace(code))
	if strings.Contains(code, "pc28") {
		return "28"
	}
	return ""
}

func remoteMatchesCodeHints(code, remoteName string) bool {
	name := NormalizeLotteryName(remoteName)
	if !chainMatchesRemoteName(code, name) {
		return false
	}
	if interval := intervalHintFromCode(code); interval != "" && !strings.Contains(name, interval) {
		return false
	}
	if hint := productHintFromCode(code); hint != "" && !strings.Contains(strings.ToLower(name), strings.ToLower(hint)) {
		return false
	}
	// 无间隔后缀的 tron_lhc 等，排除带 1分/3分/5分 的细分类
	if intervalHintFromCode(code) == "" && strings.Contains(code, "_lhc") && !strings.Contains(code, "_lhc_") {
		for _, iv := range []string{"1分", "3分", "5分"} {
			if strings.Contains(name, iv) {
				return false
			}
		}
	}
	return true
}

func chainMatchesRemoteName(code, normalizedRemoteName string) bool {
	chain := chainHintFromCode(code)
	if chain == "" {
		return true
	}
	if strings.Contains(normalizedRemoteName, chain) {
		return true
	}
	// 第三方 new_lott 常用「以太」而非「以太坊」
	if chain == "以太坊" && strings.Contains(normalizedRemoteName, "以太") {
		return true
	}
	return false
}

func looksCorrupted(name string) bool {
	return stringsContainsQuestion(name)
}

func stringsContainsQuestion(s string) bool {
	for _, r := range s {
		if r == '?' || r == '？' {
			return true
		}
	}
	return false
}
