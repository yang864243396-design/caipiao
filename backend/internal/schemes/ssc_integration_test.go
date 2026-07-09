package schemes

import (
	"encoding/json"
	"testing"
)

func TestSSCNormalizeAndEvaluate(t *testing.T) {
	cases := []struct {
		name     string
		template string
		typeID   string
		subID    string
		betMode  string
		content  string
		balls    []string
		wantHit  bool
	}{
		{
			name: "ssc_dingwei", template: "ssc_std", typeID: "dingwei", subID: "dingwei_wan",
			betMode: "dingwei", content: "3,9", balls: []string{"3", "9", "2", "7", "5"}, wantHit: true,
		},
		{
			name: "ssc_qian3_fs", template: "ssc_std", typeID: "qian3", subID: "qian3_zhixuan_fs",
			betMode: "fushi", content: "3\n9\n2", balls: []string{"3", "9", "2", "7", "5"}, wantHit: true,
		},
		{
			name: "ssc_qian3_hz", template: "ssc_std", typeID: "qian3", subID: "qian3_zhixuan_hz",
			betMode: "hezhi", content: "14", balls: []string{"3", "9", "2", "7", "5"}, wantHit: true,
		},
		{
			name: "ssc_longhu", template: "ssc_std", typeID: "longhu", subID: "lh_wanqian_dou",
			betMode: "longhu", content: "虎", balls: []string{"3", "9", "2", "7", "5"}, wantHit: true,
		},
		{
			name: "ssc_dxds", template: "ssc_std", typeID: "dxds", subID: "qian2_dxds",
			betMode: "dxds", content: "大", balls: []string{"3", "9", "2", "7", "5"}, wantHit: true,
		},
		{
			name: "ssc_budingwei", template: "ssc_std", typeID: "budingwei", subID: "qian3_2ma",
			betMode: "budingwei", content: "3,9", balls: []string{"3", "9", "2", "7", "5"}, wantHit: true,
		},
		{
			name: "ssc_ren2", template: "ssc_std", typeID: "renxuan", subID: "ren2_zhixuan_fs",
			betMode: "fushi", content: "3\n9\n\n\n\n", balls: []string{"3", "9", "2", "7", "5"}, wantHit: true,
		},
		{
			name: "ssc_qian3_zu3", template: "ssc_std", typeID: "qian3", subID: "qian3_zu3",
			betMode: "zu3", content: "3,2", balls: []string{"3", "3", "2", "7", "5"}, wantHit: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			raw, err := NormalizeBetPayload(BetPayload{
				PlayTemplate: tc.template,
				TypeID:       tc.typeID,
				SubID:        tc.subID,
				BetMode:      tc.betMode,
				GroupContent: tc.content,
			})
			if err != nil {
				t.Fatalf("NormalizeBetPayload: %v", err)
			}
			hit, odds := EvaluateBetPayload(raw, tc.balls)
			if hit != tc.wantHit {
				t.Fatalf("hit=%v want=%v odds=%v payload=%s", hit, tc.wantHit, odds, string(raw))
			}
			if odds <= 0 {
				t.Fatalf("odds=%v", odds)
			}
			var p BetPayload
			_ = json.Unmarshal(raw, &p)
			if p.GroupContent == "" {
				t.Fatal("empty groupContent after normalize")
			}
		})
	}
}

func TestSSCSynthDrawBallCounts(t *testing.T) {
	cases := []struct {
		code string
		want int
	}{
		{"tron_ffc_1m", 5},
		{"eth_jisu", 5},
		{"taiwan_ssc_5m", 5},
	}
	for _, tc := range cases {
		got := synthDrawBalls(tc.code, "20231103099")
		if len(got) != tc.want {
			t.Fatalf("%s: want %d balls, got %d %v", tc.code, tc.want, len(got), got)
		}
	}
}
