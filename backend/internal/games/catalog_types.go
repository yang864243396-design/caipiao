package games

// LegacyLotteryCodes 旧 9 彩种 purge 键（规划 §5.2）。
var LegacyLotteryCodes = []string{
	"tencent_ffc",
	"tencent_10",
	"qiqu_tencent",
	"us_ffc",
	"cq_ssc",
	"xj_ssc",
	"tj_ssc",
	"fc_3d",
	"pl3",
}

const (
	expectedCatalogSeedCount = 46
	expectedSubPlayCount     = 493 // 00111 guaji_prod_rules_v2_sync 后 6 套 play_template 总量
)
