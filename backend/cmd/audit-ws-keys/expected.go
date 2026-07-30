package main

import "strings"

// expectedWSKey 与 migrations/00123_fix_guaji_ws_keys_p0_p1.sql 及 historysync 映射一致。
var expectedWSKey = map[string]string{
	"hash_jisu":      "lottery_log033",
	"tron_k3_jisu":   "lottery_log033",
	"tron_pk10_jisu": "lottery_log033",

	"hash_ffc_1m": "lottery_log103",
	"hash_ffc_3m": "lottery_log303",
	"hash_ffc_5m": "lottery_log503",
	"tron_jisu":   "lottery_log05",
	"bnb_ffc_1m":  "bsc_lottery_log01",
	"bnb_k3_1m":   "bsc_lottery_log01",
	"bnb_syxw":    "bsc_lottery_log01",
	// 名字带"极速"但实测与 bnb_ffc_1m 同期号，属币安 1 分钟线（00138/00139）
	"bnb_pk10_jisu": "bsc_lottery_log01",

	// 波场分分彩 00 区块（独立 type）；03 线 lottery_log103/303/503 见 hash / 衍生
	"tron_ffc_1m": "lottery1_wsds",
	"tron_ffc_3m": "lottery3_wsds",
	"tron_ffc_5m": "lottery5_wsds",
	// 波场秒彩
	"tron_ffc_3s":  "lottery_v2_broadcast",
	"tron_ffc_6s":  "lottery_log101",
	"tron_ffc_15s": "lottery_log125",

	"tron_k3_1m":   "lottery_log103",
	"tron_lhc_1m":  "lottery_log103",
	"tron_syxw":    "lottery_log103",
	"tron_k3_3m":   "lottery_log303",
	"tron_lhc_3m":  "lottery_log303",
	"tron_syxw_3m": "lottery_log303",
	"tron_k3_5m":   "lottery_log503",
	"tron_lhc_5m":  "lottery_log503",
	"tron_syxw_5m": "lottery_log503",
	"tron_lhc":     "lottery_log503",

	"bnb_ffc_3m":  "bsc_lottery_log03",
	"bnb_k3_3m":   "bsc_lottery_log03",
	"bnb_syxw_3m": "bsc_lottery_log03",
	"bnb_ffc_5m":  "bsc_lottery_log05",
	"bnb_k3_5m":   "bsc_lottery_log05",
	"bnb_syxw_5m": "bsc_lottery_log05",
	"bnb_pk10_5m": "bsc_lottery_log05",
}

// knownPending 尚无可靠 WS 或未配置 key；live 审计跳过，不记 FAIL。
var knownPending = map[string]string{}

// wsKeyByRestPath：REST 历史线 → 同一条线的 WS 广播 key。
//
// 两份映射（historysync 的 REST 路径、lottery_catalog.guaji_ws_key）指向同一条彩种线，
// 但分开维护，抄错一处也不报错——币安极速赛车 REST 抄了波场极速线，99k 条开奖入错彩种、
// 101 笔注单全 cancel，没人发现。有了这张表，两份映射就能互相校验（见 expected_family_test.go）。
// 只列出 REST 与 WS 同名的线；tron_ffc_1m/3m/5m 那几条 REST 名与 WS 名无对应关系，留空跳过。
var wsKeyByRestPath = map[string]string{
	"lottery_log033s":   "lottery_log033",
	"lottery_log05s":    "lottery_log05",
	"lottery_log103s":   "lottery_log103",
	"lottery_log303s":   "lottery_log303",
	"lottery_log503s":   "lottery_log503",
	"lottery_log101s":   "lottery_log101",
	"lottery_log125s":   "lottery_log125",
	"bsc_lottery_logs":  "bsc_lottery_log01",
	"bsc_lottery_log3s": "bsc_lottery_log03",
	"bsc_lottery_log5s": "bsc_lottery_log05",
}

func wsKeyCandidates(wsKey, restPath string) []string {
	seen := map[string]bool{}
	add := func(k string) {
		k = strings.TrimSpace(k)
		if k != "" {
			seen[k] = true
		}
	}
	add(wsKey)
	if alt := strings.TrimSuffix(wsKey, "s"); alt != wsKey {
		add(alt)
	}
	add(wsKeyByRestPath[restPath])
	out := make([]string, 0, len(seen))
	for k := range seen {
		out = append(out, k)
	}
	return out
}
