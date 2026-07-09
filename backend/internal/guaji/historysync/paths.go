package historysync

import "strings"

// HistoryAPIPathForCode 返回第三方历史开奖 REST 路径（不含 /api/ 前缀）。
// 映射依据接口文档 §5（_tmp_guaji_parsed.md）与 hash.iyes.dev 实测。
func HistoryAPIPathForCode(lotteryCode string) string {
	if p, ok := historyAPIByCode[strings.TrimSpace(lotteryCode)]; ok {
		return p
	}
	return ""
}

// historyAPIByCode：平台彩种 code → GET /api/{path}
var historyAPIByCode = map[string]string{
	// §5.1 lottery_log033s：哈希极速、波场极速快三、波场极速赛车
	"hash_jisu":        "lottery_log033s",
	"tron_k3_jisu":     "lottery_log033s",
	"tron_pk10_jisu":   "lottery_log033s",
	"bnb_pk10_jisu":    "lottery_log033s",

	// §5.2 lottery_log103s：波场1分（00115 对换后 hash 系）、波场1分快三、波场11选5、波场1分六合彩
	"hash_ffc_1m":  "lottery_log103s",
	"tron_k3_1m":   "lottery_log103s",
	"tron_syxw":    "lottery_log103s",
	"tron_lhc_1m":  "lottery_log103s",

	// §5.3 lottery_log303s
	"hash_ffc_3m":  "lottery_log303s",
	"tron_k3_3m":   "lottery_log303s",
	"tron_syxw_3m": "lottery_log303s",
	"tron_lhc_3m":  "lottery_log303s",

	// §5.4 lottery_log503s
	"hash_ffc_5m":  "lottery_log503s",
	"tron_k3_5m":   "lottery_log503s",
	"tron_syxw_5m": "lottery_log503s",
	"tron_lhc_5m":  "lottery_log503s",
	"tron_lhc":     "lottery_log503s",

	// §5.5 lottery_log05s：波场极速彩
	"tron_jisu": "lottery_log05s",

	// §5.6 lottery_logs：哈希1分（00115 对换后 tron 系）、秒级波场分分彩共用区块线
	"tron_ffc_1m":   "lottery_logs",
	"tron_ffc_3s":   "lottery_logs",
	"tron_ffc_6s":   "lottery_logs",
	"tron_ffc_15s":  "lottery_logs",

	// §5.7 lottery_log3s：哈希3分（00115 对换后 tron 系）
	"tron_ffc_3m": "lottery_log3s",

	// §5.8 lottery_log5s：哈希5分（00115 对换后 tron 系）
	"tron_ffc_5m": "lottery_log5s",

	// §5.9 eth_block_logs：以太坊极速
	"eth_jisu": "eth_block_logs",

	// §5.10 eth_lottery_logs：以太1分系（分分/11选5/快三/赛车等共用）
	"eth_ffc_1m":     "eth_lottery_logs",
	"eth_ffc_new":    "eth_lottery_logs",
	"eth_syxw":       "eth_lottery_logs",
	"eth_k3":         "eth_lottery_logs",
	"eth_pk10_jisu":  "eth_lottery_logs",

	// §5.11 eth_lottery_log3s
	"eth_ffc_3m":   "eth_lottery_log3s",
	"eth_syxw_3m":  "eth_lottery_log3s",
	"eth_k3_3m":    "eth_lottery_log3s",

	// §5.12 eth_lottery_log5s
	"eth_ffc_5m":   "eth_lottery_log5s",
	"eth_syxw_5m":  "eth_lottery_log5s",
	"eth_k3_5m":    "eth_lottery_log5s",
	"eth_pk10_5m":  "eth_lottery_log5s",

	// §5.13–5.15 bsc（币安）
	"bnb_ffc_1m":    "bsc_lottery_logs",
	"bnb_syxw":      "bsc_lottery_logs",
	"bnb_k3_1m":     "bsc_lottery_logs",
	"bnb_ffc_3m":    "bsc_lottery_log3s",
	"bnb_syxw_3m":   "bsc_lottery_log3s",
	"bnb_k3_3m":     "bsc_lottery_log3s",
	"bnb_ffc_5m":    "bsc_lottery_log5s",
	"bnb_syxw_5m":   "bsc_lottery_log5s",
	"bnb_k3_5m":     "bsc_lottery_log5s",
	"bnb_pk10_5m":   "bsc_lottery_log5s",

	// §5.16 台湾
	"taiwan_ssc_5m": "tw_lottery_logs",
	"taiwan_pk10":   "tw_lottery_logs",
	"taiwan_pc28":   "tw_lottery_logs",
}
