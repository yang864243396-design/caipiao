package main

import "strings"

// expectedWSKey 与 migrations/00123_fix_guaji_ws_keys_p0_p1.sql 及 historysync 映射一致。
var expectedWSKey = map[string]string{
	"hash_jisu":      "lottery_log033",
	"tron_k3_jisu":   "lottery_log033",
	"tron_pk10_jisu": "lottery_log033",
	"bnb_pk10_jisu":  "lottery_log033",

	"hash_ffc_1m": "lottery_log103",
	"hash_ffc_3m": "lottery_log303",
	"hash_ffc_5m": "lottery_log503",
	"tron_jisu":   "lottery_log05",
	"bnb_ffc_1m":  "bsc_lottery_log01",
	"bnb_k3_1m":   "bsc_lottery_log01",
	"bnb_syxw":    "bsc_lottery_log01",

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
var knownPending = map[string]string{
	"tron_ffc_1m":  "REST lottery_logs(1014*) WS 无 lottery_logs",
	"tron_ffc_3m":  "REST lottery_log3s(3014*) WS 无对应 key",
	"tron_ffc_5m":  "REST lottery_log5s(5014*) WS 无对应 key",
	"tron_ffc_3s":  "guaji_ws_key 未配置",
	"tron_ffc_6s":  "guaji_ws_key 未配置",
	"tron_ffc_15s": "guaji_ws_key 未配置",
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
	switch restPath {
	case "lottery_log033s":
		add("lottery_log033")
	case "lottery_log05s":
		add("lottery_log05")
	case "lottery_log103s":
		add("lottery_log103")
	case "lottery_log303s":
		add("lottery_log303")
	case "lottery_log503s":
		add("lottery_log503")
	case "bsc_lottery_logs":
		add("bsc_lottery_log01")
	case "bsc_lottery_log3s":
		add("bsc_lottery_log03")
	case "bsc_lottery_log5s":
		add("bsc_lottery_log05")
	}
	out := make([]string, 0, len(seen))
	for k := range seen {
		out = append(out, k)
	}
	return out
}
