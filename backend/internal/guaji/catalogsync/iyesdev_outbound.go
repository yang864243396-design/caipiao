package catalogsync

// IyesDevOutboundByCode：hash.iyes.dev GET /api/games/new_lott 实测 game_id（2026-06-21）。
// 与文档 §8 平台彩种序号（00087）不同；迁移 00098 写入 DB。
var IyesDevOutboundByCode = map[string]int{
	"hash_ffc_1m":    27,
	"hash_ffc_3m":    28,
	"hash_ffc_5m":    29,
	"hash_jisu":      25,
	"tron_jisu":      26,
	"tron_ffc_1m":    21,
	"tron_ffc_3m":    22,
	"tron_ffc_5m":    23,
	"eth_jisu":       37,
	"eth_ffc_1m":     38,
	"eth_ffc_3m":     39,
	"eth_ffc_5m":     40,
	"bnb_ffc_1m":     41,
	"bnb_ffc_3m":     42,
	"bnb_ffc_5m":     43,
	"tron_syxw":      44,
	"tron_syxw_3m":   45,
	"tron_syxw_5m":   46,
	"eth_syxw":       47,
	"eth_syxw_3m":    48,
	"eth_syxw_5m":    49,
	"bnb_syxw":       50,
	"bnb_syxw_3m":    51,
	"bnb_syxw_5m":    52,
	"eth_pk10_jisu":  53,
	"eth_pk10_5m":    54,
	"bnb_pk10_jisu":  55,
	"bnb_pk10_5m":    56,
	"tron_pk10_jisu": 57,
	"eth_k3":         58,
	"eth_k3_3m":      59,
	"eth_k3_5m":      60,
	"tron_k3_jisu":   61,
	"tron_k3_1m":     62,
	"tron_k3_3m":     63,
	"tron_k3_5m":     64,
	"bnb_k3_1m":      65,
	"bnb_k3_3m":      66,
	"bnb_k3_5m":      67,
	"eth_ffc_new":    68,
	"tron_ffc_3s":    75,
	"tron_ffc_6s":    76,
	"tron_ffc_15s":   77,
	"tron_lhc_1m":    78,
	"tron_lhc_3m":    79,
	"tron_lhc_5m":    80,
}

// iyesDevRemoteNames 第三方 new_lott 名称（与上表 id 一一对应）。
var iyesDevRemoteNames = map[int]string{
	21: "哈希一分彩", 22: "哈希三分彩", 23: "哈希五分彩",
	25: "哈希极速彩", 26: "波场极速彩",
	27: "波场一分彩", 28: "波场三分彩", 29: "波场五分彩",
	37: "以太坊极速彩", 38: "以太坊一分彩", 39: "以太坊三分彩", 40: "以太坊五分彩",
	41: "币安一分彩", 42: "币安三分彩", 43: "币安五分彩",
	44: "波场11选5", 45: "波场3分11选5", 46: "波场5分11选5",
	47: "以太坊11选5", 48: "以太坊3分11选5", 49: "以太坊5分11选5",
	50: "币安11选5", 51: "币安3分11选5", 52: "币安5分11选5",
	53: "以太极速赛车", 54: "以太5分赛车", 55: "币安极速飞艇", 56: "币安5分飞艇", 57: "波场极速赛车",
	58: "以太坊快三", 59: "以太坊3分快三", 60: "以太坊5分快三",
	61: "波场极速快三", 62: "波场1分快三", 63: "波场3分快三", 64: "波场5分快三",
	65: "币安1分快三", 66: "币安3分快三", 67: "币安5分快三",
	68: "新以太坊分分彩",
	75: "波场3秒彩", 76: "波场6秒彩", 77: "波场15秒彩",
	78: "波场1分六合彩", 79: "波场3分六合彩", 80: "波场5分六合彩",
}

// IyesDevRemoteName 返回 hash.iyes.dev 第三方彩种展示名（按 game_id）。
func IyesDevRemoteName(gameID int) string {
	return iyesDevRemoteNames[gameID]
}

// IyesDevRemoteLotteries 返回用于单元测试的第三方彩种列表（全量已对接条目）。
func IyesDevRemoteLotteries() []RemoteLottery {
	out := make([]RemoteLottery, 0, len(iyesDevRemoteNames))
	for id, name := range iyesDevRemoteNames {
		out = append(out, RemoteLottery{ID: id, Name: name})
	}
	return out
}

// DocOutboundByCode 文档 §8 平台彩种序号（00087），仅作对照；iyes.dev 环境勿直接使用。
var DocOutboundByCode = map[string]int{
	"tron_ffc_1m": 1, "tron_ffc_3m": 2, "tron_ffc_5m": 3,
	"hash_ffc_1m": 4, "hash_ffc_3m": 5, "hash_ffc_5m": 6,
	"eth_ffc_1m": 7, "eth_ffc_3m": 8, "eth_ffc_5m": 9,
	"bnb_ffc_1m": 10, "bnb_ffc_3m": 11, "bnb_ffc_5m": 12,
	"eth_ffc_new": 13, "tron_jisu": 14, "hash_jisu": 15, "eth_jisu": 16,
	"tron_lhc_1m": 17, "tron_lhc_3m": 18, "tron_lhc_5m": 19,
	"tron_syxw": 21, "tron_syxw_3m": 22, "tron_syxw_5m": 23,
	"eth_syxw": 24, "eth_syxw_3m": 25, "eth_syxw_5m": 26,
	"bnb_syxw": 27, "bnb_syxw_3m": 28, "bnb_syxw_5m": 29,
	"eth_pk10_jisu": 30, "eth_pk10_5m": 31, "bnb_pk10_jisu": 32, "bnb_pk10_5m": 33,
	"tron_pk10_jisu": 34,
	"eth_k3": 36, "eth_k3_3m": 37, "eth_k3_5m": 38,
	"tron_k3_jisu": 39, "tron_k3_1m": 40, "tron_k3_3m": 41, "tron_k3_5m": 42,
	"bnb_k3_1m": 43, "bnb_k3_3m": 44, 	"bnb_k3_5m": 45,
}
