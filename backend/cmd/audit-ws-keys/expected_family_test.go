package main

import (
	"testing"

	"caipiao/backend/internal/guaji/historysync"
)

// 一个彩种的开奖有两条独立映射：REST 历史线（historysync.HistoryAPIPathForCode）
// 和 WS 广播 key（lottery_catalog.guaji_ws_key，期望值在 expectedWSKey）。
// 两条必须落在同一条彩种线上，否则两边都不报错，只是开奖号入错彩种：
// 币安极速赛车的 REST 抄成了波场极速线，写进 99k 条别人的开奖，注单期号永远查不到开奖。
//
// live 版 audit 抓不到这一类：它拿 REST 路径反推出候选 WS key 再比期号，
// 等于用其中一份映射验证它自己，两份不同族反而更容易"对齐"。这条离线断言才是真正的交叉校验。
func TestRestPathAndWSKeySameLine(t *testing.T) {
	for code, want := range expectedWSKey {
		restPath := historysync.HistoryAPIPathForCode(code)
		if restPath == "" {
			continue // 仅 WS 入库，无 REST 线可比
		}
		lineKey, ok := wsKeyByRestPath[restPath]
		if !ok {
			continue // REST 名与 WS 名无对应关系（如 lottery_logs ↔ lottery1_wsds）
		}
		if lineKey != want {
			t.Errorf("%s 两份映射不同族：REST %s 属 %s 线，但 guaji_ws_key 期望 %s",
				code, restPath, lineKey, want)
		}
	}
}

// 反向：REST 有映射的彩种必须在 expectedWSKey 里有期望值，否则新增彩种会绕过上面的交叉校验。
func TestRestMappedCodesHaveExpectedWSKey(t *testing.T) {
	for code := range historyAPIByCodeForTest() {
		if historysync.HistoryAPIPathForCode(code) == "" {
			continue
		}
		if _, ok := expectedWSKey[code]; !ok {
			t.Errorf("%s 配了 REST 历史线但缺 expectedWSKey，两份映射无从校验", code)
		}
	}
}

// historyAPIByCode 在 historysync 包内不导出，这里用已知彩种列表覆盖检查范围。
func historyAPIByCodeForTest() map[string]struct{} {
	codes := map[string]struct{}{}
	for code := range expectedWSKey {
		codes[code] = struct{}{}
	}
	// expectedWSKey 之外、但确实配了 REST 线的彩种也要纳入（缺 expectedWSKey 时报错）
	for _, code := range []string{
		"hash_jisu", "tron_k3_jisu", "tron_pk10_jisu", "bnb_pk10_jisu",
		"hash_ffc_1m", "hash_ffc_3m", "hash_ffc_5m",
		"tron_jisu", "tron_ffc_1m", "tron_ffc_3m", "tron_ffc_5m",
		"tron_ffc_3s", "tron_ffc_6s", "tron_ffc_15s",
		"tron_k3_1m", "tron_k3_3m", "tron_k3_5m",
		"tron_syxw", "tron_syxw_3m", "tron_syxw_5m",
		"tron_lhc_1m", "tron_lhc_3m", "tron_lhc_5m", "tron_lhc",
		"bnb_ffc_1m", "bnb_ffc_3m", "bnb_ffc_5m",
		"bnb_k3_1m", "bnb_k3_3m", "bnb_k3_5m",
		"bnb_syxw", "bnb_syxw_3m", "bnb_syxw_5m",
		"bnb_pk10_5m",
	} {
		codes[code] = struct{}{}
	}
	return codes
}
