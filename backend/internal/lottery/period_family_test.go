package lottery

import "testing"

func TestComparePeriodFamily(t *testing.T) {
	cases := []struct {
		name      string
		bet, draw string
		want      PeriodFamilyStatus
	}{
		{
			// 迁移 00136 之前：hash_jisu 下注取到区块高度族、开奖是日期族，位数就不同
			"极速彩错配：区块高度族 vs 日期族",
			"10514140800529", "105202607281314", PeriodFamilyMismatch,
		},
		{
			"波场极速彩正常：同为区块高度族且相邻",
			"10514140800529", "10514140900560", PeriodFamilyOK,
		},
		{
			"哈希极速彩正常：同为日期族且相邻",
			"105202607281314", "105202607281312", PeriodFamilyOK,
		},
		{
			"时时彩跨日仍算同族",
			"20260729001", "20260728288", PeriodFamilyOK,
		},
		{
			// 已知盲区：同体系内错配位数一致，本函数判不出，靠 audit-ws-keys
			// 与 scheme-audit 的命中率统计兜底。这里钉住现状，免得后人误以为覆盖了
			"已知盲区：同体系内错配判为 ok",
			"105202601281314", "105202607281312", PeriodFamilyOK,
		},
		{"缺开奖期号", "20260728001", "", PeriodFamilyUnknown},
		{"非数字期号", "abc", "20260728001", PeriodFamilyUnknown},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, note := ComparePeriodFamily(tc.bet, tc.draw)
			if got != tc.want {
				t.Fatalf("got %s (%s), want %s", got, note, tc.want)
			}
			if got != PeriodFamilyOK && note == "" {
				t.Fatal("异常结论必须带说明")
			}
		})
	}
}
