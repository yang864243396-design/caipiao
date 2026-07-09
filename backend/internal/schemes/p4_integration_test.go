package schemes

import (
	"encoding/json"
	"testing"
)

func TestP4NormalizeAndEvaluate(t *testing.T) {
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
			name: "syxw_dingwei", template: "syxw_std", typeID: "dingwei", subID: "dingwei_wan",
			betMode: "dingwei", content: "01,03", balls: []string{"01", "03", "05", "07", "09"}, wantHit: true,
		},
		{
			name: "syxw_renxuan", template: "syxw_std", typeID: "renxuan_fs", subID: "rx_2z2",
			betMode: "fushi", content: "01,03,05", balls: []string{"01", "03", "05", "07", "09"}, wantHit: true,
		},
		{
			name: "pk10_longhu", template: "pk10_std", typeID: "longhu", subID: "lh_1v10",
			betMode: "longhu", content: "虎", balls: []string{"3", "7", "1", "9", "5", "2", "8", "4", "6", "10"}, wantHit: true,
		},
		{
			name: "pk10_qian2", template: "pk10_std", typeID: "qian2", subID: "qian2_zhixuan_fs",
			betMode: "fushi", content: "3\n7", balls: []string{"3", "7", "1", "9", "5", "2", "8", "4", "6", "10"}, wantHit: true,
		},
		{
			name: "k3_hezhi", template: "k3_std", typeID: "hezhi", subID: "k3_hezhi",
			betMode: "hezhi", content: "12", balls: []string{"2", "4", "6"}, wantHit: true,
		},
		{
			name: "k3_dantiao", template: "k3_std", typeID: "lianhao_qita", subID: "dantiao",
			betMode: "dantiao", content: "4,6", balls: []string{"2", "4", "6"}, wantHit: true,
		},
		{
			name: "pc28_dxds", template: "pc28_std", typeID: "pc28_20", subID: "dxds",
			betMode: "dxds", content: "大", balls: []string{"3", "5", "7"}, wantHit: true,
		},
		{
			name: "pc28_teshu", template: "pc28_std", typeID: "pc28_28", subID: "teshu",
			betMode: "teshu", content: "顺子", balls: []string{"1", "2", "3"}, wantHit: true,
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

func TestP4SynthDrawBallCounts(t *testing.T) {
	cases := []struct {
		code string
		want int
	}{
		{"tron_syxw", 5},
		{"eth_pk10_jisu", 10},
		{"eth_k3", 3},
		{"tron_k3_1m", 3},
	}
	for _, tc := range cases {
		balls := synthDrawBalls(tc.code, "20231103099")
		if len(balls) != tc.want {
			t.Fatalf("%s: want %d balls, got %d %v", tc.code, tc.want, len(balls), balls)
		}
	}
}
