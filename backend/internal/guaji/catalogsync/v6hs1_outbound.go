package catalogsync

// V6hs1OutboundByCode：正式环境 www.v6hs1.com GET /api/games/new_lott 实测 game_id（2026-06-30）。
// 保留在售 33 彩种；以太坊系列已 maintenance 下架（迁移 00110）。
var V6hs1OutboundByCode = map[string]int{
	"hash_ffc_1m":    25,
	"hash_ffc_3m":    26,
	"hash_ffc_5m":    27,
	"hash_jisu":      23,
	"tron_jisu":      24,
	"tron_ffc_1m":    19,
	"tron_ffc_3m":    20,
	"tron_ffc_5m":    21,
	"bnb_ffc_1m":     39,
	"bnb_ffc_3m":     40,
	"bnb_ffc_5m":     41,
	"tron_syxw":      42,
	"tron_syxw_3m":   43,
	"tron_syxw_5m":   44,
	"bnb_syxw":       48,
	"bnb_syxw_3m":    49,
	"bnb_syxw_5m":    50,
	"bnb_pk10_jisu":  53,
	"bnb_pk10_5m":    54,
	"tron_pk10_jisu": 55,
	"tron_k3_jisu":   59,
	"tron_k3_1m":     60,
	"tron_k3_3m":     61,
	"tron_k3_5m":     62,
	"bnb_k3_1m":      63,
	"bnb_k3_3m":      64,
	"bnb_k3_5m":      65,
	"tron_ffc_3s":    73,
	"tron_ffc_6s":    74,
	"tron_ffc_15s":   75,
	"tron_lhc_1m":    76,
	"tron_lhc_3m":    77,
	"tron_lhc_5m":    78,
}

// v6hs1RemoteNames 正式环境 new_lott 名称（与上表 id 一一对应）。
var v6hs1RemoteNames = map[int]string{
	19: "哈希一分彩", 20: "哈希三分彩", 21: "哈希五分彩",
	23: "哈希极速彩", 24: "波场极速彩",
	25: "波场一分彩", 26: "波场三分彩", 27: "波场五分彩",
	39: "币安一分彩", 40: "币安三分彩", 41: "币安五分彩",
	42: "波场11选5", 43: "波场3分11选5", 44: "波场5分11选5",
	48: "币安11选5", 49: "币安3分11选5", 50: "币安5分11选5",
	53: "币安极速飞艇", 54: "币安5分飞艇", 55: "波场极速赛车",
	59: "波场极速快三", 60: "波场1分快三", 61: "波场3分快三", 62: "波场5分快三",
	63: "币安1分快三", 64: "币安3分快三", 65: "币安5分快三",
	73: "波场3秒彩", 74: "波场6秒彩", 75: "波场15秒彩",
	76: "波场1分六合彩", 77: "波场3分六合彩", 78: "波场5分六合彩",
}

// V6hs1RemoteName 返回正式环境第三方彩种展示名（按 game_id）。
func V6hs1RemoteName(gameID int) string {
	return v6hs1RemoteNames[gameID]
}

// V6hs1RemoteLotteries 返回正式环境已对接彩种列表（单元测试 / catalog audit）。
func V6hs1RemoteLotteries() []RemoteLottery {
	out := make([]RemoteLottery, 0, len(v6hs1RemoteNames))
	for id, name := range v6hs1RemoteNames {
		out = append(out, RemoteLottery{ID: id, Name: name})
	}
	return out
}

// V6hs1OffSaleCodes 正式环境无对应、平台 maintenance 下架的彩种 code。
var V6hs1OffSaleCodes = []string{
	"eth_ffc_1m", "eth_ffc_3m", "eth_ffc_5m",
	"eth_ffc_new", "eth_jisu",
	"eth_syxw", "eth_syxw_3m", "eth_syxw_5m",
	"eth_pk10_jisu", "eth_pk10_5m",
	"eth_k3", "eth_k3_3m", "eth_k3_5m",
}
